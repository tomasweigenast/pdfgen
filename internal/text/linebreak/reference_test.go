//go:build metricdiff

package linebreak

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// The reference file holds line breaking results produced by another
// implementation for a corpus of cases. Run with:
//
//	LINEBREAK_REFERENCE=ref.json go test -tags metricdiff -run TestReference ./internal/text/linebreak/
type referenceFile struct {
	Fonts   map[string]string `json:"fonts"`
	Results []referenceResult `json:"results"`
}

type referenceCase struct {
	ID        int      `json:"id"`
	Tag       string   `json:"tag"`
	Text      string   `json:"text"`
	CSS       string   `json:"css"`
	Lang      string   `json:"lang"`
	Width     *float64 `json:"width"`
	Mode      string   `json:"mode"`
	LineStart bool     `json:"line_start"`
	Minimum   bool     `json:"minimum"`
}

type referenceLine struct {
	Text     string  `json:"text"`
	Length   int     `json:"length"`
	Resume   *int    `json:"resume"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Baseline float64 `json:"baseline"`
	Runs     any     `json:"runs"`
}

type referenceResult struct {
	Case   referenceCase              `json:"case"`
	Style  map[string]json.RawMessage `json:"style"`
	Strut  []float64                  `json:"strut"`
	Lines  []referenceLine            `json:"lines"`
	Breaks *struct {
		CanBreak       bool  `json:"can_break"`
		NextBreakPoint *int  `json:"next_break_point"`
		WordBoundaries []int `json:"word_boundaries"`
		LastWordEnd    *int  `json:"last_word_end"`
	} `json:"breaks"`
	Error string `json:"error"`
}

func referenceStyle(t *testing.T, raw map[string]json.RawMessage) *Style {
	t.Helper()
	get := func(key string, v any) {
		if err := json.Unmarshal(raw[key], v); err != nil {
			t.Fatalf("style %s: %v", key, err)
		}
	}
	var size float64
	get("font_size", &size)
	var families []string
	get("font_family", &families)
	s := NewStyle(size, families...)
	var ls any
	get("letter_spacing", &ls)
	if f, ok := ls.(float64); ok {
		s.LetterSpacing = f
	}
	get("word_spacing", &s.WordSpacing)
	get("white_space", &s.WhiteSpace)
	get("word_break", &s.WordBreak)
	get("overflow_wrap", &s.OverflowWrap)
	get("hyphens", &s.Hyphens)
	get("hyphenate_character", &s.HyphenateCharacter)
	var limits []int
	get("hyphenate_limit_chars", &limits)
	s.HyphenateLimitChars = HyphenateLimitChars{limits[0], limits[1], limits[2]}
	var lang *string
	get("lang", &lang)
	if lang != nil {
		s.Lang = *lang
	}
	get("direction", &s.Direction)
	get("font_weight", &s.Font.Weight)
	get("font_style", &s.Font.Style)
	get("font_stretch", &s.Font.Stretch)
	var kerning string
	get("font_kerning", &kerning)
	s.Features.Kerning = props.Keyword(kerning)

	type dim struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	var lh any
	get("line_height", &lh)
	switch v := lh.(type) {
	case float64:
		s.LineHeight = LineHeight{Number: true, Value: v}
	case map[string]any:
		s.LineHeight = LineHeight{Value: v["value"].(float64)}
	}
	var zone dim
	get("hyphenate_limit_zone", &zone)
	s.HyphenateLimitZone = LengthPercentage{Value: zone.Value, Percent: zone.Unit == "%"}
	var tab any
	get("tab_size", &tab)
	switch v := tab.(type) {
	case float64:
		s.TabSize = TabSize{Spaces: int(v)}
	case map[string]any:
		s.TabSize = TabSize{Length: v["value"].(float64), IsLength: true}
	}
	return s
}

func referenceRuns(raw any) (advances, offsets []int32) {
	runs, ok := raw.([]any)
	if !ok {
		return nil, nil
	}
	for _, r := range runs {
		m := r.(map[string]any)
		for _, a := range m["advances"].([]any) {
			advances = append(advances, int32(a.(float64)))
		}
		for _, a := range m["x_offsets"].([]any) {
			offsets = append(offsets, int32(a.(float64)))
		}
	}
	return advances, offsets
}

func layoutRuns(l *Layout) (advances, offsets []int32) {
	l.Reactivate(l.Style)
	for _, r := range l.FirstLine().Runs {
		for _, g := range r.Glyphs {
			advances = append(advances, g.XAdvance)
			offsets = append(offsets, g.XOffset)
		}
	}
	return advances, offsets
}

type lineResult struct {
	text                    string
	length, resume          int
	width, height, baseline float64
	advances, offsets       []int32
}

func runCase(ctx *Context, c referenceCase, style *Style) []lineResult {
	var out []lineResult
	record := func(r SplitResult, text string) {
		adv, off := layoutRuns(r.Layout)
		out = append(out, lineResult{r.Layout.Text, r.Length, r.ResumeIndex, r.Width, r.Height, r.Baseline, adv, off})
	}
	if c.Mode == "split" {
		record(SplitFirstLine(ctx, c.Text, style, c.Width, 0, c.LineStart, c.Minimum), c.Text)
		return out
	}
	skip := 0
	for len(out) < 200 {
		part := c.Text[skip:]
		if part == "" {
			break
		}
		r := SplitFirstLine(ctx, part, style, c.Width, 0, c.LineStart, c.Minimum)
		record(r, part)
		if r.ResumeIndex < 0 {
			break
		}
		skip += r.ResumeIndex
	}
	return out
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestReference(t *testing.T) {
	path := os.Getenv("LINEBREAK_REFERENCE")
	if path == "" {
		t.Skip("LINEBREAK_REFERENCE is not set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ref referenceFile
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	fc := fonts.NewFontConfiguration()
	for family, file := range ref.Fonts {
		rule := fonts.FontFaceRule{Family: family, Src: []fonts.Source{{Value: file}}}
		if err := fc.AddFontFace(rule, os.ReadFile); err != nil {
			t.Fatalf("font %s: %v", family, err)
		}
	}
	ctx := NewContext(fc)

	type tally struct{ total, breaks, metrics, glyphs int }
	byTag := map[string]*tally{}
	var failures []string
	only := os.Getenv("LINEBREAK_CASE")
	for _, res := range ref.Results {
		c := res.Case
		if only != "" && fmt.Sprint(c.ID) != only {
			continue
		}
		tl := byTag[c.Tag]
		if tl == nil {
			tl = &tally{}
			byTag[c.Tag] = tl
		}
		tl.total++
		if res.Error != "" {
			continue
		}
		if c.Mode == "breaks" {
			if compareBreaks(c, res, &failures) {
				tl.breaks++
				tl.metrics++
				tl.glyphs++
			}
			continue
		}
		style := referenceStyle(t, res.Style)
		got := runCase(ctx, c, style)
		breaksOK := len(got) == len(res.Lines)
		metricsOK, glyphsOK := breaksOK, breaksOK
		var diff []string
		for i := 0; i < min(len(got), len(res.Lines)); i++ {
			g, w := got[i], res.Lines[i]
			wantResume := -1
			if w.Resume != nil {
				wantResume = *w.Resume
			}
			if g.text != w.Text || g.length != w.Length || g.resume != wantResume {
				breaksOK = false
				diff = append(diff, fmt.Sprintf("line %d: got %q len %d resume %d, want %q len %d resume %d",
					i, g.text, g.length, g.resume, w.Text, w.Length, wantResume))
			}
			if !near(g.width, w.Width) || !near(g.height, w.Height) || !near(g.baseline, w.Baseline) {
				metricsOK = false
				diff = append(diff, fmt.Sprintf("line %d: got %vx%v (%v), want %vx%v (%v)",
					i, g.width, g.height, g.baseline, w.Width, w.Height, w.Baseline))
			}
			wa, wo := referenceRuns(w.Runs)
			if fmt.Sprint(wa) != fmt.Sprint(g.advances) || fmt.Sprint(wo) != fmt.Sprint(g.offsets) {
				glyphsOK = false
				diff = append(diff, fmt.Sprintf("line %d glyphs: got %v %v, want %v %v", i, g.advances, g.offsets, wa, wo))
			}
		}
		if len(got) != len(res.Lines) {
			diff = append(diff, fmt.Sprintf("got %d lines, want %d", len(got), len(res.Lines)))
			for i := range got {
				diff = append(diff, fmt.Sprintf("  got %d: %q", i, got[i].text))
			}
			for i := range res.Lines {
				diff = append(diff, fmt.Sprintf("  want %d: %q", i, res.Lines[i].Text))
			}
		}
		metricsOK = metricsOK && breaksOK
		glyphsOK = glyphsOK && metricsOK
		if breaksOK {
			tl.breaks++
		}
		if metricsOK {
			tl.metrics++
		}
		if glyphsOK {
			tl.glyphs++
		}
		if len(diff) > 0 {
			failures = append(failures, fmt.Sprintf("case %d [%s] %q css=%q width=%v:\n    %s",
				c.ID, c.Tag, c.Text, c.CSS, ptr(c.Width), strings.Join(diff, "\n    ")))
		}
	}

	tags := make([]string, 0, len(byTag))
	for tag := range byTag {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	var sum tally
	for _, tag := range tags {
		tl := byTag[tag]
		t.Logf("%-20s %4d cases: breaks %4d, +metrics %4d, +glyphs %4d", tag, tl.total, tl.breaks, tl.metrics, tl.glyphs)
		sum.total += tl.total
		sum.breaks += tl.breaks
		sum.metrics += tl.metrics
		sum.glyphs += tl.glyphs
	}
	t.Logf("%-20s %4d cases: breaks %4d, +metrics %4d, +glyphs %4d", "total", sum.total, sum.breaks, sum.metrics, sum.glyphs)
	limit := 40
	if s := os.Getenv("LINEBREAK_FAILURES"); s != "" {
		fmt.Sscan(s, &limit)
	}
	for i, f := range failures {
		if i >= limit {
			break
		}
		t.Log(f)
	}
}

func ptr(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func compareBreaks(c referenceCase, res referenceResult, failures *[]string) bool {
	b := res.Breaks
	runes := []rune(c.Text)
	can := CanBreakText(c.Text, c.Lang)
	next := nextBreakPointInText(runes, c.Lang)
	start, end, ok := nextWordBoundaries(runes, c.Lang)
	last := -1
	if e, ok := LastWordEnd(c.Text, c.Lang); ok {
		last = len([]rune(c.Text[:e]))
	}
	wantNext := -1
	if b.NextBreakPoint != nil {
		wantNext = *b.NextBreakPoint
	}
	wantLast := -1
	if b.LastWordEnd != nil {
		wantLast = *b.LastWordEnd
	}
	gotWords := []int(nil)
	if ok {
		gotWords = []int{start, end}
	}
	if can == b.CanBreak && next == wantNext && fmt.Sprint(gotWords) == fmt.Sprint(b.WordBoundaries) && last == wantLast {
		return true
	}
	*failures = append(*failures, fmt.Sprintf("case %d [breaks] %q %s: got %v %d %v %d, want %v %d %v %d",
		c.ID, c.Text, c.Lang, can, next, gotWords, last, b.CanBreak, wantNext, b.WordBoundaries, wantLast))
	return false
}
