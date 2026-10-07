package boxes

import (
	"errors"
	"slices"
	"strings"

	nethtml "golang.org/x/net/html"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// Builder turns the elements of a document and their computed styles into
// the box tree that layout takes (CSS 2.1 §9.2, CSS Display 3 §2).
type Builder struct {
	StyleFor *cascade.StyleFor
	// GetImage loads the images of replaced elements, list markers and
	// generated content. Nil leaves them out.
	GetImage images.Getter
	// InlineSVG makes the image of an svg element of the document. When it
	// is nil or fails, the element is left out.
	InlineSVG func(el *html.Element) (images.Image, error)
	// Targets collects the anchors of the document and the content that
	// waits for them.
	Targets      *targets.Collector
	CounterStyle counters.Registry
	// Footnotes receives the footnotes taken out of the flow, in document
	// order.
	Footnotes []Box

	// root is set when the root element has no box: the document is built
	// again with only a block for the root element.
	root *html.Element
	err  error
}

// Err returns the error that stopped the loading of images, if any. It is
// fatal: rendering must stop.
func (b *Builder) Err() error { return b.err }

// Build builds the box tree of a document: the boxes of the elements, their
// white space processed and text transformed, with the anonymous boxes that
// the formatting model requires. The root box is marked as the box of the
// root element and gets the overflow of the viewport. The error is fatal and
// comes from loading an image.
func (b *Builder) Build(doc *html.Document) (Box, error) {
	list := b.ElementToBox(doc.Root, nil)
	if len(list) == 0 {
		b.root = doc.Root
		list = b.ElementToBox(doc.Root, nil)
		b.root = nil
	}
	box := list[0]

	b.Targets.CheckPendingTargets()
	ProcessWhitespace(box, false)
	ProcessTextTransform(box)

	box.Base().IsForRootElement = true
	box = CreateAnonymousBoxes(box)
	box = setViewportOverflow(box)
	return box, b.err
}

// style returns the computed style of an element or pseudo-element, nil
// for a pseudo-element without declarations.
func (b *Builder) style(el *html.Element, pseudo string) *computed.Style {
	s := b.StyleFor.Get(el, pseudo)
	if s == nil || b.root == nil {
		return s
	}
	// Without a root box, only the root element is displayed, as a block.
	if el == b.root {
		s.Set(props.PropDisplay, props.Display{"block", "flow"})
	} else {
		s.Set(props.PropDisplay, props.Display{"none"})
	}
	return s
}

// image loads an image for a box of the given style, or returns nil.
func (b *Builder) image(url, mimeType string, style *computed.Style) images.Image {
	if b.GetImage == nil || b.err != nil {
		return nil
	}
	img, err := b.GetImage(url, mimeType, orientationOf(style))
	if err != nil {
		b.err = err
		return nil
	}
	return img
}

func orientationOf(style *computed.Style) images.Orientation {
	o := style.ImageOrientation()
	switch o.Keyword {
	case "from-image":
		return images.OrientationFromImage
	case "none":
		return images.Orientation{}
	}
	return images.Orientation{Angle: int(o.Angle), Flip: o.Flip}
}

// elementTag is the tag boxes give to an element: its local name for HTML
// elements, and the namespace in braces before it for the others.
func elementTag(el *html.Element) string {
	if ns := el.Namespace(); ns != "" && ns != html.XHTMLNamespace {
		return "{" + ns + "}" + el.LocalName()
	}
	return el.LocalName()
}

// boxKind gives the kind of the box that a display value makes.
func boxKind(display props.Display) Kind {
	if len(display) >= 2 {
		inline := display[0] == "inline"
		switch display[1] {
		case "flow":
			if inline {
				return KindInline
			}
			return KindBlock
		case "flow-root":
			if inline {
				return KindInlineBlock
			}
			return KindBlock
		case "table":
			if inline {
				return KindInlineTable
			}
			return KindTable
		case "flex":
			if inline {
				return KindInlineFlex
			}
			return KindFlex
		case "grid":
			if inline {
				return KindInlineGrid
			}
			return KindGrid
		}
	}
	if len(display) == 1 {
		switch display[0] {
		case "table-row":
			return KindTableRow
		case "table-row-group", "table-header-group", "table-footer-group":
			return KindTableRowGroup
		case "table-column":
			return KindTableColumn
		case "table-column-group":
			return KindTableColumnGroup
		case "table-cell":
			return KindTableCell
		case "table-caption":
			return KindTableCaption
		}
	}
	return KindInline
}

// MakeBox returns a box of the kind that the display of style makes.
func MakeBox(tag string, style *computed.Style, el *html.Element, children []Box) Parent {
	return New(boxKind(style.Display()), tag, style, el, children)
}

func isNone(display props.Display) bool { return len(display) == 1 && display[0] == "none" }

// ElementToBox returns the boxes of an element and its descendants:
// usually one, none for an element that is not displayed, and more for a
// footnote or note, or for columns that span several columns. state is the
// state of the walk, nil to start one. White space is not processed and no
// anonymous box is added.
func (b *Builder) ElementToBox(el *html.Element, state *State) []Box {
	style := b.style(el, "")
	if style == nil || isNone(style.Display()) {
		return nil
	}

	if style.Float() == "footnote" {
		if style.FootnoteDisplay() == "block" {
			style.Set(props.PropDisplay, props.Display{"block", "flow"})
		} else {
			style.Set(props.PropDisplay, props.Display{"inline", "flow"})
		}
	}

	tag := elementTag(el)
	box := MakeBox(tag, style, el, nil)
	box.Base().FirstLetterStyle = b.style(el, "first-letter")
	box.Base().FirstLineStyle = b.style(el, "first-line")

	if state == nil {
		state = NewState()
	}
	UpdateCounters(state, style)

	var children []Box
	// The counters whose scope is created by the children of the element.
	state.pushScope()

	var markers []Box
	if slices.Contains(style.Display(), "list-item") {
		if m := b.markerToBox(el, state, style); m != nil {
			markers = []Box{m}
			children = append(children, m)
		}
	}
	children = append(children, b.beforeAfterToBox(el, "before", state)...)

	// The counters of an anchor are those after ::before and before
	// ::after (CSS GCPM 3 §4.1).
	if anchor := style.Anchor(); anchor != "" {
		b.Targets.StoreTarget(string(anchor), state.CounterValues, box.(targets.Target))
	}

	addText := func(text string) {
		if text == "" {
			return
		}
		if n := len(children); n > 0 {
			if t, ok := children[n-1].(*TextBox); ok {
				t.Text += text
				return
			}
		}
		children = append(children, AnonymousText(box, text))
	}
	doc := el.Document()
	for n := el.Node().FirstChild; n != nil; n = n.NextSibling {
		switch n.Type {
		case nethtml.TextNode:
			addText(n.Data)
		case nethtml.ElementNode:
			child := doc.Element(n)
			if child == nil {
				continue
			}
			children = append(children, b.childBoxes(child, state)...)
		}
	}

	children = append(children, b.beforeAfterToBox(el, "after", state)...)
	state.popScope()

	setChildren(box, children)
	b.setContentLists(box, style, state.CounterValues)

	if len(markers) > 0 && len(box.ParentBase().Children) == 1 && style.ListStylePosition() == "outside" {
		// The marker alone would give no height to the list item, whose
		// interaction with the marker is undefined (CSS Lists 3 §3.1.1): a
		// zero width space gives it a line.
		box.ParentBase().Children = append(box.ParentBase().Children, AnonymousText(box, "\u200b"))
	}

	if style.Float() == "footnote" {
		incrementLast(state.CounterValues, "footnote")
		marker := b.pseudoBox(el, box, "footnote-marker", state)
		box.ParentBase().Children = slices.Insert(box.ParentBase().Children, 0, Box(marker))
	} else if style.Position().Keyword == "note()" {
		incrementLast(state.CounterValues, "note")
		marker := b.pseudoBox(el, box, "note-marker", state)
		box.ParentBase().Children = slices.Insert(box.ParentBase().Children, 0, Box(marker))
		callback := b.pseudoBox(el, box, "note-callback", state)
		box.ParentBase().Children = append(box.ParentBase().Children, callback)
		box.Base().NoteCallback = callback
	}

	return b.handleElement(el, box)
}

// childBoxes returns the boxes of a child element. A footnote is moved to
// the footnotes and replaced by its call, and a note is preceded by its
// call (CSS GCPM 3 §2).
func (b *Builder) childBoxes(child *html.Element, state *State) []Box {
	list := b.ElementToBox(child, state)
	if len(list) == 0 {
		return nil
	}
	first := list[0].Base()
	if first.Style.Float() == "footnote" {
		footnote := list[0]
		first.Style = first.Style.Copy()
		first.Style.Set(props.PropFloat, props.Keyword("none"))
		b.Footnotes = append(b.Footnotes, footnote)
		call := b.pseudoBox(first.Element, footnote, "footnote-call", state)
		call.Base().Footnote = footnote
		return []Box{call}
	}
	if first.IsNote() {
		note := list[0]
		first.Style = first.Style.Copy()
		call := b.pseudoBox(first.Element, note, "note-call", state)
		first.NoteCall, call.Base().Note = call, note
		return []Box{call, note}
	}
	return list
}

// pseudoBox returns the box of a pseudo-element of the GCPM, such as
// ::footnote-call, with its content. A pseudo-element without style gets
// the anonymous style of box.
func (b *Builder) pseudoBox(el *html.Element, box Box, pseudo string, state *State) Parent {
	style := b.style(el, pseudo)
	if style == nil {
		style = box.Base().Style.Anonymous()
	}
	p := MakeBox(elementTag(el)+"::"+pseudo, style, el, nil)
	p.ParentBase().Children = b.ContentToBoxes(style, p, &state.QuoteDepth, state.CounterValues, nil, nil)
	return p
}

// beforeAfterToBox returns the box of the ::before or ::after
// pseudo-element of an element, if it has content (CSS Content 3 §1).
func (b *Builder) beforeAfterToBox(el *html.Element, pseudo string, state *State) []Box {
	style := b.style(el, pseudo)
	if style == nil || isNone(style.Display()) {
		return nil
	}
	switch style.Content().Keyword {
	case "normal", "inhibit", "none":
		return nil
	}
	box := MakeBox(elementTag(el)+"::"+pseudo, style, el, nil)
	UpdateCounters(state, style)

	var children []Box
	if slices.Contains(style.Display(), "list-item") {
		if m := b.markerToBox(el, state, style); m != nil {
			children = append(children, m)
		}
	}
	children = append(children, b.ContentToBoxes(style, box, &state.QuoteDepth, state.CounterValues, nil, nil)...)
	box.ParentBase().Children = children

	if style.BookmarkLevel().Keyword != "none" {
		b.computeBookmarkLabel(box, style.BookmarkLabel(), state.CounterValues)
	}
	return []Box{box}
}

// markerToBox returns the box of the ::marker pseudo-element of a list
// item, or nil when it has no content (CSS Lists 3 §3). An outside marker
// is an absolutely positioned block, an inside one an inline box.
func (b *Builder) markerToBox(el *html.Element, state *State, parentStyle *computed.Style) Box {
	style := b.style(el, "marker")
	if style == nil {
		style = parentStyle.Anonymous()
	}
	var box Box = MakeBox(elementTag(el)+"::marker", style, el, nil)
	if isNone(style.Display()) {
		return nil
	}

	var children []Box
	if k := style.Content().Keyword; k != "normal" && k != "inhibit" {
		children = b.ContentToBoxes(style, box, &state.QuoteDepth, state.CounterValues, nil, nil)
	} else {
		if image := style.ListStyleImage(); image.Type == "url" {
			// The image may not be available.
			if img := b.image(image.URL, "", style); img != nil {
				box = AnonymousInlineReplaced(box, img)
				children = append(children, box)
			}
		}
		if listStyle := style.ListStyleType(); len(children) == 0 && !(listStyle.Kind == props.ListStyleKeyword && listStyle.Name == "none") {
			value := lastValue(state.CounterValues, "list-item")
			if text := b.CounterStyle.RenderMarker(listStyle, value); text != "" {
				box = AnonymousText(box, text)
				children = append(children, box)
			}
		}
	}
	if len(children) == 0 {
		return nil
	}

	if parentStyle.ListStylePosition() == "outside" {
		marker := AnonymousFrom(KindBlock, box, children)
		// The position of a marker cannot be changed by author styles
		// (CSS Pseudo 4 §3.1).
		base := marker.Base()
		base.Style = base.Style.Copy()
		base.Style.Set(props.PropPosition, props.Positioning{Keyword: "absolute"})
		base.IsOutsideMarker = true
		return marker
	}
	return AnonymousFrom(KindInline, box, children)
}

// handleElement applies what HTML gives to some elements: replaced
// content, links and table columns (HTML Living Standard §15).
func (b *Builder) handleElement(el *html.Element, box Parent) []Box {
	switch el.Kind() {
	case html.KindImg:
		info := el.Img()
		if info.Src != "" {
			if img := b.image(info.Src, "", box.Base().Style); img != nil {
				return []Box{makeReplacedBox(el, box, img)}
			}
		}
		// The image is not valid: its alternative text replaces it.
		if info.Alt != "" {
			box.ParentBase().Children = []Box{AnonymousText(box, info.Alt)}
			return []Box{box}
		}
		return nil
	case html.KindLink:
		box.Base().Link = el.Link()
	case html.KindEmbed:
		// Embedded content has no fallback.
		if info, ok := el.Embed(); ok {
			if img := b.image(info.URL, info.MIMEType, box.Base().Style); img != nil {
				return []Box{makeReplacedBox(el, box, img)}
			}
		}
		return nil
	case html.KindObject:
		// The children of the element are its fallback.
		if info, ok := el.Object(); ok {
			if img := b.image(info.URL, info.MIMEType, box.Base().Style); img != nil {
				return []Box{makeReplacedBox(el, box, img)}
			}
		}
	case html.KindColgroup:
		if group, ok := box.(*TableColumnGroupBox); ok && !el.HasChildElement("col") {
			span := group.Span()
			columns := make([]Box, span)
			for i := range columns {
				columns[i] = AnonymousFrom(KindTableColumn, group, nil)
			}
			group.Children = columns
		}
	case html.KindCol:
		// A column with span is repeated (CSS 2.1 §17.2).
		if col, ok := box.(*TableColumnBox); ok && col.Span() > 1 {
			columns := make([]Box, col.Span())
			for i := range columns {
				columns[i] = col.Copy()
			}
			return columns
		}
	case html.KindSVG:
		img, err := b.inlineSVG(el)
		if err != nil {
			logging.Errorf("Failed to load inline SVG: %s", err)
			return nil
		}
		return []Box{makeReplacedBox(el, box, img)}
	}
	return []Box{box}
}

// incrementLast increments the innermost instance of a counter in scope.
func incrementLast(values targets.CounterValues, name string) {
	if v := values[name]; len(v) > 0 {
		v[len(v)-1]++
	}
}

var errSVGUnsupported = errors.New("SVG images are not supported")

func (b *Builder) inlineSVG(el *html.Element) (images.Image, error) {
	if b.InlineSVG == nil {
		return nil, errSVGUnsupported
	}
	img, err := b.InlineSVG(el)
	if err == nil && img == nil {
		err = errSVGUnsupported
	}
	return img, err
}

// makeReplacedBox replaces the box of an element by a replaced box with
// the given content, block-level or inline-level as the box was.
func makeReplacedBox(el *html.Element, box Box, img images.Image) Box {
	base := box.Base()
	var replaced Box
	if slices.Contains(base.Style.Display(), "block") {
		replaced = NewBlockReplacedBox(elementTag(el), base.Style, el, img)
	} else {
		replaced = NewInlineReplacedBox(elementTag(el), base.Style, el, img)
	}
	replaced.Base().StringSet = base.StringSet
	replaced.Base().BookmarkLabel = base.BookmarkLabel
	return replaced
}

// setViewportOverflow propagates the overflow of the root element, or of
// the body element when the root has none, to the viewport (CSS 2.1
// §11.1.1).
func setViewportOverflow(root Box) Box {
	chosen := root
	if strings.EqualFold(root.Base().ElementTag, "html") && root.Base().Style.Overflow() == "visible" {
		for _, child := range ChildrenOf(root) {
			if strings.EqualFold(child.Base().ElementTag, "body") {
				chosen = child
				break
			}
		}
	}
	root.Base().ViewportOverflow = string(chosen.Base().Style.Overflow())
	chosen.Base().Style.Set(props.PropOverflow, props.Keyword("visible"))
	return root
}
