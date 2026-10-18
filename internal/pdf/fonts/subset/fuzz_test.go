package subset

import (
	"os"
	"testing"
)

func FuzzParseAndSubset(f *testing.F) {
	for _, name := range []string{"testfont.otf", "testfont.otb"} {
		data, err := os.ReadFile("../../../../testdata/resources/" + name)
		if err != nil {
			f.Fatalf("read font seed %q: %v", name, err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 2<<20 {
			t.Skip()
		}
		font, err := Parse(data, 0)
		if err != nil {
			return
		}
		font.Subset([]uint16{0}, Options{})
	})
}
