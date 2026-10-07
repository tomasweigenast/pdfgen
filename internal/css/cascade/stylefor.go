package cascade

import (
	"fmt"
	"math"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Origin is where a stylesheet comes from (CSS Cascade 5 §6.3).
type Origin uint8

const (
	// UserAgent is the default stylesheet of the renderer.
	UserAgent Origin = iota
	// User is a stylesheet given by the user of the renderer.
	User
	// Author is a stylesheet of the document.
	Author
)

// precedence orders declarations by origin and importance (CSS Cascade 5
// §6.1): user agent, user, author, important author, important user.
func precedence(origin Origin, important bool) int {
	switch {
	case origin == UserAgent:
		return 1
	case origin == User && !important:
		return 2
	case origin == Author && !important:
		return 3
	case origin == Author:
		return 4
	}
	return 5
}

// SheetEntry is a stylesheet with its origin. A non-nil Specificity
// replaces the specificity of every selector of the stylesheet, as for
// presentational hints.
type SheetEntry struct {
	Sheet       *Stylesheet
	Origin      Origin
	Specificity *[3]int
}

// Options configure ComputeAll.
type Options struct {
	// Environment loads the stylesheets of the document and receives their
	// fonts, counter styles and color profiles. Missing registries are
	// created.
	Environment *Environment
	// UserStylesheets are the stylesheets given by the user.
	UserStylesheets []*Stylesheet
	// UAStylesheets replace the default user agent stylesheets when not
	// nil.
	UAStylesheets []*Stylesheet
	// PresentationalHints turns presentational HTML attributes into styles.
	PresentationalHints bool
	// Forms adds the user agent styles of form controls.
	Forms bool
	// Targets receives the anchors of the document.
	Targets *targets.Collector
}

type styleKey struct {
	element *html.Element
	pseudo  string
}

func (p PageType) key() string { return fmt.Sprintf("%#v", p) }

type pageKey struct {
	page   string
	pseudo string
}

// StyleFor holds the computed styles of the elements, pseudo-elements and
// page types of a document.
type StyleFor struct {
	ctx    *computed.Context
	doc    *html.Document
	sheets []SheetEntry

	cascaded      map[styleKey]computed.Cascaded
	cascadedOrder []styleKey
	styles        map[styleKey]*computed.Style

	pageCascaded map[pageKey]computed.Cascaded
	pageOrder    []pageKey
	pageStyles   map[pageKey]*computed.Style
}

// ComputeAll finds the stylesheets of a document and computes the styles of
// its elements and pseudo-elements. The error is a fatal fetching error.
func ComputeAll(doc *html.Document, opts Options) (*StyleFor, error) {
	env := opts.Environment
	if env == nil {
		env = &Environment{}
	}
	if env.CounterStyle == nil {
		env.CounterStyle = counters.Registry{}
	}
	if env.FontConfig == nil {
		env.FontConfig = fonts.NewFontConfiguration()
	}
	for name, style := range UACounterStyle() {
		env.CounterStyle[name] = style
	}

	var sheets []SheetEntry
	ua := opts.UAStylesheets
	if ua == nil {
		ua = UAStylesheets(opts.Forms)
	}
	for _, s := range ua {
		sheets = append(sheets, SheetEntry{Sheet: s, Origin: UserAgent})
	}
	if opts.PresentationalHints {
		sheets = append(sheets, SheetEntry{Sheet: PresentationalHintsStylesheet(), Origin: Author, Specificity: &[3]int{}})
	}
	for _, s := range FindStylesheets(doc, env) {
		sheets = append(sheets, SheetEntry{Sheet: s, Origin: Author})
	}
	for _, s := range opts.UserStylesheets {
		sheets = append(sheets, SheetEntry{Sheet: s, Origin: User})
	}
	if err := env.Err(); err != nil {
		return nil, err
	}
	return NewStyleFor(doc, sheets, opts.PresentationalHints, computed.NewContext(env.FontConfig), opts.Targets), nil
}

// apply cascades a declaration: it replaces the current one unless that
// one is heavier.
func apply(style computed.Cascaded, d validation.Declaration, w computed.Weight) {
	if old, ok := style[d.Name]; !ok || old.Weight.Compare(w) <= 0 {
		style[d.Name] = computed.Declared{Value: d.Value, Weight: w}
	}
}

// NewStyleFor cascades the declarations of the stylesheets and of the style
// attributes for every element, then computes the styles of the elements,
// in tree order, and of their pseudo-elements.
func NewStyleFor(doc *html.Document, sheets []SheetEntry, presentationalHints bool, ctx *computed.Context, collector *targets.Collector) *StyleFor {
	logging.Progressf("Step 3 - Applying CSS")
	sf := &StyleFor{
		ctx:          ctx,
		doc:          doc,
		sheets:       sheets,
		cascaded:     map[styleKey]computed.Cascaded{},
		styles:       map[styleKey]*computed.Style{},
		pageCascaded: map[pageKey]computed.Cascaded{},
		pageStyles:   map[pageKey]*computed.Style{},
	}

	unlayered := math.Inf(1)
	withAttributes := map[*html.Element]bool{}
	for _, a := range styleAttributes(doc, presentationalHints) {
		style := sf.cascadedFor(styleKey{a.element, ""})
		withAttributes[a.element] = true
		for _, d := range a.declarations {
			apply(style, d, computed.Weight{Precedence: precedence(Author, d.Important), Layer: unlayered, Specificity: a.specificity})
		}
	}

	// Styles are computed in tree order, so that parents come before their
	// children. Elements with the same parent style, attributes and rules
	// share their style.
	shared := map[string]*computed.Style{}
	var key strings.Builder
	for _, el := range doc.Elements() {
		parent := el.ParentElement()
		key.Reset()
		if parent != nil {
			fmt.Fprintf(&key, "%p", sf.styles[styleKey{parent, ""}])
		}
		if withAttributes[el] {
			fmt.Fprintf(&key, "|%p", el)
		}
		for i := range sheets {
			entry := &sheets[i]
			for _, m := range entry.Sheet.Matcher.Match(el) {
				block := m.Payload.(*ruleBlock)
				layer := unlayered
				if block.layered {
					layer = float64(indexOf(entry.Sheet.Layers(), block.layer))
				}
				fmt.Fprintf(&key, "|%p%v%p%v", entry.Sheet, m.Specificity, block, layer)
				specificity := m.Specificity
				if entry.Specificity != nil {
					specificity = *entry.Specificity
				}
				style := sf.cascadedFor(styleKey{el, m.PseudoElement})
				for _, d := range block.declarations {
					apply(style, d, computed.Weight{Precedence: precedence(entry.Origin, d.Important), Layer: layer, Specificity: specificity})
				}
			}
		}
		if s, ok := shared[key.String()]; ok {
			sf.styles[styleKey{el, ""}] = s
			continue
		}
		sf.setComputedStyles(el, parent, "", collector)
		shared[key.String()] = sf.styles[styleKey{el, ""}]
	}

	// Pseudo-elements inherit from their element, so they come last. Only
	// those with declarations exist.
	for _, k := range sf.cascadedOrder {
		if k.pseudo != "" {
			sf.setComputedStyles(k.element, k.element, k.pseudo, collector)
		}
	}
	sf.cascaded, sf.cascadedOrder = nil, nil
	return sf
}

func (sf *StyleFor) cascadedFor(k styleKey) computed.Cascaded {
	style, ok := sf.cascaded[k]
	if !ok {
		style = computed.Cascaded{}
		sf.cascaded[k] = style
		sf.cascadedOrder = append(sf.cascadedOrder, k)
	}
	return style
}

// setComputedStyles makes the style of an element or pseudo-element from
// its cascaded values, parent being the element it inherits from.
func (sf *StyleFor) setComputedStyles(el, parent *html.Element, pseudo string, collector *targets.Collector) {
	var parentStyle, root *computed.Style
	if el != sf.doc.Root || pseudo != "" {
		parentStyle = sf.styles[styleKey{parent, ""}]
		root = sf.styles[styleKey{sf.doc.Root, ""}]
	}
	k := styleKey{el, pseudo}
	style := computed.NewStyle(sf.ctx, parentStyle, root, sf.cascaded[k], pseudo, sf.doc.BaseURL)
	sf.styles[k] = style
	if collector != nil {
		if anchor := style.Anchor(); anchor != "" {
			collector.CollectAnchor(string(anchor))
		}
	}
}

// Context returns the context shared by the computed styles, whose page
// sizes the layout sets once the first page is made.
func (sf *StyleFor) Context() *computed.Context { return sf.ctx }

// Sheets returns the stylesheets of the document with their origins.
func (sf *StyleFor) Sheets() []SheetEntry { return sf.sheets }

// Get returns the computed style of an element, or of one of its
// pseudo-elements, or nil for a pseudo-element without declarations. The
// padding of tables with collapsing borders and the margins of internal
// table boxes are set to zero, as they do not apply (CSS 2.1 §17.6.2,
// §17.5).
func (sf *StyleFor) Get(el *html.Element, pseudo string) *computed.Style {
	style := sf.styles[styleKey{el, pseudo}]
	if style == nil {
		return nil
	}
	display := style.Display()
	if containsKeyword(display, "table") && style.BorderCollapse() == "collapse" {
		for _, id := range []props.PropID{props.PropPaddingTop, props.PropPaddingBottom, props.PropPaddingLeft, props.PropPaddingRight} {
			style.Set(id, props.Px(0))
		}
	}
	if len(display) == 1 && strings.HasPrefix(string(display[0]), "table-") && display[0] != "table-caption" {
		for _, id := range []props.PropID{props.PropMarginTop, props.PropMarginBottom, props.PropMarginLeft, props.PropMarginRight} {
			style.Set(id, props.Px(0))
		}
	}
	return style
}

func containsKeyword(display props.Display, k props.Keyword) bool {
	for _, d := range display {
		if d == k {
			return true
		}
	}
	return false
}

// AddPageDeclarations cascades the declarations of the @page rules that
// match a page type, for the page and its margin boxes.
func (sf *StyleFor) AddPageDeclarations(page PageType) {
	pk := page.key()
	for _, entry := range sf.sheets {
		for _, rule := range entry.Sheet.PageRules() {
			for _, sel := range rule.Selectors {
				if !sel.Matches(page) {
					continue
				}
				specificity := sel.Specificity
				if entry.Specificity != nil {
					specificity = *entry.Specificity
				}
				k := pageKey{pk, rule.MarginBox}
				style, ok := sf.pageCascaded[k]
				if !ok {
					style = computed.Cascaded{}
					sf.pageCascaded[k] = style
					sf.pageOrder = append(sf.pageOrder, k)
				}
				for _, d := range rule.Declarations {
					apply(style, d, computed.Weight{Precedence: precedence(entry.Origin, d.Important), Layer: math.Inf(1), Specificity: specificity})
				}
			}
		}
	}
}

// SetPageTypeComputedStyles computes the styles of a page type and of its
// margin boxes. Pages inherit from the root element, and margin boxes from
// their page.
func (sf *StyleFor) SetPageTypeComputedStyles(page PageType) {
	sf.AddPageDeclarations(page)
	pk := page.key()
	root := sf.styles[styleKey{sf.doc.Root, ""}]
	pageStyle := computed.NewStyle(sf.ctx, root, root, sf.pageCascaded[pageKey{pk, ""}], "", sf.doc.BaseURL)
	sf.pageStyles[pageKey{pk, ""}] = pageStyle
	for _, k := range sf.pageOrder {
		if k.page == pk && k.pseudo != "" {
			sf.pageStyles[k] = computed.NewStyle(sf.ctx, pageStyle, root, sf.pageCascaded[k], k.pseudo, sf.doc.BaseURL)
		}
	}
}

// Page returns the style of a page type, or of one of its margin boxes such
// as "@top-left", once SetPageTypeComputedStyles has computed them. It is
// nil for a margin box without declarations.
func (sf *StyleFor) Page(page PageType, marginBox string) *computed.Style {
	return sf.pageStyles[pageKey{page.key(), marginBox}]
}

// SetComputedStyles computes the style of an element or pseudo-element
// that inherits from parent, replacing any existing one, from the
// declarations cascaded for it.
func (sf *StyleFor) SetComputedStyles(el, parent *html.Element, pseudo string) {
	sf.setComputedStyles(el, parent, pseudo, nil)
}
