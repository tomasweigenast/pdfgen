package fonts

import (
	"crypto/md5"
	"math"
	"regexp"
	"strings"

	gtfont "github.com/go-text/typesetting/font"

	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts/subset"
	textfonts "github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// maxCID is the highest code of a two-byte Identity-H font.
const maxCID = 1<<16 - 1

// Font is a face embedded in a document. Content streams show glyphs by CID,
// codes that [Font.CID] hands out in the order glyphs are first drawn, so
// that the subset can number its glyphs the same way and CIDToGIDMap can be
// /Identity (ISO 32000-1 section 9.7.4.2).
//
// With full fonts, or when a font cannot be subset, the whole font is
// embedded and the CID of a glyph is its glyph index.
type Font struct {
	// Face is the font data and the shaping face the layout used.
	Face *textfonts.Face
	// Hash is the six-letter tag that names the font resource. It is also
	// the subset tag of BaseFont (ISO 32000-1 section 9.6.4).
	Hash string
	// Family is the family name of the face.
	Family string
	// Ascent and Descent are the vertical metrics in thousandths of an em;
	// Descent is negative.
	Ascent, Descent int
	// Flags are the font descriptor flags (ISO 32000-1 table 123).
	Flags int
	// ItalicAngle, StemV and StemH fill the font descriptor. The face has no
	// data for them, so StemV and StemH are fixed guesses.
	ItalicAngle  float64
	StemV, StemH int

	name       string
	glyphCount int
	parsed     *subset.Font // nil when the font cannot be subset
	identity   bool

	order []uint16          // CID -> glyph index, in subset mode
	cids  map[uint16]uint16 // glyph index -> CID, in subset mode

	widths  map[uint16]float64 // by CID, in thousandths of an em
	text    map[uint16]string  // by CID
	missing map[rune]uint16    // CID of the stand-in of a character no font has
	// missingOrder keeps the missing characters in the order they were
	// first drawn.
	missingOrder []rune
}

var nameChars = regexp.MustCompile(`[^A-Za-z0-9-]`)

func newFont(face *textfonts.Face, full bool) *Font {
	f := &Font{
		Face:     face,
		Family:   face.Family,
		StemV:    80,
		StemH:    80,
		widths:   map[uint16]float64{},
		text:     map[uint16]string{},
		missing:  map[rune]uint16{},
		cids:     map[uint16]uint16{0: 0},
		order:    []uint16{0},
		identity: full,
	}

	// Never derive the tag from something unstable: it ends up in the file.
	sum := md5.Sum([]byte(face.ID))
	tag := make([]byte, 6)
	for i := range tag {
		tag[i] = 'A' + sum[i]%26
	}
	f.Hash = string(tag)
	family := nameChars.ReplaceAllString(strings.ReplaceAll(face.Family, " ", "-"), "")
	if family == "" {
		family = "Font"
	}
	f.name = f.Hash + "+" + family

	size := 1000.0
	m := face.Metrics(size)
	f.Ascent = int(math.RoundToEven(textfonts.FromUnits(m.Ascent)))
	f.Descent = -int(math.RoundToEven(textfonts.FromUnits(m.Descent)))

	f.Flags = 1 << (3 - 1) // symbolic: the encoding is custom
	desc := face.GoText.Describe()
	if desc.Aspect.Style != gtfont.StyleNormal {
		f.Flags |= 1 << (7 - 1)
	}
	for _, word := range strings.Split(face.Family, " ") {
		if word == "Serif" {
			f.Flags |= 1 << (2 - 1)
		}
	}

	count, err := subset.GlyphCount(face.Raw, face.Index)
	if err == nil {
		f.glyphCount = count
	}
	if !full {
		f.parsed, err = subset.Parse(face.Raw, face.Index)
		if err != nil {
			logging.Warnf("Unable to subset %q, embedding the whole font: %v", f.Family, err)
			f.identity = true
		}
	}
	if len(face.Variations) > 0 {
		logging.Warnf("Unable to instantiate %q variable font, embedding its default instance", f.Family)
	}
	return f
}

// BaseFont returns the PostScript name of the font as written in the PDF:
// the subset tag, a plus sign and the family.
func (f *Font) BaseFont() string { return f.name }

// IsCFF reports whether the font has PostScript outlines.
func (f *Font) IsCFF() bool { return f.Face.IsCFF }

// CID returns the code that shows glyph gid, assigning the next free code the
// first time a glyph is used. Code 0 is .notdef.
func (f *Font) CID(gid uint16) uint16 {
	if f.identity {
		return gid
	}
	if cid, ok := f.cids[gid]; ok {
		return cid
	}
	if len(f.order) > maxCID {
		logging.Warnf("Too many glyphs used from %q, expect missing characters", f.Family)
		return 0
	}
	cid := uint16(len(f.order))
	f.order = append(f.order, gid)
	f.cids[gid] = cid
	return cid
}

// MissingCID returns the code of a stand-in glyph for r, a character no font
// provides, and records width (in thousandths of an em) as its advance. The
// stand-in is drawn as .notdef, and the same code is returned for every use
// of r. Glyph 0 gets the same width when it has none, so that text keeps the
// layout the shaper computed.
func (f *Font) MissingCID(r rune, width float64) uint16 {
	cid, ok := f.missing[r]
	if !ok {
		switch {
		case f.identity:
			// A code beyond the glyphs of the font; the encoding CMap sends
			// it to .notdef.
			next := f.glyphCount + len(f.missing)
			if next > maxCID {
				logging.Warnf("Too many glyphs missing from %q, expect text selection problems", f.Family)
				next = maxCID
			}
			cid = uint16(next)
		case len(f.order) > maxCID:
			logging.Warnf("Too many glyphs missing from %q, expect text selection problems", f.Family)
			cid = maxCID
		default:
			cid = uint16(len(f.order))
			f.order = append(f.order, 0)
		}
		f.missing[r] = cid
		f.missingOrder = append(f.missingOrder, r)
	}
	f.widths[cid] = width
	if _, ok := f.widths[0]; !ok {
		f.widths[0] = width
	}
	return cid
}

// Width returns the advance recorded for cid, in thousandths of an em.
func (f *Font) Width(cid uint16) (float64, bool) {
	w, ok := f.widths[cid]
	return w, ok
}

// SetWidth records the advance of cid in thousandths of an em.
func (f *Font) SetWidth(cid uint16, w float64) { f.widths[cid] = w }

// LogicalWidth returns the advance of cid in thousandths of an em. When none
// was recorded it is measured on the face at size px and rounded like the
// shaper rounds, and recorded.
func (f *Font) LogicalWidth(cid uint16, size float64) float64 {
	if w, ok := f.widths[cid]; ok {
		return w
	}
	gid := cid
	if !f.identity {
		if int(cid) < len(f.order) {
			gid = f.order[cid]
		}
	}
	advance := f.Face.HB(textfonts.ToUnits(size)).GlyphHAdvance(gtfont.GID(gid))
	w := math.RoundToEven(float64(advance) * 1000 / textfonts.UnitsPerPx / size)
	f.widths[cid] = w
	return w
}

// SetText records the text that cid stands for, for the ToUnicode map. It is
// kept only the first time: the same glyph may be drawn for several clusters,
// and the first is what gets copied.
func (f *Font) SetText(cid uint16, text string) {
	if _, ok := f.text[cid]; !ok {
		f.text[cid] = text
	}
}

// HasText reports whether the text of cid is known.
func (f *Font) HasText(cid uint16) bool {
	_, ok := f.text[cid]
	return ok
}
