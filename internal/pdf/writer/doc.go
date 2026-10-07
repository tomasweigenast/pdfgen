// Package writer is a PDF object model and serializer.
//
// Objects are numbered in creation order, dictionaries keep their insertion
// order and every created object is written, reachable or not, so the output
// is a deterministic function of the calls made to build it. The file
// structure follows ISO 32000-1 §7.5: a classic cross-reference table, or
// object streams with a cross-reference stream (§7.5.7, §7.5.8) when the
// version allows it and compression is enabled.
package writer
