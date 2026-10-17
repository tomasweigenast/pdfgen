package layout

import (
	"fmt"
	"iter"
	"math"
	"slices"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// iterLineBoxes lays out the lines of the line box of a block container
// that is the content of containing, one after the other, starting at
// positionY. Each line holds as much content as fits in the width available
// to it, which floats reduce. Lines are laid out as the caller asks for
// them: stopping early leaves the following lines, and the floats and
// positioned boxes they contain, out of the page.
func iterLineBoxes(ctx *Context, box, containing boxes.Box, opts blockOptions, positionY float64) iter.Seq[lineResult] {
	return func(yield func(lineResult) bool) {
		linebox, ok := box.(*boxes.LineBox)
		if !ok {
			ctx.fail(fmt.Errorf("lines cannot be laid out in a %s box", box.Kind()))
			return
		}
		cb := containingBlockOf(containing)
		resolvePercentages(linebox, cb)
		skip := opts.SkipStack
		if skip == nil {
			linebox.TextIndent = resolveOnePercentage(linebox.Style, props.PropTextIndent, cb.Width).Or(0)
		} else {
			linebox.TextIndent = 0
		}
		firstLetter, firstLine := opts.FirstLetterStyle, opts.FirstLineStyle
		for {
			result := getNextLinebox(ctx, linebox, containing, positionY, opts.BottomSpace, skip,
				opts.Positioned, firstLetter, firstLine)
			firstLine = nil
			if result.Line == nil {
				return
			}
			positionY = result.Line.PositionY + result.Line.Height.V
			if !yield(result) || result.ResumeAt == nil {
				return
			}
			skip = result.ResumeAt
			linebox.TextIndent = 0
			firstLetter = nil
		}
	}
}

// excludedShapes returns the floats of the current block formatting context.
func excludedShapes(ctx *Context) []boxes.Box {
	if ctx.ExcludedShapes == nil {
		return nil
	}
	return *ctx.ExcludedShapes
}

// replaceExcludedShapes sets the floats of the current block formatting
// context.
func replaceExcludedShapes(ctx *Context, shapes []boxes.Box) {
	if len(ctx.bfcStack) == 0 {
		return
	}
	frame := ctx.bfcStack[len(ctx.bfcStack)-1]
	frame.Shapes = shapes
	ctx.ExcludedShapes = &frame.Shapes
}

// getNextLinebox returns the line that follows skip in linebox, a copy of
// it with the children that fit, and where the next one resumes; the line
// is nil when the rest of the content is white space that disappears.
//
// A float in the line, or one that the line makes room for, reduces the
// width available to it, which can move it down or make it taller than
// planned and so reach other floats: the line is laid out again until it
// settles (CSS 2.1 §9.5).
func getNextLinebox(
	ctx *Context,
	linebox *boxes.LineBox,
	containing boxes.Box,
	positionY, bottomSpace float64,
	skip ResumeAt,
	positioned *positionedBoxes,
	firstLetterStyle, firstLineStyle *computed.Style,
) lineResult {
	if positioned == nil {
		positioned = &positionedBoxes{}
	}
	skip, exhausted := skipFirstWhitespace(linebox, skip)
	if exhausted {
		return lineResult{}
	}
	skip = firstLetterToBox(linebox, skip, firstLetterStyle)

	lb := linebox.Base()
	lb.PositionY = positionY

	if len(excludedShapes(ctx)) != 0 {
		// The size of the line is needed to find a place that avoids the
		// floats.
		lb.Width = units.F(inlineMinContentWidth(ctx, linebox, true, skip, true, false))
		height, _ := lb.Style.Strut()
		lb.Height = units.F(height)
	} else {
		lb.Width, lb.Height = units.F(0), units.F(0)
	}
	positionX, positionY, availableWidth := avoidCollisions(ctx, linebox, containing, false)

	candidateHeight := lb.Height.V
	shapes := slices.Clone(excludedShapes(ctx))
	rtl := containing.Base().Style.Direction() == "rtl"

	var (
		line             *boxes.LineBox
		resume           ResumeAt
		attempt          *positionedBoxes
		linePlaceholders []*boxes.AbsolutePlaceholder
		waitingFloats    []boxes.Box
	)
	for {
		originalPositionX, originalPositionY := positionX, positionY
		lb.PositionX, lb.PositionY = positionX, positionY
		originalWidth := lb.Width.V
		maxX := positionX + availableWidth
		positionX += linebox.TextIndent

		attempt = &positionedBoxes{}
		linePlaceholders, waitingFloats = nil, nil
		var lineChildren []boxes.Box
		split := splitInlineBox(ctx, linebox, containing, inlineOptions{
			PositionX:        positionX,
			MaxX:             maxX,
			BottomSpace:      bottomSpace,
			SkipStack:        skip,
			Positioned:       attempt,
			LinePlaceholders: &linePlaceholders,
			WaitingFloats:    &waitingFloats,
			LineChildren:     &lineChildren,
			FirstLetterStyle: firstLetterStyle,
			FirstLineStyle:   firstLineStyle,
		})
		var ok bool
		line, ok = split.Box.(*boxes.LineBox)
		if !ok {
			ctx.fail(fmt.Errorf("splitting a line made a %v box", split.Box))
			return lineResult{}
		}
		resume = split.ResumeAt
		lb.Width, lb.Height = line.Width, line.Height

		if isPhantomLinebox(line) && !split.PreservedLineBreak {
			// A line without content takes no room (CSS 2.1 §9.4.2).
			line.Height = units.F(0)
			break
		}

		removeLastWhitespace(ctx, line)

		_, _, newAvailableWidth := avoidCollisions(ctx, linebox, containing, false)
		offsetX := textAlign(ctx, line, newAvailableWidth, resume == nil || split.PreservedLineBreak)
		if rtl {
			offsetX = -offsetX - line.Width.V
		}

		bottom, top := lineBoxVerticality(line)
		baseline := -top
		line.Baseline = &baseline
		line.PositionY = top
		line.Height = units.F(bottom - top)
		offsetY := positionY - top
		line.MarginTop, line.MarginBottom = units.F(0), units.F(0)

		line.Translate(offsetX, offsetY, false)
		// position_y - top + top is not always position_y.
		line.PositionY = positionY

		if line.Height.V <= candidateHeight {
			break
		}
		candidateHeight = line.Height.V

		newShapes := excludedShapes(ctx)
		replaceExcludedShapes(ctx, slices.Clone(shapes))
		positionX, positionY, availableWidth = avoidCollisions(ctx, line, containing, false)

		if hasDeclarations(firstLineStyle) {
			wrapper := line.CopyWithChildren(line.Children).(*boxes.LineBox)
			wrapper.ElementTag += "::first-line"
			wrapper.Style = wrapper.Style.Copy()
			applyDeclared(wrapper.Style, firstLineStyle, false)
			line.Children = []boxes.Box{wrapper}
			adjustLineHeight(wrapper)
		}

		var settled bool
		if rtl {
			settled = positionX+line.Width.V == originalPositionX+originalWidth && positionY == originalPositionY
		} else {
			settled = positionX == originalPositionX && positionY == originalPositionY
		}
		if settled {
			replaceExcludedShapes(ctx, newShapes)
			break
		}
	}

	positioned.Absolute = append(positioned.Absolute, attempt.Absolute...)
	positioned.Fixed = append(positioned.Fixed, attempt.Fixed...)

	for _, placeholder := range linePlaceholders {
		if isInlineLevelDisplay(placeholder.Base().Style) {
			// The static position of an inline-level box is in the line.
			placeholder.Translate(0, positionY-placeholder.Base().PositionY, false)
		} else {
			// The static position of a block-level box is at the start of
			// the next line.
			placeholder.Translate(
				line.PositionX-placeholder.Base().PositionX,
				positionY+line.Height.V-placeholder.Base().PositionY, false)
		}
	}

	var floatChildren []boxes.Box
	waitingY := line.PositionY + line.Height.V
	for _, waiting := range waitingFloats {
		waiting.Base().PositionY = waitingY
		result := floatLayout(ctx, waiting, containing, positioned, bottomSpace, nil)
		if result.Box == nil {
			continue
		}
		floatChildren = append(floatChildren, result.Box)
		if result.ResumeAt != nil {
			ctx.addBrokenOutOfFlow(result.Box, waiting, containing, result.ResumeAt)
		}
	}
	if len(floatChildren) != 0 {
		line.Children = slices.Concat(line.Children, floatChildren)
	}

	return lineResult{Line: line, ResumeAt: resume}
}

// isInlineLevelDisplay reports whether the display that a style specified,
// before blockification, is inline-level.
func isInlineLevelDisplay(style *computed.Style) bool {
	display, ok := style.Specified(props.PropDisplay).(props.Display)
	if !ok {
		display = style.Display()
	}
	return slices.Contains(display, "inline")
}

// removeLastWhitespace removes the spaces at the end of a line, which hang
// outside of it, and narrows the boxes that contain them (CSS Text 3 §4.1.2).
func removeLastWhitespace(ctx *Context, line *boxes.LineBox) {
	ancestors := []boxes.Box{}
	var box boxes.Box = line
	for box.Kind() == boxes.KindLine || box.Kind() == boxes.KindInline {
		ancestors = append(ancestors, box)
		children := boxes.ChildrenOf(box)
		if len(children) == 0 {
			return
		}
		box = children[len(children)-1]
	}
	text, ok := box.(*boxes.TextBox)
	if !ok || !collapsesSpaces(text) {
		return
	}

	var spaceWidth float64
	if stripped := strings.TrimRight(text.Text, " "); stripped != "" {
		if len(stripped) == len(text.Text) {
			return
		}
		text.Text = stripped
		result := splitTextBox(ctx, text, math.Inf(1), 0, true)
		if result.Box == nil {
			return
		}
		spaceWidth = text.Width.V - result.Box.Width.V
		text.Width = result.Box.Width
	} else {
		spaceWidth = text.Width.V
		text.Width = units.F(0)
		text.Text = ""
	}

	// The spaces of a right-to-left line are on the left of the text:
	// move the line so that the text stays against the right edge.
	if text.Layout != nil && text.Layout.FirstLine().RTL {
		for _, child := range line.Children {
			child.Translate(-spaceWidth, 0, true)
		}
	}
	for _, ancestor := range ancestors {
		ancestor.Base().Width = units.F(ancestor.Base().Width.V - spaceWidth)
	}
}

// isPhantomLinebox reports whether a line has no content and no box with
// margins, borders or padding: it takes no space and is not displayed (CSS
// 2.1 §9.4.2).
func isPhantomLinebox(line boxes.Box) bool {
	for _, child := range boxes.ChildrenOf(line) {
		if inline, ok := child.(*boxes.InlineBox); ok {
			if !isPhantomLinebox(inline) {
				return false
			}
			s := inline.Style
			for _, size := range [...]float64{
				lengthValue(s.MarginTop()), float64(s.BorderTopWidth()), lengthValue(s.PaddingTop()),
				lengthValue(s.MarginRight()), float64(s.BorderRightWidth()), lengthValue(s.PaddingRight()),
				lengthValue(s.MarginBottom()), float64(s.BorderBottomWidth()), lengthValue(s.PaddingBottom()),
				lengthValue(s.MarginLeft()), float64(s.BorderLeftWidth()), lengthValue(s.PaddingLeft()),
			} {
				if size != 0 {
					return false
				}
			}
		} else if child.IsInNormalFlow() {
			return false
		}
	}
	return true
}

// lengthValue returns the number of a length that is not a keyword or a
// math function, and zero for the others.
func lengthValue(l props.Length) float64 {
	if l.IsKeyword() || l.Calc != nil {
		return 0
	}
	return l.Dimension.Value
}
