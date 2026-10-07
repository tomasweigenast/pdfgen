package fonts

import (
	"github.com/go-text/typesetting/font"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// Description selects a font: the computed font-family, font-weight,
// font-style, font-stretch, font-size and font-variation-settings of an
// element.
type Description struct {
	Families []string
	// Weight is 1 to 1000. Zero means 400.
	Weight int
	// Style is "normal", "italic" or "oblique". Empty means normal.
	Style string
	// Stretch is a font-stretch keyword. Empty means normal.
	Stretch string
	// Size is the font size in px.
	Size float64
	// Variations are the font-variation-settings.
	Variations props.FontSettings
}

var stretchRatio = map[string]font.Stretch{
	"ultra-condensed": font.StretchUltraCondensed,
	"extra-condensed": font.StretchExtraCondensed,
	"condensed":       font.StretchCondensed,
	"semi-condensed":  font.StretchSemiCondensed,
	"normal":          font.StretchNormal,
	"semi-expanded":   font.StretchSemiExpanded,
	"expanded":        font.StretchExpanded,
	"extra-expanded":  font.StretchExtraExpanded,
	"ultra-expanded":  font.StretchUltraExpanded,
}

func stretchOf(keyword string) font.Stretch {
	if s, ok := stretchRatio[keyword]; ok {
		return s
	}
	return font.StretchNormal
}

func (d Description) weight() int {
	if d.Weight == 0 {
		return 400
	}
	return d.Weight
}

// aspect converts the description to the style, weight and stretch used to
// query system fonts. Oblique is matched as italic, as the only slanted
// style fontscan knows.
func (d Description) aspect() font.Aspect {
	a := font.Aspect{Style: font.StyleNormal, Weight: font.Weight(d.weight()), Stretch: stretchOf(d.Stretch)}
	if d.Style == "italic" || d.Style == "oblique" {
		a.Style = font.StyleItalic
	}
	return a
}

// faceAspect is the style, weight and stretch a registered face is matched
// with.
type faceAspect struct {
	style   string
	weight  int
	stretch font.Stretch
}

// bestMatch picks the registered face closest to a description, following
// the font matching algorithm of CSS Fonts 4 §5.2: stretch first, then
// style, then weight. Among equal faces the one declared last wins.
func bestMatch(candidates []*registered, d Description) *registered {
	if len(candidates) == 0 {
		return nil
	}
	want := stretchOf(d.Stretch)
	set := filterNearest(candidates, float64(want),
		func(r *registered) float64 { return float64(r.aspect.stretch) }, want <= font.StretchNormal)
	set = filterStyle(set, d.Style)
	set = filterWeight(set, d.weight())
	return set[len(set)-1]
}

// filterNearest keeps the candidates closest to want on the preferred side
// of it, or on the other side when the preferred one is empty. A candidate
// at want belongs to both sides.
func filterNearest(set []*registered, want float64, value func(*registered) float64, preferLower bool) []*registered {
	for _, lowerSide := range []bool{preferLower, !preferLower} {
		var best []*registered
		var bestValue float64
		for _, r := range set {
			v := value(r)
			if lowerSide && v > want || !lowerSide && v < want {
				continue
			}
			switch {
			case best == nil || lowerSide && v > bestValue || !lowerSide && v < bestValue:
				best, bestValue = []*registered{r}, v
			case v == bestValue:
				best = append(best, r)
			}
		}
		if best != nil {
			return best
		}
	}
	return set
}

func filterStyle(set []*registered, style string) []*registered {
	order := []string{"normal", "oblique", "italic"}
	switch style {
	case "italic":
		order = []string{"italic", "oblique", "normal"}
	case "oblique":
		order = []string{"oblique", "italic", "normal"}
	}
	for _, s := range order {
		var out []*registered
		for _, r := range set {
			if r.aspect.style == s {
				out = append(out, r)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return set
}

func filterWeight(set []*registered, want int) []*registered {
	exact := func(w int) []*registered {
		var out []*registered
		for _, r := range set {
			if r.aspect.weight == w {
				out = append(out, r)
			}
		}
		return out
	}
	if out := exact(want); len(out) > 0 {
		return out
	}
	// The weights 400 and 500 look for each other before moving away.
	switch want {
	case 400:
		if out := exact(500); len(out) > 0 {
			return out
		}
	case 500:
		if out := exact(400); len(out) > 0 {
			return out
		}
	}
	return filterNearest(set, float64(want),
		func(r *registered) float64 { return float64(r.aspect.weight) }, want <= 500)
}
