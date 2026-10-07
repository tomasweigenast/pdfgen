package matrix

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNewLaysOutPDFOperands(t *testing.T) {
	m := New(1, 2, 3, 4, 5, 6)
	want := Matrix{{1, 2, 0}, {3, 4, 0}, {5, 6, 1}}
	if m != want {
		t.Errorf("New = %v, want %v", m, want)
	}
	a, b, c, d, e, f := m.Values()
	if [6]float64{a, b, c, d, e, f} != [6]float64{1, 2, 3, 4, 5, 6} {
		t.Errorf("Values = %v %v %v %v %v %v", a, b, c, d, e, f)
	}
}

func TestMulAppliesLeftOperandFirst(t *testing.T) {
	translate := New(1, 0, 0, 1, 10, 20)
	scale := New(2, 0, 0, 3, 0, 0)

	x, y := translate.Mul(scale).TransformPoint(1, 1)
	if x != 22 || y != 63 {
		t.Errorf("translate then scale: got (%v, %v), want (22, 63)", x, y)
	}
	x, y = scale.Mul(translate).TransformPoint(1, 1)
	if x != 12 || y != 23 {
		t.Errorf("scale then translate: got (%v, %v), want (12, 23)", x, y)
	}
}

func TestIdentityIsNeutral(t *testing.T) {
	m := New(2, 3, 5, 7, 11, 13)
	if got := m.Mul(Identity); got != m {
		t.Errorf("m·I = %v, want %v", got, m)
	}
	if got := Identity.Mul(m); got != m {
		t.Errorf("I·m = %v, want %v", got, m)
	}
}

func TestDeterminant(t *testing.T) {
	tests := []struct {
		name string
		m    Matrix
		want float64
	}{
		{"identity", Identity, 1},
		{"scale", New(2, 0, 0, 3, 7, 9), 6},
		{"rotation by 90 degrees", New(0, 1, -1, 0, 0, 0), 1},
		{"flip", New(1, 0, 0, -1, 0, 0), -1},
		{"singular", New(1, 2, 2, 4, 0, 0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Determinant(); !approx(got, tt.want) {
				t.Errorf("Determinant = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvertRoundTrips(t *testing.T) {
	tests := []Matrix{
		Identity,
		New(2, 0, 0, 3, 5, -7),
		New(math.Cos(0.3), math.Sin(0.3), -math.Sin(0.3), math.Cos(0.3), 4, 9),
		New(1, 0.5, 0.25, 1, -3, 2),
	}
	for _, m := range tests {
		inv, ok := m.Invert()
		if !ok {
			t.Fatalf("Invert(%v) reported singular", m)
		}
		prod := m.Mul(inv)
		for i := range prod {
			for j := range prod[i] {
				if !approx(prod[i][j], Identity[i][j]) {
					t.Errorf("m·m⁻¹ = %v, want identity", prod)
				}
			}
		}
		x, y := m.TransformPoint(3, 4)
		bx, by := inv.TransformPoint(x, y)
		if !approx(bx, 3) || !approx(by, 4) {
			t.Errorf("round trip of (3, 4) gave (%v, %v)", bx, by)
		}
	}
}

func TestInvertSingular(t *testing.T) {
	if _, ok := New(1, 2, 2, 4, 0, 0).Invert(); ok {
		t.Error("Invert of a singular matrix reported success")
	}
	if _, ok := New(0, 0, 0, 0, 5, 5).Invert(); ok {
		t.Error("Invert of a zero matrix reported success")
	}
}

func TestTransformPointRotation(t *testing.T) {
	x, y := New(0, 1, -1, 0, 0, 0).TransformPoint(1, 0)
	if !approx(x, 0) || !approx(y, 1) {
		t.Errorf("got (%v, %v), want (0, 1)", x, y)
	}
}
