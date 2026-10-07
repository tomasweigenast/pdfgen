package fonts

import "github.com/tomasweigenast/pdfgen/internal/css/props"

// SourceKind tells how a source of an @font-face rule is located.
type SourceKind uint8

const (
	// SourceURL is a url() source, loaded with the configured fetcher.
	SourceURL SourceKind = iota
	// SourceLocal is a local() source, the full name or PostScript name of
	// an installed font.
	SourceLocal
)

// Source is one entry of the src descriptor.
type Source struct {
	Kind SourceKind
	// Value is the absolute URL for SourceURL and the font name for
	// SourceLocal. An empty value is skipped.
	Value string
	// Format is the format() hint. It is informational: the font data is
	// sniffed, not trusted.
	Format string
}

// Range is an inclusive range of code points of the unicode-range
// descriptor.
type Range struct{ Start, End rune }

// FontFaceRule holds the validated descriptors of an @font-face rule.
// Unset descriptors keep the properties of the font file.
type FontFaceRule struct {
	Family string
	// Src is tried in order until one source loads.
	Src []Source
	// Weight is 100 to 900, with normal as 400 and bold as 700. Zero means
	// unset.
	Weight int
	// Style is "normal", "italic" or "oblique". Empty means unset.
	Style string
	// Stretch is a font-stretch keyword. Empty means unset.
	Stretch string
	// UnicodeRange limits the code points the face is used for. Empty means
	// all of them.
	UnicodeRange []Range
	// Variant is the font-variant descriptor.
	Variant FontVariant
	// FeatureSettings is the font-feature-settings descriptor.
	FeatureSettings props.FontSettings
}

func (r FontFaceRule) covers(c rune) bool {
	if len(r.UnicodeRange) == 0 {
		return true
	}
	for _, u := range r.UnicodeRange {
		if u.Start <= c && c <= u.End {
			return true
		}
	}
	return false
}

// features returns the OpenType features the rule applies to its font. Like
// every font-face rule, kerning is on unless font-feature-settings turns it
// off.
func (r FontFaceRule) features() []Feature {
	return Features(FeatureStyle{FontVariant: r.Variant, Settings: r.FeatureSettings})
}
