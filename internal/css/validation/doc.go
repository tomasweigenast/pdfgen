// Package validation checks CSS declarations and descriptors, and expands
// shorthands into the longhand values stored in a style.
//
// Each property has a validator that parses the component values of a
// declaration into a props.Value, following the grammar of the property in
// its specification (CSS 2.1 and the CSS modules that extend it). Validators
// are installed in the Validate field of props.Table when the package is
// imported. Invalid declarations are ignored with a warning, as browsers do.
package validation
