package validation

import (
	"fmt"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// validator parses the value tokens of a declaration, which have no
// whitespace or comments, into a value. A nil value without an error means
// that the tokens are not valid for the property.
type validator = func(tokens []syntax.Node, baseURL string) (props.Value, error)

// item validates the tokens of one comma-separated item of a list.
type item[E any] = func(tokens []syntax.Node, baseURL string) (E, bool, error)

var (
	// proprietary and unstable name the properties that can be used with the
	// vendor prefix: proprietary ones only with it, unstable ones also
	// without it (https://www.w3.org/TR/CSS/#proprietary,
	// https://www.w3.org/TR/CSS/#unstable-syntax).
	proprietary = map[string]bool{}
	unstable    = map[string]bool{}
)

type flag uint8

const (
	flagProprietary flag = 1 << iota
	flagUnstable
)

// register installs fn as the validator of the named properties. A name that
// is not in props.Table is a programming error.
func register(fn validator, flags flag, names ...string) {
	for _, name := range names {
		id, ok := props.ID(name)
		if !ok {
			panic(fmt.Sprintf("validation: %q is not a known property", name))
		}
		if props.Table[id].Validate != nil {
			panic(fmt.Sprintf("validation: %q is validated twice", name))
		}
		props.Table[id].Validate = fn
		if flags&flagProprietary != 0 {
			proprietary[name] = true
		}
		if flags&flagUnstable != 0 {
			unstable[name] = true
		}
	}
}

// Validate parses the tokens of a non-shorthand property, as written in a
// declaration and without whitespace. It reports unknown and unsupported
// properties, and returns a *props.Pending for values with var().
func Validate(tokens []syntax.Node, name, baseURL string) (props.Value, error) {
	return validateNonShorthand(tokens, name, baseURL, false)
}

// validateNonShorthand validates a longhand. With required set the property
// is known to exist, as for the longhands that an expander produced.
func validateNonShorthand(tokens []syntax.Node, name, baseURL string, required bool) (props.Value, error) {
	custom := isCustomProperty(name)
	id, known := props.ID(name)
	if !required && !known && !custom {
		return nil, tk.Invalid("unknown property")
	}
	if !required && !custom && props.Table[id].Validate == nil {
		return nil, tk.Invalid("property not supported yet")
	}

	for _, token := range tokens {
		if tk.CheckVar(token) {
			return &props.Pending{Name: name, Tokens: tokens}, nil
		}
	}
	if custom {
		return props.Tokens(tokens), nil
	}
	if !known || props.Table[id].Validate == nil {
		return nil, tk.Invalid("property not supported yet")
	}

	switch keyword := tk.GetSingleKeyword(tokens); keyword {
	case "initial", "inherit":
		return props.Keyword(keyword), nil
	}
	value, err := props.Table[id].Validate(tokens, baseURL)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, tk.ErrInvalid
	}
	return value, nil
}

func isCustomProperty(name string) bool { return len(name) >= 2 && name[:2] == "--" }
