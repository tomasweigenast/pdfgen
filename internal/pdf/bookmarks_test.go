package pdf_test

import (
	"bytes"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

var (
	titleRE = regexp.MustCompile(`/Title \((.*)\)`)
	countRE = regexp.MustCompile(`/Count ([0-9-]*)`)
)

func matches(re *regexp.Regexp, out []byte) []string {
	var got []string
	for _, m := range re.FindAllSubmatch(out, -1) {
		got = append(got, string(m[1]))
	}
	return got
}

func numbered(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = strconv.Itoa(i + 1)
	}
	return out
}

// outlineCounts returns the /Count of every outline item, in the order of the
// objects, and the /Count of the outline root.
func outlineCounts(out []byte) (items []string, root string) {
	counts := matches(countRE, out)
	// The first count is the one of the page tree.
	counts = counts[1:]
	return counts[:len(counts)-1], counts[len(counts)-1]
}

func TestOutlineCounts(t *testing.T) {
	tests := []struct {
		name, html string
		titles     []string
		items      []string
		root       string
	}{
		{
			name: "skipped levels",
			html: `<h1>a</h1><h4>b</h4><h3>c</h3><h2>d</h2><h1>e</h1>`,
			// a
			// |_ b
			// |_ c
			// L_ d
			// e
			titles: []string{"a", "b", "c", "d", "e"},
			items:  []string{"3", "0", "0", "0", "0"},
			root:   "5",
		},
		{
			name: "levels over several pages",
			html: `
				<style>
					h1, h2, h3, span { height: 90pt; margin: 0 0 10pt 0 }
				</style>
				<h1>1</h1>
				<h1>2</h1>
				<h2 style="position: relative; left: 20pt">3</h2>
				<h2>4</h2>
				<h3>5</h3>
				<span style="display: block; page-break-before: always"></span>
				<h2>6</h2>
				<h1>7</h1>
				<h2>8</h2>
				<h3>9</h3>
				<h1>10</h1>
				<h2>11</h2>`,
			// 1
			// 2
			// |_ 3
			// |_ 4
			// |  L_ 5
			// L_ 6
			// 7
			// L_ 8
			//    L_ 9
			// 10
			// L_ 11
			titles: numbered(11),
			items:  []string{"0", "4", "0", "1", "0", "0", "2", "1", "0", "1", "0"},
			root:   "11",
		},
		{
			name: "first level is h2",
			html: `<h2>1</h2><h4>2</h4><h2>3</h2><h3>4</h3><h4>5</h4>`,
			// 1
			// L_ 2
			// 3
			// L_ 4
			//    L_ 5
			titles: numbered(5),
			items:  []string{"1", "0", "2", "1", "0"},
			root:   "5",
		},
		{
			name: "skips are restored",
			html: `<h2>1</h2><h4>2</h4><h3>3</h3><h5>4</h5><h1>5</h1><h2>6</h2><h2>7</h2><h4>8</h4><h1>9</h1>`,
			// 1
			// |_ 2
			// L_ 3
			//    L_ 4
			// 5
			// |_ 6
			// L_ 7
			//    L_ 8
			// 9
			titles: numbered(9),
			items:  []string{"3", "0", "1", "0", "3", "0", "1", "0", "0"},
			root:   "9",
		},
		{
			name: "closed item",
			html: `<h1>a</h1><h2>b</h2><h3>c</h3><h2 style="bookmark-state: closed">d</h2><h3>e</h3><h4>f</h4><h1>g</h1>`,
			// a
			// |_ b
			// |  |_ c
			// |_ d (closed)
			// |  |_ e
			// |     |_ f
			// g
			titles: []string{"a", "b", "c", "d", "e", "f", "g"},
			items:  []string{"3", "1", "0", "-2", "1", "0", "0"},
			root:   "5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := writePDF(t, tt.html, pdf.Options{})
			if got := matches(titleRE, out); !slices.Equal(got, tt.titles) {
				t.Errorf("titles = %q, want %q", got, tt.titles)
			}
			items, root := outlineCounts(out)
			if !slices.Equal(items, tt.items) {
				t.Errorf("item counts = %q, want %q", items, tt.items)
			}
			if root != tt.root {
				t.Errorf("root count = %s, want %s", root, tt.root)
			}
		})
	}
}

func TestNoOutlinesWithoutBookmarks(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<body>`, pdf.Options{})
	if bytes.Contains(out, []byte("Outlines")) {
		t.Error("PDF has /Outlines")
	}
}

func TestOutlineTitleIsUTF16WhenNotASCII(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, "<h1>a\u00a0nbsp…</h1>", pdf.Options{})
	got := matches(regexp.MustCompile(`/Title <(\w*)>`), out)
	if want := []string{"feff006100a0006e0062007300702026"}; !slices.Equal(got, want) {
		t.Errorf("titles = %q, want %q", got, want)
	}
}

func TestOutlineDestinationScalesWithZoom(t *testing.T) {
	destRE := regexp.MustCompile(`/Dest \[(.*)\]`)
	destY := func(zoom float64) float64 {
		t.Helper()
		out := writePDF(t, `<h2>a</h2>`, pdf.Options{Zoom: zoom})
		if got := matches(titleRE, out); !slices.Equal(got, []string{"a"}) {
			t.Fatalf("titles = %q", got)
		}
		dests := matches(destRE, out)
		if len(dests) != 1 {
			t.Fatalf("got %d destinations, want 1", len(dests))
		}
		fields := strings.Fields(dests[0])
		y, err := strconv.ParseFloat(fields[len(fields)-2], 64)
		if err != nil {
			t.Fatalf("parse %q: %v", dests[0], err)
		}
		return math.Round(y)
	}
	testutil.AssertNoLogs(t)
	y := destY(1)
	if got := destY(1.5); got != 1.5*y {
		t.Errorf("y at zoom 1.5 = %v, want %v", got, 1.5*y)
	}
}

func TestOutlineLabels(t *testing.T) {
	tests := []struct {
		name, html string
		titles     []string
	}{
		{"counter in label", `<h1 style="bookmark-label: 'h1 on page ' counter(page)">a</h1>`, []string{"h1 on page 1"}},
		{
			"generated content",
			`<style>
				div:before, div:after {
					content: '';
					bookmark-level: 1;
					bookmark-label: 'x';
				}
			</style>
			<div>a</div>`,
			[]string{"x", "x"},
		},
		{
			"inline box split in lines",
			`<div style="display:inline; white-space:pre;
				bookmark-level:1; bookmark-label:'a'">
			a
			a
			a
			</div>
			<div style="bookmark-level:1; bookmark-label:'b'">
				<div>b</div>
				<div style="break-before:always">c</div>
			</div>`,
			[]string{"a", "b"},
		},
		{"contents", `<div style="bookmark-level:1; bookmark-label:contents">a</div>`, []string{"a"}},
		{
			"contents ignore text-transform",
			`<div style="bookmark-level:1; bookmark-label:contents; text-transform:uppercase">a</div>`,
			[]string{"a"},
		},
		{
			"white space is collapsed",
			`<h1>a</h1>
			<h1> b c d </h1>
			<h1> e
				 f </h1>
			<h1> g <span> h </span> i </h1>`,
			[]string{"a", "b c d", "e f", "g h i"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := writePDF(t, tt.html, pdf.Options{})
			counts := matches(countRE, out)
			if got := counts[len(counts)-1]; got != strconv.Itoa(len(tt.titles)) {
				t.Errorf("root count = %s, want %d", got, len(tt.titles))
			}
			if got := matches(titleRE, out); !slices.Equal(got, tt.titles) {
				t.Errorf("titles = %q, want %q", got, tt.titles)
			}
		})
	}
}

func TestOutlineDestinationOnSmallPage(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<style>@page { size: 10pt 10pt }</style>
		<h1>a</h1>`, pdf.Options{})
	counts := matches(countRE, out)
	if got := counts[len(counts)-1]; got != "1" {
		t.Errorf("root count = %s, want 1", got)
	}
	if got := matches(titleRE, out); !slices.Equal(got, []string{"a"}) {
		t.Errorf("titles = %q, want [a]", got)
	}
	if !bytes.Contains(out, []byte("/XYZ 0 10 0")) {
		t.Error("PDF has no /XYZ 0 10 0")
	}
}
