package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestBoxesImplementTargetInterfaces(t *testing.T) {
	var box boxes.Box = boxes.NewBlockBox("div", plainStyle(), nil, nil)
	if _, ok := box.(targets.Source); !ok {
		t.Error("boxes are not target sources")
	}
	if _, ok := box.(targets.Target); !ok {
		t.Error("boxes are not target boxes")
	}
	if _, ok := box.(targets.Linked); !ok {
		t.Error("boxes are not linked boxes")
	}
}

func TestSourceReadsTheAttributesOfTheElement(t *testing.T) {
	s := newStyled(t, `<a href="#x" title=""></a>`)
	b := boxes.NewInlineBox("a", s.style("a"), s.el("a"), nil)
	if v, ok := b.Attr("href"); !ok || v != "#x" {
		t.Errorf("href = %q, %v", v, ok)
	}
	if v, ok := b.Attr("title"); !ok || v != "" {
		t.Errorf("title = %q, %v", v, ok)
	}
	if _, ok := b.Attr("rel"); ok {
		t.Error("found a missing attribute")
	}
	if b.BaseURL() != csstest.BaseURL(t) {
		t.Errorf("BaseURL = %q", b.BaseURL())
	}
	if _, ok := boxes.NewBlockBox("", plainStyle(), nil, nil).Attr("x"); ok {
		t.Error("a box without element has attributes")
	}
}

func TestCollectorStoresCounterValuesOnTheTargetBox(t *testing.T) {
	c := targets.NewCollector()
	c.CollectAnchor("top")
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	c.StoreTarget("top", targets.CounterValues{"n": {1, 2}}, box)

	got := box.CachedCounterValues()
	if len(got["n"]) != 2 || got["n"][1] != 2 {
		t.Errorf("counter values = %v", got)
	}
	if c.Target("top").TargetBox != targets.Target(box) {
		t.Error("the collector does not hold the box")
	}

	other := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	other.SetCachedCounterValues(targets.CounterValues{"m": {7}})
	c.CollectAnchor("other")
	c.StoreTarget("other", targets.CounterValues{"n": {9}}, other)
	if _, ok := other.CachedCounterValues()["n"]; ok {
		t.Error("counter values replaced the ones the box already had")
	}
}

func TestMissingCountersLinkTheBoxToItself(t *testing.T) {
	c := targets.NewCollector()
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	if box.MissingLink() != nil {
		t.Fatal("a new box has a missing link")
	}
	c.CollectMissingCounters(box, "content", func(targets.CounterValues) {}, map[string]bool{"page": true}, nil)
	if got, ok := box.MissingLink().(boxes.Box); !ok || got != boxes.Box(box) {
		t.Errorf("MissingLink = %v", box.MissingLink())
	}
	if c.CounterItem(box, "content") == nil {
		t.Error("the box was not collected")
	}
	box.SetMissingLink(nil)
	if box.MissingLink() != nil || box.MissingLinkBox != nil {
		t.Error("clearing the link failed")
	}
}
