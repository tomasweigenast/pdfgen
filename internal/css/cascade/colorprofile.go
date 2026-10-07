package cascade

import (
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf16"
)

// ColorProfile is an ICC profile declared by @color-profile (CSS Color 5
// §11).
type ColorProfile struct {
	// Src is the URL the profile was loaded from.
	Src string
	// RenderingIntent is the rendering-intent descriptor.
	RenderingIntent string
	// Components are the names of the components, from the components
	// descriptor.
	Components [][]string
	// Content is the ICC profile.
	Content []byte
	// Name is the model or description of the profile.
	Name string
}

var errInvalidProfile = errors.New("invalid ICC profile")

// newColorProfile checks the ICC header (ICC.1:2010 §7.2) and reads the
// name of a profile.
func newColorProfile(data []byte, src, intent string, components [][]string) (*ColorProfile, error) {
	if len(data) < 132 || string(data[36:40]) != "acsp" {
		return nil, errInvalidProfile
	}
	size := binary.BigEndian.Uint32(data[0:4])
	if int(size) > len(data) {
		return nil, errInvalidProfile
	}
	p := &ColorProfile{Src: src, RenderingIntent: intent, Components: components, Content: data}
	p.Name = iccText(data, "dmdd")
	if p.Name == "" {
		p.Name = iccText(data, "desc")
	}
	return p, nil
}

// iccText returns the text of a tag of type desc or mluc (ICC.1:2010
// §10.13, §10.15), or "" when the profile has no such tag.
func iccText(data []byte, signature string) string {
	count := int(binary.BigEndian.Uint32(data[128:132]))
	for i := 0; i < count; i++ {
		entry := 132 + 12*i
		if entry+12 > len(data) {
			return ""
		}
		if string(data[entry:entry+4]) != signature {
			continue
		}
		offset := int(binary.BigEndian.Uint32(data[entry+4:]))
		length := int(binary.BigEndian.Uint32(data[entry+8:]))
		if offset < 0 || length < 12 || offset+length > len(data) {
			return ""
		}
		tag := data[offset : offset+length]
		switch string(tag[:4]) {
		case "desc":
			n := int(binary.BigEndian.Uint32(tag[8:12]))
			if 12+n > len(tag) {
				return ""
			}
			return strings.TrimRight(string(tag[12:12+n]), "\x00")
		case "mluc":
			if len(tag) < 28 {
				return ""
			}
			n := int(binary.BigEndian.Uint32(tag[20:24]))
			start := int(binary.BigEndian.Uint32(tag[24:28]))
			if start+n > len(tag) {
				return ""
			}
			units := make([]uint16, n/2)
			for j := range units {
				units[j] = binary.BigEndian.Uint16(tag[start+2*j:])
			}
			return strings.TrimRight(string(utf16.Decode(units)), "\x00")
		}
		return ""
	}
	return ""
}
