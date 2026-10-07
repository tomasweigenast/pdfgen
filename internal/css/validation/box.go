package validation

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

var auto = props.LengthKeyword("auto")

func init() {
	register(singleToken(plain(lengthPercentageOrAuto)), 0,
		"top", "right", "left", "bottom",
		"inset-block-start", "inset-block-end", "inset-inline-start", "inset-inline-end",
		"margin-top", "margin-right", "margin-bottom", "margin-left",
		"margin-block-start", "margin-block-end", "margin-inline-start", "margin-inline-end",
		"text-underline-offset")
	register(singleToken(plain(widthHeight)), 0, "height", "width", "block-size", "inline-size")
	register(singleToken(plain(minWidthHeight)), 0, "min-width", "min-height", "min-block-size", "min-inline-size")
	register(singleToken(plain(maxWidthHeight)), 0, "max-width", "max-height", "max-block-size", "max-inline-size")
	register(singleToken(plain(lengthOrPercentage)), 0,
		"padding-top", "padding-right", "padding-bottom", "padding-left",
		"padding-block-start", "padding-block-end", "padding-inline-start", "padding-inline-end")
	register(keywordOf("padding-box", "border-box", "content-box"), 0, "box-sizing")
	register(keywordOf("visible", "hidden", "scroll", "auto"), 0, "overflow")
	register(keywordOf("clip", "ellipsis"), 0, "text-overflow")
	register(keywordOf("left", "right", "inline-start", "inline-end", "both", "none"), 0, "clear")
	register(keywordOf("left", "right", "inline-start", "inline-end", "footnote", "none"), 0, "float")
	register(singleToken(plain(lengthOrNone(false))), 0, "outline-offset")
	register(validateClip(), 0, "clip")
	register(validateDisplay, 0, "display")
	register(validatePosition(), 0, "position")
	register(singleToken(plain(validateOpacity)), 0, "opacity")
	register(singleToken(plain(validateZIndex)), 0, "z-index")
	register(keywordOf("fixed", "auto"), 0, "table-layout")
	register(keywordOf("top", "bottom"), 0, "caption-side")
	register(keywordOf("show", "hide"), 0, "empty-cells")
	register(keywordOf("separate", "collapse"), 0, "border-collapse")
	register(keywordOf("ltr", "rtl"), 0, "direction")
	register(keywordOf("visible", "hidden", "collapse"), 0, "visibility")
	register(singleToken(plain(verticalAlign)), 0, "vertical-align")
	register(singleToken(plain(validateAppearance)), 0, "appearance")
}

func lengthPercentageOrAuto(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, true, true); ok {
		return l
	}
	if tk.GetKeyword(token) == "auto" {
		return auto
	}
	return nil
}

func widthHeight(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	if tk.GetKeyword(token) == "auto" {
		return auto
	}
	return nil
}

// minWidthHeight allows auto as the automatic minimum size of flex items
// (CSS Flexbox 1 §4.5).
func minWidthHeight(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "auto" {
		return auto
	}
	return lengthOrPercentage(token)
}

func lengthOrPercentage(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	return nil
}

func maxWidthHeight(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	if tk.GetKeyword(token) == "none" {
		return props.Px(math.Inf(1))
	}
	return nil
}

// lengthOrNone returns a validator of a length, with a percentage when
// percentage is set, that may be negative.
func lengthOrNone(percentage bool) func(syntax.Node) props.Value {
	return func(token syntax.Node) props.Value {
		if l, ok := tk.GetLength(token, true, percentage); ok {
			return l
		}
		return nil
	}
}

func verticalAlign(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, true, true); ok {
		return l
	}
	switch k := tk.GetKeyword(token); k {
	case "baseline", "middle", "sub", "super", "text-top", "text-bottom", "top", "bottom":
		return props.LengthKeyword(props.Keyword(k))
	}
	return nil
}

func validateClip() validator {
	return singleToken(plain(func(token syntax.Node) props.Value {
		f := tk.NewFunction(token)
		arguments := f.SplitComma()
		if f.Name == "rect" && len(arguments) == 4 {
			values := make(props.Clip, 4)
			for i, argument := range arguments {
				if tk.GetKeyword(argument) == "auto" {
					values[i] = auto
				} else if l, ok := tk.GetLength(argument, true, false); ok {
					values[i] = l
				} else {
					return nil
				}
			}
			return values
		}
		if tk.GetKeyword(token) == "auto" {
			return props.Clip{}
		}
		return nil
	}))
}

// validateDisplay parses the one, two or three keywords of display (CSS
// Display 3 §2) into outside, inside and optional list-item keywords.
func validateDisplay(tokens []syntax.Node, _ string) (props.Value, error) {
	for _, token := range tokens {
		if _, ok := token.(*syntax.IdentToken); !ok {
			return nil, nil
		}
	}

	if len(tokens) == 1 {
		switch value := tk.GetCustomIdent(tokens[0]); value {
		case "none", "table-caption", "table-row-group", "table-cell",
			"table-header-group", "table-footer-group", "table-row",
			"table-column-group", "table-column":
			return props.Display{props.Keyword(value)}, nil
		case "inline-table":
			return props.Display{"inline", "table"}, nil
		case "inline-flex":
			return props.Display{"inline", "flex"}, nil
		case "inline-grid":
			return props.Display{"inline", "grid"}, nil
		case "inline-block":
			return props.Display{"inline", "flow-root"}, nil
		}
	}

	var outside, inside, listItem props.Keyword
	for _, token := range tokens {
		switch value := props.Keyword(tk.GetCustomIdent(token)); value {
		case "block", "inline":
			if outside != "" {
				return nil, nil
			}
			outside = value
		case "flow", "flow-root", "table", "flex", "grid":
			if inside != "" {
				return nil, nil
			}
			inside = value
		case "list-item":
			if listItem != "" {
				return nil, nil
			}
			listItem = value
		default:
			return nil, nil
		}
	}

	if outside == "" {
		outside = "block"
	}
	if inside == "" {
		inside = "flow"
	}
	if listItem != "" {
		if inside == "flow" || inside == "flow-root" {
			return props.Display{outside, inside, listItem}, nil
		}
		return nil, nil
	}
	return props.Display{outside, inside}, nil
}

func validatePosition() validator {
	return singleToken(plain(func(token syntax.Node) props.Value {
		if fn, ok := token.(*syntax.FunctionBlock); ok && (fn.Name == "running" || fn.Name == "note") {
			if len(fn.Arguments) == 1 {
				if ident, ok := fn.Arguments[0].(*syntax.IdentToken); ok {
					return props.Positioning{Keyword: props.Keyword(fn.Name + "()"), Name: ident.Value}
				}
			}
		}
		switch k := tk.GetKeyword(token); k {
		case "static", "relative", "absolute", "fixed":
			return props.Positioning{Keyword: props.Keyword(k)}
		}
		return nil
	}))
}

func validateOpacity(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, true, false); ok {
		return props.Number(math.Min(1, math.Max(0, n.Value)))
	}
	if p, ok := tk.GetPercentage(token, true); ok {
		return props.Number(math.Min(1, math.Max(0, p.Value/100)))
	}
	return nil
}

func validateZIndex(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "auto" {
		return props.IntOrKeyword{Keyword: "auto"}
	}
	if n, ok := tk.GetNumber(token, true, true); ok {
		return props.IntOrKeyword{N: int(n.Value)}
	}
	return nil
}

func validateAppearance(token syntax.Node) props.Value {
	switch k := tk.GetKeyword(token); k {
	case "none", "auto":
		return props.Keyword(k)
	}
	return nil
}
