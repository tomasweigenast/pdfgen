package layout

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func shrinkToFit(ctx *Context, box boxes.Box, available float64) float64 {
	return firstMin(firstMax(minContentWidth(ctx, box, false), available), maxContentWidth(ctx, box, false))
}

func minContentWidth(ctx *Context, box boxes.Box, outer bool) float64 {
	return intrinsicContentWidth(ctx, box, outer, true)
}

func maxContentWidth(ctx *Context, box boxes.Box, outer bool) float64 {
	return intrinsicContentWidth(ctx, box, outer, false)
}

func intrinsicContentWidth(ctx *Context, box boxes.Box, outer, minimum bool) float64 {
	if !box.Base().IsTableWrapper && box.Kind() != boxes.KindTableCell {
		switch box := box.(type) {
		case boxes.BlockContainer:
			return blockContentWidth(ctx, box, outer, minimum)
		case *boxes.InlineBox, *boxes.LineBox:
			if minimum {
				return inlineMinContentWidth(ctx, box, outer, nil, false, true)
			}
			return inlineMaxContentWidth(ctx, box, outer, true)
		case boxes.Replaced:
			return replacedContentWidth(box, outer, minimum)
		}
	}
	if ctx.err == nil {
		ctx.err = fmt.Errorf("intrinsic widths for %s are not supported", box.Kind())
	}
	return 0
}

func blockContentWidth(ctx *Context, box boxes.BlockContainer, outer, minimum bool) float64 {
	style := box.Base().Style
	specified := style.Width()
	var width float64
	if indefiniteIntrinsicLength(specified) {
		haveChild := false
		for _, child := range box.ParentBase().Children {
			if child.IsAbsolutelyPositioned() || child.IsNote() {
				continue
			}
			childWidth := intrinsicContentWidth(ctx, child, true, minimum)
			if !haveChild {
				width = childWidth
				haveChild = true
			} else {
				width = firstMax(width, childWidth)
			}
		}
	} else {
		width = specified.Dimension.Value
		if style.BoxSizing() != "content-box" {
			var percentages float64
			for _, padding := range []props.Length{style.PaddingLeft(), style.PaddingRight()} {
				if padding.IsKeyword() || padding.Calc != nil {
					continue
				}
				if padding.Dimension.Unit == "%" {
					percentages += padding.Dimension.Value
				} else {
					width -= padding.Dimension.Value
				}
			}
			left, right := intrinsicBorders(box)
			width -= left
			width -= right
			width = (100 - firstMin(100, percentages)) * firstMax(0, width) / 100
		}
	}
	return adjustIntrinsicWidth(box, outer, width, true, true)
}

func indefiniteIntrinsicLength(length props.Length) bool {
	return length.IsKeyword() || length.Calc != nil || length.Dimension.Unit == "%"
}

func preferredMinMax(box boxes.Box, width float64) float64 {
	style := box.Base().Style
	minimum, maximum := float64(0), math.Inf(1)
	if length := style.MinWidth(); !indefiniteIntrinsicLength(length) {
		minimum = length.Dimension.Value
	}
	if length := style.MaxWidth(); !indefiniteIntrinsicLength(length) {
		maximum = length.Dimension.Value
	}
	if replaced, ok := box.(boxes.Replaced); ok {
		_, _, ratio := replaced.ReplacedBase().Replacement.IntrinsicSize(1, float64(style.FontSize()))
		if !ratio.Auto {
			minHeight, maxHeight := style.MinHeight(), style.MaxHeight()
			minPending := minHeight.Calc != nil
			if !indefiniteIntrinsicLength(minHeight) {
				minimum = firstMax(minimum, minHeight.Dimension.Value*ratio.V)
			}
			if !maxHeight.IsKeyword() && !minPending && maxHeight.Dimension.Unit != "%" {
				maximum = firstMin(maximum, maxHeight.Dimension.Value*ratio.V)
			}
		}
	}
	return firstMax(minimum, firstMin(width, maximum))
}

func intrinsicBorders(box boxes.Box) (left, right float64) {
	b := box.Base()
	if b.Style.BorderCollapse() == "collapse" && b.CollapsedBordersResolved {
		return b.BorderLeftWidth, b.BorderRightWidth
	}
	return float64(b.Style.BorderLeftWidth()), float64(b.Style.BorderRightWidth())
}

func intrinsicMarginWidth(box boxes.Box, width float64, left, right bool) float64 {
	style := box.Base().Style
	values := make([]props.Length, 0, 4)
	if left {
		values = append(values, style.MarginLeft(), style.PaddingLeft())
	}
	if right {
		values = append(values, style.MarginRight(), style.PaddingRight())
	}
	var percentages float64
	for _, value := range values {
		if value.IsKeyword() || value.Calc != nil {
			continue
		}
		if value.Dimension.Unit == "%" {
			percentages += value.Dimension.Value
		} else {
			width += value.Dimension.Value
		}
	}
	leftBorder, rightBorder := intrinsicBorders(box)
	if left {
		width += leftBorder
	}
	if right {
		width += rightBorder
	}
	if percentages < 100 {
		return width / (1 - percentages/100)
	}
	return 0
}

func adjustIntrinsicWidth(box boxes.Box, outer bool, width float64, left, right bool) float64 {
	width = preferredMinMax(box, width)
	if outer {
		return intrinsicMarginWidth(box, width, left, right)
	}
	return width
}

func inlineMinContentWidth(
	ctx *Context,
	box boxes.Box,
	outer bool,
	skip ResumeAt,
	firstLine, isLineStart bool,
) float64 {
	widths := inlineLineWidths(ctx, box.(boxes.Parent), outer, isLineStart, true, skip, firstLine)
	width := widths[0]
	if !firstLine {
		for _, candidate := range widths[1:] {
			width = firstMax(width, candidate)
		}
	}
	return adjustIntrinsicWidth(box, outer, width, true, true)
}

func inlineMaxContentWidth(ctx *Context, box boxes.Box, outer, isLineStart bool) float64 {
	widths := inlineLineWidths(ctx, box.(boxes.Parent), outer, isLineStart, false, nil, false)
	widths[len(widths)-1] -= trailingWhitespaceSize(ctx, box)
	width := widths[0]
	for _, candidate := range widths[1:] {
		width = firstMax(width, candidate)
	}
	return adjustIntrinsicWidth(box, outer, width, true, true)
}

type intrinsicLine struct {
	width float64
	// empty records a break boundary even when the adjacent text has no width.
	empty bool
}

func inlineLineWidths(
	ctx *Context,
	box boxes.Parent,
	outer, isLineStart, minimum bool,
	skip ResumeAt,
	firstLine bool,
) []float64 {
	var textIndent float64
	if box.Kind() == boxes.KindLine {
		indent := box.Base().Style.TextIndent()
		if indent.Calc != nil {
			textIndent = percentage(indent, box.Base().Style, units.F(0)).Or(0)
		} else if indent.Dimension.Unit != "%" {
			textIndent = indent.Dimension.Value
		}
	}
	widths := []float64{}
	var current float64
	index, childSkip := firstResume(skip)
	for _, child := range box.ParentBase().Children[index:] {
		if child.IsAbsolutelyPositioned() || child.IsNote() {
			continue
		}
		lines := []intrinsicLine{}
		switch child := child.(type) {
		case *boxes.InlineBox:
			childWidths := inlineLineWidths(ctx, child, outer, isLineStart, minimum, childSkip, firstLine)
			for _, width := range childWidths {
				lines = append(lines, intrinsicLine{width: width, empty: width == 0})
			}
			if len(lines) == 1 {
				lines[0] = intrinsicLine{width: adjustIntrinsicWidth(child, outer, lines[0].width, true, true)}
			} else {
				lines[0].width = adjustIntrinsicWidth(child, outer, lines[0].width, true, false)
				lines[0].empty = lines[0].width == 0
				last := &lines[len(lines)-1]
				last.width = adjustIntrinsicWidth(child, outer, last.width, false, true)
				last.empty = last.width == 0
			}
		case *boxes.TextBox:
			textSkip, _ := firstResume(childSkip)
			text := child.Text[textSkip:]
			if isLineStart && collapsesSpaces(child) {
				text = strings.TrimLeft(text, " ")
			}
			var maxWidth *float64
			if minimum {
				maxWidth = new(float64)
			}
			for resume := 0; ; {
				split := linebreak.SplitFirstLine(
					ctx.Text,
					text[resume:],
					child.Style.LineBreakStyle(),
					maxWidth,
					child.JustificationSpacing,
					isLineStart,
					true,
				)
				lines = append(lines, intrinsicLine{width: split.Width, empty: split.Width == 0})
				if firstLine && split.ResumeIndex > 0 {
					return append(widths, current+split.Width+textIndent)
				}
				if firstLine || split.ResumeIndex < 0 {
					break
				}
				resume += split.ResumeIndex
			}
			last, _ := utf8.DecodeLastRuneInString(text)
			if text == "" {
				last = 0
			}
			boundary := "a"
			if last != 0 {
				boundary = string(last) + boundary
			}
			if minimum && wrapsIntrinsicText(child) && linebreak.CanBreakText(boundary, string(child.Style.Lang())) {
				lines = append(lines, intrinsicLine{empty: true})
			}
		default:
			if minimum && wrapsIntrinsicText(box) {
				lines = append(lines,
					intrinsicLine{empty: true},
					intrinsicLine{width: minContentWidth(ctx, child, true)},
					intrinsicLine{empty: true},
				)
			} else {
				lines = append(lines, intrinsicLine{width: intrinsicContentWidth(ctx, child, true, minimum)})
			}
		}
		current += lines[0].width
		if len(lines) > 1 {
			widths = append(widths, current+textIndent)
			if firstLine {
				return widths
			}
			textIndent = 0
			for _, line := range lines[1 : len(lines)-1] {
				widths = append(widths, line.width)
			}
			current = lines[len(lines)-1].width
		}
		isLineStart = lines[len(lines)-1].empty
		childSkip = nil
	}
	return append(widths, current+textIndent)
}

func wrapsIntrinsicText(box boxes.Box) bool {
	switch box.Base().Style.WhiteSpace() {
	case "normal", "pre-wrap", "pre-line":
		return true
	}
	return false
}

func replacedContentWidth(box boxes.Replaced, outer, minimum bool) float64 {
	r := box.ReplacedBase()
	width := r.Style.Width()
	var contribution float64
	if width.IsAuto() {
		maxWidth := r.Style.MaxWidth()
		unknownMaxWidth := !maxWidth.IsKeyword() && maxWidth.Calc == nil && maxWidth.Dimension.Unit == "%"
		if !minimum || !unknownMaxWidth {
			height := units.Auto
			if specified := r.Style.Height(); !indefiniteIntrinsicLength(specified) {
				height = units.F(specified.Dimension.Value)
			}
			intrinsicWidth, intrinsicHeight, intrinsicRatio := intrinsicSize(r)
			defaultWidth, defaultHeight := float64(0), float64(0)
			if !minimum {
				defaultWidth, defaultHeight = 300, 150
			}
			contribution, _ = defaultImageSizing(
				intrinsicWidth,
				intrinsicHeight,
				intrinsicRatio,
				units.Auto,
				height,
				defaultWidth,
				defaultHeight,
			)
		}
	} else if !indefiniteIntrinsicLength(width) {
		contribution = width.Dimension.Value
	}
	return adjustIntrinsicWidth(box, outer, contribution, true, true)
}

func trailingWhitespaceSize(ctx *Context, box boxes.Box) float64 {
	var lastParent boxes.Parent
	for box.Kind() == boxes.KindInline || box.Kind() == boxes.KindLine {
		parent := box.(boxes.Parent)
		children := parent.ParentBase().Children
		if len(children) == 0 {
			return 0
		}
		lastParent, box = parent, children[len(children)-1]
	}
	text, ok := box.(*boxes.TextBox)
	if !ok || text.Text == "" || !collapsesSpaces(box) || text.Style.FontSize() == 0 ||
		!strings.HasSuffix(text.Text, " ") {
		return 0
	}
	stripped := strings.TrimRight(text.Text, " ")
	if stripped != "" {
		var old textResult
		resume, oldResume := 0, 0
		for {
			oldResume = resume
			old = splitTextBox(ctx, text, math.Inf(1), resume, true)
			if old.ResumeIndex == nil {
				break
			}
			resume = *old.ResumeIndex
		}
		trimmed := splitTextBox(ctx, text.CopyWithText(stripped), math.Inf(1), oldResume, true)
		if trimmed.Box == nil {
			return old.Box.Width.V
		}
		return old.Box.Width.V - trimmed.Box.Width.V
	}
	split := linebreak.SplitFirstLine(ctx.Text, text.Text, text.Style.LineBreakStyle(), nil,
		text.JustificationSpacing, true, false)
	width := split.Width
	if lastParent != nil {
		children := lastParent.ParentBase().Children
		if len(children) >= 2 {
			width += trailingWhitespaceSize(ctx, children[len(children)-2])
		}
	}
	return width
}
