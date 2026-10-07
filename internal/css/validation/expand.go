package validation

import (
	"fmt"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// pair is a longhand with its validated value.
type pair struct {
	name  string
	value props.Value
}

// expander turns the tokens of a shorthand into its longhands.
type expander func(tokens []syntax.Node, name, baseURL string) ([]pair, error)

var expanders = map[string]expander{}

func registerExpander(fn expander, names ...string) {
	for _, name := range names {
		if _, dup := expanders[name]; dup {
			panic(fmt.Sprintf("validation: %q is expanded twice", name))
		}
		expanders[name] = fn
	}
}

// part is the tokens that a shorthand assigns to a longhand. The name is a
// suffix of the shorthand name when it starts with a hyphen, as in "-width".
type part struct {
	name   string
	tokens []syntax.Node
}

// partsFunc splits the tokens of a shorthand among its longhands. The parts
// found before an error are returned with it, because duplicates among them
// are reported first.
type partsFunc func(tokens []syntax.Node, name, baseURL string) ([]part, error)

func hasVar(tokens []syntax.Node) bool {
	for _, t := range tokens {
		if tk.CheckVar(t) {
			return true
		}
	}
	return false
}

func fullName(shorthand, expanded string) string {
	if strings.HasPrefix(expanded, "-") {
		return shorthand + expanded
	}
	return expanded
}

// generic builds an expander from a function that only has to find which
// tokens belong to which longhand. It handles initial, inherit and var(), and
// gives the longhands that were not set their initial value.
func generic(expanded []string, fn partsFunc) expander {
	return func(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
		var pairs []pair
		if keyword := tk.GetSingleKeyword(tokens); keyword == "inherit" || keyword == "initial" {
			for _, n := range expanded {
				pairs = append(pairs, pair{fullName(name, n), props.Keyword(keyword)})
			}
			return pairs, nil
		}
		if hasVar(tokens) {
			pending := &props.Pending{Name: name, Tokens: tokens}
			for _, n := range expanded {
				pairs = append(pairs, pair{fullName(name, n), pending})
			}
			return pairs, nil
		}

		parts, err := fn(tokens, name, baseURL)
		results := map[string][]syntax.Node{}
		for _, p := range parts {
			if !contains(expanded, p.name) {
				panic(fmt.Sprintf("validation: %q does not expand to %q", name, p.name))
			}
			if _, dup := results[p.name]; dup {
				return nil, tk.Invalid("got multiple %s values in a %s shorthand", strings.Trim(p.name, "-"), name)
			}
			results[p.name] = p.tokens
		}
		if err != nil {
			return nil, err
		}

		for _, n := range expanded {
			actual := fullName(name, n)
			if t, ok := results[n]; ok {
				value, err := validateNonShorthand(t, actual, baseURL, true)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair{actual, value})
			} else {
				pairs = append(pairs, pair{actual, props.Keyword("initial")})
			}
		}
		return pairs, nil
	}
}

// Solve validates the value of a pending declaration for one longhand, once
// the tokens have had their var() references substituted. The first invalid
// value is reported with a warning; later ones for the same declaration are
// not.
func Solve(p *props.Pending, tokens []syntax.Node, longhand, baseURL string) (value props.Value, err error) {
	defer func() {
		if err == nil {
			return
		}
		if _, ok := err.(*tk.InvalidValues); !ok || p.Reported {
			return
		}
		line, column := 0, 0
		if len(p.Tokens) > 0 {
			line, column = p.Tokens[0].Pos()
		}
		parts := make([]string, len(tokens))
		for i, t := range tokens {
			parts[i] = t.Serialize()
		}
		warnf("Ignored `%s: %s` at %d:%d, %s.", p.Name, strings.Join(parts, " "), line, column, err.Error())
		p.Reported = true
	}()
	if len(tokens) == 0 {
		return nil, tk.Invalid("no value")
	}
	if exp, ok := expanders[p.Name]; ok {
		pairs, err := exp(tokens, p.Name, baseURL)
		if err != nil {
			return nil, err
		}
		for _, pr := range pairs {
			if pr.name == longhand {
				return pr.value, nil
			}
		}
		return nil, tk.Invalid("no longhand %s", longhand)
	}
	return validateNonShorthand(tokens, p.Name, baseURL, false)
}
