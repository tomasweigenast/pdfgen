package linebreak

import (
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/shape"
)

type breakResult uint8

const (
	breakNoneFit breakResult = iota
	breakSomeFit
	breakAllFit
	breakEmptyFit
	breakLineSeparator
)

// tabState follows the last tab of a line, whose width depends on the
// position of the text after it.
type tabState struct {
	glyph *Glyph // nil when there is no tab to follow
	index int    // index of the tab stop
	width int    // width of the line before the tab
	pos   int    // position of the tab stop
}

// breakState is the state of line filling within a paragraph. Character
// offsets index the whole layout.
type breakState struct {
	items []*item
	// glyphs are the glyphs of items[0] as shaped when it became the first
	// item, and logWidths the widths of its characters.
	glyphs          []Glyph
	logWidths       []int
	logWidthsOffset int
	startOffset     int // first character of items[0]

	lineStartIndex  int // in bytes
	lineStartOffset int
	lineOfPar       int

	lineWidth      int
	remainingWidth int // negative when lines do not wrap
	hyphenWidth    int // negative until needed

	lastTab tabState
	baseRTL bool
}

func (l *Layout) breakParagraph(start int, items []*item) {
	st := &breakState{
		items:           items,
		startOffset:     start,
		lineStartIndex:  l.offsets[start],
		lineStartOffset: start,
		lineOfPar:       1,
		lineWidth:       -1,
		remainingWidth:  -1,
		hyphenWidth:     -1,
		baseRTL:         l.Style.rtl(),
	}
	if len(items) == 0 {
		l.lines = append(l.lines, &Line{layout: l, Start: st.lineStartIndex, RTL: st.baseRTL, paragraphStart: true})
		return
	}
	for len(st.items) > 0 && len(l.lines) < 2 {
		l.lines = append(l.lines, l.processLine(st))
	}
}

func (l *Layout) canBreakAt(offset int, wrap wrapMode) bool {
	if offset == len(l.runes) {
		return true
	}
	if wrap == wrapChar {
		return l.attrs[offset].charBreak
	}
	return l.attrs[offset].lineBreak
}

func (l *Layout) canBreakIn(start, n int, allowAtStart bool) bool {
	i := 1
	if allowAtStart {
		i = 0
	}
	for ; i < n; i++ {
		if l.canBreakAt(start+i, l.wrap) {
			return true
		}
	}
	return false
}

func (l *Layout) breakNeedsHyphen(st *breakState, pos int) bool {
	a := l.attrs[st.startOffset+pos]
	return a.insertsHyphen || a.removesPreceding
}

func (l *Layout) ensureHyphenWidth(st *breakState) {
	if st.hyphenWidth >= 0 {
		return
	}
	st.hyphenWidth = 0
	face := st.items[0].face
	if face == nil {
		return
	}
	gid, ok := face.GoText.NominalGlyph('\u2010')
	if !ok {
		gid, ok = face.GoText.NominalGlyph('-')
	}
	if ok {
		hb := face.HB(fonts.ToUnits(l.Style.Font.Size))
		st.hyphenWidth = int(hb.GlyphHAdvance(gid))
	}
}

// findBreakExtraWidth returns the width a break at pos adds to the line:
// the hyphen it shows, or minus the width of the space it hides.
func (l *Layout) findBreakExtraWidth(st *breakState, pos int) int {
	a := l.attrs[st.startOffset+pos]
	if a.insertsHyphen {
		l.ensureHyphenWidth(st)
		if a.removesPreceding && pos > 0 {
			return st.hyphenWidth - st.logWidths[st.logWidthsOffset+pos-1]
		}
		return st.hyphenWidth
	}
	if pos > 0 && l.attrs[st.startOffset+pos-1].white {
		return -st.logWidths[st.logWidthsOffset+pos-1]
	}
	return 0
}

// lineWidthSoFar returns the width of the runs already on the line.
func lineWidthSoFar(st *breakState, line *Line) int {
	if st.remainingWidth > -1 {
		return st.lineWidth - st.remainingWidth
	}
	return line.Width()
}

// shapeRun shapes an item placed on a line. Tabs extend to the next tab
// stop.
func (l *Layout) shapeRun(line *Line, st *breakState, it *item) []Glyph {
	if l.runes[it.start] != '\t' {
		return l.shapeItem(it)
	}
	current := lineWidthSoFar(st, line)
	stop := l.tabStop
	if stop <= 0 {
		stop = 1
	}
	glyphs := []Glyph{{Empty: true}}
	i := st.lastTab.index
	for ; ; i++ {
		pos := stop * (i + 1)
		if pos >= current+1 {
			glyphs[0].XAdvance = int32(pos - current)
			st.lastTab = tabState{glyph: &glyphs[0], index: i, width: current, pos: pos}
			break
		}
	}
	return glyphs
}

func (l *Layout) computeLogWidths(st *breakState) {
	st.logWidths = l.logicalWidths(st.items[0], st.glyphs)
}

// approximateCharWidth returns the average advance of the characters of
// an English pangram, which bounds how far from the end of a line break
// opportunities are measured precisely.
func (l *Layout) approximateCharWidth(it *item) int {
	const sample = "The quick brown fox jumps over the lazy dog."
	size := fonts.ToUnits(l.Style.Font.Size)
	key := faceSize{it.face, size}
	if w, ok := l.ctx.charWidths[key]; ok {
		return w
	}
	runes := []rune(sample)
	glyphs := l.ctx.Shaper.Shape(shape.Run{
		Text: runes, End: len(runes), Face: it.face, Size: fonts.FromUnits(size),
	})
	total := 0
	for _, g := range glyphs {
		total += int(g.XAdvance)
	}
	w := total / len(runes)
	l.ctx.charWidths[key] = w
	return w
}

func (l *Layout) insertRun(line *Line, st *breakState, it *item, glyphs []Glyph, lastRun bool) *Run {
	if glyphs == nil {
		if lastRun && st.logWidthsOffset == 0 && st.glyphs != nil && !it.needHyphen {
			glyphs = st.glyphs
			st.glyphs = nil
		} else {
			glyphs = l.shapeRun(line, st, it)
		}
	}
	if lastRun {
		st.glyphs = nil
	}
	run := &Run{
		Face:    it.face,
		Size:    fonts.FromUnits(fonts.ToUnits(l.Style.Font.Size)),
		Start:   l.offsets[it.start],
		End:     l.offsets[it.end],
		Level:   it.level,
		Script:  it.script,
		Glyphs:  glyphs,
		item:    it,
		tab:     l.runes[it.start] == '\t',
		spacing: it.spacing,
	}
	line.Runs = append(line.Runs, run)
	line.Length += run.End - run.Start
	if st.lastTab.glyph != nil && (len(glyphs) == 0 || &glyphs[0] != st.lastTab.glyph) {
		// Text after a left-aligned tab does not move the tab stop; the
		// tab stops being followed once it has no width.
		width := max(st.lastTab.pos-st.lastTab.width, 0)
		st.lastTab.glyph.XAdvance = int32(width)
		if width == 0 {
			st.lastTab.glyph = nil
		}
	}
	return run
}

// processItem puts as much as possible of st.items[0] on the line. With
// forceFit, something is always put on the line even if it overflows;
// with noBreakAtEnd, the line cannot end after the whole item.
func (l *Layout) processItem(line *Line, st *breakState, forceFit, noBreakAtEnd, isLastItem bool) breakResult {
	it := st.items[0]
	processingNew := false
	if st.glyphs == nil {
		st.glyphs = l.shapeRun(line, st, it)
		st.logWidthsOffset = 0
		processingNew = true
	}

	if l.runes[it.start] == '\u2028' {
		l.insertRun(line, st, it, nil, true)
		st.logWidthsOffset += it.numChars()
		return breakLineSeparator
	}

	if st.remainingWidth < 0 && !noBreakAtEnd {
		l.insertRun(line, st, it, nil, true)
		return breakAllFit
	}

	n := it.numChars()
	width := 0
	if processingNew {
		width = glyphsWidth(st.glyphs)
	} else {
		for i := 0; i < n; i++ {
			width += st.logWidths[st.logWidthsOffset+i]
		}
	}

	if l.runes[it.start] == '\t' {
		l.insertRun(line, st, it, nil, true)
		st.remainingWidth = max(st.remainingWidth-width, 0)
		return breakAllFit
	}

	wrap := l.wrap
	extra := 0
	if !noBreakAtEnd && l.canBreakAt(st.startOffset+n, wrap) {
		if processingNew {
			l.computeLogWidths(st)
			processingNew = false
		}
		if !isLastItem {
			extra = l.findBreakExtraWidth(st, n)
		}
	}

	if (width+extra <= st.remainingWidth || (n == 1 && len(line.Runs) == 0)) && !noBreakAtEnd {
		glyphs := l.shapeRun(line, st, it)
		width = glyphsWidth(glyphs)
		if width+extra <= st.remainingWidth || (n == 1 && len(line.Runs) == 0) {
			l.insertRun(line, st, it, glyphs, true)
			st.remainingWidth = max(st.remainingWidth-width, 0)
			return breakAllFit
		}
	}

	breakWidth, breakExtra := width, extra
	breakNumChars := n
	var breakGlyphs []Glyph
	safeDistance := l.approximateCharWidth(it) * 3
	if processingNew {
		l.computeLogWidths(st)
	}

	limit := n + 1
	if noBreakAtEnd {
		limit = n
	}
	width = 0
	for numChars := 0; numChars < limit; numChars++ {
		if !isLastItem || numChars != n {
			extra = l.findBreakExtraWidth(st, numChars)
		} else {
			extra = 0
		}

		// Widths only grow, so once far past the end of the line no
		// better break opportunity is left.
		if min(width+extra, width) > st.remainingWidth+safeDistance && breakNumChars < n {
			break
		}

		if l.canBreakAt(st.startOffset+numChars, wrap) && (numChars > 0 || len(line.Runs) > 0) {
			if numChars == 0 || width+extra < st.remainingWidth-safeDistance {
				breakNumChars, breakWidth, breakExtra = numChars, width, extra
			} else {
				candidate := it
				if numChars < n {
					head := *it
					head.end = it.start + numChars
					head.needHyphen = l.breakNeedsHyphen(st, numChars)
					candidate = &head
				}
				glyphs := l.shapeRun(line, st, candidate)
				newWidth := glyphsWidth(glyphs)
				switch {
				case numChars > 0 && (candidate != it || !isLastItem) && l.attrs[st.startOffset+numChars-1].white:
					extra = -st.logWidths[st.logWidthsOffset+numChars-1]
				case candidate == it && !isLastItem && l.breakNeedsHyphen(st, numChars):
					l.ensureHyphenWidth(st)
					extra = st.hyphenWidth
				default:
					extra = 0
				}
				if breakNumChars == n || newWidth+extra <= st.remainingWidth || newWidth+extra < breakWidth+breakExtra {
					breakNumChars, breakWidth, breakExtra = numChars, newWidth, extra
					breakGlyphs = glyphs
				}
			}
		}

		if numChars < n {
			width += st.logWidths[st.logWidthsOffset+numChars]
		}
	}

	if !forceFit && breakWidth+breakExtra > st.remainingWidth {
		st.glyphs = nil
		return breakNoneFit
	}
	if st.remainingWidth >= 0 {
		st.remainingWidth = max(st.remainingWidth-(breakWidth+breakExtra), 0)
	}
	switch breakNumChars {
	case n:
		if l.canBreakAt(st.startOffset+n, wrap) && l.breakNeedsHyphen(st, n) {
			it.needHyphen = true
		}
		l.insertRun(line, st, it, nil, true)
		return breakAllFit
	case 0:
		return breakEmptyFit
	default:
		head := it.split(breakNumChars)
		l.insertRun(line, st, head, breakGlyphs, false)
		st.logWidthsOffset += breakNumChars
		return breakSomeFit
	}
}

func (l *Layout) processLine(st *breakState) *Line {
	line := &Line{layout: l, Start: st.lineStartIndex, RTL: st.baseRTL, paragraphStart: st.lineOfPar == 1}
	st.lineWidth = l.width
	st.remainingWidth = st.lineWidth
	st.lastTab = tabState{}

	haveBreak := false
	breakRemaining, breakStartOffset, breakRunCount := 0, 0, 0
	wrapped := false

loop:
	for len(st.items) > 0 {
		it := st.items[0]
		oldNumChars := it.numChars()
		oldRemaining := st.remainingWidth
		firstItemInLine := len(line.Runs) == 0
		lastItemInLine := len(st.items) == 1

		switch l.processItem(line, st, !haveBreak, false, lastItemInLine) {
		case breakAllFit:
			if l.runes[it.start] != '\t' && l.canBreakIn(st.startOffset, oldNumChars, !firstItemInLine) {
				haveBreak = true
				breakRemaining = oldRemaining
				breakStartOffset = st.startOffset
				breakRunCount = len(line.Runs) - 1
			}
			st.items = st.items[1:]
			st.startOffset += oldNumChars

		case breakEmptyFit:
			wrapped = true
			break loop

		case breakSomeFit:
			st.startOffset += oldNumChars - it.numChars()
			wrapped = true
			break loop

		case breakNoneFit:
			// Go back to the last item that could be broken.
			for len(line.Runs) > breakRunCount {
				run := line.Runs[len(line.Runs)-1]
				if st.lastTab.glyph != nil && len(run.Glyphs) > 0 && &run.Glyphs[0] == st.lastTab.glyph {
					st.lastTab = tabState{}
				}
				line.Runs = line.Runs[:len(line.Runs)-1]
				line.Length -= run.End - run.Start
				st.items = append([]*item{run.item}, st.items...)
			}
			st.startOffset = breakStartOffset
			st.remainingWidth = breakRemaining
			lastItemInLine = len(st.items) == 1

			it = st.items[0]
			oldNumChars = it.numChars()
			switch l.processItem(line, st, true, true, lastItemInLine) {
			case breakAllFit:
				st.items = st.items[1:]
				st.startOffset += oldNumChars
			case breakEmptyFit:
				wrapped = true
			case breakSomeFit:
				st.startOffset += oldNumChars - it.numChars()
				wrapped = true
			}
			break loop

		case breakLineSeparator:
			st.items = st.items[1:]
			st.startOffset += oldNumChars
			wrapped = true
			break loop
		}
	}

	l.postprocess(line, st, wrapped)
	st.lineOfPar++
	st.lineStartIndex += line.Length
	st.lineStartOffset = st.startOffset
	return line
}

func (l *Layout) postprocess(line *Line, st *breakState, wrapped bool) {
	if len(line.Runs) > 0 {
		last := line.Runs[len(line.Runs)-1]
		l.addMissingHyphen(line, st, last)
		if wrapped {
			l.zeroLineFinalSpace(st, last)
		}
	}
	line.Runs = reorderRuns(line.Runs)
	adjustLineLetterSpacing(line)
}

// addMissingHyphen shows a hyphen at the end of a line that ends where a
// break inserts one, when the last run fit without being broken.
func (l *Layout) addMissingHyphen(line *Line, st *breakState, run *Run) {
	chars := 0
	for _, r := range line.Runs {
		chars += r.item.numChars()
	}
	if run.item.needHyphen || !l.attrs[st.lineStartOffset+chars].insertsHyphen {
		return
	}
	run.item.needHyphen = true
	run.Glyphs = l.shapeRun(line, st, run.item)
}

// zeroLineFinalSpace hides the white space a wrapped line ends with.
func (l *Layout) zeroLineFinalSpace(st *breakState, run *Run) {
	glyphs := run.Glyphs
	if len(glyphs) < 1 || st.startOffset == 0 || !l.attrs[st.startOffset-1].white {
		return
	}
	i := len(glyphs) - 1
	neighbour := i - 1
	if run.RTL() {
		i, neighbour = 0, 1
	}
	if len(glyphs) >= 2 && glyphs[i].Cluster == glyphs[neighbour].Cluster {
		return
	}
	glyphs[i].XAdvance = 0
	glyphs[i].Empty = true
}

// reorderRuns puts runs in visual order (UAX #9 rule L2).
func reorderRuns(runs []*Run) []*Run {
	allEven, allOdd := true, true
	for _, r := range runs {
		if r.Level%2 == 1 {
			allEven = false
		} else {
			allOdd = false
		}
	}
	switch {
	case allEven:
		return runs
	case allOdd:
		out := make([]*Run, len(runs))
		for i, r := range runs {
			out[len(runs)-1-i] = r
		}
		return out
	}
	return reorderRecurse(runs)
}

func reorderRecurse(runs []*Run) []*Run {
	if len(runs) == 0 {
		return nil
	}
	minLevel := runs[0].Level
	for _, r := range runs {
		minLevel = min(minLevel, r.Level)
	}
	var result []*Run
	start := 0
	flush := func(end int) {
		if end <= start {
			return
		}
		inner := reorderRecurse(runs[start:end])
		if minLevel%2 == 1 {
			result = append(inner, result...)
		} else {
			result = append(result, inner...)
		}
	}
	for i, r := range runs {
		if r.Level != minLevel {
			continue
		}
		flush(i)
		if minLevel%2 == 1 {
			result = append([]*Run{r}, result...)
		} else {
			result = append(result, r)
		}
		start = i + 1
	}
	flush(len(runs))
	return result
}

// adjustLineLetterSpacing removes the letter-spacing on the outer sides of
// the first and last runs of a line and of the runs next to tabs, moving
// it to the tabs.
func adjustLineLetterSpacing(line *Line) {
	runs := line.Runs
	reversed := false
	if line.RTL {
		for _, r := range runs {
			if r.tab {
				reversed = true
				break
			}
		}
	}
	if reversed {
		runs = reversedRuns(runs)
	}
	var last *Run
	tabAdjustment := 0
	for i, run := range runs {
		var next *Run
		if i+1 < len(runs) {
			next = runs[i+1]
		}
		if run.tab {
			padRight(run.Glyphs, tabAdjustment)
			tabAdjustment = 0
			last = run
			continue
		}
		if len(run.Glyphs) == 0 {
			last = run
			continue
		}
		visualNext, visualLast := next, last
		if reversed {
			visualNext, visualLast = last, next
		}
		left, right := distributeSpacing(run.spacing)
		if run.Glyphs[0].XAdvance == 0 {
			padLeft(run.Glyphs, -left)
		} else if visualLast == nil || visualLast.tab {
			padLeft(run.Glyphs, -left)
			tabAdjustment += left
		}
		if run.Glyphs[len(run.Glyphs)-1].XAdvance == 0 {
			padRight(run.Glyphs, -right)
		} else if visualNext == nil || visualNext.tab {
			padRight(run.Glyphs, -right)
			tabAdjustment += right
		}
		last = run
	}
}

func reversedRuns(runs []*Run) []*Run {
	out := make([]*Run, len(runs))
	for i, r := range runs {
		out[len(runs)-1-i] = r
	}
	return out
}

func padRight(glyphs []Glyph, adjustment int) {
	i := len(glyphs) - 1
	for i >= 0 && glyphs[i].XAdvance == 0 {
		i--
	}
	if i < 0 {
		return
	}
	glyphs[i].XAdvance = int32(max(int(glyphs[i].XAdvance)+adjustment, 0))
}

func padLeft(glyphs []Glyph, adjustment int) {
	i := 0
	for i < len(glyphs) && glyphs[i].XAdvance == 0 {
		i++
	}
	if i == len(glyphs) {
		return
	}
	glyphs[i].XAdvance += int32(adjustment)
	glyphs[i].XOffset += int32(adjustment)
}
