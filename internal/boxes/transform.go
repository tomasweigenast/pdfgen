package boxes

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
)

// SetTransformationMatrices sets TransformationMatrix on every box of the
// tree rooted at root that has a transform, walking absolute placeholders and
// the margin boxes of pages. Inline boxes are skipped: transforms do not
// apply to elements that can be split into several boxes (CSS Transforms 1,
// introduction). The matrix of a box does not include those of its ancestors.
// Calling it again recomputes the same values.
func SetTransformationMatrices(root Box) {
	if ph, ok := root.(*AbsolutePlaceholder); ok {
		SetTransformationMatrices(ph.Box)
		return
	}
	if _, inline := root.(*InlineBox); !inline {
		root.Base().TransformationMatrix = transformationMatrix(root)
	}
	for _, c := range AllChildren(root) {
		SetTransformationMatrices(c)
	}
}

// transformationMatrix returns the matrix of b that maps the border box
// positions before the transform to those after it, or nil when b has none.
func transformationMatrix(b Box) *matrix.Matrix {
	f := b.Base()
	ops := f.Style.Transform()
	if len(ops) == 0 {
		return nil
	}
	width, height := f.BorderWidth(), f.BorderHeight()
	origin := f.Style.TransformOrigin()
	originX := f.BorderBoxX() + resolveLength(origin.X, f.Style, width)
	originY := f.BorderBoxY() + resolveLength(origin.Y, f.Style, height)

	m := matrix.New(1, 0, 0, 1, originX, originY)
	for _, op := range ops {
		m = opMatrix(op, f.Style, width, height).Mul(m)
	}
	m = matrix.New(1, 0, 0, 1, -originX, -originY).Mul(m)
	return &m
}

func opMatrix(op props.TransformOp, style *computed.Style, width, height float64) matrix.Matrix {
	n := op.Numbers
	switch op.Name {
	case "scale":
		if len(n) == 2 {
			return matrix.New(n[0], 0, 0, n[1], 0, 0)
		}
	case "rotate":
		if len(n) == 1 {
			cos, sin := math.Cos(n[0]), math.Sin(n[0])
			return matrix.New(cos, sin, -sin, cos, 0, 0)
		}
	case "translate":
		return matrix.New(1, 0, 0, 1,
			resolveLength(op.Translate.X, style, width),
			resolveLength(op.Translate.Y, style, height))
	case "skew":
		if len(n) == 2 {
			return matrix.New(1, math.Tan(n[1]), math.Tan(n[0]), 1, 0, 0)
		}
	case "matrix":
		if len(n) == 6 {
			return matrix.New(n[0], n[1], n[2], n[3], n[4], n[5])
		}
	}
	logging.Warnf("Invalid transform function: %s", op.Name)
	return matrix.Identity
}

// resolveLength returns l in pixels, percentages being relative to referTo.
func resolveLength(l props.Length, style *computed.Style, referTo float64) float64 {
	switch {
	case l.IsKeyword():
		return 0
	case l.Calc != nil:
		resolved, err := tk.ResolveMath(l.Calc.Function, style, "", &referTo)
		if err != nil {
			logging.Warnf("Invalid math function: %s", l.Calc.Function.Serialize())
			return 0
		}
		switch t := resolved.(type) {
		case *syntax.PercentageToken:
			return referTo * t.Value / 100
		case *syntax.DimensionToken:
			return style.ToPixels(t.Value, t.LowerUnit, "")
		case *syntax.NumberToken:
			return t.Value
		}
		return 0
	case l.Dimension.Unit == "%":
		return referTo * l.Dimension.Value / 100
	}
	return l.Dimension.Value
}
