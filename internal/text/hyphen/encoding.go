package hyphen

import (
	"fmt"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
)

// lookupEncoding resolves the encoding name found on the first line of a
// dictionary. Names follow Hunspell's conventions, so "ISO8859-1" and
// "microsoft-cp1251" are accepted next to the IANA spellings.
func lookupEncoding(name string) (encoding.Encoding, error) {
	lname := strings.ToLower(name)
	switch lname {
	case "utf-8", "utf8":
		return encoding.Nop, nil
	case "microsoft-cp1251":
		return charmap.Windows1251, nil
	}
	if rest, ok := strings.CutPrefix(lname, "iso8859-"); ok {
		name = "ISO-8859-" + rest
	}
	enc, err := ianaindex.IANA.Encoding(name)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("hyphen: unsupported encoding %q", name)
	}
	return enc, nil
}
