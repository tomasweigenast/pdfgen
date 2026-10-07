// Package tokens parses the component values of CSS declarations into typed
// values: keywords, numbers, lengths, angles, colors, images, urls and the
// functions of generated content. It also resolves math functions such as
// calc() and finds the var() references that a value depends on.
//
// It implements the value grammars of CSS Values and Units 4, CSS Images 4
// and CSS Content 3 that the property validators of the validation package
// and the computed-value code share.
package tokens
