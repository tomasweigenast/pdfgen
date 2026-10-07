package cascade

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/media"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// StringToCSS writes s as a CSS string, quotes included.
func StringToCSS(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\A`, "\r", `\D`, "\f", `\C`).Replace(s)
	return `"` + s + `"`
}

// URLToCSS writes s as a CSS url() function.
func URLToCSS(s string) string { return "url(" + StringToCSS(s) + ")" }

// FindStylesheets returns the stylesheets of the style elements and of the
// stylesheet links of a document, in tree order, leaving out those for
// other media or types (HTML Living Standard §4.2.6).
func FindStylesheets(doc *html.Document, env *Environment) []*Stylesheet {
	if env == nil {
		env = &Environment{}
	}
	var sheets []*Stylesheet
	for _, el := range doc.Elements() {
		tag := el.LocalName()
		if tag != "style" && tag != "link" {
			continue
		}
		mimeType := "text/css"
		if t, ok := el.Lookup("type"); ok {
			mimeType, _, _ = strings.Cut(t, ";")
			mimeType = strings.TrimSpace(mimeType)
		}
		if mimeType != "text/css" {
			continue
		}
		mediaAttr := strings.TrimSpace(el.Get("media"))
		if mediaAttr == "" {
			mediaAttr = "all"
		}
		var query media.Query
		for _, t := range strings.Split(mediaAttr, ",") {
			query = append(query, strings.TrimSpace(t))
		}
		if !media.Evaluate(query, env.mediaType()) {
			continue
		}
		switch {
		case tag == "style":
			sheets = append(sheets, ParseStylesheet(el.ChildText(), doc.BaseURL, env))
		case el.Get("href") != "":
			if !el.HasLinkType("stylesheet") || el.HasLinkType("alternate") {
				continue
			}
			href, ok := html.ParseURL(el.Get("href"), doc.BaseURL, false)
			if !ok {
				continue
			}
			sheet, err := LoadStylesheet(href, env, true)
			if err != nil {
				env.check(err)
				logging.Errorf("Failed to load stylesheet at %s: %s", href, err)
				continue
			}
			sheets = append(sheets, sheet)
		}
	}
	return sheets
}

// attributeDeclarations are the declarations that attributes of an element
// give, with their specificity.
type attributeDeclarations struct {
	element      *html.Element
	specificity  [3]int
	declarations []validation.Declaration
}

// prefixed returns the name of a proprietary property as written in CSS.
func prefixed(name string) string { return validation.VendorPrefix + name }

// styleAttributes collects the declarations of style attributes, with the
// specificity of an id, and those that other attributes imply: lang, id and
// name anchors, sizes of svg elements, and presentational hints when
// enabled (HTML Living Standard §15), with a zero specificity.
func styleAttributes(doc *html.Document, presentationalHints bool) []attributeDeclarations {
	var out []attributeDeclarations
	for _, el := range doc.Elements() {
		if style, ok := el.Lookup("style"); ok && style != "" {
			declarations := validation.PreprocessDeclarations(doc.BaseURL, syntax.ParseBlocksContents(style, false, false))
			out = append(out, attributeDeclarations{el, [3]int{1, 0, 0}, declarations})
		}
		hint := func(target *html.Element, css string) {
			node := syntax.ParseOneDeclaration(css, false)
			out = append(out, attributeDeclarations{target, [3]int{}, validation.PreprocessDeclarations(doc.BaseURL, []syntax.Node{node})})
		}
		if lang := el.Get("lang"); lang != "" {
			hint(el, prefixed("lang")+`:"`+lang+`"`)
		}
		if id := el.Get("id"); id != "" {
			hint(el, prefixed("anchor")+`:"`+id+`"`)
		}
		tag := el.LocalName()
		isHTML := el.Namespace() == html.XHTMLNamespace
		if isHTML && tag == "a" {
			if name := el.Get("name"); name != "" {
				hint(el, prefixed("anchor")+`:"`+name+`"`)
			}
		}
		if el.Namespace() == html.SVGNamespace && tag == "svg" {
			// Only the size of the outermost svg element comes from its
			// attributes, so that CSS defines the intrinsic size (SVG 2
			// §6.6).
			for _, attribute := range []string{"width", "height"} {
				if value := el.Get(attribute); value != "" {
					token := syntax.ParseOneComponentValue(value, false)
					if _, ok := tk.GetLength(token, true, true); ok {
						hint(el, attribute+":"+value)
					}
				}
			}
		}
		if presentationalHints && isHTML {
			presentationalHintsOf(el, doc.BaseURL, hint)
		}
	}
	return out
}

// legacyColor returns the CSS color of a legacy color attribute value.
func legacyColor(s string) string {
	c, _ := html.ParseLegacyColor(s)
	return c
}

func alignHint(el *html.Element, hint func(*html.Element, string)) {
	switch align := strings.ToLower(el.Get("align")); align {
	case "middle":
		hint(el, "text-align:center")
	case "center", "left", "right", "justify":
		hint(el, "text-align:"+align)
	}
}

func backgroundHints(el *html.Element, baseURL string, hint func(*html.Element, string)) {
	if background, ok := html.ParseURL(el.Get("background"), baseURL, false); ok {
		hint(el, "background-image:"+URLToCSS(background))
	}
	if bgcolor := el.Get("bgcolor"); bgcolor != "" {
		hint(el, "background-color:"+legacyColor(bgcolor))
	}
}

// fontSizes maps the size attribute of font to font sizes.
var fontSizes = map[int]string{
	1: "x-small", 2: "small", 3: "medium", 4: "large", 5: "x-large",
	6: "xx-large", 7: "48px",
}

// presentationalHintsOf gives the declarations that presentational
// attributes of an HTML element imply (HTML Living Standard §15.3).
func presentationalHintsOf(el *html.Element, baseURL string, hint func(*html.Element, string)) {
	switch el.LocalName() {
	case "body":
		for _, attribute := range []string{"marginheight", "topmargin"} {
			if v, ok := html.MapToPixelLength(el.Get(attribute)); ok {
				hint(el, "margin-top:"+v)
				hint(el, "margin-bottom:"+v)
				break
			}
		}
		for _, attribute := range []string{"marginwidth", "leftmargin"} {
			if v, ok := html.MapToPixelLength(el.Get(attribute)); ok {
				hint(el, "margin-left:"+v)
				hint(el, "margin-right:"+v)
				break
			}
		}
		backgroundHints(el, baseURL, hint)
		if text := el.Get("text"); text != "" {
			hint(el, "color:"+legacyColor(text))
		}
	case "center":
		hint(el, "text-align:center")
	case "div":
		alignHint(el, hint)
	case "font":
		if color := el.Get("color"); color != "" {
			hint(el, "color:"+legacyColor(color))
		}
		if face := el.Get("face"); face != "" {
			hint(el, "font-family:"+StringToCSS(face))
		}
		if size := el.Get("size"); size != "" {
			size = strings.Trim(size, " \t\n\f\r")
			plus, minus := strings.HasPrefix(size, "+"), strings.HasPrefix(size, "-")
			if plus || minus {
				size = size[1:]
			}
			if n, ok := html.ParseInteger(size); ok {
				if plus {
					n += 3
				} else if minus {
					n -= 3
				}
				hint(el, "font-size:"+fontSizes[max(1, min(7, n))])
			}
		}
	case "table":
		if v, ok := html.MapToPixelLength(el.Get("cellspacing")); ok {
			hint(el, "border-spacing:"+v)
		}
		if v, ok := html.MapToPixelLength(el.Get("cellpadding")); ok {
			for _, cell := range descendants(el) {
				if t := cell.LocalName(); t == "td" || t == "th" {
					hint(cell, "padding:"+v)
				}
			}
		}
		if v, ok := html.MapToDimensionPropertyIgnoringZero(el.Get("width")); ok {
			hint(el, "width:"+v)
		}
		if v, ok := html.MapToDimensionProperty(el.Get("height")); ok {
			hint(el, "height:"+v)
		}
		backgroundHints(el, baseURL, hint)
		if color := el.Get("bordercolor"); color != "" {
			hint(el, "border-color:"+legacyColor(color))
		}
		if v, ok := html.MapToPixelLength(el.Get("border")); ok {
			hint(el, "border-width:"+v)
		}
	case "tr", "td", "th", "thead", "tbody", "tfoot":
		alignHint(el, hint)
		backgroundHints(el, baseURL, hint)
		switch el.LocalName() {
		case "td", "th":
			if v, ok := html.MapToDimensionPropertyIgnoringZero(el.Get("height")); ok {
				hint(el, "height:"+v)
			}
			if v, ok := html.MapToDimensionPropertyIgnoringZero(el.Get("width")); ok {
				hint(el, "width:"+v)
			}
		case "tr":
			if v, ok := html.MapToDimensionProperty(el.Get("height")); ok {
				hint(el, "height:"+v)
			}
		}
	case "caption":
		alignHint(el, hint)
	case "col":
		if v, ok := html.MapToDimensionProperty(el.Get("width")); ok {
			hint(el, "width:"+v)
		}
	case "hr":
		size, _ := html.ParseNonNegativeInteger(el.Get("size"))
		_, hasColor := el.Lookup("color")
		_, hasNoShade := el.Lookup("noshade")
		switch {
		case hasColor || hasNoShade:
			if size >= 1 {
				hint(el, "border-width:"+strconv.FormatFloat(float64(size)/2, 'f', -1, 64)+"px")
			}
		case size == 1:
			hint(el, "border-bottom-width:0")
		case size > 1:
			hint(el, fmt.Sprintf("height:%dpx", size-2))
		}
		if v, ok := html.MapToDimensionProperty(el.Get("width")); ok {
			hint(el, "width:"+v)
		}
		if color := el.Get("color"); color != "" {
			hint(el, "color:"+legacyColor(color))
		}
	case "iframe", "applet", "embed", "img", "input", "object":
		tag := el.LocalName()
		if tag == "input" && strings.ToLower(el.Get("type")) != "image" {
			return
		}
		if align := strings.ToLower(el.Get("align")); align == "middle" || align == "center" {
			hint(el, "vertical-align:middle")
		}
		if v, ok := html.MapToDimensionProperty(el.Get("hspace")); ok {
			hint(el, "margin-left:"+v)
			hint(el, "margin-right:"+v)
		}
		if v, ok := html.MapToDimensionProperty(el.Get("vspace")); ok {
			hint(el, "margin-top:"+v)
			hint(el, "margin-bottom:"+v)
		}
		if v, ok := html.MapToDimensionProperty(el.Get("width")); ok {
			hint(el, "width:"+v)
		}
		if v, ok := html.MapToDimensionProperty(el.Get("height")); ok {
			hint(el, "height:"+v)
		}
		if tag == "img" || tag == "object" || tag == "input" {
			if v, ok := html.MapToPixelLength(el.Get("border")); ok {
				hint(el, "border-width:"+v)
				hint(el, "border-style:solid")
			}
		}
	case "ol":
		// CSS Lists 3 §8.
		if n, ok := html.ParseInteger(el.Get("start")); ok && el.Get("start") != "" {
			hint(el, fmt.Sprintf("counter-reset:list-item %d", n))
			hint(el, "counter-increment:list-item -1")
		}
	case "li":
		if n, ok := html.ParseInteger(el.Get("value")); ok && el.Get("value") != "" {
			hint(el, fmt.Sprintf("counter-reset:list-item %d", n))
			hint(el, "counter-increment:none")
		}
	}
}

// descendants returns the descendants of el in tree order, el included.
func descendants(el *html.Element) []*html.Element {
	out := []*html.Element{el}
	for _, child := range el.Children() {
		out = append(out, descendants(child)...)
	}
	return out
}
