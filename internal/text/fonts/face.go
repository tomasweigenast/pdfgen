package fonts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/go-text/typesetting/harfbuzz"
	tdfont "github.com/tdewolff/font"
)

// Axis is a variation axis of a variable font.
type Axis struct {
	Tag               font.Tag
	Min, Default, Max float32
}

// Face is a loaded font at one point of its variation space.
type Face struct {
	// ID is a stable identifier derived from the font data, the face index
	// and the variation coordinates. Faces with the same ID embed the same
	// glyph outlines.
	ID string

	// Raw is the sfnt data, after WOFF and WOFF2 decompression. For a
	// collection it is the whole collection, and Index selects the face.
	Raw   []byte
	Index int

	GoText *font.Face
	IsCFF  bool

	// Variations are the coordinates applied to GoText, in design units.
	Variations []font.Variation

	// Features are the OpenType features declared by the @font-face rule
	// that provided the font. They are applied before the features of the
	// style, so the style wins on conflicts.
	Features []Feature

	Family string

	loader *opentype.Loader
	axes   []Axis
	hb     map[int32]*harfbuzz.Font
}

// NewFace parses a font from raw bytes. WOFF and WOFF2 data is decompressed,
// and index selects a face inside a collection.
func NewFace(data []byte, index int) (*Face, error) {
	raw, err := Decompress(data)
	if err != nil {
		return nil, err
	}
	loaders, err := opentype.NewLoaders(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}
	if index < 0 || index >= len(loaders) {
		return nil, fmt.Errorf("font index %d out of range (%d faces)", index, len(loaders))
	}
	gt, err := font.NewFont(loaders[index])
	if err != nil {
		return nil, fmt.Errorf("loading font: %w", err)
	}
	return newFace(raw, index, loaders[index], gt), nil
}

func newFace(raw []byte, index int, ld *opentype.Loader, gt *font.Font) *Face {
	sum := sha256.Sum256(raw)
	f := &Face{
		ID:     hex.EncodeToString(sum[:8]) + "-" + strconv.Itoa(index),
		Raw:    raw,
		Index:  index,
		GoText: font.NewFace(gt),
		IsCFF:  ld.HasTable(opentype.MustNewTag("CFF ")) || ld.HasTable(opentype.MustNewTag("CFF2")),
		Family: gt.Describe().Family,
		loader: ld,
		hb:     map[int32]*harfbuzz.Font{},
	}
	f.axes = readAxes(ld)
	return f
}

func readAxes(ld *opentype.Loader) []Axis {
	table, err := ld.RawTable(opentype.MustNewTag("fvar"))
	if err != nil {
		return nil
	}
	fvar, _, err := tables.ParseFvar(table)
	if err != nil {
		return nil
	}
	axes := make([]Axis, len(fvar.Axis))
	for i, a := range fvar.Axis {
		axes[i] = Axis{Tag: a.Tag, Min: a.Minimum, Default: a.Default, Max: a.Maximum}
	}
	return axes
}

// Decompress returns the sfnt data inside a WOFF or WOFF2 file, and
// returns any other data unchanged.
func Decompress(data []byte) ([]byte, error) {
	if len(data) < 4 || string(data[:3]) != "wOF" {
		return data, nil
	}
	switch data[3] {
	case 'F':
		out, err := tdfont.ParseWOFF(data)
		if err != nil {
			return nil, fmt.Errorf("decoding WOFF: %w", err)
		}
		return out, nil
	case '2':
		out, err := tdfont.ParseWOFF2(data)
		if err != nil {
			return nil, fmt.Errorf("decoding WOFF2: %w", err)
		}
		return out, nil
	}
	return data, nil
}

// Axes returns the variation axes of the font, which is empty for a static
// font.
func (f *Face) Axes() []Axis { return f.axes }

// IsVariable reports whether the font has variation axes.
func (f *Face) IsVariable() bool { return len(f.axes) > 0 }

// HasGlyph reports whether the cmap maps r to a glyph.
func (f *Face) HasGlyph(r rune) bool {
	_, ok := f.GoText.NominalGlyph(r)
	return ok
}

// Instance returns the face at the given variation coordinates, sharing the
// font data with f. It returns f when the font is not variable.
func (f *Face) Instance(vars []font.Variation) *Face {
	if !f.IsVariable() {
		return f
	}
	inst := *f
	inst.GoText = font.NewFace(f.GoText.Font)
	inst.GoText.SetVariations(vars)
	inst.Variations = vars
	inst.hb = map[int32]*harfbuzz.Font{}
	inst.ID = f.baseID() + variationsKey(vars)
	return &inst
}

func (f *Face) baseID() string {
	if i := strings.IndexByte(f.ID, '@'); i >= 0 {
		return f.ID[:i]
	}
	return f.ID
}

func variationsKey(vars []font.Variation) string {
	if len(vars) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('@')
	for i, v := range vars {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(v.Tag.String())
		b.WriteByte('=')
		b.WriteString(strconv.FormatFloat(float64(v.Value), 'g', -1, 32))
	}
	return b.String()
}

// HB returns a HarfBuzz font scaled to size, given in 1/1024 px. The same
// value is returned for the same size, so callers must not change its
// scale.
func (f *Face) HB(size int32) *harfbuzz.Font {
	if h, ok := f.hb[size]; ok {
		return h
	}
	h := harfbuzz.NewFont(f.GoText)
	h.XScale, h.YScale = size, size
	f.hb[size] = h
	return h
}
