package targets

import (
	"maps"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// State tells whether the box of an anchor is known.
type State uint8

const (
	// Pending is an anchor that exists in the document but whose box has
	// not been built yet.
	Pending State = iota
	// UpToDate is an anchor whose box has been built.
	UpToDate
	// Undefined is an anchor that no element defines.
	Undefined
)

// CounterValues maps counter names to the values of their nested
// instances, outermost first.
type CounterValues map[string][]int

// Clone returns a deep copy of the counter values.
func (c CounterValues) Clone() CounterValues {
	if c == nil {
		return nil
	}
	out := make(CounterValues, len(c))
	for k, v := range c {
		out[k] = append([]int(nil), v...)
	}
	return out
}

// ParseAgain rebuilds the content of a box once a target or a page based
// counter is known. pageCounters holds the page counters to mix in, and is
// nil when the content is rebuilt before pagination.
type ParseAgain func(pageCounters CounterValues)

// Source is the box whose content refers to a target. It gives the
// attributes of its element, for attr() links, and the base URL of its
// style.
type Source interface {
	Attr(name string) (string, bool)
	BaseURL() string
}

// Target is the box of an anchor. It keeps the counter values in effect at
// the anchor, which target-counter() reads.
type Target interface {
	CachedCounterValues() CounterValues
	SetCachedCounterValues(CounterValues)
}

// Linked is a box whose content uses counters that are only known during
// pagination. The missing link is the box itself, and connects the boxes
// made from it on each page to the original one.
type Linked interface {
	MissingLink() any
	SetMissingLink(any)
}

type sourceKey struct {
	box   any
	token string
}

// TargetLookupItem follows the box of one anchor and the boxes whose
// content is waiting for it.
type TargetLookupItem struct {
	State State
	// TargetBox is the box of the anchor, used by target-counter(),
	// target-counters() and target-text().
	TargetBox Target
	// PageMakerIndex is the position of the anchor during pagination, the
	// page number minus one, valid when Paginated is set.
	PageMakerIndex int
	Paginated      bool
	// CachedPageCounterValues are the page counters of the anchor during
	// pagination.
	CachedPageCounterValues CounterValues

	parseAgainKeys []sourceKey
	parseAgain     map[sourceKey]ParseAgain
}

func newTargetLookupItem(state State) *TargetLookupItem {
	return &TargetLookupItem{
		State:                   state,
		CachedPageCounterValues: CounterValues{},
		parseAgain:              map[sourceKey]ParseAgain{},
	}
}

func (item *TargetLookupItem) addParseAgain(key sourceKey, fn ParseAgain) {
	if _, ok := item.parseAgain[key]; ok {
		return
	}
	item.parseAgainKeys = append(item.parseAgainKeys, key)
	item.parseAgain[key] = fn
}

// CounterLookupItem follows a box whose content uses page based counters
// or target counters that are missing until pagination.
type CounterLookupItem struct {
	ParseAgain ParseAgain
	// MissingCounters are the names of the counters not known when the box
	// was built.
	MissingCounters map[string]bool
	// MissingTargetCounters are, for each anchor, the counters of the
	// target that were missing.
	MissingTargetCounters map[string]map[string]bool
	// PageMakerIndex is the position of the box during pagination, the page
	// number minus one, valid when Paginated is set.
	PageMakerIndex int
	Paginated      bool
	// Pending marks the item for a new layout of its page.
	Pending bool
	// CachedPageCounterValues are the page counters of the box during
	// pagination.
	CachedPageCounterValues CounterValues
}

// Collector gathers the anchors of a document and the boxes whose content
// depends on them, so that target-counter(), target-counters() and
// target-text() can be resolved once the anchors are built and paginated.
type Collector struct {
	// Collecting is true while the box tree is built. Missing page counters
	// are then collected; during pagination they are mixed in instead.
	Collecting bool
	// HadPendingTargets is set when a target was needed before its anchor
	// was built.
	HadPendingTargets bool

	targetKeys  []string
	targets     map[string]*TargetLookupItem
	counterKeys []sourceKey
	counters    map[sourceKey]*CounterLookupItem
}

// NewCollector returns an empty collector in collecting state.
func NewCollector() *Collector {
	return &Collector{
		Collecting: true,
		targets:    map[string]*TargetLookupItem{},
		counters:   map[sourceKey]*CounterLookupItem{},
	}
}

// Target returns the lookup item of an anchor, or nil.
func (c *Collector) Target(anchor string) *TargetLookupItem { return c.targets[anchor] }

// CounterItem returns the lookup item of a box and the property its content
// comes from, or nil.
func (c *Collector) CounterItem(box any, property string) *CounterLookupItem {
	return c.counters[sourceKey{box, property}]
}

// CollectAnchor records an anchor of the document. An anchor defined twice
// keeps its first element, with a warning.
func (c *Collector) CollectAnchor(name string) {
	if _, ok := c.targets[name]; ok {
		logging.Warnf("Anchor defined twice: %s", logging.Repr(name))
		return
	}
	c.targetKeys = append(c.targetKeys, name)
	c.targets[name] = newTargetLookupItem(Pending)
}

// AnchorName returns the anchor a target link points to: a string starting
// with "#" or an internal url().
func AnchorName(link props.ContentItem) (string, bool) {
	switch link.Type {
	case "string":
		if len(link.Args) > 0 && strings.HasPrefix(link.Args[0], "#") {
			return link.Args[0][1:], true
		}
	case "url":
		if len(link.Args) == 2 && link.Args[0] == "internal" {
			return link.Args[1], true
		}
	}
	return "", false
}

// resolveAttr replaces an attr() link by the value of the attribute of the
// source element, as a string or a url.
func resolveAttr(link props.ContentItem, source Source) props.ContentItem {
	if link.Type != "attr()" || len(link.Args) != 3 {
		return link
	}
	name, kind, fallback := link.Args[0], link.Args[1], link.Args[2]
	value, ok := source.Attr(name)
	if !ok {
		value = fallback
	}
	switch kind {
	case "string":
		return props.ContentItem{Type: "string", Args: []string{value}}
	case "url":
		l, ok := urls.GetURLTuple(value, source.BaseURL())
		if !ok {
			return props.ContentItem{Type: "url", Args: []string{"external", ""}}
		}
		k := "external"
		if l.Kind == urls.LinkInternal {
			k = "internal"
		}
		return props.ContentItem{Type: "url", Args: []string{k, l.Target}}
	}
	return link
}

// LookupTarget returns the lookup item of the anchor a target link points
// to. A pending anchor records parseAgain, called once the anchor is built;
// an undefined one is reported and gets a fresh undefined item.
func (c *Collector) LookupTarget(link props.ContentItem, source Source, property string, parseAgain ParseAgain) *TargetLookupItem {
	link = resolveAttr(link, source)
	name, _ := AnchorName(link)
	item, ok := c.targets[name]
	if !ok {
		item = newTargetLookupItem(Undefined)
	}
	switch item.State {
	case Pending:
		c.HadPendingTargets = true
		item.addParseAgain(sourceKey{source, property}, parseAgain)
	case Undefined:
		logging.Errorf("Content discarded: target points to undefined anchor %s", describeLink(link))
	}
	return item
}

// describeLink formats a link the way the warning names it.
func describeLink(link props.ContentItem) string {
	parts := make([]string, len(link.Args))
	for i, a := range link.Args {
		parts[i] = logging.Repr(a)
	}
	if link.Type == "url" {
		return "(" + logging.Repr(link.Type) + ", (" + strings.Join(parts, ", ") + "))"
	}
	return "(" + logging.Repr(link.Type) + ", " + strings.Join(parts, ", ") + ")"
}

// StoreTarget records the box of a collected anchor that is pending. The
// counter values at the anchor are copied to the box unless it already has
// some.
func (c *Collector) StoreTarget(name string, counterValues CounterValues, box Target) {
	item, ok := c.targets[name]
	if !ok || item.State != Pending {
		return
	}
	item.State = UpToDate
	item.TargetBox = box
	if box.CachedCounterValues() == nil {
		box.SetCachedCounterValues(counterValues.Clone())
	}
}

// CollectMissingCounters records a box whose content uses counters that are
// unknown while the box tree is built, probably page based ones. It does
// nothing during pagination or when nothing is missing.
func (c *Collector) CollectMissingCounters(parent Linked, property string, parseAgain ParseAgain, missing map[string]bool, missingTargets map[string]map[string]bool) {
	if !c.Collecting {
		return
	}
	if len(missing) == 0 && len(missingTargets) == 0 {
		return
	}
	if parent.MissingLink() == nil {
		parent.SetMissingLink(parent)
	}
	key := sourceKey{parent, property}
	if _, ok := c.counters[key]; ok {
		return
	}
	c.counterKeys = append(c.counterKeys, key)
	c.counters[key] = &CounterLookupItem{
		ParseAgain:              parseAgain,
		MissingCounters:         missing,
		MissingTargetCounters:   missingTargets,
		CachedPageCounterValues: CounterValues{},
	}
}

// CheckPendingTargets rebuilds the content waiting for targets, if a target
// was needed before its anchor, and ends the collecting state.
func (c *Collector) CheckPendingTargets() {
	if c.HadPendingTargets {
		for _, name := range c.targetKeys {
			item := c.targets[name]
			for _, key := range item.parseAgainKeys {
				item.parseAgain[key](nil)
			}
		}
		c.HadPendingTargets = false
	}
	c.Collecting = false
}

// CacheTargetPageCounters records the page and page counters of an anchor
// during pagination. When they change, the boxes whose content uses target
// counters of that anchor are rebuilt: their page is marked changed through
// markChanged, or, when it has not been made yet (index at or beyond
// pages), the item is marked pending.
func (c *Collector) CacheTargetPageCounters(name string, pageCounters CounterValues, pageIndex, pages int, markChanged func(pageIndex int)) {
	if c.Collecting {
		return
	}
	item, ok := c.targets[name]
	if !ok || item.State != UpToDate {
		return
	}
	item.PageMakerIndex, item.Paginated = pageIndex, true
	if equalCounters(item.CachedPageCounterValues, pageCounters) {
		return
	}
	item.CachedPageCounterValues = pageCounters.Clone()

	for _, key := range c.counterKeys {
		if key.token != "content" {
			continue
		}
		counterItem := c.counters[key]
		missing, ok := counterItem.MissingTargetCounters[name]
		if !ok {
			continue
		}
		if !counterItem.Paginated || counterItem.PageMakerIndex >= pages {
			counterItem.Pending = true
			continue
		}
		for counter := range missing {
			if _, ok := pageCounters[counter]; ok {
				markChanged(counterItem.PageMakerIndex)
				counterItem.ParseAgain(counterItem.CachedPageCounterValues)
				break
			}
		}
	}
}

func equalCounters(a, b CounterValues) bool {
	return maps.EqualFunc(a, b, func(x, y []int) bool {
		if len(x) != len(y) {
			return false
		}
		for i := range x {
			if x[i] != y[i] {
				return false
			}
		}
		return true
	})
}
