package cascade

import (
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
)

// PageMarginBoxes are the at-rules allowed in @page: the margin boxes, the
// footnote area and the note area (CSS Paged Media 3 §5.2, CSS GCPM 3).
var PageMarginBoxes = map[string]bool{
	"bottom-center": true, "bottom-left": true, "bottom-left-corner": true,
	"bottom-right": true, "bottom-right-corner": true, "left-bottom": true,
	"left-middle": true, "left-top": true, "right-bottom": true,
	"right-middle": true, "right-top": true, "top-center": true,
	"top-left": true, "top-left-corner": true, "top-right": true,
	"top-right-corner": true, "footnote": true, "note-area": true,
}

// PageIndex is the :nth() page pseudo-class: An+B, counted among the pages
// of the named page group Of when it is not empty.
type PageIndex struct {
	A, B int
	Of   string
}

// PageSelector is one selector of an @page rule (CSS Paged Media 3 §4.1).
// Empty and false fields match any page.
type PageSelector struct {
	// Side is "left", "right" or "".
	Side  string
	Blank bool
	First bool
	Index *PageIndex
	// Name is the page type name, or "".
	Name        string
	Specificity [3]int
}

// ParsePageSelectors parses the prelude of an @page rule. It returns false
// when the selector is invalid or unsupported.
func ParsePageSelectors(prelude []syntax.Node) ([]PageSelector, bool) {
	tokens := tk.RemoveWhitespace(prelude)
	if len(tokens) == 0 {
		return []PageSelector{{}}, true
	}
	var selectors []PageSelector
	for len(tokens) > 0 {
		var s PageSelector
		if ident, ok := tokens[0].(*syntax.IdentToken); ok {
			tokens = tokens[1:]
			s.Name = ident.Value
			s.Specificity[0] = 1
		}
		switch len(tokens) {
		case 1:
			return nil, false
		case 0:
			return append(selectors, s), true
		}

	compound:
		for len(tokens) > 0 {
			literal, ok := tokens[0].(*syntax.LiteralToken)
			tokens = tokens[1:]
			if !ok {
				return nil, false
			}
			switch literal.Value {
			case ":":
				if len(tokens) == 0 {
					return nil, false
				}
				switch t := tokens[0].(type) {
				case *syntax.IdentToken:
					tokens = tokens[1:]
					switch t.LowerValue {
					case "left", "right":
						if s.Side != "" && s.Side != t.LowerValue {
							return nil, false
						}
						s.Side = t.LowerValue
						s.Specificity[2]++
						continue
					case "blank":
						s.Blank = true
						s.Specificity[1]++
						continue
					case "first":
						s.First = true
						s.Specificity[1]++
						continue
					}
				case *syntax.FunctionBlock:
					tokens = tokens[1:]
					index, ok := parseNthPage(t)
					if !ok {
						return nil, false
					}
					s.Index = index
					s.Specificity[1]++
					if index.Of != "" {
						s.Specificity[0]++
					}
					continue
				}
				return nil, false
			case ",":
				if len(tokens) > 0 && s.Specificity != [3]int{} {
					break compound
				}
				return nil, false
			default:
				return nil, false
			}
		}
		selectors = append(selectors, s)
	}
	return selectors, true
}

// parseNthPage parses :nth(An+B) and :nth(An+B of name) (CSS GCPM 3 §6.3).
func parseNthPage(fn *syntax.FunctionBlock) (*PageIndex, bool) {
	if fn.Name != "nth" {
		return nil, false
	}
	nth, group := fn.Arguments, []syntax.Node(nil)
	hasGroup := false
	for i, argument := range fn.Arguments {
		if ident, ok := argument.(*syntax.IdentToken); ok && ident.Value == "of" {
			// The token before "of" is expected to be whitespace.
			nth = fn.Arguments[:max(i-1, 0)]
			group, hasGroup = fn.Arguments[i+1:], true
			break
		}
	}
	a, b, ok := syntax.ParseNth(nth)
	if !ok {
		return nil, false
	}
	index := &PageIndex{A: a, B: b}
	if hasGroup {
		group = tk.RemoveWhitespace(group)
		if len(group) != 1 {
			return nil, false
		}
		ident, ok := group[0].(*syntax.IdentToken)
		if !ok {
			return nil, false
		}
		index.Of = ident.Value
	}
	return index, true
}

// PageGroup is a named page group a page belongs to, with the index of the
// page in the group.
type PageGroup struct {
	Name  string
	Index int
}

// PageType describes a page for @page selectors: its side, whether it is
// blank, its name, its index in the document and its page groups.
type PageType struct {
	Side   string
	Blank  bool
	Name   string
	Index  int
	Groups []PageGroup
}

// nthMatches reports whether the 1-based position matches An+B.
func nthMatches(a, b, position int) bool {
	offset := position - b
	if a == 0 {
		return offset == 0
	}
	return offset%a == 0 && (offset == 0 || (offset > 0) == (a > 0))
}

// Matches reports whether a selector matches pages of type p.
func (s PageSelector) Matches(p PageType) bool {
	if s.Side != "" && s.Side != p.Side {
		return false
	}
	if s.Blank && !p.Blank {
		return false
	}
	if s.First && p.Index != 0 {
		return false
	}
	if s.Name != "" && s.Name != p.Name {
		return false
	}
	if s.Index == nil {
		return true
	}
	if s.Index.Of == "" {
		return nthMatches(s.Index.A, s.Index.B, p.Index+1)
	}
	if s.Index.Of != p.Name {
		return false
	}
	for _, g := range p.Groups {
		if g.Name == s.Index.Of && nthMatches(s.Index.A, s.Index.B, g.Index+1) {
			return true
		}
	}
	return false
}

// PageRule is an @page rule, or a margin rule inside one, with its
// validated declarations.
type PageRule struct {
	Selectors []PageSelector
	// MarginBox is the at-keyword of a margin rule with its "@", such as
	// "@top-left", or "" for the page itself.
	MarginBox    string
	Declarations []validation.Declaration
}
