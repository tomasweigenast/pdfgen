package computed

import "github.com/tomasweigenast/pdfgen/internal/css/props"

// Style holds the computed values of the properties of one element. The typed
// getters in style_gen.go read from it.
type Style struct {
	values   []props.Value
	computed bitset
	parent   *Style
}

// New returns an empty style whose inherited properties come from parent,
// which may be nil.
func New(parent *Style) *Style {
	n := props.NumProps()
	return &Style{
		values:   make([]props.Value, n),
		computed: newBitset(n),
		parent:   parent,
	}
}

// Parent returns the style of the parent element, or nil.
func (s *Style) Parent() *Style { return s.parent }

// Set stores the computed value of a property.
func (s *Style) Set(id props.PropID, v props.Value) {
	s.values[id] = v
	s.computed.set(id)
}

// value returns the computed value of a property: the stored one, the one of
// the parent for inherited properties, or the initial value.
func (s *Style) value(id props.PropID) props.Value {
	if s.computed.has(id) {
		return s.values[id]
	}
	if id.Inherited() && s.parent != nil {
		return s.parent.value(id)
	}
	return id.Initial()
}

// get returns the value of a property as the type its getter declares. It
// returns the zero value of T if the stored value has another type.
func get[T props.Value](s *Style, id props.PropID) T {
	v, _ := s.value(id).(T)
	return v
}

type bitset []uint64

func newBitset(n int) bitset { return make(bitset, (n+63)/64) }

func (b bitset) has(id props.PropID) bool { return b[id/64]&(1<<(id%64)) != 0 }

func (b bitset) set(id props.PropID) { b[id/64] |= 1 << (id % 64) }
