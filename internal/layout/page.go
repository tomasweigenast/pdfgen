package layout

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// maxPages stops a layout that does not progress from filling the memory
// with pages.
const maxPages = 1 << 17

type pageResult struct {
	Page     *boxes.PageBox
	ResumeAt ResumeAt
	NextPage PageBreak
}

// setPageWidthOrHeight sets the width and horizontal margins of a page box,
// or its height and vertical margins, as for a non-replaced block in normal
// flow: "if the values are over-constrained, instead of ignoring any margins,
// the containing block is resized to coincide with the margin edges of the
// page box" (CSS Paged Media 3 §5.2, CSS 2.1 §10.3.3).
func setPageWidthOrHeight(o *orientedBox, containingSize float64) {
	remaining := containingSize - o.paddingPlusBorder
	switch {
	case o.inner.Auto:
		if o.marginA.Auto {
			o.marginA = units.F(0)
		}
		if o.marginB.Auto {
			o.marginB = units.F(0)
		}
		o.inner = units.F(remaining - o.marginA.V - o.marginB.V)
	case o.marginA.Auto && o.marginB.Auto:
		half := (remaining - o.inner.V) / 2
		o.marginA, o.marginB = units.F(half), units.F(half)
	case o.marginA.Auto:
		o.marginA = units.F(remaining - o.inner.V - o.marginB.V)
	case o.marginB.Auto:
		o.marginB = units.F(remaining - o.inner.V - o.marginA.V)
	}
	o.restore()
}

func pageWidth(ctx *Context, page *boxes.PageBox, containingWidth float64) {
	withMinMaxWidth(&page.BoxFields, func() {
		setPageWidthOrHeight(newOrientedBox(ctx, page, false), containingWidth)
	})
}

func pageHeight(ctx *Context, page *boxes.PageBox, containingHeight float64) {
	withMinMaxHeight(&page.BoxFields, func() {
		setPageWidthOrHeight(newOrientedBox(ctx, page, true), containingHeight)
	})
}

// makePage takes just enough content from the beginning of the document, or
// from where the previous page stopped, to fill one page. It returns the
// laid-out page without its margin boxes, where the next page resumes
// (nil after the last one) and the break requested before it. pageNumber
// starts at 1. state holds the page counters and groups at the start of the
// page; it is updated to the state at its end.
func makePage(ctx *Context, root boxes.Box, pageType cascade.PageType, resume ResumeAt, pageNumber int, state *PageState) pageResult {
	style := ctx.StyleFor.Page(pageType, "")
	if style == nil {
		ctx.fail(fmt.Errorf("page %d has no computed style", pageNumber))
		return pageResult{}
	}
	// The overflow of the root or body element propagates to the viewport.
	if overflow := root.Base().ViewportOverflow; overflow != "" {
		style.Set(props.PropOverflow, props.Keyword(overflow))
	}
	page := boxes.NewPageBox(pageType, style)

	device := style.Size()
	deviceWidth, deviceHeight := float64(device.X), float64(device.Y)
	resolvePercentages(page, containingBlock{Width: units.F(deviceWidth), Height: units.F(deviceHeight)})

	page.PositionX, page.PositionY = 0, 0
	pageWidth(ctx, page, deviceWidth)
	pageHeight(ctx, page, deviceHeight)

	// Viewport units refer to the first page, so they are only known once it
	// has its size.
	if pageNumber == 1 {
		pageContext := ctx.StyleFor.Context()
		pageContext.PageBox = &computed.Size{Width: deviceWidth, Height: deviceHeight}
		pageContext.PageArea = &computed.Size{Width: page.Width.V, Height: page.Height.V}
	}

	ctx.PageBottom = page.ContentBoxY() + page.Height.V
	initialContainingBlock := boxes.Box(page)

	footnoteAreaStyle := ctx.StyleFor.Page(pageType, "@footnote")
	if footnoteAreaStyle == nil {
		footnoteAreaStyle = style.Anonymous()
	}
	footnoteArea := boxes.NewFootnoteAreaBox(page, footnoteAreaStyle)
	resolvePercentages(footnoteArea, containingBlockOf(page))
	footnoteArea.PositionX = page.ContentBoxX()
	footnoteArea.PositionY = ctx.PageBottom

	var previousResume ResumeAt
	if pageType.Blank {
		previousResume = resume
		parent, ok := root.(boxes.Parent)
		if !ok {
			ctx.fail(fmt.Errorf("the root box is a %s box without children", root.Kind()))
			return pageResult{}
		}
		root = parent.CopyWithChildren(nil)
	}

	// CSS Display 3 §2.1: the root box is block-level.
	if _, ok := root.(boxes.BlockLevel); !ok {
		ctx.fail(fmt.Errorf("the root box is a %s box, which is not block-level", root.Kind()))
		return pageResult{}
	}
	ctx.CreateBFC(nil, nil)
	ctx.CurrentPage = pageNumber
	ctx.CurrentPageFootnotes = nil
	ctx.FootnoteArea = footnoteArea

	reported := ctx.ReportedFootnotes
	ctx.ReportedFootnotes = nil
	for i, footnote := range reported {
		ctx.Footnotes = append(ctx.Footnotes, footnote)
		overflow := ctx.layoutFootnote(footnote)
		if overflow && i != 0 {
			ctx.reportFootnote(footnote)
			ctx.ReportedFootnotes = slices.Clone(reported[i:])
			break
		}
	}

	root.Base().PositionX = page.ContentBoxX()
	root.Base().PositionY = page.ContentBoxY()
	noteArea := makeNoteArea(ctx, page, style, root, state, initialContainingBlock)
	if noteArea != nil && noteArea.IsInNormalFlow() && len(boxes.ChildrenOf(noteArea)) != 0 {
		root.Base().PositionY = noteArea.Base().PositionY + noteArea.MarginHeight()
	}

	// Display out-of-flow boxes broken on the previous page.
	pageIsEmpty := true
	positioned := &positionedBoxes{}
	var outOfFlow []boxes.Box
	preloaded := make(map[boxes.Box][]boxes.Box)
	var preloadedRoots []boxes.Box
	previousBroken := ctx.brokenOutOfFlow
	ctx.brokenOutOfFlow = nil
	for _, record := range previousBroken {
		if record.BFCRoot != nil {
			ctx.CreateBFC(record.BFCRoot, nil)
		}
		record.Original.Base().PositionY = root.ContentBoxY()
		var fragment boxes.Box
		var fragmentResume ResumeAt
		if record.Original.IsFloated() {
			result := floatLayout(ctx, record.Original, record.ContainingBlock, positioned, 0, record.ResumeAt)
			fragment, fragmentResume = result.Box, result.ResumeAt
			if _, seen := preloaded[record.BFCRoot]; !seen {
				preloadedRoots = append(preloadedRoots, record.BFCRoot)
			}
			preloaded[record.BFCRoot] = append(preloaded[record.BFCRoot], fragment)
		} else {
			fragment, fragmentResume = absoluteBoxLayout(ctx, record.Original, record.ContainingBlock, &positioned.Fixed, 0, record.ResumeAt)
		}
		if fragment != nil {
			outOfFlow = append(outOfFlow, fragment)
		}
		pageIsEmpty = false
		if len(fragmentResume) != 0 && fragment != nil {
			ctx.addBrokenOutOfFlow(fragment, record.Original, record.ContainingBlock, fragmentResume)
		}
		if record.BFCRoot != nil {
			ctx.FinishBFC(nil)
		}
	}
	// The floats broken on the previous page exclude the in-flow content.
	for _, contextBox := range preloadedRoots {
		if contextBox == nil {
			ctx.bfcFrames[nil].Shapes = preloaded[nil]
		} else {
			ctx.bfcFrames[contextBox] = &bfcFrame{Root: contextBox, Shapes: preloaded[contextBox]}
		}
	}

	// Display in-flow content.
	initialRoot, initialResume := root, resume
	previousRunning := ctx.takeRunningElements(pageNumber)
	for _, name := range sortedNames(previousRunning) {
		for _, element := range previousRunning[name] {
			if element.IsNote() {
				ctx.RenderedNotes = append(ctx.RenderedNotes, element)
			}
		}
	}
	result := blockLevelLayout(ctx, root, initialContainingBlock, blockOptions{
		SkipStack:   resume,
		PageIsEmpty: pageIsEmpty,
		Positioned:  positioned,
	})
	if ctx.Err() != nil {
		ctx.FinishBFC(nil)
		return pageResult{Page: page}
	}
	root, resume, nextPage := result.Box, result.ResumeAt, result.NextPage
	if root == nil {
		// Only the out-of-flow content progressed. Lay out the root box,
		// emptied at the point where it resumes, and force its fragmentation
		// so that it and its descendants cover the whole page.
		root, resume = layoutEmptyContinuation(ctx, initialRoot, initialResume, initialContainingBlock, positioned)
		if root == nil {
			ctx.fail(fmt.Errorf("page %d: the root box did not fit", pageNumber))
			ctx.FinishBFC(nil)
			return pageResult{Page: page}
		}
	}
	rootParent := root.(boxes.Parent).ParentBase()
	rootParent.Children = append(outOfFlow, rootParent.Children...)

	running := ctx.runningElementsOf(pageNumber, previousRunning)
	if !sameRunningElements(previousRunning, running) {
		for _, name := range sortedNames(previousRunning) {
			runnings := running[name]
			if len(runnings) == 0 {
				continue
			}
			previous, hadPrevious := previousRunning[name]
			if !hadPrevious || previous == nil || len(previous) > len(runnings) {
				ctx.PageMaker[pageNumber-1].Remake.ContentChanged = true
			}
		}
	}

	laidFootnotes := boxes.CreateAnonymousBoxes(footnoteArea.DeepCopy())
	footnoteResult := blockLevelLayout(ctx, laidFootnotes, footnoteArea.Page, blockOptions{
		BottomSpace: math.Inf(-1),
		PageIsEmpty: true,
		Positioned:  positioned,
	})
	if footnoteResult.Box == nil {
		ctx.fail(fmt.Errorf("page %d: the footnote area did not fit", pageNumber))
		ctx.FinishBFC(nil)
		return pageResult{Page: page}
	}
	laidFootnotes = footnoteResult.Box
	laidFootnotes.Translate(0, -laidFootnotes.MarginHeight(), false)

	for _, placeholder := range positioned.Fixed {
		page.FixedBoxes = append(page.FixedBoxes, placeholder.Box)
	}
	for _, placeholder := range positioned.Absolute {
		absoluteLayout(ctx, placeholder, page, &positioned.Fixed, 0, nil)
	}
	for i := 0; i < len(positioned.Fixed); i++ {
		absoluteLayout(ctx, positioned.Fixed[i], page, &positioned.Fixed, 0, nil)
	}

	ctx.FinishBFC(nil)

	page.Children = []boxes.Box{root, laidFootnotes}
	if noteArea != nil {
		page.Children = append(page.Children, noteArea)
	}

	updatePageCounters(ctx, page, style, pageNumber, state)

	if pageType.Blank {
		resume = previousResume
		nextPage = ctx.PageMaker[pageNumber-1].NextPage
	}

	// Line boxes of this page are never needed again.
	clear(ctx.lineResume)
	return pageResult{Page: page, ResumeAt: resume, NextPage: nextPage}
}

// layoutEmptyContinuation lays out a root box that resumes at a point where
// only out-of-flow content was placed on the page. The box where the
// continuation ends loses its children and is forced to fragment, which
// makes it fill the page.
func layoutEmptyContinuation(ctx *Context, root boxes.Box, resume ResumeAt, containing boxes.Box, positioned *positionedBoxes) (boxes.Box, ResumeAt) {
	root = root.DeepCopy()
	box, parent := root, root
	for skip := resume; len(skip) == 1; {
		index, inner, _ := onlyResume(skip)
		children := boxes.ChildrenOf(box)
		if index >= len(children) {
			break
		}
		box, parent = children[index], box
		skip = inner
	}
	if p, ok := parent.(boxes.Parent); ok {
		p.ParentBase().Children = nil
	}
	parent.Base().ForceFragmentation = true
	result := blockLevelLayout(ctx, root, containing, blockOptions{
		SkipStack:   resume,
		PageIsEmpty: true,
		Positioned:  positioned,
	})
	return result.Box, resume
}

// makeNoteArea builds and lays out the area that holds the notes of a page
// (CSS GCPM 3 §2.4). Notes in the flow push the root box down.
func makeNoteArea(ctx *Context, page *boxes.PageBox, pageStyle *computed.Style, root boxes.Box, state *PageState, containing boxes.Box) boxes.Box {
	style := ctx.StyleFor.Page(page.PageType, "@note-area")
	if style == nil {
		style = pageStyle.Anonymous()
	}
	area := boxes.Blockify(boxes.MakeBox("@note-area", style, nil, nil), "")
	area.Base().PositionX = page.ContentBoxX()
	area.Base().PositionY = page.ContentBoxY()
	if isGeneratedContent(area.Base().Style.Content()) {
		if parent, ok := area.(boxes.Parent); ok {
			parent.ParentBase().Children = ctx.Builder.ContentToBoxes(area.Base().Style, area,
				&state.Build.QuoteDepth, state.Build.CounterValues, page, ctx)
			area = boxes.CreateAnonymousBoxes(area)
		}
	}

	// The boxes positioned inside the area are not laid out: the area has a
	// list of its own that nothing resolves.
	positioned := &positionedBoxes{}
	switch {
	case area.IsInNormalFlow():
		result := blockLevelLayout(ctx, area, containing, blockOptions{PageIsEmpty: true, Positioned: positioned})
		area = result.Box
	case area.IsFloated():
		result := floatLayout(ctx, area, containing, positioned, 0, nil)
		area = result.Box
		if area != nil && len(boxes.ChildrenOf(area)) != 0 {
			if root != nil {
				ctx.bfcFrames[root] = &bfcFrame{Root: root, Shapes: []boxes.Box{area}}
			}
		}
	default:
		area, _ = absoluteBoxLayout(ctx, area, page, &positioned.Fixed, 0, nil)
	}
	return area
}

// takeRunningElements removes the running elements stored for a page and
// returns them by name.
func (ctx *Context) takeRunningElements(page int) map[string][]boxes.Box {
	taken := make(map[string][]boxes.Box, len(ctx.RunningElements))
	for name, pages := range ctx.RunningElements {
		taken[name] = pages[page]
		delete(pages, page)
	}
	return taken
}

// runningElementsOf returns the running elements stored for a page by name,
// for each name known before or since.
func (ctx *Context) runningElementsOf(page int, before map[string][]boxes.Box) map[string][]boxes.Box {
	found := make(map[string][]boxes.Box, len(ctx.RunningElements))
	for name, pages := range ctx.RunningElements {
		found[name] = pages[page]
	}
	for name := range before {
		if _, ok := found[name]; !ok {
			found[name] = nil
		}
	}
	return found
}

func sameRunningElements(a, b map[string][]boxes.Box) bool {
	if len(a) != len(b) {
		return false
	}
	for name, boxesA := range a {
		boxesB, ok := b[name]
		if !ok || !slices.Equal(boxesA, boxesB) {
			return false
		}
	}
	return true
}

func sortedNames(m map[string][]boxes.Box) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// updatePageCounters applies the counters of the page to the page state and
// records what its content owes to pages not yet made: the anchors it holds
// and the content that waits for page based counters. What changed is marked
// in the page maker for the next pass of the layout.
func updatePageCounters(ctx *Context, page *boxes.PageBox, style *computed.Style, pageNumber int, state *PageState) {
	standardizePageBasedCounters(style, "")
	boxes.UpdateCounters(state.Build, style)
	pageCounters := state.Build.CounterValues

	collector := ctx.Targets
	remake := &ctx.PageMaker[pageNumber-1].Remake

	// The page values of a target and of a content are evaluated once, for
	// the first line box that has them: otherwise targets and pseudo-elements
	// that span pages loop forever.
	var cachedAnchors []string
	var cachedLookups []*targets.CounterLookupItem
	for _, previous := range ctx.PageMaker[:pageNumber-1] {
		cachedAnchors = append(cachedAnchors, previous.Remake.Anchors...)
		cachedLookups = append(cachedLookups, previous.Remake.ContentLookups...)
	}

	for child := range boxes.Descendants(page, true) {
		anchor := string(child.Base().Style.Anchor())
		if anchor != "" && !slices.Contains(cachedAnchors, anchor) {
			remake.Anchors = append(remake.Anchors, anchor)
			cachedAnchors = append(cachedAnchors, anchor)
			// Remaking the pages of the boxes that point to the anchor
			// includes the current one.
			collector.CacheTargetPageCounters(anchor, pageCounters, pageNumber-1, len(ctx.PageMaker), func(index int) {
				ctx.PageMaker[index].Remake.ContentChanged = true
			})
		}

		// String-set and bookmark-label make no boxes, so only content asks
		// for a new page.
		link := child.Base().MissingLinkBox
		if link == nil {
			continue
		}
		lookup := collector.CounterItem(link, "content")
		if lookup == nil {
			continue
		}
		parseAgain := false

		refresh := !slices.Contains(cachedLookups, lookup)
		if refresh {
			remake.ContentLookups = append(remake.ContentLookups, lookup)
			cachedLookups = append(cachedLookups, lookup)
			lookup.PageMakerIndex, lookup.Paginated = pageNumber-1, true
		}

		// Page based back-references, marked pending when the anchor's
		// counters changed.
		if lookup.Pending {
			if !equalCounterValues(pageCounters, lookup.CachedPageCounterValues) {
				lookup.CachedPageCounterValues = pageCounters.Clone()
			}
			lookup.Pending = false
			parseAgain = true
		}

		// Local counters: when the counters of the page that the box mixes
		// in changed, its content changes.
		if len(lookup.MissingCounters) != 0 {
			if lookup.MissingCounters["pages"] {
				remake.PagesWanted = true
			}
			if refresh && !equalCounterValues(pageCounters, lookup.CachedPageCounterValues) {
				lookup.CachedPageCounterValues = pageCounters.Clone()
				for name := range lookup.MissingCounters {
					if _, ok := pageCounters[name]; ok {
						parseAgain = true
						break
					}
				}
			}
		}

		// Targeted counters.
		for _, anchorName := range sortedKeys(lookup.MissingTargetCounters) {
			if !lookup.MissingTargetCounters[anchorName]["pages"] {
				continue
			}
			item := collector.Target(anchorName)
			if item != nil && item.Paginated && item.PageMakerIndex >= 0 &&
				item.PageMakerIndex < len(ctx.PageMaker) && slices.Contains(cachedAnchors, anchorName) {
				ctx.PageMaker[item.PageMakerIndex].Remake.PagesWanted = true
			}
		}

		if parseAgain {
			remake.ContentChanged = true
			lookup.ParseAgain(pageCounters)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func equalCounterValues(a, b targets.CounterValues) bool {
	return maps.EqualFunc(a, b, slices.Equal[[]int])
}

// includesResumeAt reports whether a continuation is inside the content of
// a page group, whose continuation is the path to the element that starts
// the group.
func includesResumeAt(resume, groupResume ResumeAt) bool {
	if len(groupResume) == 0 {
		return true
	}
	index, inner, ok := onlyResume(groupResume)
	if !ok {
		return false
	}
	next, found := resume[index]
	if resume == nil || !found {
		return false
	}
	if inner == nil {
		return true
	}
	return includesResumeAt(next, inner)
}

// lastResume returns the entry of a continuation that is the last one.
func lastResume(resume ResumeAt) (index int, inner ResumeAt) {
	last := -1
	for key := range resume {
		last = max(last, key)
	}
	return last, resume[last]
}

// updatePageGroups ends the page groups that the continuation left, counts
// the page in the others and starts the group of the next page when it has a
// name of its own (CSS GCPM 3, document sequence selectors). It returns the
// name of the page.
func updatePageGroups(groups *[]pageGroup, resume ResumeAt, next PageBreak, root boxes.Box, blank bool) string {
	kept := (*groups)[:0]
	for _, group := range *groups {
		if includesResumeAt(resume, group.ResumeAt) {
			group.Index++
			kept = append(kept, group)
		}
	}
	*groups = kept

	nextName := ""
	if next.Page != nil {
		nextName = *next.Page
	}
	if !blank {
		if (len(resume) != 0 && next.Break == "any") || nextName == "" {
			// There is no forced break and no named page.
			return nextName
		}
		if n := len(*groups); n != 0 && (*groups)[n-1].Name == nextName {
			// The next page name is that of the element we are in.
			return nextName
		}
	}

	// Find the box that has the named page: the first in-flow child of the
	// element where the document resumes.
	groupResume := resume.Clone()
	if len(groupResume) == 0 {
		groupResume = ResumeAt{0: nil}
	}
	current := groupResume
	element := root
	var parent boxes.Box
	var childIndex int
	for {
		var childResume ResumeAt
		childIndex, childResume = lastResume(current)
		parent = element
		children := boxes.ChildrenOf(element)
		if childIndex < 0 || childIndex >= len(children) {
			return nextName
		}
		element = children[childIndex]
		if childResume == nil {
			break
		}
		current = childResume
	}

	if blank {
		// The page is blank: no group starts, and the page has the name of
		// the parent.
		return string(parent.Base().Style.Page())
	}

	// Find the descendant with the named page.
	for {
		if string(element.Base().Style.Page()) == nextName {
			*groups = append(*groups, pageGroup{Name: nextName, Index: 0, ResumeAt: groupResume})
			return nextName
		}
		children := boxes.ChildrenOf(element)
		descended := false
		for i, child := range children {
			if !child.IsInNormalFlow() {
				continue
			}
			current[childIndex] = ResumeAt{i: nil}
			current = current[childIndex]
			childIndex = i
			element = child
			descended = true
			break
		}
		if !descended {
			return nextName
		}
	}
}

// pageSide returns the side that a break forces for the next page, or "".
func pageSide(root boxes.Box, pageBreak string) string {
	switch pageBreak {
	case "left", "right":
		return pageBreak
	case "recto", "verso":
		ltr := root.Base().Style.Direction() == "ltr"
		if ltr != (pageBreak == "verso") {
			return "right"
		}
		return "left"
	}
	return ""
}

// remakePage lays out the page at an index of the page maker, from the state
// it stores, and updates the entry of the following page with the state at the
// end of this one. Pages are made again one by one when what they show
// changes, which is why the margin boxes are not part of the result.
func remakePage(index int, ctx *Context, root boxes.Box) pageResult {
	entry := ctx.PageMaker[index]
	resume, next, rightPage := entry.ResumeAt, entry.NextPage, entry.RightPage

	// The state of the page maker entry is only read: the page works on a
	// copy that becomes the entry of the next page.
	state := entry.State.Clone()
	nextSide := pageSide(root, next.Break)
	blank := (nextSide == "left" && rightPage) ||
		(nextSide == "right" && !rightPage) ||
		(len(ctx.ReportedFootnotes) != 0 && resume == nil)
	side := "left"
	if rightPage {
		side = "right"
	}
	name := updatePageGroups(&state.Groups, resume, next, root, blank)
	groups := make([]cascade.PageGroup, len(state.Groups))
	for i, group := range state.Groups {
		groups[i] = cascade.PageGroup{Name: group.Name, Index: group.Index}
	}
	pageType := cascade.PageType{Side: side, Blank: blank, Name: name, Index: index, Groups: groups}
	ctx.StyleFor.SetPageTypeComputedStyles(pageType)

	ctx.ForcedBreak = next.Break != "any" || (next.Page != nil && *next.Page != "")

	result := makePage(ctx, root, pageType, resume, index+1, &state)
	if ctx.Err() != nil {
		return result
	}
	rightPage = !rightPage

	changed := true
	if index+1 < len(ctx.PageMaker) {
		following := ctx.PageMaker[index+1]
		changed = !following.ResumeAt.Equal(result.ResumeAt) ||
			!equalPageBreaks(following.NextPage, result.NextPage) ||
			following.RightPage != rightPage ||
			!following.State.Equal(state)
	}
	if changed {
		// A page that has a successor is marked changed to make sure it is
		// made; the last one must not be, which would loop forever.
		item := PageMaker{
			ResumeAt:  result.ResumeAt,
			NextPage:  result.NextPage,
			RightPage: rightPage,
			State:     state,
			Remake:    RemakeState{ContentChanged: result.ResumeAt != nil},
		}
		if index+1 >= len(ctx.PageMaker) {
			ctx.PageMaker = append(ctx.PageMaker, item)
		} else {
			ctx.PageMaker[index+1] = item
		}
	}
	return result
}

func equalPageBreaks(a, b PageBreak) bool {
	if a.Break != b.Break || (a.Page == nil) != (b.Page == nil) {
		return false
	}
	return a.Page == nil || *a.Page == *b.Page
}

// makeAllPages lays out the pages of the document, making again only those
// whose content changed or that show the number of pages. previous are the
// pages of the last pass.
func makeAllPages(ctx *Context, root boxes.Box, previous []*boxes.PageBox) []*boxes.PageBox {
	var pages []*boxes.PageBox
	ctx.RenderedNotes = nil
	for i := 0; ; i++ {
		if i >= maxPages {
			ctx.fail(fmt.Errorf("the document has more than %d pages", maxPages))
			return pages
		}
		remake := &ctx.PageMaker[i].Remake
		var resume ResumeAt
		var reported []boxes.Box
		if len(previous) == 0 || i >= len(previous) || i+1 >= len(ctx.PageMaker) || remake.ContentChanged || remake.PagesWanted {
			remake.ContentChanged, remake.PagesWanted = false, false
			remake.Anchors, remake.ContentLookups = nil, nil
			result := remakePage(i, ctx, root)
			if ctx.Err() != nil {
				return pages
			}
			resume = result.ResumeAt
			reported = ctx.ReportedFootnotes
			pages = append(pages, result.Page)
		} else {
			resume = ctx.PageMaker[i+1].ResumeAt
			pages = append(pages, previous[i])
		}
		if resume == nil && len(reported) == 0 {
			// Throw away the obsolete pages and content.
			ctx.PageMaker = ctx.PageMaker[:i+2]
			ctx.brokenOutOfFlow = nil
			ctx.ReportedFootnotes = nil
			return pages
		}
	}
}
