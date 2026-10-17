package layout

import (
	"maps"
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
)

// ResumeAt identifies unfinished children, recursively, with UTF-8 byte
// offsets at text leaves. A nil continuation means the fragment is complete.
type ResumeAt map[int]ResumeAt

// Clone copies the continuation and all nested paths.
func (r ResumeAt) Clone() ResumeAt {
	if r == nil {
		return nil
	}
	cloned := make(ResumeAt, len(r))
	for index, child := range r {
		cloned[index] = child.Clone()
	}
	return cloned
}

// Equal reports whether the continuations resume the same content.
func (r ResumeAt) Equal(other ResumeAt) bool {
	if (r == nil) != (other == nil) || len(r) != len(other) {
		return false
	}
	for index, child := range r {
		value, found := other[index]
		if !found || !child.Equal(value) {
			return false
		}
	}
	return true
}

// PageBreak carries the requested side and name of the following page.
// A nil Page leaves the name unspecified; an empty name selects unnamed pages.
type PageBreak struct {
	Break string
	Page  *string
}

type pageGroup struct {
	Name     string
	Index    int
	ResumeAt ResumeAt
}

// PageState contains the counter scopes and named groups at a page boundary.
type PageState struct {
	Build  *boxes.State
	Groups []pageGroup
}

// Clone makes a page state independent of subsequent pagination attempts.
func (s PageState) Clone() PageState {
	cloned := PageState{Groups: make([]pageGroup, len(s.Groups))}
	if s.Build != nil {
		cloned.Build = s.Build.Clone()
	}
	for i, group := range s.Groups {
		cloned.Groups[i] = group
		cloned.Groups[i].ResumeAt = group.ResumeAt.Clone()
	}
	return cloned
}

// Equal compares counter values, scopes, quotes and named group continuations.
func (s PageState) Equal(other PageState) bool {
	if (s.Build == nil) != (other.Build == nil) || len(s.Groups) != len(other.Groups) {
		return false
	}
	if s.Build != nil {
		a, b := s.Build, other.Build
		if a.QuoteDepth != b.QuoteDepth || !maps.EqualFunc(a.CounterValues, b.CounterValues, slices.Equal) {
			return false
		}
		if !slices.EqualFunc(a.CounterScopes, b.CounterScopes, maps.Equal) {
			return false
		}
	}
	for i, a := range s.Groups {
		b := other.Groups[i]
		if a.Name != b.Name || a.Index != b.Index || !a.ResumeAt.Equal(b.ResumeAt) {
			return false
		}
	}
	return true
}

// RemakeState records which content can invalidate an already laid-out page.
type RemakeState struct {
	ContentChanged bool
	PagesWanted    bool
	Anchors        []string
	ContentLookups []*targets.CounterLookupItem
}

// PageMaker stores the starting state for one page. The last entry stores
// the state after the last page, used when generating its margin boxes.
type PageMaker struct {
	ResumeAt  ResumeAt
	NextPage  PageBreak
	RightPage bool
	State     PageState
	Remake    RemakeState
}
