package variants

import _ "embed"

// SRGBProfile is the sRGB ICC profile used for output intents.
//
//go:embed sRGB2014.icc
var SRGBProfile []byte
