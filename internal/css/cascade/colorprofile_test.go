package cascade

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestColorProfileReadsDescriptionTags(t *testing.T) {
	for _, tt := range []struct {
		name, signature, tagType, text string
	}{
		{"device description", "dmdd", "desc", "Display P3"},
		{"description fallback", "desc", "desc", "sRGB"},
		{"unicode description", "dmdd", "mluc", "Gráficos"},
	} {
		data := colorProfileData(tt.signature, tt.tagType, tt.text)
		profile, err := newColorProfile(data, "profile.icc", "perceptual", [][]string{{"red", "green", "blue"}})
		if err != nil {
			t.Fatalf("newColorProfile(%s): %v", tt.name, err)
		}
		if profile.Name != tt.text || profile.Src != "profile.icc" || profile.RenderingIntent != "perceptual" || len(profile.Components) != 1 {
			t.Errorf("profile = %+v, want name %q and supplied metadata", profile, tt.text)
		}
	}
}

func TestColorProfileRejectsInvalidHeaders(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		make([]byte, 131),
		colorProfileWithHeader(132, 133, "acsp"),
		colorProfileWithHeader(132, 132, "bad!"),
	} {
		if _, err := newColorProfile(data, "", "", nil); err == nil {
			t.Errorf("newColorProfile accepted %d invalid bytes", len(data))
		}
	}
}

func colorProfileData(signature, tagType, text string) []byte {
	var tag []byte
	switch tagType {
	case "desc":
		tag = make([]byte, 12+len(text)+1)
		copy(tag[:4], "desc")
		binary.BigEndian.PutUint32(tag[8:12], uint32(len(text)+1))
		copy(tag[12:], text)
	case "mluc":
		encoded := utf16.Encode([]rune(text))
		tag = make([]byte, 28+2*len(encoded))
		copy(tag[:4], "mluc")
		binary.BigEndian.PutUint32(tag[20:24], uint32(2*len(encoded)))
		binary.BigEndian.PutUint32(tag[24:28], 28)
		for i, unit := range encoded {
			binary.BigEndian.PutUint16(tag[28+2*i:], unit)
		}
	}
	data := colorProfileWithHeader(144+len(tag), 144+len(tag), "acsp")
	binary.BigEndian.PutUint32(data[0:4], uint32(len(data)))
	binary.BigEndian.PutUint32(data[128:132], 1)
	copy(data[132:136], signature)
	binary.BigEndian.PutUint32(data[136:140], 144)
	binary.BigEndian.PutUint32(data[140:144], uint32(len(tag)))
	copy(data[144:], tag)
	return data
}

func colorProfileWithHeader(size, length int, signature string) []byte {
	data := make([]byte, size)
	if len(data) >= 40 {
		copy(data[36:40], signature)
	}
	binary.BigEndian.PutUint32(data[0:4], uint32(length))
	return data
}
