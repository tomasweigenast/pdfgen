package computed

import (
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// maxVarDepth bounds the nesting of functions var() substitution goes
// through, so that pathological values cannot exhaust the stack.
const maxVarDepth = 64

// resolveVar substitutes the var() references in token (CSS Variables 1
// §3). The boolean is false when token holds no reference. A var() whose
// custom property is not set and that has no fallback becomes nothing.
func (s *Style) resolveVar(token syntax.Node, depth int) ([]syntax.Node, bool) {
	if !tk.CheckVar(token) || depth > maxVarDepth {
		return nil, false
	}

	fn, isFunction := token.(*syntax.FunctionBlock)
	if !isFunction || fn.LowerName != "var" {
		var items []syntax.Node
		if isFunction {
			items = fn.Arguments
		} else {
			items = token.(*syntax.ParenthesesBlock).Content
		}
		substituted := make([]syntax.Node, 0, len(items))
		for _, item := range items {
			switch item.(type) {
			case *syntax.FunctionBlock, *syntax.ParenthesesBlock:
				if resolved, ok := s.resolveVar(item, depth+1); ok {
					substituted = append(substituted, resolved...)
					continue
				}
			}
			substituted = append(substituted, item)
		}
		line, column := token.Pos()
		var rebuilt syntax.Node
		if isFunction {
			rebuilt = syntax.NewFunctionBlock(line, column, fn.Name, substituted)
		} else {
			rebuilt = syntax.NewParenthesesBlock(line, column, substituted)
		}
		if resolved, ok := s.resolveVar(rebuilt, depth+1); ok && len(resolved) > 0 {
			return resolved, true
		}
		return []syntax.Node{rebuilt}, true
	}

	arguments := tk.NewFunction(token).SplitCommaGroups(true)
	if len(arguments) == 0 || len(arguments[0]) != 1 {
		return []syntax.Node{}, true
	}
	name := ""
	if ident, ok := arguments[0][0].(*syntax.IdentToken); ok {
		name = ident.Value
	}
	if value := s.Custom(name); len(value) > 0 {
		return value, true
	}
	if len(arguments) < 2 {
		return []syntax.Node{}, true
	}
	fallback := []syntax.Node{}
	for _, t := range arguments[1] {
		if resolved, ok := s.resolveVar(t, depth+1); ok {
			fallback = append(fallback, resolved...)
		} else {
			fallback = append(fallback, t)
		}
	}
	return fallback, true
}
