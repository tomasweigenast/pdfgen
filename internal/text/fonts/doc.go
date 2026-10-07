// Package fonts finds, loads and describes fonts, including @font-face rules
// (CSS Fonts Level 4).
//
// Text is measured in 1/1024 px units: font sizes are truncated to that
// grid and the shaper is scaled to it, so advances and metrics are integers
// that convert exactly to pixels. See [ToUnits].
package fonts
