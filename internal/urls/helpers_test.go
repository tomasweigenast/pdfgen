package urls

import (
	"compress/flate"
	"encoding/base64"
	"io"
)

func encodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func flateWriter(w io.Writer) (*flate.Writer, error) { return flate.NewWriter(w, flate.DefaultCompression) }
