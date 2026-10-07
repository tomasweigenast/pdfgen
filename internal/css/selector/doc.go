// Package selector compiles and matches CSS selectors (Selectors Level 3 and
// the Level 4 additions used for print stylesheets: :is(), :where(), :has(),
// :not() with lists, :nth-*() with "of S" and :lang() with lists).
//
// Selectors are parsed from CSS Syntax tokens, compiled to predicates over
// the Element interface and stored in a Matcher, which indexes them by the id,
// class, local name or namespace of their subject. Pseudo-classes that depend
// on user interaction never match. The pseudo-element of a selector is
// reported next to its specificity, so the caller decides which ones to
// accept.
//
// Selector lists follow the forgiving parsing of Selectors Level 4 §18.1
// inside :is(), :where() and :has(), and are strict everywhere else.
package selector
