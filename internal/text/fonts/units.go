package fonts

// UnitsPerPx is the number of font units in one CSS pixel. Sizes, advances
// and font metrics are expressed in 1/1024 px.
const UnitsPerPx = 1024

// ToUnits converts a size in px to font units, truncating toward zero.
func ToUnits(px float64) int32 { return int32(px * UnitsPerPx) }

// FromUnits converts font units to px.
func FromUnits(u int32) float64 { return float64(u) / UnitsPerPx }
