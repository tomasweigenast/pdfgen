package tokens

import (
	"errors"
	"fmt"
)

// InvalidValues reports a declaration whose value is not valid for its
// property. Reason is shown in the warning, and may be empty.
type InvalidValues struct{ Reason string }

func (e *InvalidValues) Error() string {
	if e.Reason == "" {
		return "invalid value"
	}
	return e.Reason
}

// Invalid returns an *InvalidValues with a formatted reason.
func Invalid(format string, args ...any) error {
	return &InvalidValues{Reason: fmt.Sprintf(format, args...)}
}

// ErrInvalid is an *InvalidValues without a reason.
var ErrInvalid error = &InvalidValues{}

// Errors reported while resolving math functions.
var (
	// ErrPercentageInMath is returned when percentages are mixed with other
	// values and there is no reference length. It has priority over
	// ErrRelativeLengthInMath, so that properties that do not accept
	// percentages can discard the value first.
	ErrPercentageInMath = errors.New("percentage in math function without reference length")
	// ErrRelativeLengthInMath is returned when a font or viewport relative
	// unit is used and there is no style to resolve it.
	ErrRelativeLengthInMath = errors.New("relative length in math function without style")
)
