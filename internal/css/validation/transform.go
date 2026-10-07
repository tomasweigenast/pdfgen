package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	register(validateTransform, 0, "transform")
}

// validateTransform validates transform (CSS Transforms 1 §13). Only the 2D
// functions are supported; they are normalized to rotate, skew, translate,
// scale and matrix.
func validateTransform(tokens []syntax.Node, _ string) (props.Value, error) {
	if tk.GetSingleKeyword(tokens) == "none" {
		return props.Transform{}, nil
	}
	transforms := props.Transform{}
	for _, token := range tokens {
		f := tk.NewFunction(token)
		if !f.IsFunction() {
			return nil, nil
		}
		arguments := f.SplitComma()

		allNumbers := len(arguments) > 0
		for _, argument := range arguments {
			if _, ok := argument.(*syntax.NumberToken); !ok {
				allNumbers = false
			}
		}
		number := func(i int) float64 { return arguments[i].(*syntax.NumberToken).Value }

		switch len(arguments) {
		case 1:
			angle, okAngle := tk.GetAngle(arguments[0])
			length, okLength := tk.GetLength(arguments[0], true, true)
			switch {
			case f.Name == "rotate" && okAngle:
				transforms = append(transforms, props.TransformOp{Name: "rotate", Numbers: []float64{angle}})
			case (f.Name == "skewx" || f.Name == "skew") && okAngle:
				transforms = append(transforms, props.TransformOp{Name: "skew", Numbers: []float64{angle, 0}})
			case f.Name == "skewy" && okAngle:
				transforms = append(transforms, props.TransformOp{Name: "skew", Numbers: []float64{0, angle}})
			case (f.Name == "translatex" || f.Name == "translate") && okLength:
				transforms = append(transforms, props.TransformOp{Name: "translate", Translate: props.LengthPair{X: length, Y: zeroPixels}})
			case f.Name == "translatey" && okLength:
				transforms = append(transforms, props.TransformOp{Name: "translate", Translate: props.LengthPair{X: zeroPixels, Y: length}})
			case f.Name == "scalex" && allNumbers:
				transforms = append(transforms, props.TransformOp{Name: "scale", Numbers: []float64{number(0), 1}})
			case f.Name == "scaley" && allNumbers:
				transforms = append(transforms, props.TransformOp{Name: "scale", Numbers: []float64{1, number(0)}})
			case f.Name == "scale" && allNumbers:
				transforms = append(transforms, props.TransformOp{Name: "scale", Numbers: []float64{number(0), number(0)}})
			default:
				return nil, nil
			}
		case 2:
			switch {
			case f.Name == "scale" && allNumbers:
				transforms = append(transforms, props.TransformOp{Name: "scale", Numbers: []float64{number(0), number(1)}})
			case f.Name == "translate":
				lengths, ok := lengthList(arguments, true, true)
				if !ok {
					return nil, nil
				}
				transforms = append(transforms, props.TransformOp{Name: "translate", Translate: props.LengthPair{X: lengths[0], Y: lengths[1]}})
			case f.Name == "skew":
				a0, ok0 := tk.GetAngle(arguments[0])
				a1, ok1 := tk.GetAngle(arguments[1])
				if !ok0 || !ok1 {
					return nil, nil
				}
				transforms = append(transforms, props.TransformOp{Name: "skew", Numbers: []float64{a0, a1}})
			default:
				return nil, nil
			}
		case 6:
			if f.Name != "matrix" || !allNumbers {
				return nil, nil
			}
			numbers := make([]float64, 6)
			for i := range numbers {
				numbers[i] = number(i)
			}
			transforms = append(transforms, props.TransformOp{Name: "matrix", Numbers: numbers})
		default:
			return nil, nil
		}
	}
	return transforms, nil
}
