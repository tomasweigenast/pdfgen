// Package images decodes the raster images that documents embed, builds
// gradients from computed values, and draws both on PDF streams.
//
// JPEG data is embedded as it is (ISO 32000-1 §7.4.8, DCTDecode) unless the
// image has to be transformed or recompressed; every other format is decoded
// and stored with Flate and PNG predictors (§7.4.4.4), with transparency in a
// soft mask (§11.6.5.3). Decoded images are cached by URL.
//
// Gradients are painted as axial and radial shadings (§8.7.4.5) whose color
// stops are stitching functions; translucent stops add a luminosity soft mask.
package images
