package draw_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestNthPageSelectsPages(t *testing.T) {
	tests := []struct {
		rule, pixels string
	}{
		{"2n", "_R_R_R_R_R"},
		{"even", "_R_R_R_R_R"},
		{"2n+1", "R_R_R_R_R_"},
		{"odd", "R_R_R_R_R_"},
		{"2n+3", "__R_R_R_R_"},
		{"n", "RRRRRRRRRR"},
		{"n-1", "RRRRRRRRRR"},
		{"-n+3", "RRR_______"},
		{"-2n+3", "R_R_______"},
		{"-n-3", "__________"},
		{"3", "__R_______"},
		{"0n+0", "__________"},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, strings.Join(strings.Split(tt.pixels, ""), "\n"), `
				<style>
					@page { size: 1px 1px }
					@page:nth(`+tt.rule+`) { background: red }
					p { break-after: page }
				</style>`+strings.Repeat("<p></p>", 10))
		})
	}
}
