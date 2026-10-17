package layout

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/html"
)

// maxLayoutLoops is the number of passes over the document that layout does
// before it accepts counters that keep changing.
const maxLayoutLoops = 8

// initializePageMaker collects the state of the pagination that page based
// counters need: for each page, where it starts and what its counters are.
func initializePageMaker(ctx *Context, root boxes.Box) {
	style := root.Base().Style
	ltr := style.Direction() == "ltr"

	// The text direction and the writing mode should decide the side of the
	// first page (CSS Paged Media 3 §3.2).
	var rightPage bool
	switch style.BreakBefore() {
	case "right":
		rightPage = true
	case "left":
		rightPage = false
	case "recto":
		rightPage = ltr
	case "verso":
		rightPage = !ltr
	default:
		rightPage = ltr
	}
	start, _ := root.PageValues()

	// Pages have no element to open quotes or counter scopes, but updating
	// their counters shares the code of the box tree, which wants a state.
	// The number of pages is unknown until the layout ends and may change
	// while pages are made again, so it starts empty.
	state := PageState{Build: &boxes.State{
		CounterValues: targets.CounterValues{"pages": {0}},
		CounterScopes: []map[string]bool{{"pages": true}},
	}}
	ctx.PageMaker = []PageMaker{{
		NextPage:  PageBreak{Break: "any", Page: &start},
		RightPage: rightPage,
		State:     state,
	}}
}

// layoutFixedBoxes lays out the position: fixed boxes of pages on another
// page, along with the boxes positioned inside them (CSS Positioned Layout 3
// §3.2: fixed boxes are repeated on every page).
func layoutFixedBoxes(ctx *Context, pages []*boxes.PageBox, containing *boxes.PageBox) []boxes.Box {
	var laid []boxes.Box
	for _, page := range pages {
		for _, box := range page.FixedBoxes {
			// Replaced boxes are not copied during layout, and each page
			// needs its own.
			if _, ok := box.(boxes.Replaced); ok {
				box = box.Copy()
			}
			// Absolute boxes inside fixed boxes are children of the fixed
			// box, even when they are fixed themselves.
			var nested []*boxes.AbsolutePlaceholder
			fragment, _ := absoluteBoxLayout(ctx, box, containing, &nested, math.Inf(-1), nil)
			if fragment != nil {
				laid = append(laid, fragment)
			}
			for len(nested) != 0 {
				var more []*boxes.AbsolutePlaceholder
				for _, placeholder := range nested {
					absoluteLayout(ctx, placeholder, containing, &more, math.Inf(-1), nil)
				}
				nested = more
			}
		}
	}
	return laid
}

// LayoutDocument lays out a whole document: the line breaks, the page breaks
// and the size and position of every box. Counters that depend on pagination
// may need several passes, up to maxLoops, which is 8 when it is not
// positive. The root box is the root of the box tree, which is kept intact:
// the pages are made from it. The error is the first fatal error of the
// layout.
func LayoutDocument(doc *html.Document, root boxes.Box, ctx *Context, maxLoops int) ([]*boxes.PageBox, error) {
	if maxLoops <= 0 {
		maxLoops = maxLayoutLoops
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: %w", err)
	}
	initializePageMaker(ctx, root)

	var pages []*boxes.PageBox
	var originalFootnotes []boxes.Box
	actualTotalPages := 0
	for loop := 0; loop < maxLoops; loop++ {
		if loop > 0 {
			ctx.Footnotes = slices.Clone(originalFootnotes)
		}
		initialTotalPages := actualTotalPages
		if loop == 0 {
			originalFootnotes = slices.Clone(ctx.Footnotes)
		}
		pages = makeAllPages(ctx, root, pages)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("layout: %w", err)
		}
		actualTotalPages = len(pages)

		// Check whether another round is required.
		reloopContent, reloopPages := false, false
		for i := range ctx.PageMaker {
			entry := &ctx.PageMaker[i]
			entry.State.Build.CounterValues["pages"] = []int{actualTotalPages}
			if entry.Remake.ContentChanged {
				reloopContent = true
			}
			if entry.Remake.PagesWanted {
				reloopPages = initialTotalPages != actualTotalPages
			}
		}
		if !reloopContent && !reloopPages {
			break
		}
	}

	collectPageContent(ctx, pages)

	for i, page := range pages {
		root := page.Children[0]
		rootChildren := make([]boxes.Box, 0, len(boxes.ChildrenOf(root)))
		rootChildren = append(rootChildren, layoutFixedBoxes(ctx, pages[:i], page)...)
		rootChildren = append(rootChildren, boxes.ChildrenOf(root)...)
		rootChildren = append(rootChildren, layoutFixedBoxes(ctx, pages[i+1:], page)...)
		root.(boxes.Parent).ParentBase().Children = rootChildren
		ctx.CurrentPage = i + 1

		// The page maker keeps the state at the end of the page, which is
		// what its margin boxes show.
		state := ctx.PageMaker[ctx.CurrentPage].State
		children := []boxes.Box{root}
		for _, area := range page.Children[1:] {
			if len(boxes.ChildrenOf(area)) != 0 {
				children = append(children, area)
			}
		}
		children = append(children, makeMarginBoxes(ctx, page, state)...)
		page.Children = children
		if err := layoutBackgrounds(page, ctx.GetImage); err != nil {
			return nil, fmt.Errorf("layout: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("layout: %w", err)
		}
	}
	return pages, nil
}

// collectPageContent calculates the string-set and the bookmark-label that
// contain page based counters, once the pagination is final: they make no
// boxes, only appear in margin boxes and in the outline of the PDF. It also
// gathers the named strings of each page, and keeps only one bookmark for
// each original box.
func collectPageContent(ctx *Context, pages []*boxes.PageBox) {
	var seen, seenBefore, seenAfter []*html.Element
	for i, page := range pages {
		// The counters at the end of the page, with the final total.
		pageCounters := ctx.PageMaker[i+1].State.Build.CounterValues

		for child := range boxes.Descendants(page, false) {
			base := child.Base()
			if base.BookmarkLabel != "" {
				checklist := &seen
				switch {
				case strings.HasSuffix(base.ElementTag, "::before"):
					checklist = &seenBefore
				case strings.HasSuffix(base.ElementTag, "::after"):
					checklist = &seenAfter
				}
				if slices.Contains(*checklist, base.Element) {
					base.BookmarkLabel = ""
				} else {
					*checklist = append(*checklist, base.Element)
				}
			}

			if link := base.MissingLinkBox; link != nil {
				refreshContentLists(ctx, child, link, pageCounters)
			}

			for _, item := range base.StringSet {
				if ctx.StringSet[item.Name] == nil {
					ctx.StringSet[item.Name] = make(map[int][]string)
				}
				ctx.StringSet[item.Name][i+1] = append(ctx.StringSet[item.Name][i+1], item.Text)
			}
		}
	}
}

// refreshContentLists computes again the bookmark label and the named
// strings of the original box that child was made from, now that the page
// counters are known, and takes the results to the fragment.
func refreshContentLists(ctx *Context, child, link boxes.Box, pageCounters targets.CounterValues) {
	original := link.Base()
	base := child.Base()
	tokens := make([]string, 0, 1+len(original.Style.StringSet()))
	tokens = append(tokens, "bookmark-label")
	for _, entry := range original.Style.StringSet() {
		tokens = append(tokens, "string-set::"+entry.Name)
	}
	for _, token := range tokens {
		item := ctx.Targets.CounterItem(link, token)
		if item == nil {
			continue
		}
		if token == "bookmark-label" && base.BookmarkLabel == "" {
			// A label removed as a repetition stays removed.
			continue
		}
		item.ParseAgain(pageCounters)
		if token == "bookmark-label" {
			base.BookmarkLabel = original.BookmarkLabel
		}
	}
	// Computing a string again replaces the list of the original box, which
	// its fragments do not follow.
	if base != original {
		base.StringSet = slices.Clone(original.StringSet)
	}
}
