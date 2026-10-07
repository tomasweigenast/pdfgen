package validation

import (
	"log/slog"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

const testBaseURL = "https://example.org/foo/"

// getValue preprocesses the declarations in css, which must give at most
// one longhand, and returns its value. With a non-empty wantLog exactly one
// entry must have been logged and contain it; otherwise nothing is logged.
func getValue(t *testing.T, css, wantLog string) props.Value {
	t.Helper()
	return getValueLevel(t, css, wantLog, slog.LevelInfo)
}

func getValueLevel(t *testing.T, css, wantLog string, level slog.Level) props.Value {
	t.Helper()
	declarations := syntax.ParseBlocksContents(css, false, false)
	capture := logging.StartCaptureLevel(level)
	result := PreprocessDeclarations(testBaseURL, declarations)
	capture.Close()
	logs := capture.Lines()

	if wantLog != "" {
		if len(logs) != 1 || !strings.Contains(logs[0], wantLog) {
			t.Fatalf("%q: logs = %q, want one containing %q", css, logs, wantLog)
		}
	} else if len(logs) != 0 {
		t.Fatalf("%q: unexpected logs %q", css, logs)
	}
	if len(result) == 0 {
		return nil
	}
	if len(result) != 1 {
		t.Fatalf("%q: got %d declarations, want 1", css, len(result))
	}
	return result[0].Value
}

// assertInvalid checks that the declaration is ignored with a warning
// containing message.
func assertInvalid(t *testing.T, css string, message ...string) {
	t.Helper()
	want := "invalid"
	if len(message) > 0 {
		want = message[0]
	}
	if v := getValue(t, css, want); v != nil {
		t.Errorf("%q: got value %#v, want it ignored", css, v)
	}
}

func assertValue(t *testing.T, css string, want props.Value) {
	t.Helper()
	got := getValue(t, css, "")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s\n got  %#v\n want %#v", css, got, want)
	}
}

// expandToMap preprocesses the declarations in css and returns their values
// by longhand name.
func expandToMap(t *testing.T, css string) map[string]props.Value {
	t.Helper()
	capture := logging.StartCapture()
	result := PreprocessDeclarations(testBaseURL, syntax.ParseBlocksContents(css, false, false))
	capture.Close()
	if logs := capture.Lines(); len(logs) != 0 {
		t.Fatalf("%q: unexpected logs %q", css, logs)
	}
	m := map[string]props.Value{}
	for _, d := range result {
		m[d.Name] = d.Value
	}
	return m
}

// Constructors of expected values.

func dim(v float64, unit string) props.Length {
	return props.Length{Dimension: props.Dimension{Value: v, Unit: unit}}
}

func num(v float64) props.Length { return dim(v, "") }
func pct(v float64) props.Length { return dim(v, "%") }
func px(v float64) props.Length  { return props.Px(v) }
func kw(s string) props.Length   { return props.LengthKeyword(props.Keyword(s)) }

func kws(s ...string) props.Keywords {
	out := make(props.Keywords, len(s))
	for i, k := range s {
		out[i] = props.Keyword(k)
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func rgb(r, g, b, a float64) props.Color {
	return props.Color{Color: syntax.NewColor(syntax.SRGB, []float64{r, g, b}, a)}
}
