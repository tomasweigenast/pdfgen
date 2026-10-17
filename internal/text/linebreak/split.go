package linebreak

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/hyphen"
)

// SplitResult describes the first line of a text.
type SplitResult struct {
	// Layout holds the first line. Its text is the text of the line, with
	// the hyphenation character when the line is hyphenated.
	Layout *Layout
	// Length is the length in bytes of the text on the first line.
	Length int
	// ResumeIndex is the byte offset where the next line starts, -1 when
	// the whole text fits. It is greater than Length when a preserved line
	// break or a collapsed space separates the lines.
	ResumeIndex int
	// Width, Height and Baseline are the size of the first line and the
	// position of its baseline from its top, in px.
	Width, Height, Baseline float64
}

// maxWrapWidth is the widest width lines wrap at: wider widths do not fit
// in 32-bit 1/1024 px units and are treated as unconstrained.
const maxWrapWidth = 1 << 21

// NewLayout returns a layout of text that wraps at maxWidth px, or does not
// wrap when maxWidth is nil. The justification spacing is added to
// word-spacing when the layout is drawn, see Reactivate.
func NewLayout(ctx *Context, text string, style *Style, maxWidth *float64, justificationSpacing float64) *Layout {
	return createLayout(ctx, text, style, maxWidth, justificationSpacing)
}

// createLayout returns a layout of text wrapping at maxWidth.
func createLayout(ctx *Context, text string, style *Style, maxWidth *float64, justificationSpacing float64) *Layout {
	l := newLayout(ctx, style, justificationSpacing, maxWidth)
	if maxWidth != nil && style.textWrap() && *maxWidth < maxWrapWidth {
		l.setWidth(int(math.Max(0, *maxWidth) * fonts.UnitsPerPx))
	}
	l.SetText(text)
	return l
}

// Reactivate lays the text out again for drawing, without width limit and
// with the justification spacing added to word-spacing.
func (l *Layout) Reactivate(style *Style) {
	l.Style = style
	l.width = -1
	l.wrap = wrapWord
	l.setText(l.Text, true)
}

func firstLineMetrics(line *Line, text string, layout *Layout, resumeIndex int, spaceCollapse bool, style *Style, hyphenated bool) SplitResult {
	length := line.Length
	if hyphenated {
		length -= len(style.HyphenateCharacter)
	} else if resumeIndex >= 0 {
		// Lines are already split: lay the first one out again without
		// width, as it is drawn.
		layout.setWidth(-1)
		lineText := text[:length]
		if spaceCollapse {
			lineText = strings.TrimRight(lineText, " ")
		}
		layout.SetText(lineText)
		line = layout.FirstLine()
		length = line.Length
	}
	width, height := LineSize(line, style)
	return SplitResult{
		Layout:      layout,
		Length:      length,
		ResumeIndex: resumeIndex,
		Width:       width,
		Height:      height,
		Baseline:    layout.Baseline(),
	}
}

// SplitFirstLine fits as much of text as possible on a line of maxWidth
// px. A nil maxWidth means no limit. Lines break at the opportunities of
// UAX #14, words are hyphenated as hyphens asks, and words are broken when
// word-break or overflow-wrap allow it (CSS Text 3 §5). isLineStart tells
// whether text starts a line, and minimum asks for the min-content width,
// where overflow-wrap: break-word does not break words.
//
// Unless a line is hyphenated, the trailing spaces of a line are left out
// of its width when spaces collapse.
func SplitFirstLine(ctx *Context, text string, style *Style, maxWidth *float64, justificationSpacing float64, isLineStart, minimum bool) SplitResult {
	textWrap := style.textWrap()
	spaceCollapse := style.spaceCollapse()

	originalMaxWidth := maxWidth
	if !textWrap {
		maxWidth = nil
	}

	// Step 1: lay out a draft of the first line. Laying out the whole
	// text is slow, so only a part of it is used when its first line and
	// a break opportunity on the second line are likely to fit in it.
	const ratio = 4 // characters are almost always less than 4 times as wide as tall
	shortText := text
	var layout *Layout
	var line *Line
	var resumeIndex int
	fontSize := style.Font.Size
	if maxWidth != nil && !math.IsInf(*maxWidth, 1) && fontSize != 0 {
		if fontSize*ratio > *maxWidth {
			// Very narrow: keep one word, a space and one letter.
			if i := strings.IndexByte(text, ' '); i >= 0 {
				shortText = text[:byteOffset(text, utf8.RuneCountInString(text[:i])+2)]
			}
		} else {
			shortText = text[:byteOffset(text, int(*maxWidth/fontSize*ratio))]
		}
		layout = createLayout(ctx, shortText, style, maxWidth, justificationSpacing)
		line, resumeIndex = layout.firstLine()
		if resumeIndex < 0 && shortText != text {
			shortText = text
			layout.SetText(text)
			line, resumeIndex = layout.firstLine()
		} else {
			firstLineText := shortText
			if resumeIndex >= 0 {
				firstLineText = shortText[:resumeIndex]
			}
			if firstLineText != shortText {
				attrs := attrsRange(layout.logAttrs(), utf8.RuneCountInString(firstLineText)+1, utf8.RuneCountInString(shortText))
				if nextBreakPoint(attrs) >= 0 {
					text = shortText
				}
			}
		}
	} else {
		layout = createLayout(ctx, text, style, originalMaxWidth, justificationSpacing)
		line, resumeIndex = layout.firstLine()
	}

	// Step 2: keep the line when it does not have to be split.
	if maxWidth == nil {
		return firstLineMetrics(line, text, layout, resumeIndex, spaceCollapse, style, false)
	}
	available := *maxWidth
	lineWidth, _ := LineSize(line, style)
	if resumeIndex < 0 && lineWidth <= available {
		return firstLineMetrics(line, text, layout, resumeIndex, spaceCollapse, style, false)
	}

	// Step 3: the line may have been broken too early. Try to put the
	// first word of the second line on the first line.
	var firstLineText, secondLineText string
	if lineWidth <= available {
		firstLineText, secondLineText = text[:resumeIndex], text[resumeIndex:]
	} else {
		// The line cannot be broken earlier: try to hyphenate its first
		// word.
		secondLineText = text
	}
	breakPoint := -1
	if firstLineText != shortText {
		attrs := attrsRange(layout.logAttrs(), utf8.RuneCountInString(firstLineText)+1, utf8.RuneCountInString(shortText))
		breakPoint = nextBreakPoint(attrs)
	}
	// The word ends before the character preceding the break, usually a
	// space.
	nextWord := secondLineText
	if breakPoint >= 0 {
		nextWord = secondLineText[:byteOffset(secondLineText, breakPoint)]
	}
	nextWord = strings.TrimRight(nextWord, " ")
	if nextWord != "" {
		if spaceCollapse && runeAt(secondLineText, breakPoint) == ' ' {
			// The word might fit without the space after it, which only
			// works when spaces collapse.
			newFirstLineText := firstLineText + nextWord
			layout.SetText(newFirstLineText)
			line, resumeIndex = layout.firstLine()
			if resumeIndex < 0 {
				if firstLineText != "" {
					resumeIndex = len(newFirstLineText) + 1
					return firstLineMetrics(line, text, layout, resumeIndex, spaceCollapse, style, false)
				}
				resumeIndex = line.Length + 1
				if resumeIndex >= len(text) {
					resumeIndex = -1
				}
			}
		}
	} else if firstLineText != "" {
		// No word on the second line: there is nothing to hyphenate.
		return firstLineMetrics(line, text, layout, resumeIndex, spaceCollapse, style, false)
	}

	// Step 4: try to hyphenate.
	hyphens := style.hyphens()
	lang := ""
	if style.Lang != "" {
		lang = hyphen.LanguageFallback(style.Lang)
	}
	limits := style.HyphenateLimitChars
	hyphenated := false
	const softHyphen = "\u00ad"
	hyphenChar := style.HyphenateCharacter

	manual := hyphens != "none" && strings.Contains(firstLineText+secondLineText, softHyphen)
	auto := false
	nextTextIndex := 0
	if hyphens == "auto" && lang != "" {
		// Hyphenate words before the next break opportunity.
		nextText := []rune(secondLineText)
		if bp := nextBreakPointInText(nextText, lang); bp > 0 {
			nextText = nextText[:bp]
		}
		for len(nextText) > 0 {
			start, end, ok := nextWordBoundaries(nextText, lang)
			if !ok {
				break
			}
			nextWord = string(nextText[start:end])
			if end-start >= limits.Total {
				lineWidth, _ := LineSize(line, style)
				space := available - lineWidth
				zone := style.HyphenateLimitZone.Resolve(available)
				if space > zone || space < 0 {
					// Hyphenating is worth it, or the line is too long
					// anyway.
					auto = true
					nextTextIndex += start
					break
				}
			}
			nextText = nextText[end:]
			nextTextIndex += end
		}
	}

	// Words with soft hyphens are only hyphenated at them (CSS Text 3
	// §5.4).
	var attempts []string
	switch {
	case manual:
		if strings.HasSuffix(firstLineText, softHyphen) {
			// The line was broken at a soft hyphen.
			firstLineText, secondLineText = "", firstLineText
		}
		runes := []rune(secondLineText)
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] == '\u00ad' {
				attempts = append(attempts, string(runes[:i+1]))
			}
		}
	case auto:
		if dict := ctx.hyphenator(lang, limits.Before, limits.After); dict != nil {
			previous := string([]rune(secondLineText)[:nextTextIndex])
			for _, s := range dict.Iterate(nextWord) {
				attempts = append(attempts, previous+s.Head)
			}
		}
	}

	var newFirstLineText, hyphenatedText string
	if len(attempts) > 0 {
		for _, part := range attempts {
			newFirstLineText = firstLineText + part
			hyphenatedText = newFirstLineText + hyphenChar
			newLayout := createLayout(ctx, hyphenatedText, style, maxWidth, justificationSpacing)
			newLine, index := newLayout.firstLine()
			newWidth, _ := LineSize(newLine, style)
			// The shortest attempt is kept even when it overflows.
			hyphenated = index < 0 && (available-newWidth >= 0 || part == attempts[len(attempts)-1])
			if hyphenated {
				layout, line = newLayout, newLine
				resumeIndex = len(newFirstLineText)
				break
			}
		}
		if !hyphenated && firstLineText == "" {
			// Lay the hyphenated word out without width, so that no break
			// happens before or inside the hyphenation character.
			hyphenated = true
			layout.SetText(hyphenatedText)
			layout.setWidth(-1)
			line = layout.FirstLine()
			resumeIndex = len(newFirstLineText)
			if strings.HasPrefix(text, softHyphen) {
				resumeIndex += len(softHyphen)
			}
		}
	}
	if !hyphenated && strings.HasSuffix(firstLineText, softHyphen) {
		hyphenated = true
		layout.SetText(firstLineText + hyphenChar)
		layout.setWidth(-1)
		line = layout.FirstLine()
		resumeIndex = len(firstLineText)
	}

	// Step 5: break the word if it is too long for the line.
	lineWidth, _ = LineSize(line, style)
	canBreak := style.WordBreak == "break-all" || (isLineStart &&
		(style.OverflowWrap == "anywhere" || (style.OverflowWrap == "break-word" && !minimum)))
	if available-lineWidth < 0 && canBreak {
		// Words broken this way are shaped as separate words.
		hyphenated = false
		layout.SetText(text)
		layout.setWidth(int(available * fonts.UnitsPerPx))
		layout.setWrap(wrapChar)
		var index int
		line, index = layout.firstLine()
		resumeIndex = index
		if index <= 0 {
			resumeIndex = line.Length
		}
		if resumeIndex >= len(text) {
			resumeIndex = -1
		}
	}

	return firstLineMetrics(line, text, layout, resumeIndex, spaceCollapse, style, hyphenated)
}

// attrsRange returns attrs[start:end], clamped to the attributes there are.
func attrsRange(attrs []logAttr, start, end int) []logAttr {
	end = min(end, len(attrs))
	if start >= end {
		return nil
	}
	return attrs[start:end]
}

// runeAt returns the character at index i of text, the last character for
// a non-positive index.
func runeAt(text string, i int) rune {
	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}
	if i <= 0 {
		return runes[len(runes)-1]
	}
	if i >= len(runes) {
		return 0
	}
	return runes[i]
}
