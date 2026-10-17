package layout

import (
	"math"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func splitTextBox(ctx *Context, box *boxes.TextBox, width float64, skip int, lineStart bool) textResult {
	if box.Style.FontSize() == 0 || skip >= len(box.Text) {
		return textResult{}
	}
	text := box.Text[skip:]
	maxWidth := &width
	if math.IsInf(width, 1) {
		maxWidth = nil
	}
	split := linebreak.SplitFirstLine(
		ctx.Text,
		text,
		box.Style.LineBreakStyle(),
		maxWidth,
		box.JustificationSpacing,
		lineStart,
		false,
	)
	result := textResult{}
	if split.Length > 0 {
		fragment := box.CopyWithText(split.Layout.Text)
		fragment.Width, fragment.Height = units.F(split.Width), units.F(split.Height)
		fragment.Layout = split.Layout
		lineHeight, _ := box.Style.Strut()
		halfLeading := (lineHeight - split.Height) / 2
		fragment.MarginTop, fragment.MarginBottom = units.F(halfLeading), units.F(halfLeading)
		baseline := split.Baseline + halfLeading
		fragment.Baseline = &baseline
		result.Box = fragment
	}
	if split.ResumeIndex >= 0 {
		between := text[split.Length:split.ResumeIndex]
		result.PreservedLineBreak = split.Length != split.ResumeIndex && strings.Trim(between, " ") != ""
		resume := skip + split.ResumeIndex
		result.ResumeIndex = &resume
	}
	return result
}

func firstResume(resume ResumeAt) (int, ResumeAt) {
	if len(resume) == 0 {
		return 0, nil
	}
	index := math.MaxInt
	for child := range resume {
		index = min(index, child)
	}
	return index, resume[index]
}

func skipFirstWhitespace(box boxes.Box, skip ResumeAt) (ResumeAt, bool) {
	index, childSkip := firstResume(skip)
	if text, ok := box.(*boxes.TextBox); ok {
		if index == len(text.Text) {
			return nil, true
		}
		if collapsesSpaces(text) {
			for index < len(text.Text) && text.Text[index] == ' ' {
				index++
			}
		}
		if index == 0 {
			return nil, false
		}
		return ResumeAt{index: nil}, false
	}
	if box.Kind() != boxes.KindLine && box.Kind() != boxes.KindInline {
		return nil, false
	}
	children := boxes.ChildrenOf(box)
	if len(children) == 0 && index == 0 {
		return nil, false
	}
	for index < len(children) {
		result, exhausted := skipFirstWhitespace(children[index], childSkip)
		if !exhausted {
			if index == 0 && result == nil {
				return nil, false
			}
			return ResumeAt{index: result}, false
		}
		index++
		childSkip = nil
	}
	return nil, true
}

func collapsesSpaces(box boxes.Box) bool {
	switch box.Base().Style.WhiteSpace() {
	case "normal", "nowrap", "pre-line":
		return true
	}
	return false
}
