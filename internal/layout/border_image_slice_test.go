package layout_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func TestUnresolvableMathInBorderSlicesIsIgnoredAndTheDocumentLaysOut(t *testing.T) {
	tests := []struct{ property, value string }{
		{"border-image-slice", "scale(calc(0))"},
		{"border-image-slice", "min(0px, 0)"},
		{"mask-border-slice", "scale(calc(0))"},
		{"mask-border-slice", "rgb(calc(0%) 0 0)"},
	}
	for _, tt := range tests {
		t.Run(tt.property+" "+tt.value, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			pages := boxtest.RenderPages(t, `<div style="`+tt.property+`: `+tt.value+`">a</div>`)
			_, body := pageBody(t, onlyPage(t, pages))
			if len(boxes.ChildrenOf(body)) == 0 {
				t.Error("the body has no children")
			}
			lines := logs.Lines()
			if len(lines) != 1 || !strings.Contains(lines[0], "invalid value") {
				t.Errorf("logs = %q, want one invalid value warning", lines)
			}
		})
	}
}
