package pdfgen

import "github.com/tomasweigenast/pdfgen/internal/pdf/metadata"

// Version returns the version of the library, the one written in the
// Producer entry of the PDF files.
func Version() string { return metadata.Version }
