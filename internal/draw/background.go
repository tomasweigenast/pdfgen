package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawBackground paints the background color and image layers of bg. The
// painting is clipped to the boxes of the last layer when clipBox is set,
// and always to the painting area. bleed and marks are the bleed area of a
// page and its crop marks and crosses.
func drawBackground(s *stream.Stream, bg *boxes.Background, clipBox bool, bleed *boxes.Sides, marks props.Keywords) {
	if bg == nil || len(bg.Layers) == 0 {
		return
	}
	last := bg.Layers[len(bg.Layers)-1]
	drawBoxShadows(s, last.ClippedBoxes, bg.Style, false)

	s.PushState()
	if clipBox {
		for _, b := range last.ClippedBoxes {
			roundedBox(s, b)
		}
		s.Clip(false)
		s.EndPath()
	}

	if bg.Color.Alpha > 0 {
		area := last.PaintingArea
		s.PushState()
		s.SetColor(bg.Color.Color, false)
		s.Rectangle(area.X, area.Y, area.Width, area.Height)
		s.Clip(false)
		s.EndPath()
		s.Rectangle(area.X, area.Y, area.Width, area.Height)
		s.Fill(false)
		s.PopState()
	}

	// The first layer is the closest to the viewer, so layers are painted
	// in reverse order.
	var currentColor syntax.Color
	if bg.Style != nil {
		currentColor = bg.Style.ResolvedColor(props.PropColor).Color
	}
	rendering := "auto"
	if bg.Style != nil {
		rendering = string(bg.Style.ImageRendering())
	}
	for i := len(bg.Layers) - 1; i >= 0; i-- {
		layer := &bg.Layers[i]
		img := layer.Image
		if img == nil {
			if g, ok := images.GradientFromProps(layer.Source, currentColor); ok {
				img = g
			}
		}
		drawBackgroundImage(s, layer, img, rendering)
	}
	if bleed != nil && len(marks) > 0 {
		drawPageMarks(s, last.PaintingArea, *bleed, marks)
	}
	s.PopState()

	drawBoxShadows(s, last.ClippedBoxes, bg.Style, true)
}

func drawBackgroundImage(s *stream.Stream, layer *boxes.BackgroundLayer, img images.Image, rendering string) {
	if img == nil || layer.Size[0] == 0 || layer.Size[1] == 0 {
		return
	}
	paint := layer.PaintingArea
	pos := layer.PositioningArea
	posX, posY := layer.Position[0], layer.Position[1]
	imgW, imgH := layer.Size[0], layer.Size[1]
	repeatX, repeatY := layer.Repeat.X, layer.Repeat.Y

	if repeatX == "no-repeat" && repeatY == "no-repeat" {
		// A pattern is avoided when not needed: some viewers pixelate
		// vector images drawn through patterns.
		s.PushState()
		if !layer.Unbounded {
			s.Rectangle(paint.X, paint.Y, paint.Width, paint.Height)
			s.Clip(false)
			s.EndPath()
		}
		r := s.PageRectangle
		group := s.AddGroup(r[0], r[1], r[2], r[3])
		group.Transform(1, 0, 0, 1, posX+pos.X, posY+pos.Y)
		group.PushState()
		group.Rectangle(0, 0, imgW, imgH)
		group.Clip(false)
		group.EndPath()
		img.Draw(group, imgW, imgH, rendering)
		group.PopState()
		s.DrawXObject(group.ID())
		s.PopState()
		return
	}

	repeatW, posX := repeatStep(repeatX, imgW, paint.Width, pos.Width, posX)
	repeatH, posY := repeatStep(repeatY, imgH, paint.Height, pos.Height, posY)

	m := matrix.New(1, 0, 0, 1, posX+pos.X, posY+pos.Y).Mul(s.CTM())
	pattern := s.AddPattern(0, 0, imgW, imgH, repeatW, repeatH, m)
	group := pattern.AddGroup(0, 0, repeatW, repeatH)

	s.PushState()
	img.Draw(group, imgW, imgH, rendering)
	pattern.DrawXObject(group.ID())
	s.SetColorSpace("Pattern", false)
	s.SetColorSpecial(pattern.ID(), false)
	if layer.Unbounded {
		r := s.PageRectangle
		s.Rectangle(r[0], r[1], r[2], r[3])
	} else {
		s.Rectangle(paint.X, paint.Y, paint.Width, paint.Height)
	}
	s.Fill(false)
	s.PopState()
}

// repeatStep returns the distance between two tiles along one axis and the
// background position for that axis, which space ignores when it repeats.
func repeatStep(repeat props.Keyword, size, paintingSize, positioningSize, position float64) (step, pos float64) {
	switch repeat {
	case "no-repeat":
		// Doubling the painting size keeps viewers from bleeding the edge
		// of the tile into the painting area.
		return math.Max(size, 2*paintingSize), position
	case "space":
		n := math.Floor(positioningSize / size)
		if n >= 2 {
			return (positioningSize - size) / (n - 1), 0
		}
		return positioningSize, position
	default:
		return size, position
	}
}
