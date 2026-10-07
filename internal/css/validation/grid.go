package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	register(validateGridAuto, 0, "grid-auto-columns", "grid-auto-rows")
	register(validateGridAutoFlow, 0, "grid-auto-flow")
	register(validateGridTemplate, 0, "grid-template-columns", "grid-template-rows")
	register(validateGridTemplateAreas, 0, "grid-template-areas")
	register(validateGridLine, 0, "grid-row-start", "grid-row-end", "grid-column-start", "grid-column-end")
}

// inflexibleBreadth parses <inflexible-breadth> (CSS Grid 2 §7.2.3).
func inflexibleBreadth(token syntax.Node) (props.Length, bool) {
	switch k := tk.GetKeyword(token); k {
	case "auto", "min-content", "max-content":
		return props.LengthKeyword(props.Keyword(k)), true
	case "":
		return tk.GetLength(token, false, true)
	}
	return props.Length{}, false
}

// trackBreadth parses <track-breadth>: an inflexible breadth or a flexible
// length in fr.
func trackBreadth(token syntax.Node) (props.Length, bool) {
	if d, ok := token.(*syntax.DimensionToken); ok && d.Value >= 0 && d.LowerUnit == "fr" {
		return props.Length{Dimension: props.Dimension{Value: d.Value, Unit: "fr"}}, true
	}
	return inflexibleBreadth(token)
}

func trackSize(token syntax.Node) (props.TrackSize, bool) {
	if b, ok := trackBreadth(token); ok {
		return props.TrackSize{Breadth: b}, true
	}
	f := tk.NewFunction(token)
	arguments := f.SplitComma()
	switch f.Name {
	case "minmax":
		if len(arguments) == 2 {
			min, okMin := inflexibleBreadth(arguments[0])
			max, okMax := trackBreadth(arguments[1])
			if okMin && okMax {
				return props.TrackSize{Function: "minmax()", Min: min, Max: max}, true
			}
		}
	case "fit-content":
		if len(arguments) == 1 {
			if l, ok := tk.GetLength(arguments[0], false, true); ok {
				return props.TrackSize{Function: "fit-content()", Breadth: l}, true
			}
		}
	}
	return props.TrackSize{}, false
}

func fixedSize(token syntax.Node) (props.TrackSize, bool) {
	if l, ok := tk.GetLength(token, false, true); ok {
		return props.TrackSize{Breadth: l}, true
	}
	f := tk.NewFunction(token)
	arguments := f.SplitComma()
	if f.Name == "minmax" && len(arguments) == 2 {
		length, haveLength := tk.GetLength(arguments[0], false, true)
		if haveLength {
			if b, ok := trackBreadth(arguments[1]); ok {
				return props.TrackSize{Function: "minmax()", Min: length, Max: b}, true
			}
		}
		keyword := tk.GetKeyword(arguments[0])
		if keyword == "min-content" || keyword == "max-content" || keyword == "auto" || haveLength {
			if fixed, ok := tk.GetLength(arguments[1], false, true); ok {
				min := props.LengthKeyword(props.Keyword(keyword))
				if haveLength {
					min = length
				}
				return props.TrackSize{Function: "minmax()", Min: min, Max: fixed}, true
			}
		}
	}
	return props.TrackSize{}, false
}

func lineNames(token syntax.Node) ([]string, bool) {
	block, ok := token.(*syntax.SquareBracketsBlock)
	if !ok {
		return nil, false
	}
	var names []string
	for _, t := range block.Content {
		switch t := t.(type) {
		case *syntax.IdentToken:
			names = append(names, t.Value)
		case *syntax.WhitespaceToken:
		default:
			return nil, false
		}
	}
	return names, true
}

func validateGridAuto(tokens []syntax.Node, _ string) (props.Value, error) {
	list := make(props.TrackList, 0, len(tokens))
	for _, token := range tokens {
		size, ok := trackSize(token)
		if !ok {
			return nil, nil
		}
		list = append(list, size)
	}
	return list, nil
}

func validateGridAutoFlow(tokens []syntax.Node, _ string) (props.Value, error) {
	switch len(tokens) {
	case 1:
		switch k := tk.GetKeyword(tokens[0]); k {
		case "row", "column":
			return props.Keywords{props.Keyword(k)}, nil
		case "dense":
			return props.Keywords{"dense", "row"}, nil
		}
	case 2:
		k := []string{tk.GetKeyword(tokens[0]), tk.GetKeyword(tokens[1])}
		if contains(k, "dense") && (contains(k, "row") || contains(k, "column")) {
			return props.Keywords{props.Keyword(k[0]), props.Keyword(k[1])}, nil
		}
	}
	return nil, nil
}

// repeatArguments splits the arguments of a repeat() into the count and the
// tracks.
func repeatArguments(token syntax.Node) (name string, count syntax.Node, tracks []syntax.Node, ok bool) {
	f := tk.NewFunction(token)
	arguments := f.SplitCommaGroups(false)
	if len(arguments) != 2 || len(arguments[0]) != 1 || len(arguments[1]) == 0 {
		return "", nil, nil, false
	}
	return f.Name, arguments[0][0], arguments[1], true
}

// validateGridTemplate validates grid-template-columns and -rows (CSS Grid 2
// §7.2): none, subgrid with line names, or a track list with line names.
func validateGridTemplate(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 1 && tk.GetKeyword(tokens[0]) == "none" {
		return props.GridTemplate{None: true}, nil
	}
	if tk.GetKeyword(tokens[0]) == "subgrid" {
		return validateSubgrid(tokens[1:])
	}

	var items []props.GridItem
	includesAutoRepeat, includesTrack, lastIsLineName := false, false, false
	for _, token := range tokens {
		if names, ok := lineNames(token); ok {
			if lastIsLineName {
				return nil, nil
			}
			lastIsLineName = true
			items = append(items, props.GridItem{Kind: props.GridLineNames, Names: names})
			continue
		}
		if size, ok := fixedSize(token); ok {
			if !lastIsLineName {
				items = append(items, props.GridItem{Kind: props.GridLineNames})
			}
			lastIsLineName = false
			items = append(items, props.GridItem{Kind: props.GridTrack, Track: size})
			continue
		}
		if size, ok := trackSize(token); ok {
			if !lastIsLineName {
				items = append(items, props.GridItem{Kind: props.GridLineNames})
			}
			lastIsLineName = false
			items = append(items, props.GridItem{Kind: props.GridTrack, Track: size})
			includesTrack = true
			continue
		}

		name, count, tracks, ok := repeatArguments(token)
		if !ok || name != "repeat" {
			return nil, nil
		}
		repeat := props.GridItem{Kind: props.GridRepeat}
		if n, ok := tk.GetNumber(count, false, true); ok {
			if n.Value < 1 {
				return nil, nil
			}
			repeat.Count = int(n.Value)
		} else if k := tk.GetKeyword(count); k == "auto-fill" || k == "auto-fit" {
			if includesAutoRepeat {
				return nil, nil
			}
			repeat.Auto = props.Keyword(tk.GetCustomIdent(count))
			includesAutoRepeat = true
		} else {
			return nil, nil
		}

		repeatLastIsLineName := false
		for _, arg := range tracks {
			if names, ok := lineNames(arg); ok {
				if repeatLastIsLineName {
					return nil, nil
				}
				repeat.Items = append(repeat.Items, props.GridItem{Kind: props.GridLineNames, Names: names})
				repeatLastIsLineName = true
				continue
			}
			size, ok := fixedSize(arg)
			if !ok {
				if size, ok = trackSize(arg); !ok {
					return nil, nil
				}
				includesTrack = true
			}
			if !repeatLastIsLineName {
				repeat.Items = append(repeat.Items, props.GridItem{Kind: props.GridLineNames})
			}
			repeatLastIsLineName = false
			repeat.Items = append(repeat.Items, props.GridItem{Kind: props.GridTrack, Track: size})
		}
		if !lastIsLineName {
			items = append(items, props.GridItem{Kind: props.GridLineNames})
		}
		lastIsLineName = false
		if !repeatLastIsLineName {
			repeat.Items = append(repeat.Items, props.GridItem{Kind: props.GridLineNames})
		}
		items = append(items, repeat)
	}
	if includesAutoRepeat && includesTrack {
		return nil, nil
	}
	if !lastIsLineName {
		items = append(items, props.GridItem{Kind: props.GridLineNames})
	}
	return props.GridTemplate{Items: items}, nil
}

func validateSubgrid(tokens []syntax.Node) (props.Value, error) {
	template := props.GridTemplate{Subgrid: true}
	for _, token := range tokens {
		if names, ok := lineNames(token); ok {
			template.Items = append(template.Items, props.GridItem{Kind: props.GridLineNames, Names: names})
			continue
		}
		name, count, tracks, ok := repeatArguments(token)
		if !ok || name != "repeat" {
			return nil, nil
		}
		repeat := props.GridItem{Kind: props.GridRepeat}
		if n, ok := count.(*syntax.NumberToken); ok && n.IsInteger && n.Value >= 1 {
			repeat.Count = n.IntValue
		} else if tk.GetKeyword(count) == "auto-fill" {
			repeat.Auto = "auto-fill"
		} else {
			return nil, nil
		}
		for _, arg := range tracks {
			if names, ok := lineNames(arg); ok {
				repeat.Items = append(repeat.Items, props.GridItem{Kind: props.GridLineNames, Names: names})
			}
		}
		template.Items = append(template.Items, repeat)
	}
	return template, nil
}

// validateGridTemplateAreas validates grid-template-areas: strings whose rows
// have the same number of cells, and in which each named area is a rectangle.
func validateGridTemplateAreas(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 1 && tk.GetKeyword(tokens[0]) == "none" {
		return props.GridAreas{None: true}, nil
	}
	var rows [][]string
	for _, token := range tokens {
		s, ok := token.(*syntax.StringToken)
		if !ok {
			return nil, nil
		}
		var row []string
		lastIsDot := false
		for _, value := range syntax.ParseComponentValueList(s.Value, false) {
			switch v := value.(type) {
			case *syntax.IdentToken:
				row = append(row, v.Value)
				lastIsDot = false
			case *syntax.LiteralToken:
				if v.Value != "." {
					return nil, nil
				}
				if lastIsDot {
					continue
				}
				row = append(row, "")
				lastIsDot = true
			case *syntax.WhitespaceToken:
				lastIsDot = false
			default:
				return nil, nil
			}
		}
		if len(row) == 0 {
			return nil, nil
		}
		rows = append(rows, row)
	}

	for _, row := range rows {
		if len(row) != len(rows[0]) {
			return nil, nil
		}
	}
	if !areasAreRectangles(rows) {
		return nil, nil
	}
	return props.GridAreas{Rows: rows}, nil
}

// areasAreRectangles checks that every named area is one continuous
// rectangle. Empty names are the cells that belong to no area.
func areasAreRectangles(rows [][]string) bool {
	type cell struct{ x, y int }
	seen := map[cell]bool{}
	areas := map[string]bool{}
	for y, row := range rows {
		for x, area := range row {
			if seen[cell{x, y}] || area == "" {
				continue
			}
			if areas[area] {
				return false
			}
			areas[area] = true
			seen[cell{x, y}] = true
			end := x + 1
			for end < len(row) && row[end] == area {
				seen[cell{end, y}] = true
				end++
			}
			for ny := y + 1; ny < len(rows); ny++ {
				full := true
				for nx := x; nx < end; nx++ {
					if rows[ny][nx] != area {
						full = false
						break
					}
				}
				if !full {
					break
				}
				for nx := x; nx < end; nx++ {
					seen[cell{nx, ny}] = true
				}
			}
		}
	}
	return true
}

// validateGridLine validates grid-row-start and its siblings (CSS Grid 2
// §8.3): auto, a line name, an integer, or span with an integer and a name.
func validateGridLine(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 1 {
		token := tokens[0]
		if keyword := tk.GetKeyword(token); keyword != "" {
			switch keyword {
			case "auto":
				return props.GridLine{Auto: true}, nil
			case "span":
				return nil, nil
			}
			return props.GridLine{Ident: tk.GetCustomIdent(token)}, nil
		}
		if n, ok := tk.GetNumber(token, true, true); ok && n.Value != 0 {
			return props.GridLine{N: int(n.Value)}, nil
		}
		return nil, nil
	}

	number, ident := 0, ""
	span := false
	for _, token := range tokens {
		if keyword := tk.GetKeyword(token); keyword != "" {
			if keyword == "auto" {
				return nil, nil
			}
			if keyword == "span" {
				if !span {
					span = true
					continue
				}
			} else if ident == "" {
				ident = tk.GetCustomIdent(token)
				continue
			}
		} else if n, ok := tk.GetNumber(token, true, true); ok {
			if n.Value != 0 && number == 0 {
				number = int(n.Value)
				continue
			}
		}
		return nil, nil
	}
	if span {
		if number < 0 {
			return nil, nil
		}
		if ident != "" || number != 0 {
			return props.GridLine{Span: true, N: number, Ident: ident}, nil
		}
	} else if number != 0 {
		return props.GridLine{N: number, Ident: ident}, nil
	}
	return nil, nil
}
