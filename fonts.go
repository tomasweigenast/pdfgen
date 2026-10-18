package pdfgen

import "github.com/tomasweigenast/pdfgen/internal/text/fonts"

// FontConfiguration holds the fonts available to documents: the fonts
// installed on the system and those declared by @font-face rules. Share one
// between renderings to reuse the index of system fonts and the fonts
// already loaded; a zero Options makes a new one for each rendering.
//
// A FontConfiguration must not be used by several renderings at the same
// time.
type FontConfiguration struct {
	config *fonts.FontConfiguration
}

// FontOption adjusts a FontConfiguration.
type FontOption func([]fonts.Option) []fonts.Option

// WithoutSystemFonts leaves the installed fonts out, so that documents can
// only use the fonts they declare with @font-face.
func WithoutSystemFonts() FontOption {
	return func(o []fonts.Option) []fonts.Option { return append(o, fonts.WithoutSystemFonts()) }
}

// WithFontCacheDir sets the directory where the index of system fonts is
// kept between runs. The default is the pdfgen directory inside the user
// cache directory.
func WithFontCacheDir(dir string) FontOption {
	return func(o []fonts.Option) []fonts.Option { return append(o, fonts.WithCacheDir(dir)) }
}

// NewFontConfiguration returns a configuration with the installed fonts.
// They are indexed the first time a document needs one.
func NewFontConfiguration(o ...FontOption) *FontConfiguration {
	var opts []fonts.Option
	for _, option := range o {
		opts = option(opts)
	}
	return &FontConfiguration{config: fonts.NewFontConfiguration(opts...)}
}
