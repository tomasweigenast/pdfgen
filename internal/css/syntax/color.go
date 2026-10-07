package syntax

import (
	"math"
	"strconv"
	"strings"
)

// CurrentColor is the Space of the Color returned for the currentcolor
// keyword; such a color has no coordinates.
const CurrentColor = "currentcolor"

// Color spaces understood by ParseColor and Color.To.
const (
	SRGB        = "srgb"
	SRGBLinear  = "srgb-linear"
	DisplayP3   = "display-p3"
	A98RGB      = "a98-rgb"
	ProphotoRGB = "prophoto-rgb"
	Rec2020     = "rec2020"
	XYZD50      = "xyz-d50"
	XYZD65      = "xyz-d65"
	HSL         = "hsl"
	HWB         = "hwb"
	Lab         = "lab"
	LCH         = "lch"
	Oklab       = "oklab"
	Oklch       = "oklch"
	DeviceCMYK  = "device-cmyk"
)

// D50 and D65 are the XYZ coordinates of the white points, normalized to Y=1.
var (
	D50 = [3]float64{0.3457 / 0.3585, 1, (1 - 0.3457 - 0.3585) / 0.3585}
	D65 = [3]float64{0.3127 / 0.3290, 1, (1 - 0.3127 - 0.3290) / 0.3290}
)

// ColorSchemes are the color schemes recognized by light-dark().
var ColorSchemes = map[string]bool{"light": true, "dark": true}

var functionSpaces = map[string]bool{
	SRGB: true, SRGBLinear: true, DisplayP3: true, A98RGB: true,
	ProphotoRGB: true, Rec2020: true, "xyz": true, XYZD50: true, XYZD65: true,
}

// Color is a specified color in a color space: one of the space constants
// above, or the name of a custom color profile such as "--profile".
type Color struct {
	Space string
	// Coordinates holds the channel values, without ranges enforced. An
	// undefined ("none") coordinate is NaN; see IsNone.
	Coordinates []float64
	// Alpha is clipped to [0, 1].
	Alpha float64
}

// NewColor returns a Color with the alpha clipped to [0, 1].
func NewColor(space string, coordinates []float64, alpha float64) Color {
	return Color{space, coordinates, math.Max(0, math.Min(1, alpha))}
}

// IsNone reports whether the coordinate at index i is undefined.
func (c Color) IsNone(i int) bool { return math.IsNaN(c.Coordinates[i]) }

// IsCurrentColor reports whether c stands for the currentcolor keyword.
func (c Color) IsCurrentColor() bool { return c.Space == CurrentColor }

// Values returns the coordinates followed by the alpha.
func (c Color) Values() []float64 {
	return append(append([]float64(nil), c.Coordinates...), c.Alpha)
}

var none = math.NaN()

func pyMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r != 0 && (r < 0) != (b < 0) {
		r += b
	}
	return r
}

func radians(deg float64) float64 { return deg * (math.Pi / 180) }
func degrees(rad float64) float64 { return rad * (180 / math.Pi) }

func hlsToRGB(h, l, s float64) (float64, float64, float64) {
	if s == 0 {
		return l, l, l
	}
	var m2 float64
	if l <= 0.5 {
		m2 = l * (1 + s)
	} else {
		m2 = l + s - (l * s)
	}
	m1 := 2*l - m2
	v := func(hue float64) float64 {
		hue = pyMod(hue, 1)
		switch {
		case hue < 1.0/6:
			return m1 + (m2-m1)*hue*6
		case hue < 0.5:
			return m2
		case hue < 2.0/3:
			return m1 + (m2-m1)*(2.0/3-hue)*6
		}
		return m1
	}
	return v(h + 1.0/3), v(h), v(h - 1.0/3)
}

// To returns the color converted to space; undefined coordinates become 0.
// The supported conversions are hsl and hwb to srgb, lab and lch to xyz-d50,
// oklab and oklch to xyz-d65, and xyz-d50, xyz-d65, lch, oklab and oklch to
// lab. Any other combination reports false.
func (c Color) To(space string) (Color, bool) {
	coords := make([]float64, len(c.Coordinates))
	for i, v := range c.Coordinates {
		if !math.IsNaN(v) {
			coords[i] = v
		}
	}
	if space == "xyz" {
		space = XYZD65
	}
	if space == c.Space {
		return NewColor(space, coords, c.Alpha), true
	}
	if len(coords) != 3 {
		return Color{}, false
	}
	polar := func() (float64, float64) {
		return coords[1] * math.Cos(radians(coords[2])), coords[1] * math.Sin(radians(coords[2]))
	}
	out := func(a, b, d float64) (Color, bool) {
		return NewColor(space, []float64{a, b, d}, c.Alpha), true
	}
	switch space {
	case SRGB:
		switch c.Space {
		case HSL:
			return out(hlsToRGB(coords[0]/360, coords[2]/100, coords[1]/100))
		case HWB:
			white, black := coords[1], coords[2]
			if white+black >= 100 {
				v := white / (white + black)
				return out(v, v, v)
			}
			r, g, b := hlsToRGB(coords[0]/360, 0.5, 1)
			f := func(ch float64) float64 { return ((ch * (100 - white - black)) + white) / 100 }
			return out(f(r), f(g), f(b))
		}
	case XYZD50:
		switch c.Space {
		case Lab:
			return out(labToXYZ(coords[0], coords[1], coords[2], D50))
		case LCH:
			a, b := polar()
			return out(labToXYZ(coords[0], a, b, D50))
		}
	case XYZD65:
		switch c.Space {
		case Oklab:
			return out(oklabToXYZ(coords[0], coords[1], coords[2]))
		case Oklch:
			a, b := polar()
			return out(oklabToXYZ(coords[0], a, b))
		}
	case Lab:
		switch c.Space {
		case XYZD50:
			return out(xyzToLab(coords[0], coords[1], coords[2], D50))
		case XYZD65:
			return out(xyzToLab(coords[0], coords[1], coords[2], D65))
		case LCH:
			a, b := polar()
			return out(coords[0], a, b)
		case Oklab:
			x, y, z := oklabToXYZ(coords[0], coords[1], coords[2])
			return out(xyzToLab(x, y, z, D65))
		case Oklch:
			a, b := polar()
			x, y, z := oklabToXYZ(coords[0], a, b)
			return out(xyzToLab(x, y, z, D65))
		}
	}
	return Color{}, false
}

// Conversion constants from CSS Color 4 §17.
const (
	kappa   = 24389.0 / 27
	epsilon = 216.0 / 24389
)

var lmsToXYZ = [3][3]float64{
	{1.2268798733741557, -0.5578149965554813, 0.28139105017721583},
	{-0.04057576262431372, 1.1122868293970594, -0.07171106666151701},
	{-0.07637294974672142, -0.4214933239627914, 1.5869240244272418},
}

var oklabToLMS = [3][3]float64{
	{0.99999999845051981432, 0.39633779217376785678, 0.21580375806075880339},
	{1.0000000088817607767, -0.1055613423236563494, -0.063854174771705903402},
	{1.0000000546724109177, -0.089484182094965759684, -1.2914855378640917399},
}

func xyzToLab(X, Y, Z float64, d [3]float64) (L, a, b float64) {
	f := func(v float64) float64 {
		if v > epsilon {
			return math.Pow(v, 1.0/3)
		}
		return (kappa*v + 16) / 116
	}
	f0, f1, f2 := f(X/d[0]), f(Y/d[1]), f(Z/d[2])
	return 116*f1 - 16, 500 * (f0 - f1), 200 * (f1 - f2)
}

func labToXYZ(L, a, b float64, d [3]float64) (X, Y, Z float64) {
	f1 := (L + 16) / 116
	f0 := a/500 + f1
	f2 := f1 - b/200
	cube := func(f float64) float64 {
		if c := math.Pow(f, 3); c > epsilon {
			return c
		}
		return (116*f - 16) / kappa
	}
	x := cube(f0)
	var y float64
	if L > kappa*epsilon {
		y = math.Pow((L+16)/116, 3)
	} else {
		y = L / kappa
	}
	z := cube(f2)
	return x * d[0], y * d[1], z * d[2]
}

func oklabToXYZ(L, a, b float64) (X, Y, Z float64) {
	lab := [3]float64{L, a, b}
	var lms [3]float64
	for i := range lms {
		lms[i] = oklabToLMS[i][0]*lab[0] + oklabToLMS[i][1]*lab[1] + oklabToLMS[i][2]*lab[2]
	}
	var xyz [3]float64
	for i := range xyz {
		xyz[i] = lmsToXYZ[i][0]*math.Pow(lms[0], 3) + lmsToXYZ[i][1]*math.Pow(lms[1], 3) +
			lmsToXYZ[i][2]*math.Pow(lms[2], 3)
	}
	return xyz[0], xyz[1], xyz[2]
}

func significant(nodes []Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if !isBlank(n) {
			out = append(out, n)
		}
	}
	return out
}

// everyOther returns nodes[start], nodes[start+2], ...
func everyOther(nodes []Node, start int) []Node {
	var out []Node
	for i := start; i < len(nodes); i += 2 {
		out = append(out, nodes[i])
	}
	return out
}

func allCommas(nodes []Node) bool {
	for _, n := range nodes {
		if !IsLiteral(n, ",") {
			return false
		}
	}
	return true
}

func isNone(n Node) bool {
	id, ok := n.(*IdentToken)
	return ok && id.LowerValue == "none"
}

// typeSet summarizes the token types of args, ignoring "none".
type typeSet struct{ number, percentage, other bool }

func typesOf(args []Node) typeSet {
	var s typeSet
	for _, a := range args {
		switch {
		case isNone(a):
		case a.Type() == "number":
			s.number = true
		case a.Type() == "percentage":
			s.percentage = true
		default:
			s.other = true
		}
	}
	return s
}

// onlyNumbersOrPercentages is _types(args) <= {number, percentage}.
func (s typeSet) numbersOrPercentages() bool { return !s.other }

// uniform is _types(args) in (set(), {number}, {percentage}).
func (s typeSet) uniform() bool { return !s.other && !(s.number && s.percentage) }

// numeric returns the value of a number or percentage token.
func numeric(n Node) (value float64, isPercentage bool) {
	switch t := n.(type) {
	case *NumberToken:
		return t.Value, false
	case *PercentageToken:
		return t.Value, true
	}
	return 0, false
}

// scaled is the value of a number or percentage token, with percentages
// multiplied by pct; NaN for none.
func scaled(n Node, pct float64) float64 {
	if isNone(n) {
		return none
	}
	v, isPct := numeric(n)
	if isPct {
		return v * pct
	}
	return v
}

func parseAlpha4(args []Node) (float64, bool) {
	switch len(args) {
	case 0:
		return 1, true
	case 1:
		switch t := args[0].(type) {
		case *NumberToken:
			return math.Min(1, math.Max(0, t.Value)), true
		case *PercentageToken:
			return math.Min(1, math.Max(0, t.Value/100)), true
		}
	}
	return 0, false
}

func parseHue(n Node) (float64, bool) {
	switch t := n.(type) {
	case *NumberToken:
		return pyMod(t.Value, 360), true
	case *DimensionToken:
		switch t.Unit {
		case "deg":
			return pyMod(t.Value, 360), true
		case "grad":
			return pyMod(t.Value/400*360, 360), true
		case "rad":
			return pyMod(degrees(t.Value), 360), true
		case "turn":
			return pyMod(t.Value*360, 360), true
		}
	case *IdentToken:
		if t.LowerValue == "none" {
			return 0, true
		}
	}
	return 0, false
}

func isIdent(n Node) bool {
	_, ok := n.(*IdentToken)
	return ok
}

// hexChannels matches the hexadecimal color notations (CSS Color 4 §5.2):
// 4, 8, 3 or 6 digits.
func hexChannels(value string) ([]float64, bool) {
	// A trailing newline is tolerated, as by an end-anchored pattern.
	value = strings.TrimSuffix(value, "\n")
	for _, c := range value {
		if !isHexDigit(c) {
			return nil, false
		}
	}
	var digits, mult int
	switch len(value) {
	case 4:
		digits, mult = 1, 2
	case 8:
		digits, mult = 2, 1
	case 3:
		digits, mult = 1, 2
	case 6:
		digits, mult = 2, 1
	default:
		return nil, false
	}
	var channels []float64
	for i := 0; i < len(value); i += digits {
		v, _ := strconv.ParseInt(strings.Repeat(value[i:i+digits], mult), 16, 32)
		channels = append(channels, float64(v)/255)
	}
	return channels, true
}

// ParseColor4 parses a color as defined by CSS Color Level 4. It reports
// false for anything that is not a valid color. The currentcolor keyword
// gives a Color whose Space is CurrentColor.
func ParseColor4(token Node) (Color, bool) {
	switch t := token.(type) {
	case *IdentToken:
		switch t.LowerValue {
		case CurrentColor:
			return Color{Space: CurrentColor, Alpha: 1}, true
		case "transparent":
			return NewColor(SRGB, []float64{0, 0, 0}, 0), true
		}
		if rgb, ok := colorKeywords[t.LowerValue]; ok {
			return NewColor(SRGB, []float64{float64(rgb[0]) / 255, float64(rgb[1]) / 255, float64(rgb[2]) / 255}, 1), true
		}
	case *HashToken:
		if ch, ok := hexChannels(t.Value); ok {
			alpha := 1.0
			if len(ch) == 4 {
				alpha, ch = ch[3], ch[:3]
			}
			return NewColor(SRGB, ch, alpha), true
		}
	case *FunctionBlock:
		return parseColorFunction4(t)
	}
	return Color{}, false
}

func parseColorFunction4(fn *FunctionBlock) (Color, bool) {
	tokens := significant(fn.Arguments)
	name := fn.LowerName
	var space Node
	if name == "color" {
		if len(tokens) == 0 {
			return Color{}, false
		}
		space, tokens = tokens[0], tokens[1:]
	}
	oldSyntax := false
	switch n := len(tokens); {
	case (n == 5 || n == 7) && allCommas(everyOther(tokens, 1)):
		oldSyntax = true
		tokens = everyOther(tokens, 0)
	case n == 3:
	case n == 5 && IsLiteral(tokens[3], "/"):
		tokens = append(tokens[:3:3], tokens[4])
	default:
		return Color{}, false
	}
	args := tokens[:3]
	alpha, ok := parseAlpha4(tokens[3:])
	if !ok {
		return Color{}, false
	}
	switch {
	case name == "rgb" || name == "rgba":
		return parseRGB(args, alpha)
	case name == "hsl" || name == "hsla":
		return parseHSL(args, alpha)
	case name == "hwb":
		return parseHWB(args, alpha)
	case name == "lab" && !oldSyntax:
		return parseLab(args, alpha)
	case name == "lch" && !oldSyntax:
		return parseLCH(args, alpha)
	case name == "oklab" && !oldSyntax:
		return parseOklab(args, alpha)
	case name == "oklch" && !oldSyntax:
		return parseOklch(args, alpha)
	case name == "color" && !oldSyntax:
		return parseColorSpace(space, args, alpha)
	}
	return Color{}, false
}

func parseRGB(args []Node, alpha float64) (Color, bool) {
	if !typesOf(args).uniform() {
		return Color{}, false
	}
	coords := make([]float64, len(args))
	for i, a := range args {
		switch {
		case isNone(a):
			coords[i] = none
		case a.Type() == "number":
			coords[i], _ = numeric(a)
			coords[i] /= 255
		default:
			coords[i], _ = numeric(a)
			coords[i] /= 100
		}
	}
	return NewColor(SRGB, coords, alpha), true
}

func parseHSL(args []Node, alpha float64) (Color, bool) {
	if !typesOf(args[1:]).uniform() {
		return Color{}, false
	}
	return hueColor(HSL, args, alpha)
}

func parseHWB(args []Node, alpha float64) (Color, bool) {
	if !typesOf(args[1:]).numbersOrPercentages() {
		return Color{}, false
	}
	return hueColor(HWB, args, alpha)
}

func hueColor(space string, args []Node, alpha float64) (Color, bool) {
	hue, ok := parseHue(args[0])
	if !ok {
		return Color{}, false
	}
	coords := make([]float64, 3)
	coords[0] = hue
	if isIdent(args[0]) {
		coords[0] = none
	}
	for i := 1; i < 3; i++ {
		if isIdent(args[i]) {
			coords[i] = none
		} else {
			coords[i], _ = numeric(args[i])
		}
	}
	return NewColor(space, coords, alpha), true
}

func parseLab(args []Node, alpha float64) (Color, bool) {
	if !typesOf(args).numbersOrPercentages() {
		return Color{}, false
	}
	return NewColor(Lab, []float64{scaled(args[0], 1), scaled(args[1], 1.25), scaled(args[2], 1.25)}, alpha), true
}

func parseOklab(args []Node, alpha float64) (Color, bool) {
	if !typesOf(args).numbersOrPercentages() {
		return Color{}, false
	}
	return NewColor(Oklab, []float64{scaled(args[0], 0.01), scaled(args[1], 0.004), scaled(args[2], 0.004)}, alpha), true
}

// polarColor builds lch and oklch colors. The hue is undefined when the
// lightness is, which is how the reference behaviour treats both.
func polarColor(space string, args []Node, alpha, lightnessPct, chromaPct float64) (Color, bool) {
	if !typesOf(args[:2]).numbersOrPercentages() {
		return Color{}, false
	}
	hue, ok := parseHue(args[2])
	if !ok {
		return Color{}, false
	}
	if isIdent(args[0]) {
		hue = none
	}
	return NewColor(space, []float64{scaled(args[0], lightnessPct), scaled(args[1], chromaPct), hue}, alpha), true
}

func parseLCH(args []Node, alpha float64) (Color, bool) {
	return polarColor(LCH, args, alpha, 1, 1.5)
}

func parseOklch(args []Node, alpha float64) (Color, bool) {
	return polarColor(Oklch, args, alpha, 0.01, 0.004)
}

func parseColorSpace(space Node, args []Node, alpha float64) (Color, bool) {
	if !typesOf(args).numbersOrPercentages() {
		return Color{}, false
	}
	id, ok := space.(*IdentToken)
	if !ok || !functionSpaces[id.LowerValue] {
		return Color{}, false
	}
	name := id.LowerValue
	if name == "xyz" {
		name = XYZD65
	}
	return NewColor(name, scaledAll(args), alpha), true
}

// scaledAll returns the values of number and percentage tokens, with
// percentages mapped to [0, 1]; NaN for none.
func scaledAll(args []Node) []float64 {
	out := make([]float64, len(args))
	for i, a := range args {
		switch v, isPct := numeric(a); {
		case isNone(a):
			out[i] = none
		case isPct:
			out[i] = v / 100
		default:
			out[i] = v
		}
	}
	return out
}

// ParseColor parses a color as defined by CSS Color Level 5 as supported
// here: everything of Level 4, plus device-cmyk(), color() with custom
// "--name" profiles and light-dark(). colorSchemes selects the branch of
// light-dark(): the first recognized scheme wins and "light" is the default;
// nil or the single value "normal" also give "light".
func ParseColor(token Node, colorSchemes ...string) (Color, bool) {
	if c, ok := ParseColor4(token); ok {
		return c, true
	}
	scheme := "light"
	if !(len(colorSchemes) == 1 && colorSchemes[0] == "normal") {
		for _, s := range colorSchemes {
			if ColorSchemes[s] {
				scheme = s
				break
			}
		}
	}

	fn, ok := token.(*FunctionBlock)
	if !ok {
		return Color{}, false
	}
	tokens := significant(fn.Arguments)
	name := fn.LowerName
	var alphaTokens []Node
	var space Node
	if name == "color" {
		if len(tokens) == 0 {
			return Color{}, false
		}
		space, tokens = tokens[0], tokens[1:]
	}
	oldSyntax := allCommas(everyOther(tokens, 1))
	if oldSyntax {
		tokens = everyOther(tokens, 0)
	} else {
		for i, t := range tokens {
			if IsLiteral(t, "/") {
				alphaTokens = tokens[i+1:]
				tokens = tokens[:i]
				break
			}
		}
	}

	switch name {
	case "device-cmyk":
		alpha, ok := parseAlpha4(alphaTokens)
		if !ok {
			return Color{}, false
		}
		return parseDeviceCMYK(tokens, alpha, oldSyntax)
	case "color":
		alpha, ok := parseAlpha4(alphaTokens)
		if !ok {
			return Color{}, false
		}
		return parseCustomSpace(space, tokens, alpha)
	case "light-dark":
		return parseLightDark(tokens, scheme)
	}
	return Color{}, false
}

func parseDeviceCMYK(args []Node, alpha float64, oldSyntax bool) (Color, bool) {
	s := typesOf(args)
	if oldSyntax {
		if !(s.number && !s.percentage && !s.other) {
			return Color{}, false
		}
	} else if !s.numbersOrPercentages() {
		return Color{}, false
	}
	if len(args) != 4 {
		return Color{}, false
	}
	coords := make([]float64, 4)
	for i, a := range args {
		if isNone(a) {
			// An undefined channel has no value to clip.
			return Color{}, false
		}
		coords[i] = math.Max(0, math.Min(1, scaledAll([]Node{a})[0]))
	}
	return NewColor(DeviceCMYK, coords, alpha), true
}

func parseLightDark(args []Node, scheme string) (Color, bool) {
	var colors []Color
	for _, a := range args {
		// Nested arguments are always resolved for the light scheme.
		if c, ok := ParseColor(a); ok {
			colors = append(colors, c)
		}
	}
	if len(colors) != 2 {
		return Color{}, false
	}
	if scheme == "light" {
		return colors[0], true
	}
	return colors[1], true
}

func parseCustomSpace(space Node, args []Node, alpha float64) (Color, bool) {
	if !typesOf(args).numbersOrPercentages() {
		return Color{}, false
	}
	id, ok := space.(*IdentToken)
	if !ok || !strings.HasPrefix(id.Value, "--") {
		return Color{}, false
	}
	return NewColor(id.Value, scaledAll(args), alpha), true
}

// ParseColorString parses a color from CSS text; see ParseColor.
func ParseColorString(css string, colorSchemes ...string) (Color, bool) {
	return ParseColor(ParseOneComponentValue(css, true), colorSchemes...)
}

// ParseColor4String parses a color from CSS text; see ParseColor4.
func ParseColor4String(css string) (Color, bool) {
	return ParseColor4(ParseOneComponentValue(css, true))
}
