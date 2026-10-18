package fonts

import (
	"encoding/binary"
	"os"
	"testing"
)

func FuzzNewFace(f *testing.F) {
	for _, name := range []string{"testfont.otf", "testfont.woff", "testfont.otb"} {
		data, err := os.ReadFile("../../../testdata/resources/" + name)
		if err != nil {
			f.Fatalf("read font seed %q: %v", name, err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 2<<20 {
			t.Skip()
		}
		if len(data) >= 20 && string(data[:3]) == "wOF" &&
			int64(binary.BigEndian.Uint32(data[16:20])) > 32<<20 {
			t.Skip()
		}
		NewFace(data, 0)
	})
}
