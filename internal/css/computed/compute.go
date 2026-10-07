package computed

import (
	"fmt"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// computer turns the specified value of property id into its computed value
// (CSS Cascade 5 §4.4). It also receives values that are already computed,
// such as initial values, and must leave them unchanged.
type computer func(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value

func register(fn computer, names ...string) {
	for _, name := range names {
		id, ok := props.ID(name)
		if !ok {
			panic(fmt.Sprintf("computed: %q is not a known property", name))
		}
		props.Table[id].Compute = func(ctx props.ComputeContext, v props.Value) props.Value {
			return fn(ctx, id, v)
		}
	}
}

func init() {
	register(computeImages, "background-image")
	register(computeImageValue, "border-image-source")
	register(computeColor,
		"color", "background-color", "border-top-color", "border-right-color",
		"border-bottom-color", "border-left-color", "column-rule-color",
		"outline-color", "text-decoration-color")
	register(computeBoxShadow, "box-shadow")
	register(computePositions, "background-position", "object-position")
	register(computeLengthPair, "transform-origin")
	register(computePixelLengths, "border-spacing", "size", "clip")
	register(computeBreak, "break-after", "break-before")
	register(computeLength,
		"top", "right", "left", "bottom",
		"margin-top", "margin-right", "margin-bottom", "margin-left",
		"height", "width", "block-size", "inline-size",
		"min-width", "min-height", "min-block-size", "min-inline-size",
		"max-width", "max-height", "max-block-size", "max-inline-size",
		"padding-top", "padding-right", "padding-bottom", "padding-left",
		"text-indent", "hyphenate-limit-zone", "flex-basis",
		"text-underline-offset", "text-decoration-thickness")
	register(computeBleed, "bleed-left", "bleed-right", "bleed-top", "bleed-bottom")
	register(computeLetterSpacing, "letter-spacing")
	register(computeBackgroundSize, "background-size")
	register(computeImageOrientation, "image-orientation")
	register(computeBorderWidth,
		"border-top-width", "border-right-width", "border-left-width",
		"border-bottom-width", "column-rule-width", "outline-width")
	register(computeBorderImageOutset, "border-image-outset", "mask-border-outset")
	register(computePixelLength, "column-width", "outline-offset")
	register(computeLengthPair,
		"border-top-left-radius", "border-top-right-radius",
		"border-bottom-left-radius", "border-bottom-right-radius")
	register(computeGap, "column-gap", "row-gap")
	register(computeBookmarkLabel, "bookmark-label")
	register(computeStringSet, "string-set")
	register(computeContent, "content")
	register(computeDisplay, "display")
	register(computeFloat, "float")
	register(computeFontSize, "font-size")
	register(computeFontWeight, "font-weight")
	register(computeGridTemplate, "grid-template-columns", "grid-template-rows")
	register(computeGridAuto, "grid-auto-columns", "grid-auto-rows")
	register(computeLineHeight, "line-height")
	register(computeLang, "lang")
	register(computeTabSize, "tab-size")
	register(computeTransform, "transform")
	register(computeVerticalAlign, "vertical-align")
	register(computeWordSpacing, "word-spacing")
}

// length computes a length: absolute and relative lengths become pixels,
// while keywords, percentages and math functions are kept.
func length(ctx props.ComputeContext, id props.PropID, l props.Length) props.Length {
	if l.Keyword != "" || l.Calc != nil {
		return l
	}
	if l.Dimension.Value == 0 {
		return props.Px(0)
	}
	if !units.IsLength(l.Dimension.Unit) {
		return l
	}
	return props.Px(ctx.ToPixels(l.Dimension.Value, l.Dimension.Unit, id.Name()))
}

// pixels computes a length that cannot be a percentage to pixels.
func pixels(ctx props.ComputeContext, id props.PropID, l props.Length) float64 {
	return pixelsOf(length(ctx, id, l))
}

func computeLength(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	if l, ok := v.(props.Length); ok {
		return length(ctx, id, l)
	}
	return v
}

func computeLengthPair(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	if p, ok := v.(props.LengthPair); ok {
		return props.LengthPair{X: length(ctx, id, p.X), Y: length(ctx, id, p.Y)}
	}
	return v
}

func computePixelLength(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok {
		return v
	}
	computed := length(ctx, id, l)
	if id == props.PropOutlineOffset {
		return props.Pixels(pixelsOf(computed))
	}
	return computed
}

func computePixelLengths(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	switch v := v.(type) {
	case props.LengthPair:
		return props.PixelPair{X: props.Pixels(pixels(ctx, id, v.X)), Y: props.Pixels(pixels(ctx, id, v.Y))}
	case props.Clip:
		out := make(props.Clip, len(v))
		for i, l := range v {
			out[i] = length(ctx, id, l)
		}
		return out
	}
	return v
}

func computeImageValue(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	if img, ok := v.(props.Image); ok {
		return computeImage(ctx, id, img)
	}
	return v
}

func computeImages(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	images, ok := v.(props.Images)
	if !ok {
		return v
	}
	out := make(props.Images, len(images))
	for i, img := range images {
		out[i] = computeImage(ctx, id, img)
	}
	return out
}

// computeImage computes the lengths of a gradient. The gradient is copied,
// as the specified value is shared by every element it applies to.
func computeImage(ctx props.ComputeContext, id props.PropID, img props.Image) props.Image {
	gradient := func(g props.Gradient) props.Gradient {
		stops := make([]props.ColorStop, len(g.ColorStops))
		for i, stop := range g.ColorStops {
			stops[i] = stop
			if stop.Position != nil {
				p := length(ctx, id, *stop.Position)
				stops[i].Position = &p
			}
		}
		hints := make([]props.Length, len(g.ColorHints))
		for i, hint := range g.ColorHints {
			hints[i] = length(ctx, id, hint)
		}
		g.ColorStops, g.ColorHints = stops, hints
		return g
	}
	switch {
	case img.Linear != nil:
		linear := *img.Linear
		linear.Gradient = gradient(linear.Gradient)
		img.Linear = &linear
	case img.Radial != nil:
		radial := *img.Radial
		radial.Gradient = gradient(radial.Gradient)
		radial.Center = computePosition(ctx, id, radial.Center)
		if radial.SizeType == "explicit" {
			radial.SizeExplicit = [2]props.Length{
				length(ctx, id, radial.SizeExplicit[0]),
				length(ctx, id, radial.SizeExplicit[1]),
			}
		}
		img.Radial = &radial
	}
	return img
}

// colorSchemes returns the color-scheme of the style as strings.
func colorSchemes(ctx props.ComputeContext) []string {
	keywords, _ := ctx.Get(props.PropColorScheme).(props.Keywords)
	schemes := make([]string, len(keywords))
	for i, k := range keywords {
		schemes[i] = string(k)
	}
	return schemes
}

// color parses again a color that depends on the color scheme, such as
// light-dark(), for the color-scheme of the element (CSS Color 5 §8).
func color(ctx props.ComputeContext, c props.Color) props.Color {
	if c.Token == nil {
		return c
	}
	if parsed, ok := syntax.ParseColor(c.Token, colorSchemes(ctx)...); ok {
		c.Color = parsed
	}
	return c
}

func computeColor(ctx props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	if c, ok := v.(props.Color); ok {
		return color(ctx, c)
	}
	return v
}

func computeBoxShadow(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	shadows, ok := v.(props.BoxShadows)
	if !ok {
		return v
	}
	out := make(props.BoxShadows, len(shadows))
	for i, s := range shadows {
		out[i] = props.BoxShadow{
			X: length(ctx, id, s.X), Y: length(ctx, id, s.Y),
			Blur: length(ctx, id, s.Blur), Spread: length(ctx, id, s.Spread),
			Color: color(ctx, s.Color), Inset: s.Inset,
		}
	}
	return out
}

func computePosition(ctx props.ComputeContext, id props.PropID, p props.Position) props.Position {
	p.X = length(ctx, id, p.X)
	p.Y = length(ctx, id, p.Y)
	return p
}

func computePositions(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	positions, ok := v.(props.Positions)
	if !ok {
		return v
	}
	out := make(props.Positions, len(positions))
	for i, p := range positions {
		out[i] = computePosition(ctx, id, p)
	}
	return out
}

func computeBreak(_ props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	if isKeyword(v, "always") {
		return props.Keyword("page")
	}
	return v
}

// computeBleed gives auto bleeds 8px when crop marks are drawn (CSS Paged
// Media 3 §7.2).
func computeBleed(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok {
		return v
	}
	if l.IsAuto() {
		marks, _ := ctx.Get(props.PropMarks).(props.Keywords)
		if containsKeyword(marks, "crop") {
			return props.Px(8)
		}
		return props.Px(0)
	}
	return length(ctx, id, l)
}

func computeLetterSpacing(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok || l.Keyword == "normal" {
		return v
	}
	return length(ctx, id, l)
}

func computeBackgroundSize(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	sizes, ok := v.(props.BackgroundSizes)
	if !ok {
		return v
	}
	out := make(props.BackgroundSizes, len(sizes))
	for i, s := range sizes {
		if s.Keyword != "" {
			out[i] = s
			continue
		}
		out[i] = props.BackgroundSize{Width: length(ctx, id, s.Width), Height: length(ctx, id, s.Height)}
	}
	return out
}

// computeImageOrientation rounds the angle to a quarter turn, in degrees
// (CSS Images 3 §5.3).
func computeImageOrientation(_ props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	o, ok := v.(props.ImageOrientation)
	if !ok || o.Keyword != "" {
		return v
	}
	quarters := int(math.RoundToEven(o.Angle / math.Pi * 2))
	o.Angle = float64(((quarters%4)+4)%4) * 90
	return o
}

// borderStyles maps the width properties to the style that makes them zero.
var borderStyles = map[props.PropID]props.PropID{
	props.PropBorderTopWidth:    props.PropBorderTopStyle,
	props.PropBorderRightWidth:  props.PropBorderRightStyle,
	props.PropBorderBottomWidth: props.PropBorderBottomStyle,
	props.PropBorderLeftWidth:   props.PropBorderLeftStyle,
	props.PropColumnRuleWidth:   props.PropColumnRuleStyle,
	props.PropOutlineWidth:      props.PropOutlineStyle,
}

// computeBorderWidth makes the width zero when there is no border (CSS
// Backgrounds 3 §4.3).
func computeBorderWidth(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	if style, ok := ctx.Get(borderStyles[id]).(props.Keyword); ok && (style == "none" || style == "hidden") {
		return props.Pixels(0)
	}
	switch v := v.(type) {
	case props.Pixels:
		return v
	case props.Length:
		if px, ok := props.BorderWidthKeywords[v.Keyword]; ok {
			return props.Pixels(px)
		}
		return props.Pixels(pixels(ctx, id, v))
	}
	return v
}

func computeBorderImageOutset(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	quad, ok := v.(props.LengthQuad)
	if !ok {
		return v
	}
	for i, l := range quad {
		if l.Keyword == "" && l.Calc == nil && l.Dimension.Unit == "" {
			continue
		}
		quad[i] = length(ctx, id, l)
	}
	return quad
}

func computeGap(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok || l.Keyword == "normal" {
		return v
	}
	return length(ctx, id, l)
}

// contentTypes are the content items that can be computed.
var contentTypes = map[string]bool{
	"string": true, "content": true, "url": true, "quote": true, "leader()": true,
	"counter()": true, "counters()": true, "content()": true, "element()": true,
	"string()": true, "attr()": true, "target-counter()": true,
	"target-counters()": true, "target-text()": true,
}

func contentList(items []props.ContentItem) []props.ContentItem {
	out := make([]props.ContentItem, 0, len(items))
	for _, item := range items {
		if contentTypes[item.Type] {
			out = append(out, item)
		} else {
			logging.Warnf("Unable to compute content: %v", item)
		}
	}
	return out
}

func computeBookmarkLabel(_ props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	if list, ok := v.(props.ContentList); ok {
		return props.ContentList(contentList(list))
	}
	return v
}

func computeStringSet(_ props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	set, ok := v.(props.StringSet)
	if !ok {
		return v
	}
	out := make(props.StringSet, len(set))
	for i, entry := range set {
		out[i] = props.StringSetEntry{Name: entry.Name, Content: contentList(entry.Content)}
	}
	return out
}

// computeContent turns normal into inhibit for pseudo-elements and into
// contents for elements, and none into inhibit (CSS Generated Content 3
// §1.1).
func computeContent(ctx props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	c, ok := v.(props.Content)
	if !ok {
		return v
	}
	switch c.Keyword {
	case "normal":
		if ctx.PseudoType() != "" {
			return props.Content{Keyword: "inhibit"}
		}
		return props.Content{Keyword: "contents"}
	case "none":
		return props.Content{Keyword: "inhibit"}
	case "":
		return props.Content{Items: contentList(c.Items)}
	}
	return c
}

func keywordOf(v props.Value) props.Keyword {
	switch v := v.(type) {
	case props.Keyword:
		return v
	case props.Positioning:
		return v.Keyword
	}
	return ""
}

// computeDisplay blockifies absolutely positioned, floating and root boxes
// (CSS 2.1 §9.7).
func computeDisplay(ctx props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	display, ok := v.(props.Display)
	if !ok || len(display) == 0 {
		return v
	}
	float := keywordOf(ctx.Specified(props.PropFloat))
	position := keywordOf(ctx.Specified(props.PropPosition))
	if position == "absolute" || position == "fixed" || float != "none" || ctx.IsRootElement() {
		switch {
		case len(display) == 1 && display[0] == "inline-table":
			return props.Display{"block", "table"}
		case len(display) == 1 && len(display[0]) > 6 && display[0][:6] == "table-":
			return props.Display{"block", "flow"}
		case display[0] == "inline":
			if containsKeyword(props.Keywords(display), "list-item") {
				return props.Display{"block", "flow", "list-item"}
			}
			return props.Display{"block", "flow"}
		}
	}
	return display
}

// computeFloat removes the float of absolutely positioned and running
// elements (CSS 2.1 §9.7).
func computeFloat(ctx props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	switch keywordOf(ctx.Specified(props.PropPosition)) {
	case "absolute", "fixed", "running()", "note()":
		return props.Keyword("none")
	}
	return v
}

// fontSizeKeywords are the absolute size keywords, smallest first.
var fontSizeKeywords = []props.Keyword{"xx-small", "x-small", "small", "medium", "large", "x-large", "xx-large"}

// computeFontSize resolves keywords, percentages and font-relative units
// against the font size of the parent (CSS Fonts 4 §2.5).
func computeFontSize(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok {
		return v
	}
	if px, ok := props.FontSizeKeywords[l.Keyword]; ok {
		return props.Pixels(px)
	}
	parent := pixelsOf(ctx.ParentValue(id))
	switch l.Keyword {
	case "larger":
		for _, k := range fontSizeKeywords {
			if px := props.FontSizeKeywords[k]; px > parent {
				return props.Pixels(px)
			}
		}
		return props.Pixels(parent * 1.2)
	case "smaller":
		for i := len(fontSizeKeywords) - 1; i >= 0; i-- {
			if px := props.FontSizeKeywords[fontSizeKeywords[i]]; px < parent {
				return props.Pixels(px)
			}
		}
		return props.Pixels(parent * 0.8)
	}
	if l.Calc == nil && l.Dimension.Unit == "%" {
		return props.Pixels(l.Dimension.Value * parent / 100)
	}
	if l.Calc != nil {
		// Percentages in math functions also refer to the parent size.
		resolved, err := tk.ResolveMath(l.Calc.Function, ctx, id.Name(), &parent)
		if err != nil || resolved == nil {
			return props.Pixels(parent)
		}
		l, _ = tk.GetLength(resolved, true, true)
		if l.Dimension.Unit == "%" {
			return props.Pixels(l.Dimension.Value * parent / 100)
		}
	}
	return props.Pixels(pixels(ctx, id, l))
}

// fontWeightRelative gives bolder and lighter weights from the parent weight
// (CSS Fonts 4 §2.2).
var fontWeightRelative = map[props.Keyword]map[props.Integer]props.Integer{
	"bolder":  {100: 400, 200: 400, 300: 400, 400: 700, 500: 700, 600: 900, 700: 900, 800: 900, 900: 900},
	"lighter": {100: 100, 200: 100, 300: 100, 400: 100, 500: 100, 600: 400, 700: 400, 800: 700, 900: 700},
}

func computeFontWeight(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	k, ok := v.(props.Keyword)
	if !ok {
		return v
	}
	switch k {
	case "normal":
		return props.Integer(400)
	case "bold":
		return props.Integer(700)
	case "bolder", "lighter":
		parent, _ := ctx.ParentValue(id).(props.Integer)
		return fontWeightRelative[k][parent]
	}
	return v
}

func trackBreadth(ctx props.ComputeContext, id props.PropID, l props.Length) props.Length {
	switch {
	case l.Keyword != "":
		return l
	case l.Calc == nil && l.Dimension.Unit == "fr":
		return l
	}
	return length(ctx, id, l)
}

func trackSize(ctx props.ComputeContext, id props.PropID, t props.TrackSize) props.TrackSize {
	switch t.Function {
	case "minmax()":
		t.Min = trackBreadth(ctx, id, t.Min)
		t.Max = trackBreadth(ctx, id, t.Max)
	case "fit-content()":
		t.Breadth = length(ctx, id, t.Breadth)
	default:
		t.Breadth = trackBreadth(ctx, id, t.Breadth)
	}
	return t
}

func gridItems(ctx props.ComputeContext, id props.PropID, items []props.GridItem) []props.GridItem {
	out := make([]props.GridItem, len(items))
	for i, item := range items {
		switch item.Kind {
		case props.GridTrack:
			item.Track = trackSize(ctx, id, item.Track)
		case props.GridRepeat:
			item.Items = gridItems(ctx, id, item.Items)
		}
		out[i] = item
	}
	return out
}

func computeGridTemplate(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	g, ok := v.(props.GridTemplate)
	if !ok || g.None || g.Subgrid {
		return v
	}
	g.Items = gridItems(ctx, id, g.Items)
	return g
}

func computeGridAuto(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	list, ok := v.(props.TrackList)
	if !ok {
		return v
	}
	out := make(props.TrackList, len(list))
	for i, t := range list {
		out[i] = trackSize(ctx, id, t)
	}
	return out
}

// computeLineHeight keeps numbers, which are inherited as numbers, and turns
// percentages into lengths (CSS 2.1 §10.8.1).
func computeLineHeight(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok || l.Keyword == "normal" || l.Calc != nil || l.Dimension.Unit == "" {
		return v
	}
	if l.Dimension.Unit == "%" {
		fontSize := pixelsOf(ctx.Get(props.PropFontSize))
		return props.Px(l.Dimension.Value / 100 * fontSize)
	}
	return length(ctx, id, l)
}

func computeLang(_ props.ComputeContext, _ props.PropID, v props.Value) props.Value {
	if s, ok := v.(props.String); ok {
		return s
	}
	return props.String("")
}

func computeTabSize(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok || (l.Keyword == "" && l.Calc == nil && l.Dimension.Unit == "") {
		return v
	}
	return length(ctx, id, l)
}

func computeTransform(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	t, ok := v.(props.Transform)
	if !ok {
		return v
	}
	out := make(props.Transform, len(t))
	for i, op := range t {
		if op.Name == "translate" {
			op.Translate = props.LengthPair{X: length(ctx, id, op.Translate.X), Y: length(ctx, id, op.Translate.Y)}
		}
		out[i] = op
	}
	return out
}

// computeVerticalAlign raises super and sub by half an em, and resolves
// percentages against the line height (CSS 2.1 §10.8.1).
func computeVerticalAlign(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok || l.Calc != nil {
		return v
	}
	switch l.Keyword {
	case "":
	case "super":
		return props.Px(pixelsOf(ctx.Get(props.PropFontSize)) * 0.5)
	case "sub":
		return props.Px(pixelsOf(ctx.Get(props.PropFontSize)) * -0.5)
	default:
		return v
	}
	if l.Dimension.Unit == "%" {
		height, _ := ctx.Strut()
		return props.Px(height * l.Dimension.Value / 100)
	}
	return props.Px(pixels(ctx, id, l))
}

func computeWordSpacing(ctx props.ComputeContext, id props.PropID, v props.Value) props.Value {
	l, ok := v.(props.Length)
	if !ok {
		return v
	}
	if l.Keyword == "normal" {
		return props.Pixels(0)
	}
	return props.Pixels(pixels(ctx, id, l))
}
