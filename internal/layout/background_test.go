package layout

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func backgroundStyle(t testing.TB, css string) *computed.Style {
	t.Helper()
	doc := csstest.Parse(t, `<div style="`+css+`"></div>`)
	styles := csstest.StyleFor(t, doc, cascade.Options{})
	return styles.Get(csstest.Element(t, doc, "div"), "")
}

func backgroundBox(t testing.TB, css string) *boxes.BlockBox {
	t.Helper()
	box := boxes.NewBlockBox("div", backgroundStyle(t, css), nil, nil)
	box.Width, box.Height = units.F(100), units.F(80)
	return box
}

func backgroundPage() *boxes.PageBox {
	page := boxes.NewPageBox(cascade.PageType{}, computed.New(nil))
	page.Width, page.Height = units.F(200), units.F(100)
	page.MarginTop, page.MarginBottom = units.F(10), units.F(15)
	page.MarginLeft, page.MarginRight = units.F(20), units.F(30)
	return page
}

func backgroundRaster(t testing.TB, width, height int) images.Image {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	raster, err := images.NewRaster("background", data.Bytes(), images.Orientation{}, images.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return raster
}

func backgroundGetter(img images.Image) images.Getter {
	return func(string, string, images.Orientation) (images.Image, error) { return img, nil }
}

func layoutTestBackground(t testing.TB, page *boxes.PageBox, box boxes.Box, getImage images.Getter) {
	t.Helper()
	if err := layoutBoxBackgrounds(page, box, getImage, backgroundOptions{Children: true}); err != nil {
		t.Fatal(err)
	}
}

func TestBackgroundSizingAndPosition(t *testing.T) {
	img := backgroundRaster(t, 40, 20)
	for _, tt := range []struct {
		css  string
		size [2]float64
		pos  [2]float64
	}{
		{"", [2]float64{40, 20}, [2]float64{}},
		{"background-size: cover", [2]float64{160, 80}, [2]float64{}},
		{"background-size: contain", [2]float64{100, 50}, [2]float64{}},
		{"background-size: 60px auto", [2]float64{60, 30}, [2]float64{}},
		{"background-size: auto 30px", [2]float64{60, 30}, [2]float64{}},
		{"background-size: 50% 25%", [2]float64{50, 20}, [2]float64{}},
		{"background-size: calc(50% + 10px) calc(50% - 5px)", [2]float64{60, 35}, [2]float64{}},
		{"background-position: center", [2]float64{40, 20}, [2]float64{30, 30}},
		{"background-position: right 10px bottom 20px", [2]float64{40, 20}, [2]float64{50, 40}},
		{"background-position: calc(50% + 5px) calc(100% - 3px)", [2]float64{40, 20}, [2]float64{35, 57}},
		{"background-size: 120px 100px; background-position: right bottom", [2]float64{120, 100}, [2]float64{-20, -20}},
		{"image-resolution: 2dppx", [2]float64{20, 10}, [2]float64{}},
	} {
		t.Run(tt.css, func(t *testing.T) {
			box := backgroundBox(t, "background-image: url(pattern.png); "+tt.css)
			layoutTestBackground(t, backgroundPage(), box, backgroundGetter(img))
			layer := box.Background.Layers[0]
			if layer.Size != tt.size || layer.Position != tt.pos {
				t.Errorf("size, position = %v, %v; want %v, %v", layer.Size, layer.Position, tt.size, tt.pos)
			}
			if layer.Image != img || layer.Source.Type != "url" {
				t.Errorf("image or source lost: %+v", layer)
			}
		})
	}
}

func TestBackgroundRoundPreservesAutoAspectRatio(t *testing.T) {
	img := backgroundRaster(t, 4, 4)
	for _, tt := range []struct {
		css  string
		w, h float64
		size [2]float64
		pos  [2]float64
	}{
		{"top/6px round repeat", 8, 12, [2]float64{8, 8}, [2]float64{0, 0}},
		{"center/auto 8px repeat round", 8, 16, [2]float64{8, 8}, [2]float64{0, 0}},
		{"center/6px 9px round", 8, 12, [2]float64{8, 12}, [2]float64{0, 0}},
		{"center/5px 9px round", 8, 12, [2]float64{4, 12}, [2]float64{0, 0}},
		{"center/4px round repeat", 10, 8, [2]float64{5, 5}, [2]float64{0, 2}},
		{"center/auto 4px repeat round", 8, 10, [2]float64{5, 5}, [2]float64{2, 0}},
		{"center/4px repeat round", 8, 10, [2]float64{4, 5}, [2]float64{2, 0}},
	} {
		t.Run(tt.css, func(t *testing.T) {
			box := backgroundBox(t, "background: url(pattern.png) "+tt.css)
			box.Width, box.Height = units.F(tt.w), units.F(tt.h)
			layoutTestBackground(t, backgroundPage(), box, backgroundGetter(img))
			layer := box.Background.Layers[0]
			if layer.Size != tt.size || layer.Position != tt.pos {
				t.Errorf("size, position = %v, %v; want %v, %v", layer.Size, layer.Position, tt.size, tt.pos)
			}
		})
	}
}

func TestBackgroundClipAndOrigin(t *testing.T) {
	img := backgroundRaster(t, 4, 4)
	for _, tt := range []struct {
		keyword string
		want    boxes.Rect
	}{
		{"border-box", boxes.Rect{X: 14, Y: 21, Width: 136, Height: 112}},
		{"padding-box", boxes.Rect{X: 26, Y: 30, Width: 114, Height: 92}},
		{"content-box", boxes.Rect{X: 34, Y: 35, Width: 100, Height: 80}},
	} {
		t.Run(tt.keyword, func(t *testing.T) {
			box := backgroundBox(t, "background-image: url(pattern.png); border-radius: 50%; "+
				"background-clip: "+tt.keyword+"; background-origin: "+tt.keyword)
			box.PositionX, box.PositionY = 10, 20
			box.MarginTop, box.MarginRight = units.F(1), units.F(2)
			box.MarginBottom, box.MarginLeft = units.F(3), units.F(4)
			box.PaddingTop, box.PaddingRight, box.PaddingBottom, box.PaddingLeft = 5, 6, 7, 8
			box.BorderTopWidth, box.BorderRightWidth, box.BorderBottomWidth, box.BorderLeftWidth = 9, 10, 11, 12
			layoutTestBackground(t, backgroundPage(), box, backgroundGetter(img))
			layer := box.Background.Layers[0]
			if layer.PaintingArea != tt.want || layer.PositioningArea != tt.want {
				t.Errorf("painting, positioning = %v, %v; want %v", layer.PaintingArea, layer.PositioningArea, tt.want)
			}
			if len(layer.ClippedBoxes) != 1 || layer.ClippedBoxes[0].Rect != tt.want {
				t.Errorf("clipping = %+v, want rectangle %v", layer.ClippedBoxes, tt.want)
			}
			if box.BorderTopLeftRadius != (boxes.Radius{X: 68, Y: 56}) {
				t.Errorf("radius = %v, want {68 56}", box.BorderTopLeftRadius)
			}
		})
	}
}

func TestBackgroundPageBleedAndFixedAttachment(t *testing.T) {
	img := backgroundRaster(t, 4, 4)
	page := backgroundPage()
	page.Style = backgroundStyle(t, "background: url(pattern.png); background-clip: content-box; bleed: 1px 4px 2px 3px")
	layoutTestBackground(t, page, page, backgroundGetter(img))
	layer := page.Background.Layers[0]
	if want := (boxes.Rect{X: -3, Y: -1, Width: 257, Height: 128}); layer.PaintingArea != want {
		t.Errorf("painting = %v, want %v", layer.PaintingArea, want)
	}
	if len(layer.ClippedBoxes) != 0 {
		t.Errorf("page clipping = %v, want none", layer.ClippedBoxes)
	}
	page.Style.Set(props.PropBackgroundAttachment, props.Keywords{"fixed"})
	layoutTestBackground(t, page, page, backgroundGetter(img))
	if got, want := page.Background.Layers[0].PositioningArea, (boxes.Rect{Width: 250, Height: 125}); got != want {
		t.Errorf("fixed page positioning = %v, want %v", got, want)
	}
	box := backgroundBox(t, "background: url(pattern.png); background-attachment: fixed")
	layoutTestBackground(t, page, box, backgroundGetter(img))
	want := boxes.Rect{X: 20, Y: 10, Width: 200, Height: 100}
	if got := box.Background.Layers[0].PositioningArea; got != want {
		t.Errorf("fixed element positioning = %v, want %v", got, want)
	}
}

func TestBackgroundLayersCycleTheirProperties(t *testing.T) {
	box := backgroundBox(t, "background-image: url(a), none, url(b); background-size: 10px, 20px; "+
		"background-repeat: space no-repeat, no-repeat round; background-position: right bottom")
	layoutTestBackground(t, backgroundPage(), box, backgroundGetter(backgroundRaster(t, 4, 4)))
	layers := box.Background.Layers
	if len(layers) != 3 {
		t.Fatalf("%d layers, want 3", len(layers))
	}
	if layers[0].Size != ([2]float64{10, 10}) || layers[2].Size != ([2]float64{10, 10}) {
		t.Errorf("sizes = %v and %v, want {10 10}", layers[0].Size, layers[2].Size)
	}
	if layers[0].Repeat != (props.Repeat{X: "space", Y: "no-repeat"}) || layers[2].Repeat != layers[0].Repeat {
		t.Errorf("repeats = %v and %v", layers[0].Repeat, layers[2].Repeat)
	}
	if layers[1].Image != nil || layers[1].PaintingArea.Width != 100 {
		t.Errorf("empty layer lost its painting area: %+v", layers[1])
	}
}

func TestGradientBackgroundKeepsDescriptorAndGeometry(t *testing.T) {
	for _, gradient := range []string{"linear-gradient(red, blue)", "radial-gradient(red, blue)"} {
		for _, tt := range []struct {
			css  string
			size [2]float64
			pos  [2]float64
		}{
			{"", [2]float64{100, 80}, [2]float64{}},
			{"background-size: contain", [2]float64{100, 80}, [2]float64{}},
			{"background-size: cover", [2]float64{100, 80}, [2]float64{}},
			{"background-size: 50% auto; background-position: center", [2]float64{50, 80}, [2]float64{25, 0}},
			{"background-size: auto 25%; background-position: right bottom", [2]float64{100, 20}, [2]float64{0, 60}},
		} {
			t.Run(gradient+tt.css, func(t *testing.T) {
				box := backgroundBox(t, "background-image: "+gradient+"; "+tt.css)
				layoutTestBackground(t, backgroundPage(), box, nil)
				if box.Background == nil {
					t.Fatal("gradient has no background")
				}
				layer := box.Background.Layers[0]
				if layer.Image != nil || (layer.Source.Linear == nil && layer.Source.Radial == nil) {
					t.Fatalf("gradient descriptor lost: %+v", layer)
				}
				if layer.Size != tt.size || layer.Position != tt.pos {
					t.Errorf("size, position = %v, %v; want %v, %v", layer.Size, layer.Position, tt.size, tt.pos)
				}
			})
		}
	}
}

func TestTransparentBackgroundAndHiddenChildren(t *testing.T) {
	page := backgroundPage()
	box := backgroundBox(t, "")
	layoutTestBackground(t, page, box, nil)
	if box.Background != nil {
		t.Error("transparent element has a background")
	}
	layoutTestBackground(t, page, page, nil)
	if page.Background == nil || page.Background.Color.Alpha != 0 {
		t.Fatal("transparent page lost its background metadata")
	}
	child := backgroundBox(t, "visibility: visible; background: blue")
	box.Style = backgroundStyle(t, "visibility: hidden; background: url(pattern.png) red")
	box.Children = []boxes.Box{child}
	getImage := func(string, string, images.Orientation) (images.Image, error) {
		t.Error("hidden background fetched an image")
		return nil, nil
	}
	layoutTestBackground(t, page, box, getImage)
	if box.Background != nil || child.Background == nil || child.Background.Color.Alpha != 1 {
		t.Errorf("hidden parent or visible child background incorrect: %v, %v", box.Background, child.Background)
	}
}

func TestBackgroundResolvesCurrentColorAndKeepsShadows(t *testing.T) {
	box := backgroundBox(t, "color: blue; background-color: currentcolor")
	layoutTestBackground(t, backgroundPage(), box, nil)
	if got := box.Background.Color; got.Alpha != 1 || !reflect.DeepEqual(got.Coordinates, []float64{0, 0, 1}) {
		t.Errorf("background color = %+v, want blue", got)
	}
	box.Style = backgroundStyle(t, "box-shadow: 1px 2px black")
	layoutTestBackground(t, backgroundPage(), box, nil)
	if box.Background == nil {
		t.Error("transparent box with shadow lost its background")
	}
}

func TestFailedBackgroundImageIsAbsent(t *testing.T) {
	box := backgroundBox(t, "background: url(missing)")
	layoutTestBackground(t, backgroundPage(), box, backgroundGetter(nil))
	if box.Background != nil {
		t.Errorf("failed image created a background: %+v", box.Background)
	}
}

func TestBackgroundImageErrorsStopLayout(t *testing.T) {
	failure := errors.New("fetch failed")
	for _, property := range []string{"background-image", "border-image-source", "mask-border-source"} {
		t.Run(property, func(t *testing.T) {
			box := backgroundBox(t, property+": url(pattern.png)")
			getImage := func(string, string, images.Orientation) (images.Image, error) { return nil, failure }
			err := layoutBoxBackgrounds(backgroundPage(), box, getImage, backgroundOptions{Children: true})
			if !errors.Is(err, failure) {
				t.Errorf("error = %v, want wrapped fetch failure", err)
			}
		})
	}
}

func TestCanvasBackgroundComesFromTheRootOrBody(t *testing.T) {
	for _, tt := range []struct {
		name, rootTag, rootCSS string
		bodyPropagates         bool
		canvas                 bool
	}{
		{"body", "html", "", true, true},
		{"root", "html", "background: red", false, true},
		{"root gradient", "html", "background: linear-gradient(red, blue)", false, true},
		{"failed root image", "html", "background: url(missing)", true, true},
		{"different root", "section", "", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			page := backgroundPage()
			page.Style = backgroundStyle(t, "background: lime")
			body := backgroundBox(t, "background: blue")
			body.ElementTag = "body"
			root := backgroundBox(t, tt.rootCSS)
			root.ElementTag = tt.rootTag
			root.Children = []boxes.Box{body}
			page.Children = []boxes.Box{root}
			if err := layoutBackgrounds(page, backgroundGetter(nil)); err != nil {
				t.Fatal(err)
			}
			if page.Background == nil || !reflect.DeepEqual(page.Background.Color.Coordinates, []float64{0, 1, 0}) {
				t.Fatalf("page background was replaced: %+v", page.Background)
			}
			if (page.CanvasBackground != nil) != tt.canvas {
				t.Fatalf("canvas = %+v, want present %v", page.CanvasBackground, tt.canvas)
			}
			if (body.Background == nil) != tt.bodyPropagates {
				t.Errorf("body background = %+v, want propagated %v", body.Background, tt.bodyPropagates)
			}
			if !tt.canvas {
				return
			}
			canvas := page.CanvasBackground
			want := boxes.Rect{X: 20, Y: 10, Width: 200, Height: 100}
			for _, layer := range canvas.Layers {
				if layer.PaintingArea != want {
					t.Errorf("canvas painting = %+v, want %+v", layer.PaintingArea, want)
				}
			}
			if tt.bodyPropagates && !reflect.DeepEqual(canvas.Color.Coordinates, []float64{0, 0, 1}) {
				t.Errorf("canvas color = %v, want blue", canvas.Color)
			}
			if tt.name == "root gradient" && canvas.Layers[0].Size != ([2]float64{200, 100}) {
				t.Errorf("canvas gradient size = %v, want {200 100}", canvas.Layers[0].Size)
			}
		})
	}
}

func TestBackgroundBorderSourcesAndOrientation(t *testing.T) {
	img := backgroundRaster(t, 4, 4)
	box := backgroundBox(t, "background: url(c); border-image-source: url(a); "+
		"mask-border-source: url(b); image-orientation: 90deg flip; visibility: hidden")
	type request struct {
		name        byte
		orientation images.Orientation
	}
	requests := []request{}
	getImage := func(url, _ string, orientation images.Orientation) (images.Image, error) {
		requests = append(requests, request{name: url[len(url)-1], orientation: orientation})
		return img, nil
	}
	layoutTestBackground(t, backgroundPage(), box, getImage)
	want := []request{{name: 'a', orientation: images.OrientationFromImage}, {name: 'b', orientation: images.OrientationFromImage}}
	if !reflect.DeepEqual(requests, want) || box.BorderImage != img || box.MaskBorderImage != img {
		t.Fatalf("border image requests = %+v; images = %v, %v", requests, box.BorderImage, box.MaskBorderImage)
	}
	box.Style.Set(props.PropVisibility, props.Keyword("visible"))
	requests = requests[:0]
	layoutTestBackground(t, backgroundPage(), box, getImage)
	want = append(want, request{name: 'c', orientation: images.Orientation{Angle: 90, Flip: true}})
	if !reflect.DeepEqual(requests, want) {
		t.Errorf("visible image requests = %+v, want %+v", requests, want)
	}
}

func TestBackgroundErrorsReachThePage(t *testing.T) {
	failure := errors.New("image unavailable")
	page := backgroundPage()
	root := backgroundBox(t, "background-image: url(pattern.png)")
	page.Children = []boxes.Box{root}
	getImage := func(string, string, images.Orientation) (images.Image, error) { return nil, failure }
	if err := layoutBackgrounds(page, getImage); !errors.Is(err, failure) {
		t.Errorf("page background error = %v, want wrapped image error", err)
	}
}

func TestTableBackgroundsClipToTheCells(t *testing.T) {
	style := backgroundStyle(t, "background: red")
	first := boxes.NewTableCellBox("td", style, nil, nil)
	first.PositionX, first.PositionY, first.Width, first.Height = 10, 20, units.F(25), units.F(12)
	second := boxes.NewTableCellBox("td", style, nil, nil)
	second.PositionX, second.PositionY, second.Width, second.Height = 40, 20, units.F(30), units.F(20)
	row := boxes.NewTableRowBox("tr", style, nil, []boxes.Box{first, second})
	row.PositionX, row.PositionY, row.Width, row.Height = 10, 20, units.F(60), units.F(8)
	group := boxes.NewTableRowGroupBox("tbody", style, nil, []boxes.Box{row})
	group.PositionX, group.PositionY, group.Width, group.Height = 10, 20, units.F(60), units.F(8)
	column := boxes.NewTableColumnBox("col", style, nil, nil)
	column.PositionY, column.Height = 20, units.F(8)
	column.GetCells = func() []*boxes.TableCellBox { return []*boxes.TableCellBox{first, second} }
	columns := boxes.NewTableColumnGroupBox("colgroup", style, nil, []boxes.Box{column})
	columns.PositionY, columns.Height = 20, units.F(8)
	for _, tt := range []struct {
		box  boxes.Box
		area boxes.Rect
	}{
		{row, boxes.Rect{X: 10, Y: 20, Width: 60, Height: 20}},
		{group, boxes.Rect{X: 10, Y: 20, Width: 60, Height: 20}},
		{column, boxes.Rect{X: 10, Y: 20, Width: 60, Height: 8}},
		{columns, boxes.Rect{X: 10, Y: 20, Width: 60, Height: 8}},
	} {
		layoutTestBackground(t, backgroundPage(), tt.box, nil)
		layer := tt.box.Base().Background.Layers[0]
		if layer.PaintingArea != tt.area || len(layer.ClippedBoxes) != 2 {
			t.Errorf("%v: area, clipping = %v, %v; want %v, two cells", tt.box.Kind(), layer.PaintingArea, layer.ClippedBoxes, tt.area)
		}
	}
}
