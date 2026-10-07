package targets

import (
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

type box struct {
	attrs   map[string]string
	baseURL string
	values  CounterValues
	link    any
}

func (b *box) Attr(name string) (string, bool) { v, ok := b.attrs[name]; return v, ok }
func (b *box) BaseURL() string                 { return b.baseURL }
func (b *box) CachedCounterValues() CounterValues {
	return b.values
}
func (b *box) SetCachedCounterValues(v CounterValues) { b.values = v }
func (b *box) MissingLink() any                       { return b.link }
func (b *box) SetMissingLink(v any)                   { b.link = v }

func str(s string) props.ContentItem { return props.ContentItem{Type: "string", Args: []string{s}} }

func TestAnchorDefinedTwiceIsReported(t *testing.T) {
	c := NewCollector()
	logs := logging.StartCapture()
	defer logs.Close()
	c.CollectAnchor("a")
	c.CollectAnchor("a")
	want := []string{"WARNING: Anchor defined twice: 'a'"}
	if got := logs.Lines(); !slices.Equal(got, want) {
		t.Errorf("logs = %q, want %q", got, want)
	}
	if item := c.Target("a"); item == nil || item.State != Pending {
		t.Errorf("Target(a) = %+v, want pending item", item)
	}
}

func TestAnchorName(t *testing.T) {
	tests := []struct {
		link props.ContentItem
		want string
		ok   bool
	}{
		{str("#id"), "id", true},
		{str("id"), "", false},
		{props.ContentItem{Type: "url", Args: []string{"internal", "id"}}, "id", true},
		{props.ContentItem{Type: "url", Args: []string{"external", "http://x/#id"}}, "", false},
	}
	for _, tt := range tests {
		got, ok := AnchorName(tt.link)
		if got != tt.want || ok != tt.ok {
			t.Errorf("AnchorName(%v) = %q, %v; want %q, %v", tt.link, got, ok, tt.want, tt.ok)
		}
	}
}

func TestPendingTargetsAreParsedAgain(t *testing.T) {
	c := NewCollector()
	c.CollectAnchor("id4")
	source := &box{attrs: map[string]string{"data-count": "#id4"}}

	calls := 0
	item := c.LookupTarget(props.ContentItem{Type: "attr()", Args: []string{"data-count", "string", ""}}, source, "content", func(CounterValues) { calls++ })
	if item.State != Pending || !c.HadPendingTargets {
		t.Fatalf("item state = %v, pending targets = %v", item.State, c.HadPendingTargets)
	}
	// The same box and property only register once.
	c.LookupTarget(str("#id4"), source, "content", func(CounterValues) { calls++ })

	target := &box{}
	c.StoreTarget("id4", CounterValues{"div": {4}}, target)
	if item.State != UpToDate || item.TargetBox != target {
		t.Errorf("stored item = %+v", item)
	}
	if !slices.Equal(target.values["div"], []int{4}) {
		t.Errorf("target counter values = %v", target.values)
	}

	c.CheckPendingTargets()
	if calls != 1 {
		t.Errorf("parse again called %d times, want 1", calls)
	}
	if c.Collecting || c.HadPendingTargets {
		t.Error("collector still collecting after CheckPendingTargets")
	}
}

func TestUndefinedTargetIsReported(t *testing.T) {
	c := NewCollector()
	logs := logging.StartCapture()
	defer logs.Close()
	item := c.LookupTarget(str("#missing"), &box{}, "content", nil)
	if item.State != Undefined {
		t.Errorf("state = %v, want undefined", item.State)
	}
	want := []string{"ERROR: Content discarded: target points to undefined anchor ('string', '#missing')"}
	if got := logs.Lines(); !slices.Equal(got, want) {
		t.Errorf("logs = %q, want %q", got, want)
	}
}

func TestMissingCountersAreCollectedOnlyWhileBuilding(t *testing.T) {
	c := NewCollector()
	parent := &box{}
	c.CollectMissingCounters(parent, "content", nil, nil, nil)
	if c.CounterItem(parent, "content") != nil {
		t.Error("empty missing counters were collected")
	}
	c.CollectMissingCounters(parent, "content", nil, map[string]bool{"pages": true}, nil)
	if c.CounterItem(parent, "content") == nil || parent.link != parent {
		t.Error("missing counters not collected")
	}
	c.CheckPendingTargets()
	other := &box{}
	c.CollectMissingCounters(other, "content", nil, map[string]bool{"page": true}, nil)
	if c.CounterItem(other, "content") != nil {
		t.Error("missing counters collected during pagination")
	}
}

func TestTargetPageCountersUpdateDependentBoxes(t *testing.T) {
	c := NewCollector()
	c.CollectAnchor("a")
	c.StoreTarget("a", nil, &box{})

	waiting, later := &box{}, &box{}
	var parsed []CounterValues
	c.CollectMissingCounters(waiting, "content", func(v CounterValues) { parsed = append(parsed, v) },
		nil, map[string]map[string]bool{"a": {"page": true}})
	c.CollectMissingCounters(later, "content", nil,
		nil, map[string]map[string]bool{"a": {"page": true}})
	c.CheckPendingTargets()

	item := c.CounterItem(waiting, "content")
	item.PageMakerIndex, item.Paginated = 0, true

	var changed []int
	c.CacheTargetPageCounters("a", CounterValues{"page": {2}}, 1, 2, func(i int) { changed = append(changed, i) })
	if !slices.Equal(changed, []int{0}) || len(parsed) != 1 {
		t.Errorf("changed pages = %v, parsed = %d", changed, len(parsed))
	}
	if !c.CounterItem(later, "content").Pending {
		t.Error("item without page is not marked pending")
	}
	target := c.Target("a")
	if !target.Paginated || target.PageMakerIndex != 1 {
		t.Errorf("target page = %d, %v", target.PageMakerIndex, target.Paginated)
	}

	// Unchanged counters do not trigger anything.
	c.CacheTargetPageCounters("a", CounterValues{"page": {2}}, 1, 2, func(i int) { changed = append(changed, i) })
	if len(changed) != 1 {
		t.Errorf("unchanged counters marked pages %v", changed)
	}
}
