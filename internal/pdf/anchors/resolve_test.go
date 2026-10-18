package anchors_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/anchors"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

func TestResolveLinksKeepsTheFirstAnchorOfAName(t *testing.T) {
	pages := []*anchors.Data{
		{Anchors: map[string][4]float64{"a": {1, 2, 3, 4}, "b": {5, 6, 7, 8}}},
		{Anchors: map[string][4]float64{"a": {9, 9, 9, 9}, "c": {0, 1, 2, 3}}},
	}
	got := anchors.ResolveLinks(pages)
	want := [][]anchors.Destination{
		{{Name: "a", X: 1, Y: 2}, {Name: "b", X: 5, Y: 6}},
		{{Name: "c", X: 0, Y: 1}},
	}
	for i := range want {
		if len(got[i].Destinations) != len(want[i]) {
			t.Fatalf("page %d: destinations = %v, want %v", i, got[i].Destinations, want[i])
		}
		for j, d := range want[i] {
			if got[i].Destinations[j] != d {
				t.Errorf("page %d: destination %d = %v, want %v", i, j, got[i].Destinations[j], d)
			}
		}
	}
}

func TestResolveLinksDropsLinksToMissingAnchors(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	pages := []*anchors.Data{{
		Anchors: map[string][4]float64{"a": {}},
		Links: []anchors.Link{
			{Kind: urls.LinkInternal, Target: "a"},
			{Kind: urls.LinkInternal, Target: "missing"},
			{Kind: urls.LinkExternal, Target: "missing"},
		},
	}}
	got := anchors.ResolveLinks(pages)[0].Links
	if len(got) != 2 || got[0].Target != "a" || got[1].Kind != urls.LinkExternal {
		t.Errorf("links = %+v", got)
	}
	lines := logs.Lines()
	if len(lines) != 1 || lines[0] != "ERROR: No anchor #missing for internal URI reference" {
		t.Errorf("logs = %q", lines)
	}
}
