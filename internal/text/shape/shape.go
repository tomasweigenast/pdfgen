package shape

import (
	"unicode/utf8"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Glyph is a positioned glyph. Advances and offsets are in 1/1024 px, the
// unit of the fonts package, with y growing upwards as in the font.
type Glyph struct {
	// ID is the glyph index in the face. Zero is .notdef.
	ID uint32
	// Cluster is the byte offset, in the UTF-8 encoding of the shaped text,
	// of the first character of the cluster the glyph belongs to.
	Cluster int

	XAdvance, YAdvance int32
	XOffset, YOffset   int32
}

// Missing reports whether the glyph is .notdef, which means the face has
// no glyph for the character.
func (g Glyph) Missing() bool { return g.ID == 0 }

// XAdvancePx returns the horizontal advance in px.
func (g Glyph) XAdvancePx() float64 { return fonts.FromUnits(g.XAdvance) }

// YAdvancePx returns the vertical advance in px.
func (g Glyph) YAdvancePx() float64 { return fonts.FromUnits(g.YAdvance) }

// XOffsetPx returns the horizontal offset in px.
func (g Glyph) XOffsetPx() float64 { return fonts.FromUnits(g.XOffset) }

// YOffsetPx returns the vertical offset in px.
func (g Glyph) YOffsetPx() float64 { return fonts.FromUnits(g.YOffset) }

// Run is a piece of text shaped with a single face, size, script and
// direction.
type Run struct {
	// Text holds the characters of the paragraph and Start:End the part to
	// shape. The rest is context for contextual shaping and joining.
	Text       []rune
	Start, End int

	Face *fonts.Face
	// Size is the font size in px, truncated to the 1/1024 px grid.
	Size float64
	// Variations, when set, are applied to Face before shaping. Leave it
	// empty when Face already is the right instance.
	Variations []font.Variation
	// Features are applied after the features of the @font-face rule of the
	// face, so they win on conflicts.
	Features []fonts.Feature

	// Lang and Script are detected from the text when empty.
	Lang   language.Language
	Script language.Script
	RTL    bool
}

// Shaper converts a run into glyphs in visual order.
type Shaper interface {
	Shape(r Run) []Glyph
}

// GoText shapes with the go-text HarfBuzz implementation.
//
// The font is scaled to int(size*1024), so advances come out as integers in
// 1/1024 px with the rounding of the shaping engine, and clusters are
// computed per character (HB_BUFFER_CLUSTER_LEVEL_MONOTONE_CHARACTERS).
type GoText struct{}

// Shape implements Shaper.
func (GoText) Shape(r Run) []Glyph {
	if r.Face == nil || r.Start >= r.End || r.End > len(r.Text) {
		return nil
	}
	face := r.Face
	if len(r.Variations) > 0 {
		face = face.Instance(r.Variations)
	}
	runes := r.Text[r.Start:r.End]

	buf := harfbuzz.NewBuffer()
	buf.AddRunes(r.Text, r.Start, r.End-r.Start)
	buf.ClusterLevel = harfbuzz.MonotoneCharacters
	buf.Props.Direction = harfbuzz.LeftToRight
	if r.RTL {
		buf.Props.Direction = harfbuzz.RightToLeft
	}
	buf.Props.Script = r.Script
	if buf.Props.Script == 0 {
		buf.Props.Script = detectScript(runes)
	}
	buf.Props.Language = r.Lang

	features := make([]fonts.Feature, 0, len(face.Features)+len(r.Features))
	features = append(append(features, face.Features...), r.Features...)
	buf.Shape(face.HB(fonts.ToUnits(r.Size)), fonts.HBFeatures(features))

	// Clusters index the paragraph in characters; callers work in bytes of
	// the shaped text.
	offsets := make([]int, r.End-r.Start+1)
	for i, c := range runes {
		offsets[i+1] = offsets[i] + utf8.RuneLen(c)
	}

	glyphs := make([]Glyph, len(buf.Info))
	for i, info := range buf.Info {
		pos := buf.Pos[i]
		glyphs[i] = Glyph{
			ID:       uint32(info.Glyph),
			Cluster:  offsets[info.Cluster-r.Start],
			XAdvance: pos.XAdvance,
			YAdvance: pos.YAdvance,
			XOffset:  pos.XOffset,
			YOffset:  pos.YOffset,
		}
	}
	return glyphs
}

func detectScript(text []rune) language.Script {
	for _, c := range text {
		if s := language.LookupScript(c); s != language.Common && s != language.Inherited {
			return s
		}
	}
	return language.Common
}
