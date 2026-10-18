package metadata

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func TestW3CDateToPDF(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"2011", "D:2011"},
		{"2011-04", "D:201104"},
		{"2011-04-21", "D:20110421"},
		{"2011-04-21T23:00Z", "D:20110421230000Z"},
		{"2011-04-21T23:00:05Z", "D:20110421230005Z"},
		{"2011-04-21T23:00:05.25Z", "D:20110421230005Z"},
		{"2013-07-21T23:46+01:00", "D:20130721234600+01'00"},
		{"2013-07-21T23:46:10-05:30", "D:20130721234610-05'30"},
		{" 2011-04-21 ", "D:20110421"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			got, ok := w3cDateToPDF(&tt.in, "created")
			if !ok || got != tt.want {
				t.Errorf("w3cDateToPDF(%q) = %q, %v; want %q", tt.in, got, ok, tt.want)
			}
		})
	}
}

func TestW3CDateToPDFRejectsInvalidDates(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	bad := "yesterday"
	if got, ok := w3cDateToPDF(&bad, "created"); ok {
		t.Errorf("w3cDateToPDF(%q) = %q, want failure", bad, got)
	}
	if got := logs.Warnings(); len(got) != 1 {
		t.Errorf("logged %q, want one warning", got)
	}
	if _, ok := w3cDateToPDF(nil, "created"); ok {
		t.Error("a missing date is not a date")
	}
}
