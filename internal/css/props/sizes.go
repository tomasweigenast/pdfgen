package props

// FontSizeKeywords gives the size in pixels of the absolute font-size
// keywords: medium is the initial font size and the others are a ratio of it
// (CSS Fonts 3 §3.5).
var FontSizeKeywords = map[Keyword]float64{
	"xx-small": 16 * 3 / 5,
	"x-small":  16 * 3 / 4,
	"small":    16 * 8 / 9,
	"medium":   16,
	"large":    16 * 6 / 5,
	"x-large":  16 * 3 / 2,
	"xx-large": 16 * 2,
}

// BorderWidthKeywords gives the width in pixels of thin, medium and thick.
// The specification only requires thin <= medium <= thick.
var BorderWidthKeywords = map[Keyword]float64{
	"thin":   1,
	"medium": 3,
	"thick":  5,
}

// PageSize is the portrait size of a named page size.
type PageSize struct {
	Width, Height float64
	Unit          string
}

// PageSizes maps the page size keywords of the size property to their
// portrait dimensions (CSS Paged Media 3 §7.1).
var PageSizes = map[Keyword]PageSize{
	"a10": {26, 37, "mm"}, "a9": {37, 52, "mm"}, "a8": {52, 74, "mm"},
	"a7": {74, 105, "mm"}, "a6": {105, 148, "mm"}, "a5": {148, 210, "mm"},
	"a4": {210, 297, "mm"}, "a3": {297, 420, "mm"}, "a2": {420, 594, "mm"},
	"a1": {594, 841, "mm"}, "a0": {841, 1189, "mm"},
	"b10": {31, 44, "mm"}, "b9": {44, 62, "mm"}, "b8": {62, 88, "mm"},
	"b7": {88, 125, "mm"}, "b6": {125, 176, "mm"}, "b5": {176, 250, "mm"},
	"b4": {250, 353, "mm"}, "b3": {353, 500, "mm"}, "b2": {500, 707, "mm"},
	"b1": {707, 1000, "mm"}, "b0": {1000, 1414, "mm"},
	"c10": {28, 40, "mm"}, "c9": {40, 57, "mm"}, "c8": {57, 81, "mm"},
	"c7": {81, 114, "mm"}, "c6": {114, 162, "mm"}, "c5": {162, 229, "mm"},
	"c4": {229, 324, "mm"}, "c3": {324, 458, "mm"}, "c2": {458, 648, "mm"},
	"c1": {648, 917, "mm"}, "c0": {917, 1297, "mm"},
	"jis-b10": {32, 45, "mm"}, "jis-b9": {45, 64, "mm"}, "jis-b8": {64, 91, "mm"},
	"jis-b7": {91, 128, "mm"}, "jis-b6": {128, 182, "mm"}, "jis-b5": {182, 257, "mm"},
	"jis-b4": {257, 364, "mm"}, "jis-b3": {364, 515, "mm"}, "jis-b2": {515, 728, "mm"},
	"jis-b1": {728, 1030, "mm"}, "jis-b0": {1030, 1456, "mm"},
	"letter": {8.5, 11, "in"}, "legal": {8.5, 14, "in"}, "ledger": {11, 17, "in"},
}

// InitialPageSize is the portrait size used when none is specified: A4.
var InitialPageSize = PageSizes["a4"]
