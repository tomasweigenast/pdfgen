package shape

import "github.com/tomasweigenast/pdfgen/internal/text/fonts"

type metricCase struct {
	name     string
	size     float64
	text     string
	features []fonts.Feature
	glyphs   []Glyph
}

// testFontCases holds the expected shaping of testfont.otf in 1/1024 px.
var testFontCases = []metricCase{
	{
		name: "Hello fi ffi AVATAR @16", size: 16, text: "Hello fi ffi AVATAR",
		features: []fonts.Feature{},
		glyphs:   []Glyph{{ID: 40, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 69, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 76, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 76, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 79, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 4}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 5}, {ID: 70, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 6}, {ID: 73, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 70, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 70, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 73, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 12}, {ID: 33, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 13}, {ID: 54, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 14}, {ID: 33, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 15}, {ID: 52, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 16}, {ID: 33, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 17}, {ID: 50, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 18}},
	},
	{
		name: "kk liga camps @16", size: 16, text: "kk liga camps",
		features: []fonts.Feature{},
		glyphs:   []Glyph{{ID: 75, XAdvance: 8192, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 24576, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 67, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 77, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 80, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 83, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga camps liga 0 @16", size: 16, text: "kk liga camps",
		features: []fonts.Feature{{Tag: "liga", Value: 0}},
		glyphs:   []Glyph{{ID: 75, XAdvance: 8192, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 76, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 73, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 4}, {ID: 71, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 5}, {ID: 65, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 6}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 67, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 77, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 80, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 83, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga camps kern 0 @16", size: 16, text: "kk liga camps",
		features: []fonts.Feature{{Tag: "kern", Value: 0}},
		glyphs:   []Glyph{{ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 24576, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 67, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 77, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 80, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 83, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga camps smcp @16", size: 16, text: "kk liga camps",
		features: []fonts.Feature{{Tag: "smcp", Value: 1}},
		glyphs:   []Glyph{{ID: 75, XAdvance: 8192, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 24576, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 250, XAdvance: 6144, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 250, XAdvance: 6144, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 250, XAdvance: 6144, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 250, XAdvance: 6144, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga lig dlig, hlig @16", size: 16, text: "kk liga lig",
		features: []fonts.Feature{{Tag: "dlig", Value: 1}, {Tag: "hlig", Value: 1}},
		glyphs:   []Glyph{{ID: 75, XAdvance: 8192, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 24576, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 76, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 73, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 71, XAdvance: 16384, YOffset: 0, XOffset: 0, Cluster: 10}},
	},
	{
		name: "kk liga camps @13.333", size: 13.333, text: "kk liga camps",
		features: []fonts.Feature{},
		glyphs:   []Glyph{{ID: 75, XAdvance: 6826, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 20478, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 67, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 77, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 80, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 83, XAdvance: 13652, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga camps @2", size: 2, text: "kk liga camps",
		features: []fonts.Feature{},
		glyphs:   []Glyph{{ID: 75, XAdvance: 1024, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 3072, YOffset: 0, XOffset: 0, Cluster: 3}, {ID: 1, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 7}, {ID: 67, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 8}, {ID: 65, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 9}, {ID: 77, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 10}, {ID: 80, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 11}, {ID: 83, XAdvance: 2048, YOffset: 0, XOffset: 0, Cluster: 12}},
	},
	{
		name: "kk liga @1000", size: 1000, text: "kk liga",
		features: []fonts.Feature{},
		glyphs:   []Glyph{{ID: 75, XAdvance: 512000, YOffset: 0, XOffset: 0, Cluster: 0}, {ID: 75, XAdvance: 1024000, YOffset: 0, XOffset: 0, Cluster: 1}, {ID: 1, XAdvance: 1024000, YOffset: 0, XOffset: 0, Cluster: 2}, {ID: 249, XAdvance: 1536000, YOffset: 0, XOffset: 0, Cluster: 3}},
	},
}
