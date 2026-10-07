package validation

import (
	"fmt"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/selector"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// VendorPrefix is the prefix of proprietary properties and of the unstable
// ones that are also accepted without it. An empty prefix disables them.
var VendorPrefix = "-pdfgen-"

func warnf(format string, args ...any) { logging.Warnf(format, args...) }

// notPrintMedia lists the properties that do not apply to paged media.
var notPrintMedia = setOf(
	// Aural media.
	"azimuth", "cue", "cue-after", "cue-before", "elevation", "pause",
	"pause-after", "pause-before", "pitch-range", "pitch", "play-during",
	"richness", "speak-header", "speak-numeral", "speak-punctuation", "speak",
	"speech-rate", "stress", "voice-family", "volume",
	// Animations, transitions and timelines.
	"animation", "animation-composition", "animation-delay",
	"animation-direction", "animation-duration", "animation-fill-mode",
	"animation-iteration-count", "animation-name", "animation-play-state",
	"animation-range", "animation-range-end", "animation-range-start",
	"animation-timeline", "animation-timing-function", "timeline-scope",
	"transition", "transition-delay", "transition-duration",
	"transition-property", "transition-timing-function", "view-timeline",
	"view-timeline-axis", "view-timeline-inset", "view-timeline-name",
	"view-transition-name", "will-change",
	// Dynamic and interactive.
	"caret", "caret-color", "caret-shape", "cursor", "field-sizing",
	"pointer-events", "resize", "touch-action",
	// Scrolling.
	"overscroll-behavior", "overscroll-behavior-block",
	"overscroll-behavior-inline", "overscroll-behavior-x",
	"overscroll-behavior-y", "scroll-behavior", "scroll-margin",
	"scroll-margin-block", "scroll-margin-block-end",
	"scroll-margin-block-start", "scroll-margin-bottom", "scroll-margin-inline",
	"scroll-margin-inline-end", "scroll-margin-inline-start",
	"scroll-margin-left", "scroll-margin-right", "scroll-margin-top",
	"scroll-padding", "scroll-padding-block", "scroll-padding-block-end",
	"scroll-padding-block-start", "scroll-padding-bottom",
	"scroll-padding-inline", "scroll-padding-inline-end",
	"scroll-padding-inline-start", "scroll-padding-left",
	"scroll-padding-right", "scroll-padding-top", "scroll-snap-align",
	"scroll-snap-stop", "scroll-snap-type", "scroll-timeline",
	"scroll-timeline-axis", "scroll-timeline-name", "scrollbar-color",
	"scrollbar-gutter", "scrollbar-width",
)

func setOf(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// Declaration is a validated longhand declaration. A declaration made from a
// shorthand yields one per longhand.
type Declaration struct {
	// Selectors are the compiled selectors of the rule the declaration
	// belongs to, including the nested ones. They are nil when the
	// declarations were preprocessed without a rule prelude.
	Selectors []selector.Selector
	// Name is the longhand name, such as "margin-top", or a custom property
	// name starting with "--".
	Name  string
	Value props.Value
	// Important is set for declarations ending with !important.
	Important bool
}

// SelectorError reports the prelude of a rule that cannot be compiled.
type SelectorError struct {
	Prelude string
	Err     error
}

func (e *SelectorError) Error() string { return "'" + e.Prelude + "'" }
func (e *SelectorError) Unwrap() error { return e.Err }

var (
	nestingSelector = syntax.NewLiteralToken(1, 1, "&")
	rootTokens      = []syntax.Node{syntax.NewLiteralToken(1, 1, ":"), syntax.NewIdentToken(1, 1, "root")}
)

func isNesting(n syntax.Node) bool { return syntax.IsLiteral(n, nestingSelector.Value) }

// PreprocessDeclarations expands the shorthands of declarations, drops the
// unsupported properties and the invalid values, and logs a warning for each
// declaration that is ignored. Nested rules are ignored.
func PreprocessDeclarations(baseURL string, declarations []syntax.Node) []Declaration {
	out, _ := preprocess(baseURL, declarations, nil, false)
	return out
}

// PreprocessRuleDeclarations is like PreprocessDeclarations for the
// declarations of the rule with the given selector prelude. Every returned
// declaration carries the selectors of its rule, and the declarations of
// nested rules are included with their own selectors (CSS Nesting 1). An
// error is returned if a selector cannot be compiled.
func PreprocessRuleDeclarations(baseURL string, declarations, prelude []syntax.Node) ([]Declaration, error) {
	return preprocess(baseURL, declarations, prelude, true)
}

func preprocess(baseURL string, declarations, prelude []syntax.Node, hasPrelude bool) ([]Declaration, error) {
	var selectors []selector.Selector
	if hasPrelude {
		original := prelude
		replaced := prelude
		for _, t := range prelude {
			if isNesting(t) {
				// A & outside a nested rule is equivalent to :scope, which
				// is :root as there is no :scope yet.
				replaced = nil
				for _, t := range prelude {
					if isNesting(t) {
						replaced = append(replaced, rootTokens...)
					} else {
						replaced = append(replaced, t)
					}
				}
				break
			}
		}
		var err error
		selectors, err = selector.Compile(replaced, nil)
		if err != nil {
			return nil, &SelectorError{Prelude: syntax.Serialize(original), Err: err}
		}
	}

	var out []Declaration
	for _, node := range declarations {
		if e, ok := node.(*syntax.ParseError); ok {
			line, column := e.Pos()
			warnf("Error: %s at %d:%d.", e.Message, line, column)
		}

		if rule, ok := node.(*syntax.QualifiedRule); ok {
			if !hasPrelude {
				continue
			}
			nested, err := nestedPrelude(rule, prelude)
			if err != nil {
				return nil, err
			}
			more, err := preprocess(baseURL, syntax.ParseBlocksContents(rule.Content, false, false), nested, true)
			if err != nil {
				return nil, err
			}
			out = append(out, more...)
			continue
		}

		declaration, ok := node.(*syntax.Declaration)
		if !ok {
			continue
		}
		out = append(out, preprocessDeclaration(baseURL, declaration, selectors)...)
	}
	return out, nil
}

// nestedPrelude builds the selector list of a nested rule from its own
// prelude and the one of its parent.
func nestedPrelude(rule *syntax.QualifiedRule, parent []syntax.Node) ([]syntax.Node, error) {
	isToken := []syntax.Node{syntax.NewLiteralToken(1, 1, ":"), syntax.NewFunctionBlock(1, 1, "is", parent)}
	groups := [][]syntax.Node{nil}
	for _, t := range rule.Prelude {
		if syntax.IsLiteral(t, ",") {
			groups = append(groups, nil)
		} else {
			groups[len(groups)-1] = append(groups[len(groups)-1], t)
		}
	}
	var prelude []syntax.Node
	for _, group := range groups {
		hasNesting := false
		for _, t := range group {
			if isNesting(t) {
				hasNesting = true
			}
		}
		if hasNesting {
			// Replace & by the parent.
			for _, t := range rule.Prelude {
				if isNesting(t) {
					prelude = append(prelude, isToken...)
				} else {
					prelude = append(prelude, t)
				}
			}
		} else {
			// No &: the parent is prepended as an ancestor.
			prelude = append(prelude, isToken...)
			prelude = append(prelude, syntax.NewWhitespaceToken(1, 1, " "))
			prelude = append(prelude, group...)
		}
		prelude = append(prelude, syntax.NewLiteralToken(1, 1, ","))
	}
	return prelude[:len(prelude)-1], nil
}

func preprocessDeclaration(baseURL string, declaration *syntax.Declaration, selectors []selector.Selector) []Declaration {
	name := declaration.LowerName
	if isCustomProperty(declaration.Name) {
		name = declaration.Name
	}
	line, column := declaration.Pos()

	ignore := func(level, reason string) {
		msg := fmt.Sprintf("Ignored `%s:%s` at %d:%d, %s.", declaration.Name, syntax.Serialize(declaration.Value), line, column, reason)
		if level == "debug" {
			logging.Debugf("%s", msg)
		} else {
			logging.Warnf("%s", msg)
		}
	}

	if notPrintMedia[name] {
		ignore("debug", "the property does not apply for the print media")
		return nil
	}

	if VendorPrefix != "" && strings.HasPrefix(name, VendorPrefix) {
		unprefixed := name[len(VendorPrefix):]
		switch {
		case proprietary[unprefixed]:
			name = unprefixed
		case unstable[unprefixed]:
			warnf("Deprecated `%s:%s` at %d:%d, prefixes on unstable attributes are deprecated, use '%s' instead.",
				declaration.Name, syntax.Serialize(declaration.Value), line, column, unprefixed)
			name = unprefixed
		default:
			warnf("Ignored `%s:%s` at %d:%d, prefix on this attribute is not supported, use '%s' instead.",
				declaration.Name, syntax.Serialize(declaration.Value), line, column, unprefixed)
			return nil
		}
	}

	if strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "--") {
		ignore("debug", "prefixed selectors are ignored")
		return nil
	}

	tokens := tk.RemoveWhitespace(declaration.Value)
	var pairs []pair
	var err error
	if len(tokens) == 0 {
		// An empty value is allowed by the grammar but refused by every
		// property and expander.
		err = tk.Invalid("no value")
	} else if exp, ok := expanders[name]; ok {
		pairs, err = exp(tokens, name, baseURL)
	} else {
		var v props.Value
		if v, err = validateNonShorthand(tokens, name, baseURL, false); err == nil {
			pairs = []pair{{name, v}}
		}
	}
	if err != nil {
		reason := "invalid value"
		if e, ok := err.(*tk.InvalidValues); ok {
			reason = e.Error()
		} else {
			reason = err.Error()
		}
		ignore("warning", reason)
		return nil
	}

	out := make([]Declaration, len(pairs))
	for i, p := range pairs {
		out[i] = Declaration{Selectors: selectors, Name: p.name, Value: p.value, Important: declaration.Important}
	}
	return out
}
