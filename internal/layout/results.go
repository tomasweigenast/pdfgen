package layout

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
)

// BlockResult describes a laid-out fragment and the content following it.
// A nil Box means that the fragment did not fit in the available space.
type BlockResult struct {
	Box               boxes.Box
	ResumeAt          ResumeAt
	NextPage          PageBreak
	AdjoiningMargins  *marginList
	CollapsingThrough bool
	MaxLines          *int
}

type boundaryKind uint8

const (
	absentBoundary boundaryKind = iota
	characterBoundary
	leaderStartBoundary
	leaderEndBoundary
)

type boundary struct {
	Kind boundaryKind
	Rune rune
}

type floatWidths struct{ Left, Right float64 }

// InlineResult describes an inline fragment and its possible break boundaries.
type InlineResult struct {
	Box                     boxes.Box
	ResumeAt                ResumeAt
	PreservedLineBreak      bool
	FirstLetter, LastLetter boundary
	FloatWidths             floatWidths
}

type textResult struct {
	Box                *boxes.TextBox
	ResumeIndex        *int
	PreservedLineBreak bool
}

type lineResult struct {
	Line     *boxes.LineBox
	ResumeAt ResumeAt
}

type fragmentResult struct {
	Box      boxes.Box
	ResumeAt ResumeAt
}

type positionedBoxes struct {
	Absolute, Fixed []*boxes.AbsolutePlaceholder
}

type blockOptions struct {
	BottomSpace                      float64
	SkipStack                        ResumeAt
	PageIsEmpty                      bool
	Positioned                       *positionedBoxes
	AdjoiningMargins                 *marginList
	FirstLetterStyle, FirstLineStyle *computed.Style
	Discard                          bool
	MaxLines                         *int
}

type inlineOptions struct {
	PositionX, MaxX, BottomSpace     float64
	SkipStack                        ResumeAt
	Positioned                       *positionedBoxes
	LinePlaceholders                 *[]*boxes.AbsolutePlaceholder
	WaitingFloats, LineChildren      *[]boxes.Box
	FirstLetterStyle, FirstLineStyle *computed.Style
}
