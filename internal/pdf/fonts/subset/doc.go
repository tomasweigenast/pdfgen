// Package subset reduces an OpenType font to the glyphs a document uses and
// writes it back as a standalone sfnt, ready to be embedded in a PDF
// (ISO 32000-1 section 9.9, OpenType Font File).
//
// Glyphs are renumbered in the order they are given: glyph i of the result is
// glyph glyphs[i] of the source, so a PDF font that assigns CIDs in first-use
// order can use CIDToGIDMap /Identity. The output is deterministic: nothing
// in it depends on the wall clock.
//
// Supported sources are TrueType outlines (glyf) and name-keyed CFF.
// CID-keyed CFF and CFF2 are reported with [ErrUnsupported].
package subset
