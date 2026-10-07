package validation

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	registerExpander(expandTwoSides, "margin-block", "margin-inline", "padding-block", "padding-inline",
		"border-block-color", "border-block-style", "border-block-width",
		"border-inline-color", "border-inline-style", "border-inline-width",
		"inset-block", "inset-inline")
	registerExpander(expandFourSides, "border-color", "border-style", "border-width",
		"margin", "padding", "bleed", "inset")
	registerExpander(expandBorderRadius, "border-radius")
	registerExpander(expandBorder, "border")
	registerExpander(expandLogicalBorder, "border-block", "border-inline")
	registerExpander(generic([]string{"-width", "-color", "-style"}, borderSideParts), "border-top", "border-right",
		"border-bottom", "border-left", "border-block-start", "border-block-end",
		"border-inline-start", "border-inline-end", "column-rule", "outline")
	registerExpander(generic([]string{"-type", "-position", "-image"}, listStyleParts), "list-style")
	registerExpander(generic([]string{"-outset", "-repeat", "-slice", "-source", "-width"}, borderImageParts(false)), "border-image")
	registerExpander(generic([]string{"-outset", "-repeat", "-slice", "-source", "-width", "-mode"}, borderImageParts(true)), "mask-border")
}

func expandTwoSides(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	return expandSides(tokens, name, baseURL, []string{"start", "end"})
}

func expandFourSides(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	sides := []string{"top", "right", "bottom", "left"}
	if len(tokens) > 0 && tk.GetKeyword(tokens[0]) == "logical" {
		sides = []string{"block-start", "inline-start", "block-end", "inline-end"}
		tokens = tokens[1:]
	}
	return expandSides(tokens, name, baseURL, sides)
}

// expandSides sets a token on two or four sides of a box (CSS Box 4 §6.2).
func expandSides(tokens []syntax.Node, name, baseURL string, sides []string) ([]pair, error) {
	expanded := make([]string, len(sides))
	for i, side := range sides {
		switch {
		case strings.HasSuffix(name, "-color") || strings.HasSuffix(name, "-style") || strings.HasSuffix(name, "-width"):
			// border-color becomes border-top-color, not border-color-top.
			expanded[i] = name[:len(name)-6] + "-" + side + "-" + name[len(name)-5:]
		case name == "inset" && !strings.Contains(side, "-"):
			// The physical inset does not give inset-top, just top.
			expanded[i] = side
		default:
			expanded[i] = name + "-" + side
		}
	}

	if hasVar(tokens) {
		pending := &props.Pending{Name: name, Tokens: tokens}
		pairs := make([]pair, len(expanded))
		for i, n := range expanded {
			pairs[i] = pair{n, pending}
		}
		return pairs, nil
	}

	switch {
	case len(tokens) == 1:
		t := tokens[0]
		tokens = make([]syntax.Node, len(sides))
		for i := range tokens {
			tokens[i] = t
		}
	case len(sides) == 4 && len(tokens) == 2:
		tokens = []syntax.Node{tokens[0], tokens[1], tokens[0], tokens[1]}
	case len(sides) == 4 && len(tokens) == 3:
		tokens = []syntax.Node{tokens[0], tokens[1], tokens[2], tokens[1]}
	case len(tokens) != len(sides):
		return nil, tk.Invalid("Expected 1 to %d tokens, got %d", len(sides), len(tokens))
	}
	pairs := make([]pair, len(expanded))
	for i, n := range expanded {
		v, err := validateNonShorthand([]syntax.Node{tokens[i]}, n, baseURL, true)
		if err != nil {
			return nil, err
		}
		pairs[i] = pair{n, v}
	}
	return pairs, nil
}

var radiusCorners = []string{"top-left", "top-right", "bottom-right", "bottom-left"}

func radiusNames() []string {
	names := make([]string, len(radiusCorners))
	for i, c := range radiusCorners {
		names[i] = "border-" + c + "-radius"
	}
	return names
}

func expandBorderRadius(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	names := radiusNames()
	return generic(names, func(tokens []syntax.Node, _, baseURL string) ([]part, error) {
		var horizontal, vertical []syntax.Node
		inVertical := false
		for _, token := range tokens {
			if !syntax.IsLiteral(token, "/") {
				if inVertical {
					vertical = append(vertical, token)
				} else {
					horizontal = append(horizontal, token)
				}
				continue
			}
			if inVertical {
				return nil, tk.Invalid(`Expected only one "/" separator`)
			}
			if syntax.IsLiteral(tokens[len(tokens)-1], "/") {
				return nil, tk.Invalid(`Expected value after "/" separator`)
			}
			inVertical = true
		}
		if len(vertical) == 0 {
			vertical = append([]syntax.Node(nil), horizontal...)
		}

		for _, values := range []*[]syntax.Node{&horizontal, &vertical} {
			switch v := *values; len(v) {
			case 1:
				*values = []syntax.Node{v[0], v[0], v[0], v[0]}
			case 2:
				*values = []syntax.Node{v[0], v[1], v[0], v[1]}
			case 3:
				*values = []syntax.Node{v[0], v[1], v[2], v[1]}
			case 4:
			default:
				return nil, tk.Invalid("Expected 1 to 4 token components got %d", len(v))
			}
		}
		var parts []part
		for i, n := range names {
			pair := []syntax.Node{horizontal[i], vertical[i]}
			if _, err := validateNonShorthand(pair, n, baseURL, true); err != nil {
				return parts, err
			}
			parts = append(parts, part{n, pair})
		}
		return parts, nil
	})(tokens, name, baseURL)
}

func borderSideParts(tokens []syntax.Node, name, baseURL string) ([]part, error) {
	var parts []part
	for _, token := range tokens {
		var suffix string
		if _, ok := syntax.ParseColor(token); ok {
			suffix = "-color"
		} else if ok, err := accepts("border-top-width", baseURL, token); err != nil {
			return parts, err
		} else if ok {
			suffix = "-width"
		} else if ok, err := accepts("border-top-style", baseURL, token); err != nil {
			return parts, err
		} else if ok {
			suffix = "-style"
		} else {
			return parts, tk.ErrInvalid
		}
		parts = append(parts, part{suffix, []syntax.Node{token}})
	}
	return parts, nil
}

func expandBorder(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	var all []pair
	for _, side := range []string{"top", "right", "bottom", "left"} {
		pairs, err := expanders["border-"+side](tokens, name+"-"+side, baseURL)
		if err != nil {
			return nil, err
		}
		all = append(all, pairs...)
	}
	return all, nil
}

func expandLogicalBorder(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	var all []pair
	for _, side := range []string{"start", "end"} {
		pairs, err := expanders["border-top"](tokens, name+"-"+side, baseURL)
		if err != nil {
			return nil, err
		}
		all = append(all, pairs...)
	}
	return all, nil
}

func listStyleParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	var parts []part
	typeSpecified, imageSpecified := false, false
	noneCount := 0
	var noneToken syntax.Node
	for _, token := range tokens {
		if tk.GetKeyword(token) == "none" {
			// It can be either the type or the image: decided at the end,
			// according to what else is specified.
			noneCount++
			noneToken = token
			continue
		}
		var suffix string
		if ok, err := accepts("list-style-image", baseURL, token); err != nil {
			return parts, err
		} else if ok {
			suffix = "-image"
			imageSpecified = true
		} else if ok, _ := accepts("list-style-position", baseURL, token); ok {
			suffix = "-position"
		} else if ok, _ := accepts("list-style-type", baseURL, token); ok {
			suffix = "-type"
			typeSpecified = true
		} else {
			return parts, tk.ErrInvalid
		}
		parts = append(parts, part{suffix, []syntax.Node{token}})
	}
	if !typeSpecified && noneCount > 0 {
		parts = append(parts, part{"-type", []syntax.Node{noneToken}})
		noneCount--
	}
	if !imageSpecified && noneCount > 0 {
		parts = append(parts, part{"-image", []syntax.Node{noneToken}})
		noneCount--
	}
	if noneCount > 0 {
		return parts, tk.ErrInvalid
	}
	return parts, nil
}

// borderImageParts parses border-image, or mask-border when mask is set:
// source, slice / width / outset and repeat in any order (CSS Backgrounds 3
// §6.4).
func borderImageParts(mask bool) partsFunc {
	prefix := "border-image"
	if mask {
		prefix = "mask-border"
	}
	check := func(suffix string, baseURL string, tokens []syntax.Node) (bool, error) {
		return accepts(prefix+suffix, baseURL, tokens...)
	}
	return func(tokens []syntax.Node, name, baseURL string) ([]part, error) {
		var parts []part
		for len(tokens) > 0 {
			if ok, err := check("-source", baseURL, tokens[:1]); err != nil {
				return parts, err
			} else if ok {
				parts = append(parts, part{"-source", tokens[:1]})
				tokens = tokens[1:]
				continue
			}
			if mask {
				if ok, _ := accepts("mask-border-mode", baseURL, tokens[0]); ok {
					parts = append(parts, part{"-mode", tokens[:1]})
					tokens = tokens[1:]
					continue
				}
			}
			if ok, _ := check("-repeat", baseURL, tokens[:1]); ok {
				n := 1
				for n < len(tokens) {
					if ok, _ := check("-repeat", baseURL, tokens[n:n+1]); !ok {
						break
					}
					n++
				}
				parts = append(parts, part{"-repeat", tokens[:n]})
				tokens = tokens[n:]
				continue
			}
			okSlice, _ := check("-slice", baseURL, tokens[:1])
			if !okSlice && tk.GetKeyword(tokens[0]) != "fill" {
				return parts, tk.ErrInvalid
			}

			n := 1
			for n < len(tokens) {
				if ok, _ := check("-slice", baseURL, tokens[:n+1]); !ok {
					break
				}
				n++
			}
			parts = append(parts, part{"-slice", tokens[:n]})
			tokens = tokens[n:]
			if len(tokens) > 0 && syntax.IsLiteral(tokens[0], "/") {
				tokens = tokens[1:]
			} else {
				continue
			}
			if len(tokens) == 0 {
				return parts, tk.ErrInvalid
			}

			if ok, _ := check("-width", baseURL, tokens[:1]); ok {
				n := 1
				for n < len(tokens) {
					if ok, _ := check("-width", baseURL, tokens[:n+1]); !ok {
						break
					}
					n++
				}
				parts = append(parts, part{"-width", tokens[:n]})
				tokens = tokens[n:]
				if len(tokens) > 0 && syntax.IsLiteral(tokens[0], "/") {
					tokens = tokens[1:]
				} else {
					continue
				}
			} else if len(tokens) > 0 && syntax.IsLiteral(tokens[0], "/") {
				tokens = tokens[1:]
			} else {
				return parts, tk.ErrInvalid
			}
			if len(tokens) == 0 {
				return parts, tk.ErrInvalid
			}

			if ok, _ := check("-outset", baseURL, tokens[:1]); !ok {
				return parts, tk.ErrInvalid
			}
			n = 1
			for n < len(tokens) {
				if ok, _ := check("-outset", baseURL, tokens[:n+1]); !ok {
					break
				}
				n++
			}
			parts = append(parts, part{"-outset", tokens[:n]})
			tokens = tokens[n:]
		}
		return parts, nil
	}
}
