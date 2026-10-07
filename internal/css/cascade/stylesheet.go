package cascade

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/media"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/selector"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// supportedPseudoElements are the pseudo-elements rules can style; rules
// for the others are reported.
var supportedPseudoElements = map[string]bool{
	"": true, "before": true, "after": true, "marker": true,
	"first-line": true, "first-letter": true, "footnote-call": true,
	"footnote-marker": true, "note-call": true, "note-marker": true,
	"note-callback": true,
}

// Environment is what loading stylesheets needs and fills: the device, the
// fetcher, and the registries that at-rules add fonts, counter styles and
// color profiles to. The zero value loads stylesheets for print media
// without fetching anything.
type Environment struct {
	// MediaType is the media type of the device, "print" when empty.
	MediaType string
	Fetcher   urls.Fetcher
	// FontConfig receives the @font-face rules. Nil ignores them.
	FontConfig *fonts.FontConfiguration
	// CounterStyle receives the @counter-style rules. Nil ignores them.
	CounterStyle counters.Registry
	// ColorProfiles receives the @color-profile rules. Nil ignores them.
	ColorProfiles map[string]*ColorProfile
	// PageRules and Layers, when not nil, are shared by the stylesheets
	// loaded with the environment instead of each having its own.
	PageRules *[]PageRule
	Layers    *[]string

	fatal error
}

// Err returns the first fatal fetching error met while loading, which
// stops the rendering (see urls.IsFatal).
func (env *Environment) Err() error { return env.fatal }

// check records err if it is fatal.
func (env *Environment) check(err error) {
	if env.fatal == nil && urls.IsFatal(err) {
		env.fatal = err
	}
}

func (env *Environment) mediaType() string {
	if env == nil || env.MediaType == "" {
		return "print"
	}
	return env.MediaType
}

// Stylesheet is a parsed and preprocessed stylesheet: its style rules in a
// selector matcher, its page rules and its cascade layers. Rules imported
// with @import are part of the stylesheet that imports them.
type Stylesheet struct {
	BaseURL string
	Matcher *selector.Matcher

	env       *Environment
	pageRules *[]PageRule
	layers    *[]string
	selectors int
}

// PageRules returns the @page rules of the stylesheet, margin rules
// included, in source order.
func (s *Stylesheet) PageRules() []PageRule { return *s.pageRules }

// Layers returns the cascade layers in layer order, from the lowest
// priority.
func (s *Stylesheet) Layers() []string { return *s.layers }

// ruleBlock is the payload of the selectors added to the matcher: the
// declarations of the rule and its cascade layer.
type ruleBlock struct {
	declarations []validation.Declaration
	layer        string
	layered      bool
}

func newStylesheet(baseURL string, env *Environment) *Stylesheet {
	if env == nil {
		env = &Environment{}
	}
	s := &Stylesheet{BaseURL: baseURL, Matcher: &selector.Matcher{}, env: env}
	s.pageRules, s.layers = env.PageRules, env.Layers
	if s.pageRules == nil {
		s.pageRules = &[]PageRule{}
	}
	if s.layers == nil {
		s.layers = &[]string{}
	}
	return s
}

// ParseStylesheet parses and preprocesses CSS text whose relative URLs
// resolve against baseURL.
func ParseStylesheet(css, baseURL string, env *Environment) *Stylesheet {
	s := newStylesheet(baseURL, env)
	s.preprocess(syntax.ParseStylesheet(css, false, false), false, "", false)
	return s
}

// LoadStylesheet fetches a stylesheet and preprocesses it. With checkType
// set, a resource that is not text/css is reported and gives an empty
// stylesheet. The error is the fetching error.
func LoadStylesheet(url string, env *Environment, checkType bool) (*Stylesheet, error) {
	return loadStylesheet(url, env, checkType, nil, "", false)
}

// loadStylesheet fetches a stylesheet. An imported stylesheet adds its rules
// to the stylesheet that imports it, in the given layer.
func loadStylesheet(url string, env *Environment, checkType bool, into *Stylesheet, layer string, layered bool) (*Stylesheet, error) {
	if env == nil {
		env = &Environment{}
	}
	fetcher := env.Fetcher
	if fetcher == nil {
		fetcher = urls.NewDefaultFetcher()
	}
	resp, err := fetcher.Fetch(context.Background(), url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	s := into
	if s == nil {
		s = newStylesheet(resp.URL, env)
	}
	if checkType && resp.MIMEType != "text/css" {
		logging.Errorf("Unsupported stylesheet type %s for %s", resp.MIMEType, resp.URL)
		return s, nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &urls.FetchError{URL: url, Err: err}
	}
	rules, _ := syntax.ParseStylesheetBytes(data, resp.Encoding, nil, false, false)
	baseURL := s.BaseURL
	s.BaseURL = resp.URL
	s.preprocess(rules, false, layer, layered)
	if into != nil {
		s.BaseURL = baseURL
	}
	return s, nil
}

// atRuleRepr names an at-rule in messages.
func atRuleRepr(rule *syntax.AtRule) string {
	return "<AtRule @" + rule.AtKeyword + " … >"
}

// preprocess handles the rules of a stylesheet (CSS Cascade 5 §6, CSS
// Syntax 3 §9): style rules go to the matcher and at-rules to the page
// rules, the layers and the registries of the environment. @import is only
// allowed before other rules.
func (s *Stylesheet) preprocess(rules []syntax.Node, ignoreImports bool, layer string, layered bool) {
	for _, node := range rules {
		switch rule := node.(type) {
		case *syntax.ParseError:
			line, column := rule.Pos()
			logging.Warnf("Parse error at %d:%d: %s", line, column, rule.Message)
		case *syntax.QualifiedRule:
			if s.styleRule(rule, layer, layered) {
				ignoreImports = true
			}
		case *syntax.AtRule:
			if rule.Content == nil && rule.LowerAtKeyword != "import" && rule.LowerAtKeyword != "layer" {
				line, column := rule.Pos()
				logging.Warnf("Unknown empty rule %s at %d:%d", atRuleRepr(rule), line, column)
				continue
			}
			ignoreImports = s.atRule(rule, ignoreImports, layer, layered)
		}
	}
}

// styleRule adds the selectors of a qualified rule to the matcher. It
// returns false when the rule is reported as invalid.
func (s *Stylesheet) styleRule(rule *syntax.QualifiedRule, layer string, layered bool) bool {
	contents := syntax.ParseBlocksContents(rule.Content, false, false)
	declarations, err := validation.PreprocessRuleDeclarations(s.BaseURL, contents, rule.Prelude)
	if err != nil {
		logging.Warnf("Invalid or unsupported selector, %s", err.Error())
		return false
	}
	for start := 0; start < len(declarations); {
		end := start + 1
		for end < len(declarations) && sameSelectors(declarations[end].Selectors, declarations[start].Selectors) {
			end++
		}
		block := &ruleBlock{declarations: declarations[start:end:end], layer: layer, layered: layered}
		for _, sel := range declarations[start].Selectors {
			s.Matcher.Add(sel, block)
			s.selectors++
			if !supportedPseudoElements[sel.PseudoElement] {
				prelude := syntax.Serialize(rule.Prelude)
				if strings.HasPrefix(sel.PseudoElement, "-") {
					logging.Debugf("Invalid or unsupported selector, '%s', ignored prefixed pseudo-element: %s", prelude, sel.PseudoElement)
				} else {
					logging.Warnf("Invalid or unsupported selector, '%s', unknown pseudo-element: %s", prelude, sel.PseudoElement)
				}
				return false
			}
		}
		start = end
	}
	return true
}

// sameSelectors reports whether two declarations come from the same rule.
func sameSelectors(a, b []selector.Selector) bool {
	if len(a) != len(b) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0]
}

// atRule handles an at-rule and returns whether later @import rules are
// ignored.
func (s *Stylesheet) atRule(rule *syntax.AtRule, ignoreImports bool, layer string, layered bool) bool {
	line, column := rule.Pos()
	switch rule.LowerAtKeyword {
	case "import":
		if ignoreImports {
			logging.Warnf("@import rule %s not at the beginning of the the whole rule was ignored at %d:%d.",
				logging.Repr(syntax.Serialize(rule.Prelude)), line, column)
			return ignoreImports
		}
		s.importRule(rule, layer, layered)
		return ignoreImports
	case "media":
		query, ok := media.Parse(rule.Prelude)
		if !ok {
			logging.Warnf("Invalid media type %s the whole @media rule was ignored at %d:%d.",
				logging.Repr(syntax.Serialize(rule.Prelude)), line, column)
			return ignoreImports
		}
		if media.Evaluate(query, s.env.mediaType()) {
			s.preprocess(syntax.ParseRuleList(rule.Content, false, false), true, layer, layered)
		}
		return ignoreImports
	case "page":
		return s.pageRule(rule) || ignoreImports
	case "font-face":
		s.fontFace(rule)
		return true
	case "color-profile":
		s.colorProfile(rule)
		return true
	case "counter-style":
		return s.counterStyle(rule, ignoreImports)
	case "layer":
		s.layerRule(rule, layer, layered)
		return ignoreImports
	}
	logging.Warnf("Unknown rule %s at %d:%d", atRuleRepr(rule), line, column)
	return ignoreImports
}

func (s *Stylesheet) importRule(rule *syntax.AtRule, layer string, layered bool) {
	line, column := rule.Pos()
	prelude := syntax.Serialize(rule.Prelude)
	tokens := tk.RemoveWhitespace(rule.Prelude)
	url, found, invalidSyntax := "", false, true
	if len(tokens) > 0 {
		if str, ok := tokens[0].(*syntax.StringToken); ok {
			invalidSyntax = false
			url, found = urls.JoinChecked(s.BaseURL, str.Value, false, fmt.Sprintf("@import at %d:%d", line, column))
		} else if u, ok, _ := tk.GetURL(tokens[0], s.BaseURL); ok {
			if external, ok := u.External(); ok {
				invalidSyntax = false
				url, found = external, true
			}
		}
	}
	if !found {
		if invalidSyntax {
			logging.Warnf("Invalid @import rule %s, the whole rule was ignored at %d:%d.", logging.Repr(prelude), line, column)
		}
		return
	}

	newLayer, newLayered := "", false
	next := tokens[1:]
	if len(next) > 0 {
		if fn, ok := next[0].(*syntax.FunctionBlock); ok && fn.Name == "layer" {
			next = next[1:]
			name, ok := parseLayer(fn.Arguments)
			if !ok {
				logging.Warnf("Invalid layer name %s the whole @import rule was ignored at %d:%d.",
					logging.Repr(fn.Serialize()), line, column)
				return
			}
			newLayer, newLayered = name, true
		} else if ident, ok := next[0].(*syntax.IdentToken); ok && ident.Value == "layer" {
			next = next[1:]
			newLayer, newLayered = fmt.Sprintf("@anonymous%d", len(*s.layers)), true
		}
		if newLayered {
			if layered {
				newLayer = layer + "." + newLayer
			}
			addLayer(newLayer, s.layers)
		}
	}

	query, ok := media.Parse(next)
	if !ok {
		logging.Warnf("Invalid media type %s the whole @import rule was ignored at %d:%d.", logging.Repr(prelude), line, column)
		return
	}
	if !media.Evaluate(query, s.env.mediaType()) {
		return
	}
	if _, err := loadStylesheet(url, s.env, false, s, newLayer, newLayered); err != nil {
		s.env.check(err)
		logging.Errorf("Failed to load stylesheet at %s : %s", url, err)
	}
}

func (s *Stylesheet) pageRule(rule *syntax.AtRule) bool {
	line, column := rule.Pos()
	selectors, ok := ParsePageSelectors(rule.Prelude)
	if !ok {
		logging.Warnf("Unsupported @page selector %s, the whole @page rule was ignored at %d:%d.",
			logging.Repr(syntax.Serialize(rule.Prelude)), line, column)
		return false
	}
	for _, sel := range selectors {
		content := syntax.ParseBlocksContents(rule.Content, false, false)
		if declarations := validation.PreprocessDeclarations(s.BaseURL, content); len(declarations) > 0 {
			*s.pageRules = append(*s.pageRules, PageRule{Selectors: []PageSelector{sel}, Declarations: declarations})
		}
		for _, node := range content {
			margin, ok := node.(*syntax.AtRule)
			if !ok {
				continue
			}
			mLine, mColumn := margin.Pos()
			switch {
			case !PageMarginBoxes[margin.LowerAtKeyword]:
				logging.Warnf("Unknown @page rule %s at %d:%d", atRuleRepr(margin), mLine, mColumn)
				continue
			case margin.Content == nil:
				logging.Warnf("Empty @page rule %s at %d:%d", atRuleRepr(margin), mLine, mColumn)
				continue
			case len(tk.RemoveWhitespace(margin.Prelude)) > 0:
				logging.Warnf("Invalid prelude %s for @page rule %s at %d:%d",
					logging.Repr(syntax.Serialize(margin.Prelude)), atRuleRepr(margin), mLine, mColumn)
				continue
			}
			declarations := validation.PreprocessDeclarations(s.BaseURL, syntax.ParseBlocksContents(margin.Content, false, false))
			if len(declarations) > 0 {
				*s.pageRules = append(*s.pageRules, PageRule{
					Selectors:    []PageSelector{sel},
					MarginBox:    "@" + margin.LowerAtKeyword,
					Declarations: declarations,
				})
			}
		}
	}
	return true
}

// fontFace registers the font of an @font-face rule (CSS Fonts 4 §4).
func (s *Stylesheet) fontFace(rule *syntax.AtRule) {
	line, column := rule.Pos()
	content := syntax.ParseBlocksContents(rule.Content, false, false)
	descriptors := map[string]any{}
	for _, d := range validation.PreprocessDescriptors("font-face", s.BaseURL, content) {
		descriptors[d.Name] = d.Value
	}
	for _, key := range []string{"src", "font-family"} {
		if _, ok := descriptors[key]; !ok {
			logging.Warnf("Missing %s descriptor in '@font-face' rule at %d:%d", key, line, column)
			return
		}
	}
	if s.env.FontConfig == nil {
		return
	}
	face := fontFaceRule(descriptors)
	if err := s.env.FontConfig.AddFontFace(face, s.env.fetchBytes); err != nil {
		logging.Warnf("Font-face %s cannot be loaded", logging.Repr(face.Family))
	}
}

// fontFaceRule converts validated descriptors into a font-face rule.
func fontFaceRule(descriptors map[string]any) fonts.FontFaceRule {
	var rule fonts.FontFaceRule
	rule.Family, _ = descriptors["font-family"].(string)
	sources, _ := descriptors["src"].([]validation.FontSource)
	for _, src := range sources {
		if src.Local {
			rule.Src = append(rule.Src, fonts.Source{Kind: fonts.SourceLocal, Value: src.Name})
		} else {
			rule.Src = append(rule.Src, fonts.Source{Kind: fonts.SourceURL, Value: src.Link.Target})
		}
	}
	if style, ok := descriptors["font-style"].(props.Keyword); ok {
		rule.Style = string(style)
	}
	switch w := descriptors["font-weight"].(type) {
	case props.Keyword:
		rule.Weight = 400
		if w == "bold" {
			rule.Weight = 700
		}
	case props.Integer:
		rule.Weight = int(w)
	}
	if stretch, ok := descriptors["font-stretch"].(props.Keyword); ok {
		rule.Stretch = string(stretch)
	}
	if settings, ok := descriptors["font-feature-settings"].(props.FontSettings); ok {
		rule.FeatureSettings = settings
	} else {
		rule.FeatureSettings = props.FontSettings{Normal: true}
	}
	variants, _ := descriptors["font-variant"].([]validation.FontVariant)
	for _, v := range variants {
		switch v.Name {
		case "font-variant-ligatures":
			rule.Variant.Ligatures, _ = v.Value.(props.Keywords)
		case "font-variant-position":
			rule.Variant.Position, _ = v.Value.(props.Keyword)
		case "font-variant-caps":
			rule.Variant.Caps, _ = v.Value.(props.Keyword)
		case "font-variant-numeric":
			rule.Variant.Numeric, _ = v.Value.(props.Keywords)
		case "font-variant-alternates":
			rule.Variant.Alternates, _ = v.Value.(props.Keyword)
		case "font-variant-east-asian":
			rule.Variant.EastAsian, _ = v.Value.(props.Keywords)
		}
	}
	ranges, _ := descriptors["unicode-range"].([]validation.UnicodeRange)
	for _, r := range ranges {
		rule.UnicodeRange = append(rule.UnicodeRange, fonts.Range{Start: rune(r.Start), End: rune(r.End)})
	}
	return rule
}

// fetchBytes reads a resource such as a font file with the fetcher.
func (env *Environment) fetchBytes(url string) ([]byte, error) {
	f := env.Fetcher
	if f == nil {
		f = urls.NewDefaultFetcher()
	}
	resp, err := f.Fetch(context.Background(), url, nil)
	if err != nil {
		env.check(err)
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// colorProfileName returns the name an @color-profile rule defines: a
// dashed identifier or device-cmyk (CSS Color 5 §11.2).
func colorProfileName(prelude []syntax.Node) (string, bool) {
	tokens := tk.RemoveWhitespace(prelude)
	if len(tokens) != 1 {
		return "", false
	}
	ident, ok := tokens[0].(*syntax.IdentToken)
	if !ok || !(strings.HasPrefix(ident.Value, "--") || ident.Value == "device-cmyk") {
		return "", false
	}
	return ident.Value, true
}

func (s *Stylesheet) colorProfile(rule *syntax.AtRule) {
	line, column := rule.Pos()
	prelude := syntax.Serialize(rule.Prelude)
	name, ok := colorProfileName(rule.Prelude)
	if !ok {
		logging.Warnf("Invalid color profile name %s, the whole @color-profile rule was ignored at %d:%d.",
			logging.Repr(prelude), line, column)
		return
	}
	content := syntax.ParseBlocksContents(rule.Content, false, false)
	var src *urls.Link
	intent := "relative-colorimetric"
	var components [][]string
	for _, d := range validation.PreprocessDescriptors("color-profile", s.BaseURL, content) {
		switch d.Name {
		case "src":
			link := d.Value.(urls.Link)
			src = &link
		case "rendering-intent":
			intent = d.Value.(string)
		case "components":
			components = d.Value.([][]string)
		default:
			logging.Warnf("Unknown descriptor %s for profile named %s at %d:%d.",
				logging.Repr(d.Name), logging.Repr(prelude), line, column)
		}
	}
	if src == nil {
		logging.Warnf("No source for profile named %s, the whole @color-profile rule was ignored at %d:%d.",
			logging.Repr(prelude), line, column)
		return
	}
	data, err := s.env.fetchBytes(src.Target)
	var profile *ColorProfile
	if err == nil {
		profile, err = newColorProfile(data, src.Target, intent, components)
	}
	if err != nil {
		logging.Warnf("Invalid profile file for profile named %s, the whole @color-profile rule was ignored at %d:%d.",
			logging.Repr(strings.TrimSpace(prelude)), line, column)
		return
	}
	if s.env.ColorProfiles != nil {
		s.env.ColorProfiles[name] = profile
	}
}

// counterStyle registers an @counter-style rule (CSS Counter Styles 3 §3).
func (s *Stylesheet) counterStyle(rule *syntax.AtRule, ignoreImports bool) bool {
	line, column := rule.Pos()
	registry := s.env.CounterStyle
	if registry == nil {
		registry = counters.Registry{}
	}
	name, ok := counters.ParseName(rule.Prelude, registry)
	if !ok {
		logging.Warnf("Invalid counter style name %s, the whole @counter-style rule was ignored at %d:%d.",
			logging.Repr(syntax.Serialize(rule.Prelude)), line, column)
		return ignoreImports
	}

	content := syntax.ParseBlocksContents(rule.Content, false, false)
	style := &counters.Style{}
	for _, d := range validation.PreprocessDescriptors("counter-style", s.BaseURL, content) {
		switch v := d.Value.(type) {
		case validation.CounterSystem:
			style.System = &v
		case validation.CounterSymbol:
			if d.Name == "prefix" {
				style.Prefix = &v
			} else {
				style.Suffix = &v
			}
		case validation.CounterPad:
			style.Pad = &v
		case []validation.CounterRange:
			style.Range = v
		case []validation.CounterPad:
			style.AdditiveSymbols = v
		case string:
			style.Fallback = v
		case []validation.CounterSymbol:
			if d.Name == "negative" {
				style.Negative = v
			} else {
				style.Symbols = v
			}
		}
	}

	if style.System == nil || style.System.Kind != "extends" {
		system := "symbolic"
		if style.System != nil {
			system = style.System.Kind
		}
		switch system {
		case "cyclic", "fixed", "symbolic":
			if len(style.Symbols) < 1 {
				logging.Warnf("In counter style %s at %d:%d, counter style %s needs at least one symbol",
					logging.Repr(name), line, column, logging.Repr(system))
				return true
			}
		case "alphabetic", "numeric":
			if len(style.Symbols) < 2 {
				logging.Warnf("In counter style %s at %d:%d, counter style %s needs at least two symbols",
					logging.Repr(name), line, column, logging.Repr(system))
				return true
			}
		case "additive":
			if len(style.AdditiveSymbols) < 2 {
				logging.Warnf(`In counter style %s at %d:%d, counter style "additive" needs at least two additive symbols`,
					logging.Repr(name), line, column)
				return true
			}
		}
	}
	registry[name] = style
	return true
}

// layerRule handles @layer: a statement declaring the order of layers, or a
// block of rules in a named or anonymous layer (CSS Cascade 5 §6.4.2).
func (s *Stylesheet) layerRule(rule *syntax.AtRule, layer string, layered bool) {
	line, column := rule.Pos()
	prelude := tk.RemoveWhitespace(rule.Prelude)
	var names []string
	if len(prelude) > 0 {
		for _, tokens := range tk.SplitOnComma(prelude) {
			name, ok := parseLayer(tokens)
			if !ok {
				logging.Warnf("Unsupported @layer selector %s, the whole @layer rule was ignored at %d:%d.",
					logging.Repr(syntax.Serialize(rule.Prelude)), line, column)
				return
			}
			if layered {
				name = layer + "." + name
			}
			names = append(names, name)
		}
	}
	if len(names) > 1 {
		if len(rule.Content) > 0 {
			logging.Warnf("@layer rule with multiple layer names, the whole @layer rule was ignored at %d:%d.", line, column)
			return
		}
		for _, name := range names {
			addLayer(name, s.layers)
		}
		return
	}

	var name string
	if len(names) == 1 {
		name = names[0]
	} else {
		name = fmt.Sprintf("@anonymous%d", len(*s.layers))
		if layered {
			name = layer + "." + name
		}
	}
	addLayer(name, s.layers)
	if rule.Content == nil {
		return
	}
	s.preprocess(syntax.ParseRuleList(rule.Content, false, false), true, name, true)
}
