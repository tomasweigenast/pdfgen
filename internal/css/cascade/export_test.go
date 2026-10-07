package cascade

// SelectorCount returns the number of selectors added to the matcher of a
// stylesheet, imported ones included.
func SelectorCount(s *Stylesheet) int { return s.selectors }
