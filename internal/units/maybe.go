package units

// MaybeFloat is a dimension that is either a number or the keyword auto.
// The zero value is the number 0, not auto.
type MaybeFloat struct {
	V    float64
	Auto bool
}

// Auto is the auto keyword.
var Auto = MaybeFloat{Auto: true}

// F returns the number v.
func F(v float64) MaybeFloat { return MaybeFloat{V: v} }

// IsAuto reports whether m is the auto keyword.
func (m MaybeFloat) IsAuto() bool { return m.Auto }

// Or returns the number held by m, or def when m is auto.
func (m MaybeFloat) Or(def float64) float64 {
	if m.Auto {
		return def
	}
	return m.V
}
