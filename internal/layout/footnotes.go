package layout

import (
	"fmt"
	"math"
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// layoutFootnote places a footnote in the footnote area of the current page
// and reports whether the area overflows (CSS GCPM Level 3 §2.5).
func (ctx *Context) layoutFootnote(footnote boxes.Box) bool {
	if i := slices.Index(ctx.Footnotes, footnote); i >= 0 {
		ctx.Footnotes = slices.Delete(ctx.Footnotes, i, i+1)
	}
	ctx.CurrentPageFootnotes = append(ctx.CurrentPageFootnotes, footnote)
	return ctx.updateFootnoteArea()
}

// unlayoutFootnote takes a footnote back from the page and returns it to the
// list of footnotes waiting for a place.
func (ctx *Context) unlayoutFootnote(footnote boxes.Box) {
	if slices.Contains(ctx.Footnotes, footnote) {
		return
	}
	ctx.Footnotes = append(ctx.Footnotes, footnote)
	if i := slices.Index(ctx.CurrentPageFootnotes, footnote); i >= 0 {
		ctx.CurrentPageFootnotes = slices.Delete(ctx.CurrentPageFootnotes, i, i+1)
	} else if i := slices.Index(ctx.ReportedFootnotes, footnote); i >= 0 {
		ctx.ReportedFootnotes = slices.Delete(ctx.ReportedFootnotes, i, i+1)
	}
	ctx.updateFootnoteArea()
}

// reportFootnote moves a footnote of the current page to the next page.
func (ctx *Context) reportFootnote(footnote boxes.Box) {
	if i := slices.Index(ctx.CurrentPageFootnotes, footnote); i >= 0 {
		ctx.CurrentPageFootnotes = slices.Delete(ctx.CurrentPageFootnotes, i, i+1)
	}
	ctx.ReportedFootnotes = append(ctx.ReportedFootnotes, footnote)
	ctx.updateFootnoteArea()
}

// updateFootnoteArea lays out the footnotes of the current page, makes
// their area as high as they need and moves the bottom of the page above
// it. It reports whether the last footnote overflows the area. Without a
// footnote area there is nothing to update.
func (ctx *Context) updateFootnoteArea() bool {
	area := ctx.FootnoteArea
	if area == nil {
		return false
	}
	if !area.Height.Auto && !ctx.InColumn {
		ctx.PageBottom += area.MarginHeight()
	}
	area.Children = slices.Clone(ctx.CurrentPageFootnotes)
	if len(area.Children) == 0 {
		area.Height = units.F(0)
		if !ctx.InColumn {
			ctx.PageBottom -= area.MarginHeight()
		}
		return false
	}
	if area.Page == nil {
		ctx.fail(fmt.Errorf("footnote area without a page"))
		return false
	}

	copied := boxes.CreateAnonymousBoxes(area.DeepCopy())
	laid := blockLevelLayout(ctx, copied, area.Page, blockOptions{
		BottomSpace: math.Inf(-1),
		PageIsEmpty: true,
	}).Box
	if laid == nil {
		return false
	}
	area.Height = laid.Base().Height
	if !ctx.InColumn {
		ctx.PageBottom -= laid.MarginHeight()
	}
	children := boxes.ChildrenOf(laid)
	if len(children) == 0 {
		return false
	}
	last := children[len(children)-1]
	lastBottom := last.Base().PositionY + last.MarginHeight() - last.Base().MarginBottom.V
	areaBottom := laid.Base().PositionY + laid.MarginHeight() - laid.Base().MarginBottom.V
	return lastBottom > areaBottom
}
