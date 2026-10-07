package boxes

import (
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
)

// State is what the building of the box tree carries from one element to
// the next in tree order: the nesting depth of quotes and the counters in
// scope (CSS Lists 3 §4.4). Layout keeps one for the pages, whose counters
// are separate from those of the elements.
type State struct {
	QuoteDepth int
	// CounterValues are the values of the counters in scope, outermost
	// instance first.
	CounterValues targets.CounterValues
	// CounterScopes holds, for each level of the tree being walked, the
	// counters whose scope was created by the elements of that level.
	CounterScopes []map[string]bool
}

// NewState returns the state at the start of a document, where the
// footnote and note counters are in scope.
func NewState() *State {
	return &State{
		CounterValues: targets.CounterValues{"footnote": {0}, "note": {0}},
		CounterScopes: []map[string]bool{{"footnote": true, "note": true}},
	}
}

// Clone returns a deep copy of the state.
func (s *State) Clone() *State {
	c := &State{QuoteDepth: s.QuoteDepth, CounterValues: s.CounterValues.Clone()}
	if c.CounterValues == nil {
		c.CounterValues = targets.CounterValues{}
	}
	for _, scope := range s.CounterScopes {
		c.CounterScopes = append(c.CounterScopes, cloneSet(scope))
	}
	return c
}

func cloneSet(s map[string]bool) map[string]bool {
	c := make(map[string]bool, len(s))
	for k, v := range s {
		c[k] = v
	}
	return c
}

// pushScope starts the level of the children of an element.
func (s *State) pushScope() { s.CounterScopes = append(s.CounterScopes, map[string]bool{}) }

// popScope ends the scopes of the counters created by the children of an
// element.
func (s *State) popScope() {
	last := s.CounterScopes[len(s.CounterScopes)-1]
	s.CounterScopes = s.CounterScopes[:len(s.CounterScopes)-1]
	for name := range last {
		values := s.CounterValues[name]
		values = values[:len(values)-1]
		if len(values) == 0 {
			delete(s.CounterValues, name)
		} else {
			s.CounterValues[name] = values
		}
	}
}

// UpdateCounters applies counter-reset, counter-set and counter-increment,
// in that order (CSS Lists 3 §4.5). Without counter-increment, list items
// increment the list-item counter (CSS Lists 3 §4.6).
func UpdateCounters(state *State, style *computed.Style) {
	values := state.CounterValues
	siblings := state.CounterScopes[len(state.CounterScopes)-1]

	for _, op := range style.CounterReset().Items {
		if siblings[op.Name] {
			v := values[op.Name]
			values[op.Name] = v[:len(v)-1]
		} else {
			siblings[op.Name] = true
		}
		values[op.Name] = append(values[op.Name], op.Value)
	}

	for _, op := range style.CounterSet().Items {
		v := values[op.Name]
		if len(v) == 0 {
			siblings[op.Name] = true
			v = append(v, 0)
		}
		v[len(v)-1] = op.Value
		values[op.Name] = v
	}

	increment := style.CounterIncrement()
	items := increment.Items
	if increment.Auto {
		items = nil
		if slices.Contains(style.Display(), "list-item") {
			items = []props.CounterOp{{Name: "list-item", Value: 1}}
		}
	}
	for _, op := range items {
		v := values[op.Name]
		if len(v) == 0 {
			siblings[op.Name] = true
			v = append(v, 0)
		}
		v[len(v)-1] += op.Value
		values[op.Name] = v
	}
}

// lastValue returns the innermost value of a counter, 0 when it is not in
// scope.
func lastValue(values targets.CounterValues, name string) int {
	if v := values[name]; len(v) > 0 {
		return v[len(v)-1]
	}
	return 0
}
