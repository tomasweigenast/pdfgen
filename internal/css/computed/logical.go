package computed

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// logicalName returns the logical property that sets the physical property
// id, or "" when there is none (CSS Logical Properties 1 §4). The block flow
// direction is always top to bottom, as writing-mode is not supported; the
// inline direction comes from direction.
func logicalName(id props.PropID, direction string) string {
	name := id.Name()

	prefix, rest := "", name
	for _, p := range []string{"max-", "min-"} {
		if strings.HasPrefix(name, p) {
			prefix, rest = p, name[len(p):]
		}
	}
	switch rest {
	case "width":
		return prefix + "inline-size"
	case "height":
		return prefix + "block-size"
	}
	if prefix != "" {
		return ""
	}

	switch name {
	case "border-top-left-radius":
		return "border-start-" + inlineSide("left", direction) + "-radius"
	case "border-top-right-radius":
		return "border-start-" + inlineSide("right", direction) + "-radius"
	case "border-bottom-left-radius":
		return "border-end-" + inlineSide("left", direction) + "-radius"
	case "border-bottom-right-radius":
		return "border-end-" + inlineSide("right", direction) + "-radius"
	}

	prefix, side, suffix := "inset", name, ""
	for _, p := range []string{"padding", "margin", "border"} {
		if strings.HasPrefix(name, p+"-") {
			prefix, side = p, name[len(p)+1:]
		}
	}
	if prefix == "border" {
		var ok bool
		side, suffix, ok = strings.Cut(side, "-")
		if !ok || (suffix != "width" && suffix != "style" && suffix != "color") {
			return ""
		}
		suffix = "-" + suffix
	}
	switch side {
	case "top":
		return prefix + "-block-start" + suffix
	case "bottom":
		return prefix + "-block-end" + suffix
	case "left", "right":
		return prefix + "-inline-" + inlineSide(side, direction) + suffix
	}
	return ""
}

// inlineSide returns whether the physical side is the start or the end of
// the inline direction.
func inlineSide(side, direction string) string {
	if (side == "left") == (direction != "rtl") {
		return "start"
	}
	return "end"
}
