// Package stream builds PDF content streams (ISO 32000-1 chapter 8).
//
// [Content] holds the operators and their byte format. [Stream] adds the
// graphics-state tracking the renderer needs: it drops redundant color,
// alpha and font operators, tracks the transformation matrix, and registers
// graphics states, transparency groups, tiling patterns and shadings in the
// [Resources] it shares with the streams it creates.
//
// Fonts, images and marked content are not registered here: the code that
// owns them adds entries to Resources.Font and the XObject category, and
// brackets drawing with Content.BeginMarkedContent.
package stream
