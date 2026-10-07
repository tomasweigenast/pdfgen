package stream

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// Options configure a page stream.
type Options struct {
	// PageRectangle is the page box in CSS pixels: x, y, width, height.
	PageRectangle [4]float64
	// ColorSpace is the color space of transparency groups and shadings:
	// a name such as /DeviceCMYK or an [/ICCBased ref] array. Nil means
	// /DeviceRGB.
	ColorSpace writer.Object
	// ColorProfiles holds the names ("--profile") of the custom color
	// spaces defined in the /ColorSpace resource.
	ColorProfiles map[string]bool
	// Compress selects Flate for the stream data.
	Compress bool
}

type colorState struct {
	space  string
	coords []float64
}

func (c *colorState) equal(space string, coords []float64) bool {
	return c != nil && c.space == space && slices.Equal(c.coords, coords)
}

type fontState struct {
	name string
	size float64
}

// Stream is a content stream with the state needed to avoid redundant
// operators: the current color, alpha and font, and the transformation matrix
// stack. Form XObjects and patterns are streams too, created with
// [Stream.AddGroup] and [Stream.AddPattern].
type Stream struct {
	Content

	// PageRectangle is the page box in CSS pixels: x, y, width, height.
	PageRectangle [4]float64

	opts  Options
	res   *Resources
	extra *writer.Dict
	id    string

	color, strokeColor *colorState
	alpha, strokeAlpha string
	font, oldFont      *fontState
	ctmStack           []matrix.Matrix
	compress           bool
}

// New returns an empty stream that registers its resources in res.
func New(res *Resources, opts Options) *Stream {
	s := &Stream{
		PageRectangle: opts.PageRectangle,
		opts:          opts,
		res:           res,
		extra:         writer.NewDict(),
		ctmStack:      []matrix.Matrix{matrix.Identity},
		compress:      opts.Compress,
	}
	s.Content.Compress = opts.Compress
	return s
}

// Clone returns an empty stream with the same resources and options as s.
func (s *Stream) Clone() *Stream { return s.cloneWith(s.res, writer.NewDict()) }

func (s *Stream) cloneWith(res *Resources, extra *writer.Dict) *Stream {
	c := New(res, s.opts)
	c.extra = extra
	return c
}

// ID returns the name under which a form XObject or pattern was registered.
func (s *Stream) ID() string { return s.id }

// Object returns the stream as a PDF stream object. It is meant for the
// content stream of a page; groups and patterns are written by
// [Resources.Embed].
func (s *Stream) Object() *writer.Stream {
	compress := s.compress
	return &writer.Stream{Dict: s.extra.Clone(), Data: s.Bytes(), Compress: &compress}
}

// CTM returns the current transformation matrix.
func (s *Stream) CTM() matrix.Matrix { return s.ctmStack[len(s.ctmStack)-1] }

// PushState saves the graphics state.
func (s *Stream) PushState() {
	s.Content.PushState()
	s.ctmStack = append(s.ctmStack, s.CTM())
}

// PopState restores the graphics state. A "q" with nothing after it is
// removed instead of emitting an empty "q Q" pair.
func (s *Stream) PopState() {
	if s.lastIs("q") {
		s.dropLast()
	} else {
		s.Content.PopState()
	}
	s.color, s.strokeColor = nil, nil
	s.alpha, s.strokeAlpha = "", ""
	s.font = nil
	if len(s.ctmStack) > 1 {
		s.ctmStack = s.ctmStack[:len(s.ctmStack)-1]
	}
}

// Transform concatenates a matrix to the current transformation matrix.
func (s *Stream) Transform(a, b, c, d, e, f float64) {
	s.SetMatrix(a, b, c, d, e, f)
	s.ctmStack[len(s.ctmStack)-1] = matrix.New(a, b, c, d, e, f).Mul(s.CTM())
}

// BeginText begins a text object. Directly after an ET the two are merged,
// so consecutive runs share one text object and keep their font.
func (s *Stream) BeginText() {
	if s.lastIs("ET") {
		s.font = s.oldFont
		s.dropLast()
		return
	}
	s.Content.BeginText()
}

// EndText ends a text object.
func (s *Stream) EndText() {
	s.oldFont, s.font = s.font, nil
	s.Content.EndText()
}

// SetFontSize selects the named font resource and size, unless already
// selected.
func (s *Stream) SetFontSize(font string, size float64) {
	if s.font != nil && *s.font == (fontState{font, size}) {
		return
	}
	s.font = &fontState{font, size}
	s.Content.SetFontSize(font, size)
}

// SetColor sets the fill or stroke color and the matching alpha.
func (s *Stream) SetColor(c syntax.Color, stroke bool) {
	s.SetAlpha(c.Alpha, stroke, nil)

	current := &s.color
	if stroke {
		current = &s.strokeColor
	}
	if (*current).equal(c.Space, c.Coordinates) {
		return
	}
	*current = &colorState{c.Space, slices.Clone(c.Coordinates)}

	switch {
	case c.Space == syntax.SRGB || c.Space == syntax.HSL || c.Space == syntax.HWB:
		rgb, _ := c.To(syntax.SRGB)
		s.SetColorRGB(rgb.Coordinates[0], rgb.Coordinates[1], rgb.Coordinates[2], stroke)
	case c.Space == syntax.XYZD65 || c.Space == syntax.Oklab || c.Space == syntax.Oklch:
		s.setLab("lab-d65", c, stroke)
	case c.Space == syntax.XYZD50 || c.Space == syntax.Lab || c.Space == syntax.LCH:
		s.setLab("lab-d50", c, stroke)
	case c.Space == syntax.DeviceCMYK:
		s.SetColorSpace("DeviceCMYK", stroke)
		s.SetColorSpecial("", stroke, c.Coordinates...)
	case strings.HasPrefix(c.Space, "--") && s.opts.ColorProfiles[c.Space]:
		s.SetColorSpace(c.Space, stroke)
		s.SetColorSpecial("", stroke, c.Coordinates...)
	default:
		logging.Warnf("Unsupported color space %s, use sRGB instead", c.Space)
		var rgb [3]float64
		copy(rgb[:], c.Coordinates)
		s.SetColorRGB(rgb[0], rgb[1], rgb[2], stroke)
	}
}

// setLab selects a Lab color space and sets the color in it. The
// coordinates are always taken from the Lab conversion of c.
func (s *Stream) setLab(space string, c syntax.Color, stroke bool) {
	s.SetColorSpace(space, stroke)
	lab, _ := c.To(syntax.Lab)
	s.SetColorSpecial("", stroke, lab.Coordinates...)
}

// SetState registers an ExtGState dictionary under the next free "s<N>" name
// and applies it.
func (s *Stream) SetState(state *writer.Dict) { s.setState(&extGState{dict: state}) }

func (s *Stream) setState(g *extGState) {
	name := "s" + strconv.Itoa(s.res.extGState.len())
	s.res.extGState.set(name, g)
	s.SetGraphicsState(name)
}

// SetAlpha sets the stroke alpha when stroke is set and the fill alpha when
// fill is true; a nil fill means "unless stroking". The states are named
// after the alpha and shared by every use.
func (s *Stream) SetAlpha(alpha float64, stroke bool, fill *bool) {
	doFill := !stroke
	if fill != nil {
		doFill = *fill
	}
	repr := pyFloat(alpha)

	if stroke {
		key := "A" + repr
		if key != s.strokeAlpha {
			s.strokeAlpha = key
			s.useAlphaState(key, "CA", alpha)
		}
	}
	if doFill {
		key := "a" + repr
		if key != s.alpha {
			s.alpha = key
			s.useAlphaState(key, "ca", alpha)
		}
	}
}

func (s *Stream) useAlphaState(key, param string, alpha float64) {
	if !s.res.extGState.has(key) {
		s.res.extGState.set(key, &extGState{dict: writer.NewDict(param, alpha)})
	}
	s.SetGraphicsState(key)
}

// SetAlphaState applies a soft mask built from a new transparency group
// covering the given box, and returns the group so the caller can draw the
// mask into it. mode is "luminosity" or "alpha".
func (s *Stream) SetAlphaState(x, y, width, height float64, mode string) *Stream {
	group := s.AddGroup(x, y, width, height)
	s.setState(&extGState{
		dict: writer.NewDict(
			"Type", writer.Name("ExtGState"),
			"SMask", writer.Null{},
			"ca", 1,
			"AIS", false,
		),
		maskGroup: group,
		maskMode:  capitalize(mode),
	})
	return group
}

// SetBlendMode sets the blend mode of subsequent painting.
func (s *Stream) SetBlendMode(mode string) {
	s.SetState(writer.NewDict("Type", writer.Name("ExtGState"), "BM", writer.Name(mode)))
}

// AddGroup creates a form XObject with a transparency group (ISO 32000-1
// §11.6.6) covering the given box, registers it and returns its stream.
func (s *Stream) AddGroup(x, y, width, height float64) *Stream {
	res := s.res.child()
	extra := writer.NewDict(
		"Type", writer.Name("XObject"),
		"Subtype", writer.Name("Form"),
		"BBox", writer.Array{writer.Real(x), writer.Real(y), writer.Real(x + width), writer.Real(y + height)},
		"Resources", writer.Null{},
		"Group", writer.NewDict(
			"Type", writer.Name("Group"),
			"S", writer.Name("Transparency"),
			"I", true,
			"CS", s.defaultColorSpace(),
		),
	)
	group := s.cloneWith(res, extra)
	group.id = "x" + strconv.Itoa(s.res.xObject.len())
	s.res.xObject.set(group.id, group)
	return group
}

// AddPattern creates a colored tiling pattern (ISO 32000-1 §8.7.3) whose
// cell is the given box, repeated every repeatWidth by repeatHeight, and
// returns its stream.
func (s *Stream) AddPattern(x, y, width, height, repeatWidth, repeatHeight float64, m matrix.Matrix) *Stream {
	a, b, c, d, e, f := m.Values()
	res := s.res.child()
	extra := writer.NewDict(
		"Type", writer.Name("Pattern"),
		"PatternType", 1,
		"BBox", writer.Array{writer.Real(x), writer.Real(y), writer.Real(x + width), writer.Real(y + height)},
		"XStep", writer.Real(repeatWidth),
		"YStep", writer.Real(repeatHeight),
		"TilingType", 1,
		"PaintType", 1,
		"Matrix", writer.Array{writer.Real(a), writer.Real(b), writer.Real(c), writer.Real(d), writer.Real(e), writer.Real(f)},
		"Resources", writer.Null{},
	)
	pattern := s.cloneWith(res, extra)
	pattern.id = "p" + strconv.Itoa(s.res.pattern.len())
	s.res.pattern.set(pattern.id, pattern)
	return pattern
}

// AddShading registers a shading dictionary (ISO 32000-1 §8.7.4.3) and
// returns its name for [Content.PaintShading] or a shading pattern. An empty
// colorSpace uses the default color space; extend extends the shading beyond
// both ends of its domain.
func (s *Stream) AddShading(shadingType int, domain, coords []float64, extend bool, fn Function, colorSpace string) string {
	cs := s.defaultColorSpace()
	if colorSpace != "" {
		cs = writer.Name(colorSpace)
	}
	shading := writer.NewDict(
		"ShadingType", shadingType,
		"ColorSpace", cs,
		"Domain", reals(domain),
		"Coords", reals(coords),
		"Function", fn,
	)
	if extend {
		shading.Set("Extend", writer.Array{writer.Bool(true), writer.Bool(true)})
	}
	name := "s" + strconv.Itoa(s.res.shading.len())
	s.res.shading.set(name, shading)
	return name
}

func (s *Stream) defaultColorSpace() writer.Object {
	if s.opts.ColorSpace != nil {
		return s.opts.ColorSpace
	}
	return writer.Name("DeviceRGB")
}

// Function is a PDF function object, as built by [InterpolationFunction] and
// [StitchingFunction].
type Function = writer.Object

// InterpolationFunction returns an exponential interpolation function
// (type 2, ISO 32000-1 §7.10.3).
func InterpolationFunction(domain, c0, c1 []float64, n float64) Function {
	return writer.NewDict(
		"FunctionType", 2,
		"Domain", reals(domain),
		"C0", reals(c0),
		"C1", reals(c1),
		"N", n,
	)
}

// StitchingFunction returns a stitching function (type 3, ISO 32000-1
// §7.10.4) that joins subFunctions over the intervals set by bounds.
func StitchingFunction(domain, encode, bounds []float64, subFunctions []Function) Function {
	return writer.NewDict(
		"FunctionType", 3,
		"Domain", reals(domain),
		"Encode", reals(encode),
		"Bounds", reals(bounds),
		"Functions", writer.Array(subFunctions),
	)
}

func reals(v []float64) writer.Array {
	a := make(writer.Array, len(v))
	for i, f := range v {
		a[i] = writer.Real(f)
	}
	return a
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// pyFloat formats f as the shortest decimal that reads back to f, always
// with a fractional part or an exponent ("1.0", "0.5", "1e-05"), so that
// resource names stay stable across runs.
func pyFloat(f float64) string {
	if f == 0 || (math.Abs(f) >= 1e-4 && math.Abs(f) < 1e16) {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	}
	return strconv.FormatFloat(f, 'e', -1, 64)
}
