package layout

import (
	"context"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/units"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// paginate lays out a document page by page with the block, float and
// positioned layout only: pages have no margin boxes, footnote area or
// counters, and a forced break to a given side does not insert a blank
// page. Tests that need those use the complete page layout instead.
func paginate(t testing.TB, source string) []*boxes.PageBox {
	t.Helper()
	doc := csstest.Parse(t, source)
	env := csstest.Environment()
	collector := targets.NewCollector()
	styles := csstest.StyleFor(t, doc, cascade.Options{Environment: env, Targets: collector})
	builder := &boxes.Builder{
		StyleFor:     styles,
		Targets:      collector,
		CounterStyle: env.CounterStyle,
		GetImage:     images.NewGetter(context.Background(), urls.NewDefaultFetcher(), nil, images.Options{}),
	}
	root, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	ctx := NewContext(builder)

	var pages []*boxes.PageBox
	var resume ResumeAt
	rootStart, _ := root.PageValues()
	next := PageBreak{Break: "any", Page: &rootStart}
	right := root.Base().Style.Direction() == "ltr"
	for number := 1; ; number++ {
		side := "left"
		if right {
			side = "right"
		}
		name := ""
		if next.Page != nil && *next.Page != "auto" {
			name = *next.Page
		}
		pageType := cascade.PageType{Side: side, Name: name, Index: number - 1}
		ctx.ForcedBreak = next.Break != "any" || name != ""
		page, nextResume, nextPage := fixturePage(t, ctx, root, pageType, number, resume)
		if err := ctx.Err(); err != nil {
			t.Fatalf("layout page %d: %v", number, err)
		}
		pages = append(pages, page)
		if nextResume == nil {
			break
		}
		if number > 50 {
			t.Fatal("pagination did not end after 50 pages")
		}
		resume, next, right = nextResume, nextPage, !right
	}

	for i, page := range pages {
		rootBox := page.Children[0]
		var children []boxes.Box
		children = append(children, fixtureFixedBoxes(ctx, pages[:i], page)...)
		children = append(children, boxes.ChildrenOf(rootBox)...)
		children = append(children, fixtureFixedBoxes(ctx, pages[i+1:], page)...)
		rootBox.(boxes.Parent).ParentBase().Children = children
	}
	return pages
}

// fixturePage lays out one page of the document.
func fixturePage(t testing.TB, ctx *Context, root boxes.Box, pageType cascade.PageType, number int, resume ResumeAt) (*boxes.PageBox, ResumeAt, PageBreak) {
	t.Helper()
	ctx.StyleFor.SetPageTypeComputedStyles(pageType)
	style := ctx.StyleFor.Page(pageType, "")
	if overflow := root.Base().ViewportOverflow; overflow != "" {
		style.Set(props.PropOverflow, props.Keyword(overflow))
	}
	page := boxes.NewPageBox(pageType, style)

	device := style.Size()
	deviceWidth, deviceHeight := float64(device.X), float64(device.Y)
	resolvePercentages(page, size(deviceWidth, deviceHeight))
	page.PositionX, page.PositionY = 0, 0
	fixturePageSize(page, deviceWidth, deviceHeight)
	if number == 1 {
		pageContext := ctx.StyleFor.Context()
		pageContext.PageBox = &computed.Size{Width: deviceWidth, Height: deviceHeight}
		pageContext.PageArea = &computed.Size{Width: page.Width.V, Height: page.Height.V}
	}
	ctx.PageBottom = page.ContentBoxY() + page.Height.V
	ctx.CurrentPage = number
	ctx.CreateBFC(nil, nil)

	root.Base().PositionX = page.ContentBoxX()
	root.Base().PositionY = page.ContentBoxY()

	// Out-of-flow boxes that broke on the previous page come first.
	pageIsEmpty := true
	positioned := &positionedBoxes{}
	var outOfFlow []boxes.Box
	preloaded := map[boxes.Box][]boxes.Box{}
	previous := ctx.brokenOutOfFlow
	ctx.brokenOutOfFlow = nil
	for _, record := range previous {
		if record.BFCRoot != nil {
			ctx.CreateBFC(record.BFCRoot, nil)
		}
		record.Original.Base().PositionY = root.ContentBoxY()
		var fragment boxes.Box
		var fragmentResume ResumeAt
		if record.Original.IsFloated() {
			result := floatLayout(ctx, record.Original, record.ContainingBlock, positioned, 0, record.ResumeAt)
			fragment, fragmentResume = result.Box, result.ResumeAt
			preloaded[record.BFCRoot] = append(preloaded[record.BFCRoot], fragment)
		} else {
			fragment, fragmentResume = absoluteBoxLayout(ctx, record.Original, record.ContainingBlock, &positioned.Fixed, 0, record.ResumeAt)
		}
		outOfFlow = append(outOfFlow, fragment)
		pageIsEmpty = false
		if fragmentResume != nil {
			ctx.addBrokenOutOfFlow(fragment, record.Original, record.ContainingBlock, fragmentResume)
		}
		if record.BFCRoot != nil {
			ctx.FinishBFC(nil)
		}
	}
	for contextBox, shapes := range preloaded {
		if contextBox == nil {
			ctx.bfcFrames[nil].Shapes = shapes
		} else {
			ctx.bfcFrames[contextBox] = &bfcFrame{Root: contextBox, Shapes: shapes}
		}
	}

	result := blockLevelLayout(ctx, root, page, blockOptions{
		SkipStack:   resume,
		PageIsEmpty: pageIsEmpty,
		Positioned:  positioned,
	})
	rootBox := result.Box
	if rootBox == nil {
		t.Fatalf("page %d: the root box did not fit", number)
	}
	rootBox.(boxes.Parent).ParentBase().Children = append(outOfFlow, boxes.ChildrenOf(rootBox)...)

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

	page.Children = []boxes.Box{rootBox}
	return page, result.ResumeAt, result.NextPage
}

// fixturePageSize sets the width and the height of a page box, and its
// margins, as for a non-replaced block: an over-constrained page keeps its
// values and the containing block is resized to coincide with it.
func fixturePageSize(page *boxes.PageBox, deviceWidth, deviceHeight float64) {
	axis := func(inner, marginA, marginB *units.MaybeFloat, paddingAndBorder, available float64) {
		remaining := available - paddingAndBorder
		switch {
		case inner.Auto:
			if marginA.Auto {
				*marginA = units.F(0)
			}
			if marginB.Auto {
				*marginB = units.F(0)
			}
			*inner = units.F(remaining - marginA.V - marginB.V)
		case marginA.Auto && marginB.Auto:
			half := (remaining - inner.V) / 2
			*marginA, *marginB = units.F(half), units.F(half)
		case marginA.Auto:
			*marginA = units.F(remaining - inner.V - marginB.V)
		case marginB.Auto:
			*marginB = units.F(remaining - inner.V - marginA.V)
		}
	}
	b := &page.BoxFields
	withMinMaxWidth(b, func() {
		axis(&b.Width, &b.MarginLeft, &b.MarginRight,
			b.PaddingLeft+b.PaddingRight+b.BorderLeftWidth+b.BorderRightWidth, deviceWidth)
	})
	withMinMaxHeight(b, func() {
		axis(&b.Height, &b.MarginTop, &b.MarginBottom,
			b.PaddingTop+b.PaddingBottom+b.BorderTopWidth+b.BorderBottomWidth, deviceHeight)
	})
}

// fixtureFixedBoxes lays out the position: fixed boxes of pages on another
// page, with the boxes nested in them (CSS Positioned Layout 3 §3.2).
func fixtureFixedBoxes(ctx *Context, pages []*boxes.PageBox, containing *boxes.PageBox) []boxes.Box {
	var laid []boxes.Box
	for _, page := range pages {
		for _, box := range page.FixedBoxes {
			// Replaced boxes are not copied during layout, and each page needs
			// its own.
			if _, ok := box.(boxes.Replaced); ok {
				box = box.Copy()
			}
			// Absolute boxes inside fixed boxes are fixed boxes' children,
			// even when they are fixed themselves.
			var nested []*boxes.AbsolutePlaceholder
			fragment, _ := absoluteBoxLayout(ctx, box, containing, &nested, math.Inf(-1), nil)
			laid = append(laid, fragment)
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

// expectChildren returns the children of a box, failing the test unless
// there are n of them.
func expectChildren(t testing.TB, box boxes.Box, n int) []boxes.Box {
	t.Helper()
	children := boxes.ChildrenOf(unwrapPlaceholder(box))
	if len(children) != n {
		t.Fatalf("%v has %d children, want %d: %v", box, len(children), n, children)
	}
	return children
}

// unwrapPlaceholder returns the box that a placeholder stands for, which
// is the box itself for any other.
func unwrapPlaceholder(box boxes.Box) boxes.Box {
	if placeholder, ok := box.(*boxes.AbsolutePlaceholder); ok {
		return placeholder.Box
	}
	return box
}

// onlyPage fails the test unless there is exactly one page, and returns it.
func onlyPage(t testing.TB, pages []*boxes.PageBox) *boxes.PageBox {
	t.Helper()
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	return pages[0]
}

// pageBody returns the html and body boxes of a page.
func pageBody(t testing.TB, page *boxes.PageBox) (html, body boxes.Box) {
	t.Helper()
	html = expectChildren(t, page, 1)[0]
	if tag := html.Base().ElementTag; tag != "html" {
		t.Fatalf("page child is %q, want html", tag)
	}
	return html, boxes.ChildrenOf(html)[0]
}

func noLogs(t testing.TB) { testutil.AssertNoLogs(t) }
