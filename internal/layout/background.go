package layout

import (
	"fmt"
	"math"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

type backgroundOptions struct {
	Children bool
	Style    *computed.Style
}

type backgroundLayerStyle struct {
	Resolution float64
	Size       props.BackgroundSize
	Clip       props.Keyword
	Repeat     props.Repeat
	Origin     props.Keyword
	Position   props.Position
	Attachment props.Keyword
}

func boxRectangle(box boxes.Box, rectangle props.Keyword) boxes.Rect {
	b := box.Base()
	switch rectangle {
	case "border-box":
		return boxes.Rect{X: b.BorderBoxX(), Y: b.BorderBoxY(), Width: b.BorderWidth(), Height: b.BorderHeight()}
	case "padding-box":
		return boxes.Rect{X: b.PaddingBoxX(), Y: b.PaddingBoxY(), Width: b.PaddingWidth(), Height: b.PaddingHeight()}
	}
	return boxes.Rect{X: b.ContentBoxX(), Y: b.ContentBoxY(), Width: b.Width.V, Height: b.Height.V}
}

func layoutBoxBackgrounds(page *boxes.PageBox, box boxes.Box, getImage images.Getter, options backgroundOptions) error {
	resolveRadiiPercentages(box)
	if options.Children {
		for _, child := range boxes.AllChildren(box) {
			if err := layoutBoxBackgrounds(page, child, getImage, backgroundOptions{Children: true}); err != nil {
				return err
			}
		}
	}
	b := box.Base()
	style := options.Style
	if style == nil {
		style = b.Style
	}
	var err error
	if source := style.BorderImageSource(); source.Type != "none" {
		b.BorderImage, err = loadBackgroundImage(source, getImage, images.OrientationFromImage)
		if err != nil {
			return err
		}
	}
	if source := style.MaskBorderSource(); source.Type != "none" {
		b.MaskBorderImage, err = loadBackgroundImage(source, getImage, images.OrientationFromImage)
		if err != nil {
			return err
		}
	}

	layers := []boxes.BackgroundLayer{}
	color := style.ResolvedColor(props.PropBackgroundColor)
	if style.Visibility() == "hidden" {
		color = props.PropBackgroundColor.Initial().(props.Color)
	} else {
		orientation := backgroundOrientation(style)
		for _, source := range style.BackgroundImage() {
			img, err := loadBackgroundImage(source, getImage, orientation)
			if err != nil {
				return err
			}
			layers = append(layers, boxes.BackgroundLayer{Source: source, Image: img})
		}
	}

	hasImage := false
	for _, layer := range layers {
		if backgroundHasImage(layer) {
			hasImage = true
			break
		}
	}
	if color.Alpha == 0 && !hasImage && len(style.BoxShadow()) == 0 && box != page {
		b.Background = nil
		return nil
	}

	sizes, clips := style.BackgroundSize(), style.BackgroundClip()
	repeats, origins := style.BackgroundRepeat(), style.BackgroundOrigin()
	positions, attachments := style.BackgroundPosition(), style.BackgroundAttachment()
	for i, layer := range layers {
		layers[i] = layoutBackgroundLayer(box, page, layer, backgroundLayerStyle{
			Resolution: float64(style.ImageResolution()),
			Size:       sizes[i%len(sizes)],
			Clip:       clips[i%len(clips)],
			Repeat:     repeats[i%len(repeats)],
			Origin:     origins[i%len(origins)],
			Position:   positions[i%len(positions)],
			Attachment: attachments[i%len(attachments)],
		})
	}
	b.Background = &boxes.Background{Color: color, Layers: layers, Style: style}
	return nil
}

func loadBackgroundImage(source props.Image, getImage images.Getter, orientation images.Orientation) (images.Image, error) {
	if source.Type != "url" {
		return nil, nil
	}
	if getImage == nil {
		return nil, fmt.Errorf("load background image %q: no image getter", source.URL)
	}
	img, err := getImage(source.URL, "", orientation)
	if err != nil {
		return nil, fmt.Errorf("load background image %q: %w", source.URL, err)
	}
	return img, nil
}

func backgroundOrientation(style *computed.Style) images.Orientation {
	o := style.ImageOrientation()
	switch o.Keyword {
	case "from-image":
		return images.OrientationFromImage
	case "none":
		return images.Orientation{}
	}
	return images.Orientation{Angle: int(o.Angle), Flip: o.Flip}
}

func backgroundHasImage(layer boxes.BackgroundLayer) bool {
	return layer.Image != nil || layer.Source.Linear != nil || layer.Source.Radial != nil
}

func layoutBackgroundLayer(
	box boxes.Box,
	page *boxes.PageBox,
	layer boxes.BackgroundLayer,
	style backgroundLayerStyle,
) boxes.BackgroundLayer {
	layer.PaintingArea, layer.ClippedBoxes = backgroundPaintingArea(box, page, style.Clip)
	width, height, ratio := units.Auto, units.Auto, units.Auto
	if layer.Image != nil {
		width, height, ratio = layer.Image.IntrinsicSize(style.Resolution, float64(box.Base().Style.FontSize()))
	}
	zeroWidth := !width.Auto && width.V == 0
	zeroHeight := !height.Auto && height.V == 0
	if !backgroundHasImage(layer) || zeroWidth || zeroHeight {
		layer.Image = nil
		return layer
	}

	if style.Attachment == "fixed" {
		// CSS Paged Media 3 §3: fixed page images include the page margins.
		if box == page {
			layer.PositioningArea = boxes.Rect{Width: page.MarginWidth(), Height: page.MarginHeight()}
		} else {
			layer.PositioningArea = boxRectangle(page, "content-box")
		}
	} else {
		layer.PositioningArea = boxRectangle(box, style.Origin)
	}
	area := layer.PositioningArea
	var imageWidth, imageHeight float64
	switch style.Size.Keyword {
	case "cover":
		imageWidth, imageHeight = coverConstraintImageSizing(area.Width, area.Height, ratio)
	case "contain":
		imageWidth, imageHeight = containConstraintImageSizing(area.Width, area.Height, ratio)
	default:
		imageWidth, imageHeight = defaultImageSizing(
			width,
			height,
			ratio,
			percentage(style.Size.Width, box.Base().Style, units.F(area.Width)),
			percentage(style.Size.Height, box.Base().Style, units.F(area.Height)),
			area.Width,
			area.Height,
		)
	}

	position := style.Position
	refX, refY := area.Width-imageWidth, area.Height-imageHeight
	x := percentage(position.X, box.Base().Style, units.F(refX)).Or(0)
	y := percentage(position.Y, box.Base().Style, units.F(refY)).Or(0)
	if position.OriginX == "right" {
		x = refX - x
	}
	if position.OriginY == "bottom" {
		y = refY - y
	}
	if style.Repeat.X == "round" {
		n := math.Max(1, math.RoundToEven(area.Width/imageWidth))
		newWidth := area.Width / n
		x = 0
		if style.Repeat.Y != "round" && style.Size.Height.IsAuto() {
			imageHeight *= newWidth / imageWidth
		}
		imageWidth = newWidth
	}
	if style.Repeat.Y == "round" {
		n := math.Max(1, math.RoundToEven(area.Height/imageHeight))
		newHeight := area.Height / n
		y = 0
		if style.Repeat.X != "round" && style.Size.Width.IsAuto() {
			imageWidth *= newHeight / imageHeight
		}
		imageHeight = newHeight
	}
	layer.Size = [2]float64{imageWidth, imageHeight}
	layer.Position = [2]float64{x, y}
	layer.Repeat = style.Repeat
	return layer
}

func backgroundPaintingArea(box boxes.Box, page *boxes.PageBox, clip props.Keyword) (boxes.Rect, []boxes.RoundedBox) {
	if box == page {
		// CSS Paged Media 3 §3: the page background always extends to bleed.
		return page.BleedArea(), []boxes.RoundedBox{}
	}
	b := box.Base()
	clipped := []boxes.RoundedBox{}
	switch box.Kind() {
	case boxes.KindTableRowGroup:
		var height float64
		for _, row := range boxes.ChildrenOf(box) {
			for _, cell := range boxes.ChildrenOf(row) {
				clipped = append(clipped, cell.Base().RoundedBorderBox())
				height = firstMax(height, cell.Base().BorderHeight())
			}
		}
		return boxes.Rect{X: b.BorderBoxX(), Y: b.BorderBoxY(), Width: b.BorderWidth(), Height: height}, clipped
	case boxes.KindTableRow:
		children := boxes.ChildrenOf(box)
		if len(children) == 0 {
			return boxes.Rect{}, clipped
		}
		height := children[0].Base().BorderHeight()
		for _, cell := range children {
			clipped = append(clipped, cell.Base().RoundedBorderBox())
			height = firstMax(height, cell.Base().BorderHeight())
		}
		return boxes.Rect{X: b.BorderBoxX(), Y: b.BorderBoxY(), Width: b.BorderWidth(), Height: height}, clipped
	case boxes.KindTableColumnGroup, boxes.KindTableColumn:
		columns := box.(interface{ Cells() []*boxes.TableCellBox })
		cells := columns.Cells()
		if len(cells) == 0 {
			return boxes.Rect{}, clipped
		}
		minX := cells[0].BorderBoxX()
		maxX := minX + cells[0].BorderWidth()
		for _, cell := range cells {
			clipped = append(clipped, cell.RoundedBorderBox())
			minX = firstMin(minX, cell.BorderBoxX())
			maxX = firstMax(maxX, cell.BorderBoxX()+cell.BorderWidth())
		}
		return boxes.Rect{X: minX, Y: b.BorderBoxY(), Width: maxX - minX, Height: b.BorderHeight()}, clipped
	}
	area := boxRectangle(box, clip)
	switch clip {
	case "border-box":
		clipped = append(clipped, b.RoundedBorderBox())
	case "padding-box":
		clipped = append(clipped, b.RoundedPaddingBox())
	default:
		clipped = append(clipped, b.RoundedContentBox())
	}
	return area, clipped
}

func layoutBackgrounds(page *boxes.PageBox, getImage images.Getter) error {
	if err := layoutBoxBackgrounds(page, page, getImage, backgroundOptions{Children: true}); err != nil {
		return err
	}
	if len(page.Children) == 0 {
		page.CanvasBackground = nil
		return nil
	}
	root := page.Children[0]
	chosen := root
	if strings.EqualFold(root.Base().ElementTag, "html") && root.Base().Background == nil {
		for _, child := range boxes.ChildrenOf(root) {
			if strings.EqualFold(child.Base().ElementTag, "body") {
				chosen = child
				break
			}
		}
	}
	if chosen.Base().Background == nil {
		page.CanvasBackground = nil
		return nil
	}
	original := page.Background
	if err := layoutBoxBackgrounds(page, page, getImage, backgroundOptions{Style: chosen.Base().Style}); err != nil {
		return err
	}
	canvas := *page.Background
	canvas.Layers = append([]boxes.BackgroundLayer{}, canvas.Layers...)
	area := boxRectangle(page, "border-box")
	for i := range canvas.Layers {
		canvas.Layers[i].PaintingArea = area
	}
	page.CanvasBackground = &canvas
	page.Background = original
	chosen.Base().Background = nil
	return nil
}
