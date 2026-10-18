package draw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

const opacitySource = `
	<style>
		@page { size: 60px 60px }
		div { background: #000; width: 20px; height: 20px }
	</style>
	%s`

func opacityDocuments(bodies ...string) []string {
	docs := make([]string, len(bodies))
	for i, body := range bodies {
		docs[i] = fmt.Sprintf(opacitySource, body)
	}
	return docs
}

func TestOpacityZeroPaintsNothing(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div></div>`,
		`<div></div><div style="opacity: 0"></div>`,
		`<div></div><div style="opacity: 0%"></div>`,
	)...)
}

func TestOpacityBlendsWithBackdrop(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div style="background: rgb(102, 102, 102)"></div>`,
		`<div style="opacity: 0.6"></div>`,
		`<div style="opacity: 60%"></div>`,
		`<div style="opacity: 60.0%"></div>`,
	)...)
}

func TestNestedOpacityMultiplies(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div style="background: rgb(102, 102, 102)"></div>`,
		`<div style="opacity: 0.6"></div>`,
		`<div style="background: none; opacity: 0.666666">
			<div style="opacity: 0.9"></div>
		</div>`,
	)...)
}

func TestOpacityIsClampedToUnitRange(t *testing.T) {
	testutil.AssertNoLogs(t)
	t.Run("above one", func(t *testing.T) {
		testutil.AssertSameRenderings(t, 0, opacityDocuments(
			`<div></div>`,
			`<div style="opacity: 1.2"></div>`,
			`<div style="opacity: 120%"></div>`,
		)...)
	})
	t.Run("below zero", func(t *testing.T) {
		testutil.AssertSameRenderings(t, 0, opacityDocuments(
			`<div></div>`,
			`<div></div><div style="opacity: -0.2"></div>`,
			`<div></div><div style="opacity: -20%"></div>`,
		)...)
	})
}

func TestOpacityKeepsTextColorAlphaEquivalent(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`a<span style="color: rgb(0, 0, 0, 0.5)">b</span>`,
		`a<span style="opacity: 0.5">b</span>`,
	)...)
}

func TestOpacityIsPaintedAsGroup(t *testing.T) {
	pdf := string(rendertest.PDF(t, fmt.Sprintf(opacitySource, `<div style="opacity: 0.5"></div>`)))
	for _, want := range []string{"/ca 0.5", "/CA 0.5", " Do"} {
		if !strings.Contains(pdf, want) {
			t.Errorf("PDF does not contain %q", want)
		}
	}
	opaque := string(rendertest.PDF(t, fmt.Sprintf(opacitySource, `<div></div>`)))
	if strings.Contains(opaque, "/ca 0.5") {
		t.Error("an opaque page sets the alpha of an opacity group")
	}
}
