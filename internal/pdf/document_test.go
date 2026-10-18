package pdf_test

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func newDocument(t *testing.T, source string) *pdf.Document {
	t.Helper()
	doc := boxtest.FakeHTML(t, source)
	return pdf.NewDocument(doc.RenderPages(t), doc.HTML.Metadata())
}

const threePages = `
	<title>Three pages</title>
	<style>@page { size: 100px } h1 { margin: 0; break-after: page }</style>
	<h1>a</h1><h1>b</h1><h1>c</h1>`

func TestCopyKeepsTheSelectedPagesAndMetadata(t *testing.T) {
	doc := newDocument(t, threePages)
	if len(doc.Pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(doc.Pages))
	}

	odd := doc.Copy([]*pdf.Page{doc.Pages[0], doc.Pages[2]})
	if len(odd.Pages) != 2 || odd.Pages[0] != doc.Pages[0] || odd.Pages[1] != doc.Pages[2] {
		t.Errorf("copy has pages %v", odd.Pages)
	}
	if odd.Metadata != doc.Metadata {
		t.Error("copy has other metadata")
	}
	if all := doc.Copy(nil); len(all.Pages) != 3 {
		t.Errorf("copy of no pages has %d pages, want all 3", len(all.Pages))
	}
	if len(doc.Pages) != 3 {
		t.Errorf("copy changed the original, which has %d pages", len(doc.Pages))
	}
}

func TestBookmarkTreeOfACopyOnlyHasItsPages(t *testing.T) {
	doc := newDocument(t, threePages)
	second := doc.Copy([]*pdf.Page{doc.Pages[1]})
	tree := second.BookmarkTree(1, false)
	if len(tree) != 1 || tree[0].Label != "b" || tree[0].Page != 0 {
		t.Errorf("tree = %+v, want only b on page 0", tree)
	}
}

func TestBookmarkTreeTransformsPositions(t *testing.T) {
	doc := newDocument(t, `
		<style>@page { size: 100px } h1 { margin: 0; height: 10px }</style>
		<h1>a</h1><h1 style="margin-top: 20px">b</h1>`)

	tests := []struct {
		name      string
		scale     float64
		transform bool
		wantY     [2]float64
	}{
		{"unscaled", 1, false, [2]float64{0, 30}},
		{"scaled", 2, false, [2]float64{0, 60}},
		{"flipped", 1, true, [2]float64{100, 70}},
		{"flipped and scaled", 0.5, true, [2]float64{50, 35}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := doc.BookmarkTree(tt.scale, tt.transform)
			if len(tree) != 2 {
				t.Fatalf("tree = %+v, want two nodes", tree)
			}
			got := [2]float64{tree[0].Y, tree[1].Y}
			if !reflect.DeepEqual(got, tt.wantY) {
				t.Errorf("y = %v, want %v", got, tt.wantY)
			}
		})
	}
}
