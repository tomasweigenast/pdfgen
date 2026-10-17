package layout

import (
	"math"
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestMinMaxRerunsWithLimitedSize(t *testing.T) {
	for _, vertical := range []bool{false, true} {
		for _, tt := range []struct {
			name                      string
			initial, minimum, maximum float64
			want                      []float64
		}{
			{"inside", 50, 20, 80, []float64{50}},
			{"maximum", 100, 20, 80, []float64{100, 80}},
			{"minimum", 10, 20, 80, []float64{10, 20}},
			{"minimum wins", 100, 90, 80, []float64{100, 80, 90}},
			{"exact maximum", 80, 20, 80, []float64{80}},
			{"exact minimum", 20, 20, 80, []float64{20}},
			{"unordered maximum", 50, 20, math.NaN(), []float64{50}},
		} {
			t.Run(tt.name+map[bool]string{false: "/width", true: "/height"}[vertical], func(t *testing.T) {
				b := &boxes.BoxFields{
					Width: units.F(tt.initial), Height: units.F(tt.initial),
					MinWidth: units.F(tt.minimum), MinHeight: units.F(tt.minimum),
					MaxWidth: units.F(tt.maximum), MaxHeight: units.F(tt.maximum),
				}
				seen := []float64{}
				set := func() {
					value := b.Width.V
					if vertical {
						value = b.Height.V
					}
					seen = append(seen, value)
					b.MarginLeft = units.F((200 - value) / 2)
				}
				if vertical {
					withMinMaxHeight(b, set)
				} else {
					withMinMaxWidth(b, set)
				}
				if !slices.Equal(seen, tt.want) || b.MarginLeft != units.F((200-tt.want[len(tt.want)-1])/2) {
					t.Fatalf("calls = %v, margin = %+v; want %v", seen, b.MarginLeft, tt.want)
				}
			})
		}
	}
}

func TestAutoDimensionsAreNotNumericalBounds(t *testing.T) {
	if greater(units.Auto, units.F(0)) || greater(units.F(100), units.Auto) || greater(units.Auto, units.Auto) {
		t.Fatal("auto compared as a number")
	}
}
