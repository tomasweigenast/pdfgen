package computed

import (
	"cmp"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
)

// Weight orders the declarations of a property in the cascade (CSS Cascade
// 5 §6): origin and importance first, then the cascade layer, then the
// specificity of the selector. Of two declarations with equal weights, the
// later one wins.
type Weight struct {
	// Precedence combines origin and importance, from 1 for user agent
	// declarations to 5 for important user declarations.
	Precedence int
	// Layer is the position of the cascade layer in the layer order, and
	// +Inf for declarations outside any layer.
	Layer       float64
	Specificity [3]int
}

// Compare returns -1, 0 or +1 when w is lighter than, equal to or heavier
// than o.
func (w Weight) Compare(o Weight) int {
	if c := cmp.Compare(w.Precedence, o.Precedence); c != 0 {
		return c
	}
	if c := cmp.Compare(w.Layer, o.Layer); c != 0 {
		return c
	}
	for i := range w.Specificity {
		if c := cmp.Compare(w.Specificity[i], o.Specificity[i]); c != 0 {
			return c
		}
	}
	return 0
}

// Declared is the value of the declaration that won the cascade for a
// property, with its weight.
type Declared struct {
	Value  props.Value
	Weight Weight
}

// Cascaded maps the names of the properties declared for an element or a
// pseudo-element, longhands and custom properties, to their cascaded value.
type Cascaded map[string]Declared

type styleKind uint8

const (
	elementStyle styleKind = iota
	anonymousStyle
	initialStyle
)

// Style holds the computed values of the properties of an element, a
// pseudo-element, a page or an anonymous box. Values are computed the first
// time they are read, from the cascaded values, the parent style and the
// initial values. The typed getters in style_gen.go read from it.
type Style struct {
	kind      styleKind
	values    []props.Value
	computed  bitset
	computing bitset
	custom    map[string]props.Tokens
	specified map[props.PropID]props.Value

	parent, root *Style
	cascaded     Cascaded
	pseudoType   string
	baseURL      string
	ctx          *Context
	anonymous    *Style
}

func (s *Style) init() {
	n := props.NumProps()
	s.values = make([]props.Value, n)
	s.computed = newBitset(n)
	s.computing = newBitset(n)
	s.custom = map[string]props.Tokens{}
	s.specified = map[props.PropID]props.Value{}
}

// NewStyle returns the style of an element or pseudo-element whose
// declarations are cascaded. parent is the style of the parent element, or
// of the element for a pseudo-element, and nil for the root element. root
// is the style of the root element; it is ignored for the root element,
// whose root style holds the initial values.
func NewStyle(ctx *Context, parent, root *Style, cascaded Cascaded, pseudoType, baseURL string) *Style {
	if parent == nil {
		root = ctx.initial
	}
	s := &Style{
		kind:       elementStyle,
		parent:     parent,
		root:       root,
		cascaded:   cascaded,
		pseudoType: pseudoType,
		baseURL:    baseURL,
		ctx:        ctx,
	}
	s.init()
	return s
}

// New returns a style without declarations whose parent style is parent,
// which may be nil. It shares the context of its parent, or has a new one
// using the installed fonts.
func New(parent *Style) *Style {
	if parent == nil {
		return NewStyle(NewContext(nil), nil, nil, nil, "", "")
	}
	return NewStyle(parent.ctx, parent, parent.root, nil, "", parent.baseURL)
}

// Anonymous returns the style of the anonymous boxes whose parent box has
// style s: inherited properties come from s and the others have their
// initial value (CSS 2.1 §9.2.1.1). The style is made once and shared.
func (s *Style) Anonymous() *Style {
	if s.anonymous == nil {
		a := &Style{
			kind:    anonymousStyle,
			parent:  s,
			root:    s.root,
			baseURL: s.baseURL,
			ctx:     s.ctx,
		}
		a.init()
		// Borders and outlines have no style, so their widths compute to
		// zero; the other properties that need computing do not apply.
		for _, id := range []props.PropID{
			props.PropBorderTopWidth, props.PropBorderBottomWidth,
			props.PropBorderLeftWidth, props.PropBorderRightWidth,
			props.PropOutlineWidth,
		} {
			a.store(id, props.Pixels(0))
		}
		s.anonymous = a
	}
	return s.anonymous
}

// Copy returns a style with the same values and sources. Values that the
// copy computes later are its own.
func (s *Style) Copy() *Style {
	c := &Style{
		kind:       s.kind,
		parent:     s.parent,
		root:       s.root,
		cascaded:   s.cascaded,
		pseudoType: s.pseudoType,
		baseURL:    s.baseURL,
		ctx:        s.ctx,
	}
	c.init()
	copy(c.values, s.values)
	copy(c.computed, s.computed)
	for k, v := range s.custom {
		c.custom[k] = v
	}
	for k, v := range s.specified {
		c.specified[k] = v
	}
	return c
}

// Parent returns the style the inherited values come from, or nil for the
// root element.
func (s *Style) Parent() *Style { return s.parent }

// Root returns the style of the root element, or the initial style for the
// root element itself.
func (s *Style) Root() *Style { return s.root }

// Context returns the context shared by the styles of the document.
func (s *Style) Context() *Context { return s.ctx }

// PseudoType returns the pseudo-element the style is for, such as "before",
// or the margin box of a page style, such as "@top-left", or "".
func (s *Style) PseudoType() string { return s.pseudoType }

// IsRootElement reports whether s is the style of the root element.
func (s *Style) IsRootElement() bool { return s.kind == elementStyle && s.parent == nil }

// IsAnonymous reports whether s is the style of anonymous boxes.
func (s *Style) IsAnonymous() bool { return s.kind == anonymousStyle }

// BaseURL returns the URL that relative URLs of the style resolve against.
func (s *Style) BaseURL() string { return s.baseURL }

// Cascaded returns the cascaded declarations of the style.
func (s *Style) Cascaded() Cascaded { return s.cascaded }

// Set stores the computed value of a property, replacing any other.
func (s *Style) Set(id props.PropID, v props.Value) { s.store(id, v) }

// Specified returns the specified value of position, float or display, set
// once the property has been computed, or nil.
func (s *Style) Specified(id props.PropID) props.Value {
	if v, ok := s.specified[id]; ok {
		return v
	}
	if s.kind != elementStyle {
		return s.Get(id)
	}
	return nil
}

// ParentValue returns the computed value of a property of the parent
// style, or the initial value when there is no parent.
func (s *Style) ParentValue(id props.PropID) props.Value {
	if s.parent == nil {
		return id.Initial()
	}
	return s.parent.Get(id)
}

func (s *Style) store(id props.PropID, v props.Value) {
	s.values[id] = v
	s.computed.set(id)
}

func (s *Style) unstore(id props.PropID) {
	s.values[id] = nil
	s.computed.clear(id)
}

// value is read by the generated getters.
func (s *Style) value(id props.PropID) props.Value { return s.Get(id) }

// get returns the value of a property as the type its getter declares. It
// returns the zero value of T if the computed value has another type.
func get[T props.Value](s *Style, id props.PropID) T {
	v, _ := s.value(id).(T)
	return v
}

// Get returns the computed value of a property, computing it on first use.
func (s *Style) Get(id props.PropID) props.Value {
	if s.computed.has(id) {
		return s.values[id]
	}
	switch s.kind {
	case initialStyle:
		v := id.Initial()
		s.store(id, v)
		return v
	case anonymousStyle:
		return s.computeAnonymous(id)
	}
	return s.compute(id)
}

func isKeyword(v props.Value, k props.Keyword) bool {
	kw, ok := v.(props.Keyword)
	return ok && kw == k
}

func (s *Style) computeAnonymous(id props.PropID) props.Value {
	name := id.Name()
	var v props.Value
	switch {
	case id.Inherited() || id == props.PropPage:
		v = s.parent.Get(id)
	case strings.HasPrefix(name, "text-decoration-"):
		v = textDecoration(id, id.Initial(), s.parent.Get(id), false)
	case id.InitialNotComputed():
		v = s.computeValue(id, id.Initial())
	default:
		v = id.Initial()
	}
	s.store(id, v)
	return v
}

// fallback is the value of a property whose computation depends on itself:
// the inherited or initial value.
func (s *Style) fallback(id props.PropID) props.Value {
	if id.Inherited() && s.parent != nil {
		return s.parent.Get(id)
	}
	return id.Initial()
}

// compute finds the specified value of a property from the cascade, the
// parent style or the initial value, and computes it (CSS Cascade 5 §4).
func (s *Style) compute(id props.PropID) props.Value {
	if s.computing.has(id) {
		return s.fallback(id)
	}
	s.computing.set(id)
	defer s.computing.clear(id)

	// The computed values of float and display depend on the specified
	// values of position and float (CSS 2.1 §9.7).
	switch id {
	case props.PropFloat:
		s.Get(props.PropPosition)
	case props.PropDisplay:
		s.Get(props.PropFloat)
	}

	name := id.Name()
	parent := s.parent
	declared, isCascaded := s.cascaded[name]
	value, weight := declared.Value, declared.Weight
	if !isCascaded {
		if id.Inherited() {
			value = props.Keyword("inherit")
		} else {
			value = props.Keyword("initial")
		}
	}

	wanted := name
	if logical := logicalName(id, string(s.Direction())); logical != "" {
		if d, ok := s.cascaded[logical]; ok && d.Weight.Compare(weight) >= 0 {
			wanted, value = logical, d.Value
		}
	}

	if isKeyword(value, "inherit") && parent == nil {
		value = props.Keyword("initial")
	}

	if pending, ok := value.(*props.Pending); ok {
		solved, err := validation.Solve(pending, s.substitute(pending.Tokens), wanted, s.baseURL)
		if err != nil {
			if id.Inherited() && parent != nil {
				value = parent.Get(id)
				s.store(id, value)
			} else {
				value = id.Initial()
				if !id.InitialNotComputed() {
					s.store(id, value)
				}
			}
		} else {
			value = solved
		}
	}

	if isKeyword(value, "initial") {
		value = id.Initial()
		if !id.InitialNotComputed() {
			s.store(id, value)
		}
	} else if isKeyword(value, "inherit") {
		value = parent.Get(id)
		s.store(id, value)
	}

	switch {
	case strings.HasPrefix(name, "text-decoration-") && parent != nil:
		// Text decorations are not inherited but propagated (CSS Text
		// Decoration 3 §2).
		value = s.computeValue(id, value)
		value = textDecoration(id, value, parent.Get(id), isCascaded)
		s.store(id, value)
	case id == props.PropPage && isKeyword(value, "auto"):
		// The used value of auto is the value of the nearest ancestor with
		// a non-auto value, or the empty string on the root (CSS Paged
		// Media 3 §8.1).
		if parent == nil {
			value = props.Keyword("")
		} else {
			value = parent.Get(id)
		}
		s.unstore(id)
	case id == props.PropPosition || id == props.PropFloat || id == props.PropDisplay:
		s.specified[id] = value
	}

	if v, ok := s.resolveMath(id, value); ok {
		value = v
	}

	if s.computed.has(id) {
		return s.values[id]
	}
	value = s.computeValue(id, value)
	s.store(id, value)
	return value
}

// computeValue applies the computer of the property, if it has one.
func (s *Style) computeValue(id props.PropID, v props.Value) props.Value {
	if fn := props.Table[id].Compute; fn != nil {
		return fn(s, v)
	}
	return v
}

// textDecoration propagates the text decoration of the parent: the color,
// style and thickness are those of the parent unless they are cascaded, and
// the lines are added to those of the parent.
func textDecoration(id props.PropID, value, parentValue props.Value, cascaded bool) props.Value {
	switch id {
	case props.PropTextDecorationColor, props.PropTextDecorationStyle, props.PropTextDecorationThickness:
		if !cascaded {
			return parentValue
		}
	case props.PropTextDecorationLine:
		parentLines, _ := parentValue.(props.Keywords)
		lines, _ := value.(props.Keywords)
		if isNoLine(parentLines) {
			return value
		}
		if isNoLine(lines) {
			return parentValue
		}
		union := append(props.Keywords(nil), lines...)
		for _, l := range parentLines {
			if !containsKeyword(union, l) {
				union = append(union, l)
			}
		}
		return union
	}
	return value
}

func isNoLine(k props.Keywords) bool { return len(k) == 1 && k[0] == "none" }

func containsKeyword(list props.Keywords, k props.Keyword) bool {
	for _, v := range list {
		if v == k {
			return true
		}
	}
	return false
}

// Custom returns the computed value of a custom property: its tokens with
// var() references substituted. The result is empty when the property is
// not set or invalid at computed-value time, or when its value depends on
// itself (CSS Variables 1 §2.3).
func (s *Style) Custom(name string) props.Tokens {
	if v, ok := s.custom[name]; ok {
		return v
	}
	switch s.kind {
	case initialStyle:
		return nil
	case anonymousStyle:
		v := s.parent.Custom(name)
		s.custom[name] = v
		return v
	}

	declared, ok := s.cascaded[name]
	value := declared.Value
	if !ok {
		value = props.Keyword("inherit")
	}
	if isKeyword(value, "inherit") && s.parent == nil {
		value = props.Keyword("initial")
	}

	var result props.Tokens
	switch v := value.(type) {
	case *props.Pending:
		// The property is invalid while its own value is resolved, which
		// breaks dependency cycles.
		s.custom[name] = nil
		tokens := s.substitute(v.Tokens)
		delete(s.custom, name)
		if solved, err := validation.Solve(v, tokens, name, s.baseURL); err == nil {
			result, _ = solved.(props.Tokens)
		}
	case props.Tokens:
		result = v
	case props.Keyword:
		if v == "inherit" {
			result = s.parent.Custom(name)
		} else {
			result = props.Tokens{}
		}
	}
	s.custom[name] = result
	return result
}

// substitute replaces the var() references of tokens by the values of the
// custom properties.
func (s *Style) substitute(tokens []syntax.Node) []syntax.Node {
	var out []syntax.Node
	for _, t := range tokens {
		if resolved, ok := s.resolveVar(t, 0); ok {
			out = append(out, resolved...)
		} else {
			out = append(out, t)
		}
	}
	return out
}

type bitset []uint64

func newBitset(n int) bitset { return make(bitset, (n+63)/64) }

func (b bitset) has(id props.PropID) bool { return b[id/64]&(1<<(id%64)) != 0 }

func (b bitset) set(id props.PropID) { b[id/64] |= 1 << (id % 64) }

func (b bitset) clear(id props.PropID) { b[id/64] &^= 1 << (id % 64) }
