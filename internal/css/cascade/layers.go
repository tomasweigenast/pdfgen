package cascade

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// addLayer adds a layer and its parents to the layer order, a sublayer
// coming just before the layer that contains it (CSS Cascade 5 §6.4.3).
func addLayer(layer string, layers *[]string) {
	index, found := 0, false
	full := ""
	for _, part := range strings.Split(layer, ".") {
		if full != "" {
			full += "."
		}
		full += part
		if i := indexOf(*layers, full); i >= 0 {
			index, found = i, true
			continue
		}
		if !found {
			*layers = append(*layers, full)
			index, found = len(*layers)-1, true
		} else {
			insert(layers, index, full)
			index--
		}
	}
}

// insert inserts s before position i, a negative position counting from
// the end.
func insert(list *[]string, i int, s string) {
	if i < 0 {
		i = max(i+len(*list), 0)
	}
	*list = append(*list, "")
	copy((*list)[i+1:], (*list)[i:])
	(*list)[i] = s
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// parseLayer parses a layer name: identifiers separated by dots.
func parseLayer(tokens []syntax.Node) (string, bool) {
	if len(tokens) == 0 {
		return "", false
	}
	var b strings.Builder
	lastDot := true
	for _, token := range tokens {
		switch t := token.(type) {
		case *syntax.IdentToken:
			if !lastDot {
				return "", false
			}
			b.WriteString(t.Value)
			lastDot = false
		case *syntax.LiteralToken:
			if t.Value != "." || lastDot {
				return "", false
			}
			b.WriteByte('.')
			lastDot = true
		default:
			return "", false
		}
	}
	if lastDot {
		return "", false
	}
	return b.String(), true
}
