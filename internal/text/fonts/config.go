package fonts

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// Fetcher returns the content of a URL.
type Fetcher func(url string) ([]byte, error)

// ErrFontFaceUnavailable is returned by AddFontFace when none of the sources
// of a rule could be loaded.
var ErrFontFaceUnavailable = errors.New("font-face cannot be loaded")

// Option configures a FontConfiguration.
type Option func(*FontConfiguration)

// WithCacheDir sets the directory holding the index of system fonts. The
// default is the pdfgen directory inside os.UserCacheDir.
func WithCacheDir(dir string) Option {
	return func(fc *FontConfiguration) { fc.cacheDir = dir }
}

// WithoutSystemFonts disables the lookup of installed fonts, so that only
// @font-face rules provide fonts.
func WithoutSystemFonts() Option {
	return func(fc *FontConfiguration) { fc.noSystem = true }
}

// FontConfiguration holds the fonts available to a document: the installed
// fonts and the fonts declared by @font-face rules.
//
// A FontConfiguration is not safe for concurrent use.
type FontConfiguration struct {
	cacheDir string
	noSystem bool

	systemOnce sync.Once
	fontMap    *fontscan.FontMap
	footprints []fontscan.Footprint

	faces     map[string][]*registered
	rules     map[[16]byte]bool
	systemSet map[fontscan.Location]*Face
	instances map[string]*Face

	// metrics and ratios keep the measures of Strut and CharacterRatio.
	metrics map[string]Metrics
	ratios  map[string]float64
}

// registered is a face declared by an @font-face rule.
type registered struct {
	face   *Face
	rule   FontFaceRule
	aspect faceAspect
}

// NewFontConfiguration returns a configuration that has no @font-face fonts
// yet. System fonts are indexed on first use.
func NewFontConfiguration(opts ...Option) *FontConfiguration {
	fc := &FontConfiguration{
		faces:     map[string][]*registered{},
		rules:     map[[16]byte]bool{},
		systemSet: map[fontscan.Location]*Face{},
		instances: map[string]*Face{},
		metrics:   map[string]Metrics{},
		ratios:    map[string]float64{},
	}
	for _, o := range opts {
		o(fc)
	}
	return fc
}

type debugLogger struct{}

func (debugLogger) Printf(format string, args ...any) {
	logging.Logger.Debug(fmt.Sprintf(format, args...))
}

// system indexes the installed fonts once. It returns nil when there are
// none or when the index cannot be built, which is logged.
func (fc *FontConfiguration) system() *fontscan.FontMap {
	fc.systemOnce.Do(func() {
		if fc.noSystem {
			return
		}
		dir := fc.cacheDir
		if dir == "" {
			base, err := os.UserCacheDir()
			if err == nil {
				dir = filepath.Join(base, "pdfgen")
			}
		}
		fm := fontscan.NewFontMap(debugLogger{})
		if err := fm.UseSystemFonts(dir); err != nil {
			logging.Warnf("No system fonts available: %v", err)
			return
		}
		fp, err := fontscan.SystemFonts(debugLogger{}, dir)
		if err != nil {
			logging.Warnf("No system fonts available: %v", err)
			return
		}
		fc.fontMap, fc.footprints = fm, fp
	})
	return fc.fontMap
}

// AddFontFace registers the font of an @font-face rule. The sources are tried
// in order and the first one that loads is used. It returns an error wrapping
// ErrFontFaceUnavailable when none does.
func (fc *FontConfiguration) AddFontFace(rule FontFaceRule, fetch Fetcher) error {
	digest := md5.Sum(fmt.Appendf(nil, "%+v", rule))
	if fc.rules[digest] {
		return nil
	}
	for _, src := range rule.Src {
		if src.Value == "" {
			continue
		}
		data, err := fc.load(src, fetch)
		if err != nil {
			logging.Logger.Debug("Failed to load font", "source", src.Value, "error", err)
			continue
		}
		face, err := NewFace(data, 0)
		if err != nil {
			logging.Logger.Debug("Failed to load font", "source", src.Value, "error", err)
			continue
		}
		fc.register(face, rule)
		fc.rules[digest] = true
		return nil
	}
	return fmt.Errorf("%w: %q", ErrFontFaceUnavailable, rule.Family)
}

func (fc *FontConfiguration) load(src Source, fetch Fetcher) ([]byte, error) {
	if src.Kind == SourceLocal {
		return fc.loadLocal(src.Value)
	}
	if fetch == nil {
		return nil, errors.New("no fetcher")
	}
	return fetch(src.Value)
}

func (fc *FontConfiguration) register(face *Face, rule FontFaceRule) {
	face.Features = rule.features()
	face.Family = rule.Family

	own := face.GoText.Font.Describe().Aspect
	a := faceAspect{style: "normal", weight: int(own.Weight), stretch: own.Stretch}
	if own.Style == font.StyleItalic {
		a.style = "italic"
	}
	if rule.Weight != 0 {
		a.weight = rule.Weight
	}
	if rule.Style != "" {
		a.style = rule.Style
	}
	if rule.Stretch != "" {
		a.stretch = stretchOf(rule.Stretch)
	}
	key := font.NormalizeFamily(rule.Family)
	fc.faces[key] = append(fc.faces[key], &registered{face: face, rule: rule, aspect: a})

	// A new face can take over the family of measures already made.
	clear(fc.metrics)
	clear(fc.ratios)
}

// loadLocal returns the data of an installed font whose full name or
// PostScript name is name, compared without regard to case.
func (fc *FontConfiguration) loadLocal(name string) ([]byte, error) {
	if fc.system() == nil {
		return nil, errors.New("no system fonts")
	}
	want := strings.ToLower(name)
	squashed := squash(want)
	for _, fp := range fc.footprints {
		// Full names start with the family and PostScript names with the
		// family without its spaces, so other families can be skipped
		// without opening their file.
		if !strings.HasPrefix(squashed, squash(fp.Family)) {
			continue
		}
		data, err := os.ReadFile(fp.Location.File)
		if err != nil {
			continue
		}
		loaders, err := opentype.NewLoaders(bytes.NewReader(data))
		if err != nil || int(fp.Location.Index) >= len(loaders) {
			continue
		}
		raw, err := loaders[fp.Location.Index].RawTable(opentype.MustNewTag("name"))
		if err != nil {
			continue
		}
		names, _, err := tables.ParseName(raw)
		if err != nil {
			continue
		}
		const fullName, postscriptName = tables.NameID(4), tables.NameID(6)
		for _, id := range []tables.NameID{fullName, postscriptName} {
			if strings.ToLower(names.Name(id)) == want {
				return data, nil
			}
		}
	}
	return nil, fmt.Errorf("no local font named %q", name)
}

func squash(s string) string {
	return strings.NewReplacer(" ", "", "-", "", "\t", "").Replace(strings.ToLower(s))
}

// Query returns the face to draw the character r with: the face of the
// first family of the description that has a glyph for it, with the
// variations of the description applied. Families declared by @font-face
// take precedence over installed fonts of the same name. When no family has
// the character, installed fonts are searched for one that does, and
// failing that the face of the first family is returned.
//
// It returns nil only when there are no fonts at all.
func (fc *FontConfiguration) Query(d Description, lang string, r rune) *Face {
	base := fc.resolve(d, lang, r)
	if base == nil {
		return nil
	}
	return fc.instance(base, d)
}

func (fc *FontConfiguration) instance(base *Face, d Description) *Face {
	if !base.IsVariable() {
		return base
	}
	vars := base.VariationsFor(d.weight(), d.Size, d.Variations)
	key := base.ID + variationsKey(vars)
	if inst, ok := fc.instances[key]; ok {
		return inst
	}
	inst := base.Instance(vars)
	fc.instances[key] = inst
	return inst
}

func (fc *FontConfiguration) resolve(d Description, lang string, r rune) *Face {
	var primary *Face
	for _, family := range d.Families {
		if face := fc.registeredFace(family, d, r); face != nil {
			if face.HasGlyph(r) {
				return face
			}
			if primary == nil {
				primary = face
			}
			continue
		}
		if face := fc.systemFace(family, d, r); face != nil {
			if face.HasGlyph(r) {
				return face
			}
			if primary == nil {
				primary = face
			}
		}
	}
	if face := fc.fallback(d, lang, r); face != nil && (primary == nil || face.HasGlyph(r)) {
		return face
	}
	return primary
}

// registeredFace returns the @font-face face of a family that matches the
// description and whose unicode-range includes r.
func (fc *FontConfiguration) registeredFace(family string, d Description, r rune) *Face {
	var candidates []*registered
	for _, c := range fc.faces[font.NormalizeFamily(family)] {
		if c.rule.covers(r) {
			candidates = append(candidates, c)
		}
	}
	if c := bestMatch(candidates, d); c != nil {
		return c.face
	}
	return nil
}

var genericFamilies = map[string]bool{
	"serif": true, "sans-serif": true, "monospace": true, "cursive": true,
	"fantasy": true, "system-ui": true,
}

// systemFace returns the installed font of a family that best matches the
// description. A named family must match exactly, while a generic family
// stands for whatever the system maps it to.
func (fc *FontConfiguration) systemFace(family string, d Description, r rune) *Face {
	fm := fc.system()
	if fm == nil {
		return nil
	}
	fm.SetQuery(fontscan.Query{Families: []string{family}, Aspect: d.aspect()})
	fm.SetScript(language.LookupScript(r))
	gt := fm.ResolveFace(r)
	if gt == nil {
		return nil
	}
	if name, _ := fm.FontMetadata(gt.Font); !genericFamilies[strings.ToLower(family)] && name != font.NormalizeFamily(family) {
		return nil
	}
	return fc.wrapSystem(fm, gt)
}

// fallback searches the installed fonts for any face with a glyph for r,
// starting from the families of the description.
func (fc *FontConfiguration) fallback(d Description, lang string, r rune) *Face {
	fm := fc.system()
	if fm == nil {
		return nil
	}
	fm.SetQuery(fontscan.Query{Families: d.Families, Aspect: d.aspect()})
	script := language.LookupScript(r)
	fm.SetScript(script)
	gt := fm.ResolveFace(r)
	if gt != nil && lang != "" && script == language.Han {
		if _, ok := gt.NominalGlyph(r); !ok {
			if id, ok := language.NewLangID(language.NewLanguage(lang)); ok {
				if byLang := fm.ResolveFaceForLang(id); byLang != nil {
					gt = byLang
				}
			}
		}
	}
	if gt == nil {
		return nil
	}
	return fc.wrapSystem(fm, gt)
}

func (fc *FontConfiguration) wrapSystem(fm *fontscan.FontMap, gt *font.Face) *Face {
	loc := fm.FontLocation(gt.Font)
	if face, ok := fc.systemSet[loc]; ok {
		return face
	}
	face := fc.openSystem(loc, gt)
	fc.systemSet[loc] = face
	return face
}

func (fc *FontConfiguration) openSystem(loc fontscan.Location, gt *font.Face) *Face {
	data, err := os.ReadFile(loc.File)
	if err == nil {
		var loaders []*opentype.Loader
		if loaders, err = opentype.NewLoaders(bytes.NewReader(data)); err == nil && int(loc.Index) < len(loaders) {
			face := newFace(data, int(loc.Index), loaders[loc.Index], gt.Font)
			face.GoText = gt
			return face
		}
	}
	logging.Warnf("Cannot read font file %q: %v", loc.File, err)
	return &Face{ID: loc.File, GoText: gt, Family: gt.Describe().Family}
}
