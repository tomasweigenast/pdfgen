package validation

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func FuzzValidateValue(f *testing.F) {
	for _, seed := range [][2]string{
		{"color", "red"}, {"margin", "1px 2px"}, {"width", "calc(1em + 2px)"},
		{"font", "12px serif"}, {"unknown-property", "var(--missing)"},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, name, value string) {
		if len(name) > 1024 || len(value) > 16<<10 {
			t.Skip()
		}
		Validate(syntax.ParseComponentValueList(value, true), name, "")
	})
}
