package layout

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// blockLevelWidth resolves the width and auto margins using the containing
// block's direction (CSS 2.1 §10.3.3).
func blockLevelWidth(box *boxes.BoxFields, cb containingBlock, direction props.Keyword) {
	resolveBlockWidth(box, cb, direction, true)
}

func blockLevelWidthWithoutMinMax(box *boxes.BoxFields, cb containingBlock, direction props.Keyword) {
	resolveBlockWidth(box, cb, direction, false)
}

func resolveBlockWidth(box *boxes.BoxFields, cb containingBlock, direction props.Keyword, withMinMax bool) {
	paddingAndBorder := box.PaddingLeft + box.PaddingRight + box.BorderLeftWidth + box.BorderRightWidth
	if box.Width.Auto {
		width := cb.Width.V - paddingAndBorder
		if !box.MarginLeft.Auto {
			width -= box.MarginLeft.V
		}
		if !box.MarginRight.Auto {
			width -= box.MarginRight.V
		}
		box.Width = units.F(width)
	}
	if withMinMax {
		box.Width = units.F(firstMax(box.MinWidth.V, firstMin(box.MaxWidth.V, box.Width.V)))
	}

	marginWidth := paddingAndBorder + box.Width.V
	if !box.MarginLeft.Auto {
		marginWidth += box.MarginLeft.V
	}
	if !box.MarginRight.Auto {
		marginWidth += box.MarginRight.V
	}
	if marginWidth > cb.Width.V {
		if box.MarginLeft.Auto {
			box.MarginLeft = units.F(0)
		}
		if box.MarginRight.Auto {
			box.MarginRight = units.F(0)
		}
	}

	if direction == "rtl" && !box.IsColumn {
		box.PositionX += cb.Width.V - paddingAndBorder - box.Width.V
		if !box.MarginLeft.Auto {
			box.PositionX -= box.MarginLeft.V
		}
		if !box.MarginRight.Auto {
			box.PositionX -= box.MarginRight.V
		}
	}

	marginSum := cb.Width.V - paddingAndBorder - box.Width.V
	switch {
	case box.MarginLeft.Auto && box.MarginRight.Auto:
		box.MarginLeft = units.F(marginSum / 2)
		box.MarginRight = units.F(marginSum / 2)
	case box.MarginLeft.Auto:
		box.MarginLeft = units.F(marginSum - box.MarginRight.V)
	case box.MarginRight.Auto:
		box.MarginRight = units.F(marginSum - box.MarginLeft.V)
	}
}
