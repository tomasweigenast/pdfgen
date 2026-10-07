// Package images decodes the raster images that documents embed and draws
// them on PDF streams.
//
// JPEG data is embedded as it is (ISO 32000-1 §7.4.8, DCTDecode) unless the
// image has to be transformed or recompressed; every other format is decoded
// and stored with Flate and PNG predictors (§7.4.4.4), with transparency in a
// soft mask (§11.6.5.3). Decoded images are cached by URL.
package images
