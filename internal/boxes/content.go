package boxes

import (
	"maps"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// PageContext gives the content of page margin boxes the named strings and
// the running elements of a page (CSS GCPM 3 §1, §2).
type PageContext interface {
	// StringSetFor returns the values of the named string for the page,
	// selected by keyword (first, start, last or first-except).
	StringSetFor(page *PageBox, name, keyword string) []string
	// RunningElementFor returns the running elements of the given name
	// for the page, selected by keyword (first, start, last, first-except
	// or all-once).
	RunningElementFor(page *PageBox, name, keyword string) []Box
}

// contentList holds what the items of a content list are computed with.
type contentList struct {
	parent        Box
	counterValues targets.CounterValues
	// property names the property the list comes from, for the targets
	// that wait for it: "content", "bookmark-label" or "string-set::name".
	property   string
	parseAgain targets.ParseAgain
	// images allows url() items to make replaced boxes.
	images bool
	// quoteDepth and quotes are nil when quotes are ignored.
	quoteDepth *int
	quotes     *props.Quotes
	lang       string
	page       *PageBox
	pageCtx    PageContext
}

// textAdder collects the text and boxes made from a content list, merging
// consecutive texts into one text box.
type textAdder struct {
	parent  Box
	boxes   []Box
	hasText bool
}

func (a *textAdder) add(text string) {
	a.hasText = true
	if text == "" {
		return
	}
	if n := len(a.boxes); n > 0 {
		if t, ok := a.boxes[n-1].(*TextBox); ok {
			t.Text += text
			return
		}
	}
	a.boxes = append(a.boxes, AnonymousText(a.parent, text))
}

// computeContentList makes the boxes of a content list (CSS Content 3 §2,
// CSS GCPM 3). It reports false when the list produced nothing, not even an
// empty string, as when a target is not known yet. Content waiting for a
// target or for counters known only during pagination is registered in the
// target collector with c.parseAgain.
func (b *Builder) computeContentList(items []props.ContentItem, c contentList) ([]Box, bool) {
	out := &textAdder{parent: c.parent}
	missingCounters := map[string]bool{}
	missingTargetCounters := map[string]map[string]bool{}
	inPageContext := c.pageCtx != nil && c.page != nil
	// Missing counters are only collected while the tree is built; margin
	// boxes have no use for target counters.
	collectMissing := b.Targets.Collecting && !inPageContext

	parent := c.parent.Base()
	if parent.CounterValues == nil {
		// Kept for the content built later, in page margins.
		parent.CounterValues = cloneCounters(c.counterValues)
	}

items:
	for _, item := range items {
		switch item.Type {
		case "string":
			out.add(arg(item, 0))
		case "url":
			if !c.images {
				continue
			}
			if arg(item, 0) != "external" {
				// Internal references cannot be embedded.
				continue
			}
			if img := b.image(arg(item, 1), "", c.parent.Base().Style); img != nil {
				out.boxes = append(out.boxes, AnonymousInlineReplaced(c.parent, img))
			}
		case "attr()":
			if kind := arg(item, 1); kind == "string" {
				var value string
				if parent.Element != nil {
					value = parent.Element.Get(arg(item, 0))
				}
				out.add(value)
			} else {
				logging.Warnf("Only strings are allowed for content attr() functions, not %s", kind)
			}
		case "content()":
			out.add(ExtractText(c.parent, arg(item, 0)))
		case "string()":
			if !inPageContext {
				// string() is only valid in page margins.
				logging.Warnf(`"string(%s)" is only allowed in page margins`, strings.Join(item.Args, " "))
				continue
			}
			out.add(strings.Join(c.pageCtx.StringSetFor(c.page, arg(item, 0), arg(item, 1)), ""))
		case "counter()", "counters()":
			name := arg(item, 0)
			style := counterStyleOf(item)
			if style.Kind == props.ListStyleKeyword && style.Name == "none" {
				continue
			}
			if _, ok := c.counterValues[name]; collectMissing && !ok {
				missingCounters[name] = true
			}
			if item.Type == "counter()" {
				out.add(b.CounterStyle.RenderValue(lastValue(c.counterValues, name), style))
				continue
			}
			values := c.counterValues[name]
			if len(values) == 0 {
				values = []int{0}
			}
			parts := make([]string, len(values))
			for i, v := range values {
				parts[i] = b.CounterStyle.RenderValue(v, style)
			}
			out.add(strings.Join(parts, arg(item, 1)))
		case "target-counter()", "target-counters()":
			name, styleName := arg(item, 0), arg(item, 1)
			if styleName == "none" || item.Link == nil {
				continue
			}
			lookup := b.Targets.LookupTarget(*item.Link, c.parent.(targets.Source), c.property, c.parseAgain)
			if lookup.State != targets.UpToDate {
				break items
			}
			targetValues := lookup.TargetBox.CachedCounterValues()
			if _, ok := targetValues[name]; collectMissing && !ok {
				anchor, _ := targets.AnchorName(*item.Link)
				if missingTargetCounters[anchor] == nil {
					missingTargetCounters[anchor] = map[string]bool{}
				}
				missingTargetCounters[anchor][name] = true
			}
			// The page counters of the target are empty until pagination.
			local := maps.Clone(lookup.CachedPageCounterValues)
			if local == nil {
				local = targets.CounterValues{}
			}
			maps.Copy(local, targetValues)
			if item.Type == "target-counter()" {
				out.add(b.CounterStyle.RenderValueOf(lastValue(local, name), styleName))
				continue
			}
			if item.Separator == nil || item.Separator.Type != "string" {
				break items
			}
			values := local[name]
			if len(values) == 0 {
				values = []int{0}
			}
			parts := make([]string, len(values))
			for i, v := range values {
				parts[i] = b.CounterStyle.RenderValueOf(v, styleName)
			}
			out.add(strings.Join(parts, arg(*item.Separator, 0)))
		case "target-text()":
			if item.Link == nil {
				continue
			}
			lookup := b.Targets.LookupTarget(*item.Link, c.parent.(targets.Source), c.property, c.parseAgain)
			if lookup.State != targets.UpToDate {
				break items
			}
			target, _ := lookup.TargetBox.(Box)
			if target == nil {
				break items
			}
			out.add(ExtractText(target, arg(item, 0)))
		case "quote":
			if c.quoteDepth == nil || c.quotes == nil {
				continue
			}
			value := arg(item, 0)
			isOpen := strings.Contains(value, "open")
			insert := !strings.HasPrefix(value, "no-") && c.quotes.Keyword != "none"
			if !isOpen {
				*c.quoteDepth = max(0, *c.quoteDepth-1)
			}
			if insert {
				open, close := c.quotes.Open, c.quotes.Close
				if c.quotes.Keyword == "auto" {
					q := quotesForLang(c.lang)
					open, close = q.open, q.close
				}
				quotes := close
				if isOpen {
					quotes = open
				}
				if len(quotes) > 0 {
					out.add(quotes[min(*c.quoteDepth, len(quotes)-1)])
				}
			}
			if isOpen {
				*c.quoteDepth++
			}
		case "element()":
			if !inPageContext {
				logging.Warnf(`"element(%s)" is only allowed in page margins`, strings.Join(item.Args, " "))
				continue
			}
			for _, running := range c.pageCtx.RunningElementFor(c.page, arg(item, 0), arg(item, 1)) {
				out.boxes = append(out.boxes, b.runningElementCopy(running, arg(item, 1), c))
			}
		case "leader()":
			text := arg(item, 0)
			if text == "" {
				continue
			}
			textBox := AnonymousText(c.parent, text)
			leader := AnonymousFrom(KindInline, c.parent, []Box{textBox})
			// The anonymous style is shared with the text box: neither the
			// leader can break nor its spaces collapse.
			leader.Base().Style.Set(props.PropWhiteSpace, props.Keyword("pre"))
			textBox.Style.Set(props.PropWhiteSpace, props.Keyword("pre"))
			leader.Base().IsLeader = true
			out.boxes = append(out.boxes, leader)
		}
	}

	if !out.hasText && len(out.boxes) == 0 {
		return nil, false
	}
	b.Targets.CollectMissingCounters(c.parent.(targets.Linked), c.property, c.parseAgain, missingCounters, missingTargetCounters)
	return out.boxes, true
}

// runningElementCopy returns a copy of a running element placed in a page
// margin. Unless the element is shown once with all-once, the generated
// content of its descendants uses the counters of the page.
func (b *Builder) runningElementCopy(running Box, keyword string, c contentList) Box {
	box := running.DeepCopy()
	base := box.Base()
	base.Style = base.Style.Copy()
	base.Style.Set(props.PropPosition, props.Positioning{Keyword: "relative"})
	if keyword == "all-once" {
		return box
	}
	for d := range Descendants(box, false) {
		p, ok := d.(Parent)
		if !ok {
			continue
		}
		switch d.Base().Style.Content().Keyword {
		case "normal", "none":
			continue
		}
		p.ParentBase().Children = b.ContentToBoxes(d.Base().Style, d, c.quoteDepth, c.counterValues, c.page, c.pageCtx)
	}
	return box
}

func arg(item props.ContentItem, i int) string {
	if i < len(item.Args) {
		return item.Args[i]
	}
	return ""
}

func counterStyleOf(item props.ContentItem) props.ListStyleType {
	if item.Style != nil {
		return *item.Style
	}
	return props.ListStyleType{Name: "decimal"}
}

func cloneCounters(values targets.CounterValues) targets.CounterValues {
	if c := values.Clone(); c != nil {
		return c
	}
	return targets.CounterValues{}
}

// mixCounters returns the page counters, if any, updated with the counters
// cached on a box when its content was first built.
func mixCounters(pageCounters, cached targets.CounterValues) targets.CounterValues {
	local := targets.CounterValues{}
	maps.Copy(local, pageCounters)
	maps.Copy(local, cached)
	return local
}

// ContentToBoxes makes the boxes of the content property of a style for
// the box parent. Quotes change quoteDepth. page and pageCtx are set for
// the content of page margins, where string() and element() are allowed.
func (b *Builder) ContentToBoxes(style *computed.Style, parent Box, quoteDepth *int, counterValues targets.CounterValues, page *PageBox, pageCtx PageContext) []Box {
	content := style.Content()
	if content.Keyword != "" {
		return nil
	}
	origQuoteDepth := *quoteDepth
	quotes := style.Quotes()
	parseAgain := func(pageCounters targets.CounterValues) {
		local := mixCounters(pageCounters, parent.Base().CounterValues)
		children := b.ContentToBoxes(style, parent, &origQuoteDepth, local, nil, nil)
		// The parent may already be wrapped in lines, or not.
		p := parent.(Parent).ParentBase()
		if len(p.Children) == 1 {
			if line, ok := p.Children[0].(*LineBox); ok {
				line.Children = children
				return
			}
		}
		p.Children = children
	}
	boxes, _ := b.computeContentList(content.Items, contentList{
		parent:        parent,
		counterValues: counterValues,
		property:      "content",
		parseAgain:    parseAgain,
		images:        true,
		quoteDepth:    quoteDepth,
		quotes:        &quotes,
		lang:          string(style.Lang()),
		page:          page,
		pageCtx:       pageCtx,
	})
	return boxes
}

// computeStringSet sets the value of a named string of string-set on the
// box (CSS GCPM 3 §1.1).
func (b *Builder) computeStringSet(box Box, name string, items props.ContentList, counterValues targets.CounterValues) {
	parseAgain := func(pageCounters targets.CounterValues) {
		b.computeStringSet(box, name, items, mixCounters(pageCounters, box.Base().CounterValues))
	}
	list, ok := b.computeContentList(items, contentList{
		parent:        box,
		counterValues: counterValues,
		property:      "string-set::" + name,
		parseAgain:    parseAgain,
	})
	if !ok {
		return
	}
	var text strings.Builder
	for _, child := range list {
		if t, ok := child.(*TextBox); ok {
			text.WriteString(t.Text)
		}
	}
	base := box.Base()
	for i, s := range base.StringSet {
		if s.Name == name {
			base.StringSet = append(base.StringSet[:i:i], base.StringSet[i+1:]...)
			break
		}
	}
	base.StringSet = append(base.StringSet, StringSetItem{Name: name, Text: text.String()})
}

// computeBookmarkLabel sets the bookmark label of the box (CSS GCPM 3
// §3.2).
func (b *Builder) computeBookmarkLabel(box Box, items props.ContentList, counterValues targets.CounterValues) {
	parseAgain := func(pageCounters targets.CounterValues) {
		b.computeBookmarkLabel(box, items, mixCounters(pageCounters, box.Base().CounterValues))
	}
	list, _ := b.computeContentList(items, contentList{
		parent:        box,
		counterValues: counterValues,
		property:      "bookmark-label",
		parseAgain:    parseAgain,
	})
	if len(list) == 0 {
		return
	}
	var label strings.Builder
	for _, child := range list {
		label.WriteString(ContentText(child))
	}
	box.Base().BookmarkLabel = label.String()
}

// setContentLists computes the string-set and bookmark-label of a box.
func (b *Builder) setContentLists(box Box, style *computed.Style, counterValues targets.CounterValues) {
	box.Base().StringSet = []StringSetItem{}
	for _, entry := range style.StringSet() {
		b.computeStringSet(box, entry.Name, entry.Content, counterValues)
	}
	if style.BookmarkLevel().Keyword != "none" {
		b.computeBookmarkLabel(box, style.BookmarkLabel(), counterValues)
	}
}
