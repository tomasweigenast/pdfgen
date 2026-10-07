package images

// orient returns a plane of n interleaved samples per pixel rotated
// counter-clockwise by angle degrees (0, 90, 180 or 270) and then mirrored
// horizontally when flip is set, with its new width and height.
func orient(plane []byte, width, height, n, angle int, flip bool) ([]byte, int, int) {
	if angle == 0 && !flip {
		return plane, width, height
	}
	newWidth, newHeight := width, height
	if angle == 90 || angle == 270 {
		newWidth, newHeight = height, width
	}
	out := make([]byte, len(plane))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var nx, ny int
			switch angle {
			case 90:
				nx, ny = y, width-1-x
			case 180:
				nx, ny = width-1-x, height-1-y
			case 270:
				nx, ny = height-1-y, x
			default:
				nx, ny = x, y
			}
			if flip {
				nx = newWidth - 1 - nx
			}
			copy(out[(ny*newWidth+nx)*n:], plane[(y*width+x)*n:(y*width+x+1)*n])
		}
	}
	return out, newWidth, newHeight
}

// exifTransform maps an Exif orientation value to the rotation that
// displays the image upright (Exif 2.32 §4.6.4, tag 274).
func exifTransform(orientation int) (angle int, flip bool) {
	switch orientation {
	case 2:
		return 0, true
	case 3:
		return 180, false
	case 4:
		return 180, true
	case 5:
		return 270, true
	case 6:
		return 270, false
	case 7:
		return 90, true
	case 8:
		return 90, false
	}
	return 0, false
}
