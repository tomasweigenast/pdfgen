package syntax

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

// formatNumber renders a coordinate rounded to 6 digits, without a trailing
// ".0", as the shared suite expects.
func formatNumber(v float64) string {
	if math.IsNaN(v) {
		return "none"
	}
	v, _ = strconv.ParseFloat(strconv.FormatFloat(v+0.0000001, 'f', 6, 64), 64)
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	if exp := math.Floor(math.Log10(math.Abs(v))); exp < -4 || exp >= 16 {
		return strconv.FormatFloat(v, 'e', -1, 64)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func rgbString(channels []float64, alpha float64) string {
	parts := make([]string, len(channels))
	for i, c := range channels {
		parts[i] = formatNumber(c * 255)
	}
	s := "rgb"
	if alpha != 1 {
		s += "a"
	}
	s += "(" + strings.Join(parts, ", ")
	if alpha != 1 {
		s += ", " + formatNumber(alpha)
	}
	return s + ")"
}

func spaceString(space string, coords []float64, alpha float64) string {
	parts := make([]string, len(coords))
	for i, c := range coords {
		parts[i] = formatNumber(c)
	}
	s := space + "(" + strings.Join(parts, " ")
	if alpha != 1 {
		s += " / " + formatNumber(alpha)
	}
	return s + ")"
}

func colorFunctionString(c Color) string {
	s := "color(" + c.Space
	for _, v := range c.Coordinates {
		s += " " + formatNumber(v)
	}
	if c.Alpha != 1 {
		s += " / " + formatNumber(c.Alpha)
	}
	return s + ")"
}

type parser func(css string) (Color, bool)

func parse4(css string) (Color, bool) { return ParseColor4String(css) }
func parse5(css string) (Color, bool) { return ParseColorString(css) }

// asRGB renders a parsed color in the rgb() form after checking its space.
func asRGB(t *testing.T, parse parser, space string, convert bool) func(string) any {
	return func(css string) any {
		c, ok := parse(css)
		if !ok {
			return nil
		}
		if c.IsCurrentColor() {
			return CurrentColor
		}
		if c.Space != space {
			t.Errorf("%q: space %s, want %s", css, c.Space, space)
		}
		if convert {
			var ok bool
			if c, ok = c.To(SRGB); !ok {
				t.Fatalf("%q: no conversion to srgb", css)
			}
		}
		return rgbString(c.Coordinates, c.Alpha)
	}
}

func asRGB3(css string) any {
	c, current, ok := ParseColor3String(css)
	switch {
	case !ok:
		return nil
	case current:
		return "currentColor"
	}
	return rgbString([]float64{c.R, c.G, c.B}, c.A)
}

func asSpace(t *testing.T, parse parser, space string) func(string) any {
	return func(css string) any {
		c, ok := parse(css)
		if !ok {
			return nil
		}
		if c.IsCurrentColor() {
			return CurrentColor
		}
		if c.Space != space {
			t.Errorf("%q: space %s, want %s", css, c.Space, space)
		}
		return spaceString(c.Space, c.Coordinates, c.Alpha)
	}
}

func asFunction(parse parser) func(string) any {
	return func(css string) any {
		c, ok := parse(css)
		if !ok {
			return nil
		}
		return colorFunctionString(c)
	}
}

func TestColorSuites(t *testing.T) {
	tests := []struct {
		file string
		fn   func(*testing.T) func(string) any
	}{
		{"color_function_4.json", func(t *testing.T) func(string) any { return asFunction(parse4) }},
		{"color_function_4.json", func(t *testing.T) func(string) any { return asFunction(parse5) }},
		{"color_hexadecimal_3.json", func(t *testing.T) func(string) any { return asRGB3 }},
		{"color_hexadecimal_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, SRGB, false) }},
		{"color_hexadecimal_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, SRGB, false) }},
		{"color_hexadecimal_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, SRGB, false) }},
		{"color_hexadecimal_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, SRGB, false) }},
		{"color_hsl_3.json", func(t *testing.T) func(string) any { return asRGB3 }},
		{"color_hsl_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, HSL, true) }},
		{"color_hsl_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, HSL, true) }},
		{"color_hsl_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, HSL, true) }},
		{"color_hsl_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, HSL, true) }},
		{"color_hwb_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, HWB, true) }},
		{"color_hwb_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, HWB, true) }},
		{"color_keywords_3.json", func(t *testing.T) func(string) any { return asRGB3 }},
		{"color_keywords_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, SRGB, false) }},
		{"color_keywords_3.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, SRGB, false) }},
		{"color_keywords_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse4, SRGB, false) }},
		{"color_keywords_4.json", func(t *testing.T) func(string) any { return asRGB(t, parse5, SRGB, false) }},
	}
	for _, space := range []string{Lab, Oklab, LCH, Oklch} {
		for _, p := range []parser{parse4, parse5} {
			space, p := space, p
			tests = append(tests, struct {
				file string
				fn   func(*testing.T) func(string) any
			}{"color_" + space + "_4.json", func(t *testing.T) func(string) any { return asSpace(t, p, space) }})
		}
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%s/%d", tc.file, i), func(t *testing.T) {
			runSuite(t, tc.file, tc.fn(t))
		})
	}
}

func TestLightDarkAndDeviceColors(t *testing.T) {
	runSuite(t, "color_functions_5.json", func(css string) any {
		if strings.HasPrefix(css, "light-dark") {
			var out []any
			for _, scheme := range []string{"light", "dark"} {
				c, ok := ParseColorString(css, scheme)
				if !ok {
					out = append(out, nil)
					continue
				}
				out = append(out, colorFunctionString(c))
			}
			return out
		}
		c, ok := ParseColorString(css)
		if !ok {
			return nil
		}
		return colorFunctionString(c)
	})
}

func TestCurrentColorKeywordIsCaseInsensitive(t *testing.T) {
	for _, css := range []string{"currentcolor", "currentColor", "CURRENTCOLOR"} {
		if _, current, ok := ParseColor3String(css); !ok || !current {
			t.Errorf("level 3 %q", css)
		}
		if c, ok := ParseColor4String(css); !ok || !c.IsCurrentColor() {
			t.Errorf("level 4 %q", css)
		}
		if c, ok := ParseColorString(css); !ok || !c.IsCurrentColor() {
			t.Errorf("level 5 %q", css)
		}
	}
}

func TestColorFromDeclarationValue(t *testing.T) {
	decl := ParseOneDeclaration("color:#369", false).(*Declaration)
	token := decl.Value[0]
	want := []float64{0.2, 0.4, 0.6, 1}
	rgba, _, ok := ParseColor3(token)
	if !ok || rgba != (RGBA{want[0], want[1], want[2], want[3]}) {
		t.Errorf("level 3: %v", rgba)
	}
	c, ok := ParseColor4(token)
	if !ok {
		t.Fatal("level 4: invalid")
	}
	for i, v := range c.Values() {
		if math.Abs(v-want[i]) > 1e-12 {
			t.Errorf("level 4 value %d = %v, want %v", i, v, want[i])
		}
	}
}

func TestColorConversionReportsUnsupportedPairs(t *testing.T) {
	c, _ := ParseColor4String("lab(50 10 10)")
	if _, ok := c.To(SRGB); ok {
		t.Error("lab to srgb should be unsupported")
	}
	if _, ok := c.To(XYZD50); !ok {
		t.Error("lab to xyz-d50 should be supported")
	}
}
