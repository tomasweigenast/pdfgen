package fonts

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode/utf16"

	gtfont "github.com/go-text/typesetting/font"

	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts/subset"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// BuildOptions control how fonts are written.
type BuildOptions struct {
	// Compress selects Flate for the font programs and CMaps.
	Compress bool
	// Hinting keeps the hinting programs of subset TrueType fonts.
	Hinting bool
	// PDFVersion is the version of the file, such as "1.7". Up to 1.4 each
	// font descriptor gets a CIDSet, which that version requires of subsets
	// (ISO 32000-1 table 124).
	PDFVersion string
}

// Build adds the objects of every font to pdf and returns the font resource
// dictionary: font resource names (see [Font.Hash]) mapped to the Type0 font
// dictionaries. It is meant to be called once, when all pages are drawn.
func (r *Registry) Build(pdf *writer.PDF, o BuildOptions) (*writer.Dict, error) {
	files := make(map[*Font]writer.Ref, len(r.fonts))
	for _, f := range r.fonts {
		data, err := f.fontFile(o)
		if err != nil {
			return nil, fmt.Errorf("embedding font %q: %w", f.Family, err)
		}
		compress := o.Compress
		stream := &writer.Stream{Data: data, Compress: &compress}
		if bytes.HasPrefix(data, []byte("OTTO")) {
			stream.Dict = writer.NewDict("Subtype", writer.Name("OpenType"))
		} else {
			stream.Dict = writer.NewDict("Length1", len(data))
		}
		files[f] = pdf.Add(stream)
	}

	dict := writer.NewDict()
	for _, f := range r.fonts {
		dict.Set(writer.Name(f.Hash), pdf.Add(f.dictionary(pdf, files[f], r.full, o)))
	}
	return dict, nil
}

// fontFile returns the font program to embed.
func (f *Font) fontFile(o BuildOptions) ([]byte, error) {
	if f.identity {
		return subset.Full(f.Face.Raw, f.Face.Index)
	}
	return f.parsed.Subset(f.order, subset.Options{KeepNotdef: len(f.missing) > 0, Hinting: o.Hinting})
}

// tables returns the advances and the texts to write: those of the glyphs
// used, or those of every glyph of a full font.
func (f *Font) tables(full bool) (widths map[uint16]float64, text map[uint16]string) {
	if !full {
		return f.widths, f.text
	}
	upem := float64(f.Face.GoText.Upem())
	widths = make(map[uint16]float64, f.glyphCount)
	for gid := 0; gid < f.glyphCount; gid++ {
		adv := float64(f.Face.GoText.HorizontalAdvance(gtfont.GID(gid)))
		widths[uint16(gid)] = adv * 1000 / upem
	}

	// The first, lowest, code point of a glyph is the text it copies as.
	lowest := make(map[uint16]rune, f.glyphCount)
	for it := f.Face.GoText.Cmap.Iter(); it.Next(); {
		r, gid := it.Char()
		if cur, ok := lowest[uint16(gid)]; !ok || r < cur {
			lowest[uint16(gid)] = r
		}
	}
	text = make(map[uint16]string, len(lowest))
	for gid, r := range lowest {
		text[gid] = string(r)
	}
	return widths, text
}

func (f *Font) dictionary(pdf *writer.PDF, file writer.Ref, full bool, o BuildOptions) *writer.Dict {
	widths, text := f.tables(full)
	compress := o.Compress

	toUnicode := pdf.Add(&writer.Stream{Data: toUnicodeCMap(text), Compress: &compress})

	cff := f.IsCFF()
	fileKey, cidSubtype := "FontFile2", "CIDFontType2"
	if cff {
		fileKey, cidSubtype = "FontFile3", "CIDFontType0"
	}

	var maxX float64
	for _, w := range widths {
		maxX = max(maxX, w)
	}
	flags := f.Flags
	if len(widths) > 1 && distinct(f.widths) == 1 {
		flags |= 1 // FixedPitch
	}
	bbox := writer.Array{writer.Int(0), writer.Int(f.Descent), writer.Real(maxX), writer.Int(f.Ascent)}
	descriptor := writer.NewDict(
		"Type", writer.Name("FontDescriptor"),
		"FontName", writer.Name(f.name),
		"FontFamily", writer.Text(f.Family),
		"Flags", flags,
		"FontBBox", bbox,
		"ItalicAngle", f.ItalicAngle,
		"Ascent", f.Ascent,
		"Descent", f.Descent,
		"CapHeight", writer.Int(f.Ascent),
		"StemV", f.StemV,
		"StemH", f.StemH,
		fileKey, file,
	)
	if o.PDFVersion != "" && o.PDFVersion <= "1.4" && len(f.widths) > 0 {
		set := pdf.Add(&writer.Stream{Data: cidSet(f.widths), Compress: &compress})
		descriptor.Set("CIDSet", set)
	}
	descriptorRef := pdf.Add(descriptor)

	cidFont := pdf.Add(writer.NewDict(
		"Type", writer.Name("Font"),
		"Subtype", writer.Name(cidSubtype),
		"BaseFont", writer.Name(f.name),
		"CIDSystemInfo", writer.NewDict(
			"Registry", writer.Text("Adobe"),
			"Ordering", writer.Text("Identity"),
			"Supplement", 0,
		),
		"CIDToGIDMap", writer.Name("Identity"),
		"W", widthsArray(widths),
		"FontDescriptor", descriptorRef,
	))

	var encoding writer.Object = writer.Name("Identity-H")
	if f.identity && len(f.missing) > 0 {
		encoding = pdf.Add(&writer.Stream{
			Dict: writer.NewDict(
				"Type", writer.Name("CMap"),
				"CMapName", writer.Name("Identity-Notdef"),
				"CIDSystemInfo", writer.NewDict(
					"Registry", writer.Text("Adobe"),
					"Ordering", writer.Text("Identity"),
					"Supplement", 0,
				),
			),
			Data:     f.notdefCMap(),
			Compress: &compress,
		})
	}
	return writer.NewDict(
		"Type", writer.Name("Font"),
		"Subtype", writer.Name("Type0"),
		"BaseFont", writer.Name(f.name),
		"ToUnicode", toUnicode,
		"Encoding", encoding,
		"DescendantFonts", writer.Array{cidFont},
	)
}

func distinct(widths map[uint16]float64) int {
	seen := make(map[float64]bool, len(widths))
	for _, w := range widths {
		seen[w] = true
	}
	return len(seen)
}

func sortedKeys[V any](m map[uint16]V) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// widthsArray builds the /W array of a CIDFont (ISO 32000-1 section 9.7.4.3):
// a first code followed by the list of the advances of the consecutive codes
// that have one.
func widthsArray(widths map[uint16]float64) writer.Array {
	var out writer.Array
	var run writer.Array
	var prev int = -2
	for _, cid := range sortedKeys(widths) {
		if int(cid) != prev+1 {
			if run != nil {
				out = append(out, run)
			}
			out = append(out, writer.Int(cid))
			run = writer.Array{}
		}
		run = append(run, writer.Real(widths[cid]))
		prev = int(cid)
	}
	if run != nil {
		out = append(out, run)
	}
	if out == nil {
		out = writer.Array{}
	}
	return out
}

// cidSet returns the bit set of the codes in widths, the first code being the
// most significant bit of the first byte.
func cidSet(widths map[uint16]float64) []byte {
	cids := sortedKeys(widths)
	set := make([]byte, int(math.Ceil(float64(cids[len(cids)-1]+1)/8)))
	for _, cid := range cids {
		set[cid/8] |= 0x80 >> (cid % 8)
	}
	return set
}

// toUnicodeCMap returns the ToUnicode CMap (ISO 32000-1 section 9.10.3) that
// maps each code to its text in UTF-16BE.
func toUnicodeCMap(text map[uint16]string) []byte {
	lines := []string{
		"/CIDInit /ProcSet findresource begin",
		"12 dict begin",
		"begincmap",
		"/CIDSystemInfo",
		"<< /Registry (Adobe)",
		"/Ordering (UCS)",
		"/Supplement 0",
		">> def",
		"/CMapName /Adobe-Identity-UCS def",
		"/CMapType 2 def",
		"1 begincodespacerange",
		"<0000> <ffff>",
		"endcodespacerange",
	}
	cids := sortedKeys(text)
	for start := 0; start < len(cids); start += 100 {
		batch := cids[start:min(start+100, len(cids))]
		lines = append(lines, fmt.Sprintf("%d beginbfchar", len(batch)))
		for _, cid := range batch {
			var utf16Hex strings.Builder
			for _, u := range utf16.Encode([]rune(text[cid])) {
				fmt.Fprintf(&utf16Hex, "%04x", u)
			}
			lines = append(lines, fmt.Sprintf("<%04x> <%s>", cid, utf16Hex.String()))
		}
		lines = append(lines, "endbfchar")
	}
	lines = append(lines,
		"endcmap",
		"CMapName currentdict /CMap defineresource pop",
		"end",
		"end",
	)
	return []byte(strings.Join(lines, "\n"))
}

// notdefCMap returns an encoding that is the identity except for the codes of
// missing characters, which it sends to CID 0 so that they show .notdef.
func (f *Font) notdefCMap() []byte {
	stand := make(map[uint16]bool, len(f.missing))
	for _, cid := range f.missing {
		stand[cid] = true
	}
	lines := []string{
		"/CIDInit /ProcSet findresource begin",
		"12 dict begin",
		"begincmap",
		"/CIDSystemInfo",
		"3 dict dup begin",
		"/Registry (Adobe) def",
		"/Ordering (Identity) def",
		"/Supplement 0 def",
		"end def",
		"/CMapName /Identity-Notdef def",
		"/CMapType 1 def",
		"1 begincodespacerange",
		"<0000> <ffff>",
		"endcodespacerange",
	}
	cids := sortedKeys(f.text)
	for start := 0; start < len(cids); start += 100 {
		batch := cids[start:min(start+100, len(cids))]
		lines = append(lines, fmt.Sprintf("%d begincidchar", len(batch)))
		for _, cid := range batch {
			target := cid
			if stand[cid] {
				target = 0
			}
			lines = append(lines, fmt.Sprintf("<%04x> %d", cid, target))
		}
		lines = append(lines, "endcidchar")
	}
	lines = append(lines,
		"endcmap",
		"CMapName currentdict /CMap defineresource pop",
		"end",
		"end",
	)
	return []byte(strings.Join(lines, "\n"))
}
