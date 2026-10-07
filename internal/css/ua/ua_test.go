package ua

import (
	"strings"
	"testing"
)

func TestEmbeddedSheetsAreNotEmpty(t *testing.T) {
	tests := []struct {
		name  string
		sheet string
		want  string
	}{
		{"HTML", HTML, "display"},
		{"Form", Form, "input"},
		{"PresentationalHints", PresentationalHints, "align"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.sheet, tt.want) {
				t.Errorf("sheet does not contain %q (len %d)", tt.want, len(tt.sheet))
			}
		})
	}
}
