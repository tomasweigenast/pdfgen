package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// clipBorderSegment restricts the painting of s to the segment of a border
// that belongs to one side. border is the border box, widths the widths of
// the four sides (top, right, bottom, left) and radii the radii of its
// corners (top-left, top-right, bottom-right, bottom-left). A border of
// uniform width and square corners passes four equal widths and zero radii.
//
// The segment is the quadrilateral that joins the corners of the border box
// to the transition points of the adjacent sides. For dotted and dashed
// styles it is cut once more to the dots or dashes, spaced so that the
// corners are not left empty.
func clipBorderSegment(s *stream.Stream, style props.Keyword, side string, border boxes.Rect, widths [4]float64, radii [4]boxes.Radius) {
	bbx, bby, bbw, bbh := border.X, border.Y, border.Width, border.Height
	tl, tr, br, bl := radii[0], radii[1], radii[2], radii[3]
	bt, bright, bb, bleft := widths[0], widths[1], widths[2], widths[3]

	var (
		width              float64
		px1, py1, px2, py2 float64
		rounded1, rounded2 bool
		way, angle         float64
		mainOffset         float64
	)
	switch side {
	case "top":
		px1, py1, rounded1 = transitionPoint(tl.X, tl.Y, bleft, bt)
		px2, py2, rounded2 = transitionPoint(-tr.X, tr.Y, -bright, bt)
		width, way, angle, mainOffset = bt, 1, 1, bby
	case "right":
		px1, py1, rounded1 = transitionPoint(-tr.X, tr.Y, -bright, bt)
		px2, py2, rounded2 = transitionPoint(-br.X, -br.Y, -bright, -bb)
		width, way, angle, mainOffset = bright, 1, 2, bbx+bbw
	case "bottom":
		px1, py1, rounded1 = transitionPoint(bl.X, -bl.Y, bleft, -bb)
		px2, py2, rounded2 = transitionPoint(-br.X, -br.Y, -bright, -bb)
		width, way, angle, mainOffset = bb, -1, 3, bby+bbh
	case "left":
		px1, py1, rounded1 = transitionPoint(tl.X, tl.Y, bleft, bt)
		px2, py2, rounded2 = transitionPoint(bl.X, -bl.Y, bleft, -bb)
		width, way, angle, mainOffset = bleft, -1, 4, bbx
	default:
		return
	}

	horizontal := side == "top" || side == "bottom"
	var a1, b1, a2, b2, lineLength, length float64
	if horizontal {
		a1, b1 = px1-bleft/2, way*py1-width/2
		a2, b2 = -px2-bright/2, way*py2-width/2
		lineLength = bbw - px1 + px2
		length = bbw
		s.MoveTo(bbx+bbw, mainOffset)
		s.LineTo(bbx, mainOffset)
		s.LineTo(bbx+px1, mainOffset+py1)
		s.LineTo(bbx+bbw+px2, mainOffset+py2)
	} else {
		a1, b1 = -way*px1-width/2, py1-bt/2
		a2, b2 = -way*px2-width/2, -py2-bb/2
		lineLength = bbh - py1 + py2
		length = bbh
		s.MoveTo(mainOffset, bby+bbh)
		s.LineTo(mainOffset, bby)
		s.LineTo(mainOffset+px1, bby+py1)
		s.LineTo(mainOffset+px2, bby+bbh+py2)
	}

	if style == "dotted" || style == "dashed" {
		s.Clip(true)
		s.EndPath()
		d := dashedSegment{
			s: s, style: style, horizontal: horizontal,
			width: width, angle: angle, mainOffset: mainOffset,
		}
		dash := width
		if style == "dashed" {
			dash = 3 * width
		}
		if rounded1 || rounded2 {
			d.roundedCorners(border, way, dash, lineLength,
				cornerHalfLength(a1, b1), cornerHalfLength(a2, b2),
				px1, py1, px2, py2, rounded1 && rounded2)
		} else {
			d.straight(border, side, length, dash)
		}
	}
	s.Clip(true)
	s.EndPath()
}

// transitionPoint returns the point where the borders of two adjacent sides
// meet. It must lie inside the rounded padding box and in the transition
// zone that CSS Backgrounds 3 §5.2 allows; the corner of that zone is easy
// to get and gives good results. The result is true when the point is in the
// padding box, that is when the padding box is rounded.
func transitionPoint(x1, y1, x2, y2 float64) (x, y float64, rounded bool) {
	if math.Abs(x1) > math.Abs(x2) && math.Abs(y1) > math.Abs(y2) {
		return x1, y1, true
	}
	return x2, y2, false
}

// cornerHalfLength returns the length of half of an elliptical corner with
// semi-axes a and b, using Ramanujan's approximation of the perimeter of an
// ellipse.
func cornerHalfLength(a, b float64) float64 {
	x := (a - b) / (a + b)
	return math.Pi / 8 * (a + b) * (1 + 3*x*x/(10+math.Sqrt(4-3*x*x)))
}

// dashedSegment adds the dots or dashes of one side of a dotted or dashed
// border to the current path of s.
type dashedSegment struct {
	s          *stream.Stream
	style      props.Keyword
	horizontal bool
	width      float64
	angle      float64
	mainOffset float64
}

// dash adds a dot of radius r, or a dashed rectangle of the given size,
// centered on (cx, cy).
func (d *dashedSegment) dash(cx, cy, width, height, r float64) {
	switch d.style {
	case "dotted":
		ratio := r / math.Sqrt(math.Pi)
		d.s.MoveTo(cx+r, cy)
		d.s.CurveTo(cx+r, cy+ratio, cx+ratio, cy+r, cx, cy+r)
		d.s.CurveTo(cx-ratio, cy+r, cx-r, cy+ratio, cx-r, cy)
		d.s.CurveTo(cx-r, cy-ratio, cx-ratio, cy-r, cx, cy-r)
		d.s.CurveTo(cx+ratio, cy-r, cx+r, cy-ratio, cx+r, cy)
		d.s.ClosePath()
	case "dashed":
		d.s.Rectangle(cx-width/2, cy-height/2, width, height)
	}
}

// straight spaces the dashes evenly along a side with square corners, with a
// dash on each corner.
func (d *dashedSegment) straight(border boxes.Rect, side string, length, dash float64) {
	bbx, bby, bbw, bbh := border.X, border.Y, border.Width, border.Height
	width := d.width
	spaces := math.Floor(length / dash / 2)
	dashes := spaces + 1
	var space float64
	if d.style == "dotted" {
		dash = width
		if spaces != 0 {
			space = (length - dashes*dash) / spaces
		}
	} else {
		space = length / (spaces + dashes)
		if space == 0 {
			space = 1
		}
		dash = space
	}
	if !finite(dash, space) {
		return
	}
	for i := 0.; i < dashes+1; i++ {
		advance := i * (space + dash)
		switch side {
		case "top":
			d.dash(bbx+advance+dash/2, bby+width/2, dash, width, dash/2)
		case "right":
			d.dash(bbx+bbw-width/2, bby+advance+dash/2, width, dash, dash/2)
		case "bottom":
			d.dash(bbx+advance+dash/2, bby+bbh-width/2, dash, width, dash/2)
		case "left":
			d.dash(bbx+width/2, bby+advance+dash/2, width, dash, dash/2)
		}
	}
}

// roundedCorners adds the dashes of a side where at least one corner is
// rounded. The dashes follow the arc of each rounded corner, with chl1 and
// chl2 the half lengths of the corners, and then continue along the straight
// part of the side. The period is adjusted so that dashes are symmetric:
// twice as many as the period when both corners are rounded, else half a
// period less.
func (d *dashedSegment) roundedCorners(border boxes.Rect, way, dash, lineLength, chl1, chl2, px1, py1, px2, py2 float64, bothRounded bool) {
	bbx, bby, bbw, bbh := border.X, border.Y, border.Width, border.Height
	length := lineLength + chl1 + chl2
	count := math.RoundToEven(length / dash)
	if bothRounded {
		dash = length / (count + math.Mod(count, 2))
	} else {
		dash = length / (count + math.Mod(count, 2) - 0.5)
	}
	if !finite(dash, chl1, chl2) || dash <= 0 {
		return
	}
	dashes1 := int(math.Ceil((chl1 - dash/2) / dash))
	dashes2 := int(math.Ceil((chl2 - dash/2) / dash))
	line := int(math.Floor(lineLength / dash))

	line, offset := d.cornerDashes(dashes1, line, way, dash, bbx, bby, px1, py1, chl1)
	line, _ = d.cornerDashes(dashes2, line, -way, dash, bbx+bbw, bby+bbh, px2, py2, chl2)

	if lineLength <= 1e-6 {
		return
	}
	width := d.width
	for i := 0; i < line; i += 2 {
		pos := float64(i) + offset
		var x1, y1, x2, y2 float64
		if d.horizontal {
			x1 = bbx + px1 + pos*dash
			x2 = bbx + px1 + (pos+1)*dash
			y1 = d.mainOffset
			if way < 0 {
				y1 -= width
			}
			y2 = y1 + width
		} else {
			y1 = bby + py1 + pos*dash
			y2 = bby + py1 + (pos+1)*dash
			x1 = d.mainOffset
			if way > 0 {
				x1 -= width
			}
			x2 = x1 + width
		}
		d.dash(x1+(x2-x1)/2, y1+(y2-y1)/2, x2-x1, y2-y1, width/2)
	}
}

// cornerDashes adds the dashes of one rounded corner, whose origin is (x, y)
// and whose transition point is at (px, py) from it. It returns the number
// of dashes that the straight part of the side loses to the corner, and the
// offset in dashes at which the straight part starts.
func (d *dashedSegment) cornerDashes(dashes, line int, way, dash, x, y, px, py, chl float64) (int, float64) {
	angle := d.angle
	if d.style == "dotted" {
		return d.cornerDots(dashes, line, way, dash, x, y, px, py, chl)
	}

	if dashes <= 0 {
		return line + 1, -1. / 3
	}
	var angle1, angle2 float64
	var offset float64
	for i := 0; i < dashes; i += 2 {
		pos := float64(i) + 0.5
		angle1 = ((2*angle - way) + pos*way*dash/chl) / 4 * math.Pi
		angle2 = ((2*angle - way) + (pos+1)*way*dash/chl) / 4 * math.Pi
		if limit := angle * math.Pi / 2; way > 0 {
			angle2 = math.Min(angle2, limit)
		} else {
			angle2 = math.Max(angle2, limit)
		}
		if d.horizontal {
			d.s.MoveTo(x+px, d.mainOffset+py)
			d.s.LineTo(x+px-way*px*1/math.Tan(angle2), d.mainOffset)
			d.s.LineTo(x+px-way*px*1/math.Tan(angle1), d.mainOffset)
		} else {
			d.s.MoveTo(d.mainOffset+px, y+py)
			d.s.LineTo(d.mainOffset, y+py+way*py*math.Tan(angle2))
			d.s.LineTo(d.mainOffset, y+py+way*py*math.Tan(angle1))
		}
		if angle2 == angle*math.Pi/2 {
			next := ((2*angle - way) + (pos+1)*way*dash/chl) / 4 * math.Pi
			return line + 1, (angle1 - angle2) / (next - angle1)
		}
	}
	offset = 1 - (angle*math.Pi/2-angle2)/(angle2-angle1)
	return line, offset
}

// cornerDots is cornerDashes for dotted borders: the dots sit on the arc of
// the corner, every other dash position.
func (d *dashedSegment) cornerDots(dashes, line int, way, dash, x, y, px, py, chl float64) (int, float64) {
	angle := d.angle
	switch {
	case dashes <= 0:
		return line + 1, -1
	case dashes == 1:
		return line + 1, -0.5
	}

	cx, cy := d.mainOffset, d.mainOffset
	if d.horizontal {
		cx = x
	} else {
		cy = y
	}
	last := 1
	for i := 1; i < dashes; i += 2 {
		last = i
		a := ((2*angle - way) + float64(i)*way*dash/chl) / 4 * math.Pi
		d.dash(
			cx+px-(math.Abs(px)-dash/2)*math.Cos(a),
			cy+py-(math.Abs(py)-dash/2)*math.Sin(a),
			0, 0, dash/2)
	}
	next := ((2*angle - way) + float64(last+2)*way*dash/chl) / 4 * math.Pi
	offset := next/math.Pi*2 - angle
	if dashes%2 != 0 {
		line++
	}
	return line, offset
}

func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
