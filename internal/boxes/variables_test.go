package boxes_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func TestVariableInDisplayMakesLines(t *testing.T) {
	tests := []struct{ variable, display string }{
		{"inline", "var(--var)"},
		{"inline-block", "var(--var)"},
		{"inline flow", "var(--var)"},
		{"inline", "var(--var) flow"},
		{"flow", "inline var(--var)"},
	}
	for _, tt := range tests {
		t.Run(tt.variable+"/"+tt.display, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			section := body(t, boxtest.ParseAll(t, `
              <style>
                html { --var: `+tt.variable+` }
                div { display: `+tt.display+` }
              </style>
              <section><div></div></section>
            `))[0]
			if child := onlyChild(t, section); child.Kind() != boxes.KindLine {
				t.Errorf("section child is %v, want a line", child)
			}
		})
	}
}

func TestVariablesInCounterFunctions(t *testing.T) {
	tests := []struct {
		name, css, lastDiv string
		want               []string
	}{
		{
			"counter name",
			`html { --var: title }
             h1 { counter-increment: var(--var) }
             div::before { content: counter(var(--var)) }`,
			"", []string{"1", "2"},
		},
		{
			"several values",
			`html { --name: title; --counter: title, upper-roman }
             h1 { counter-increment: var(--name) }
             div::before { content: counter(var(--counter)) }`,
			`style="--counter: var(--name), lower-roman"`, []string{"I", "II", "iii"},
		},
		{
			"variable in variable",
			`html { --name: title; --counter: var(--name), upper-roman }
             h1 { counter-increment: var(--name) }
             div::before { content: counter(var(--counter)) }`,
			`style="--counter: var(--name), lower-roman"`, []string{"I", "II", "iii"},
		},
		{
			"function in variable",
			`html { --name: title; --counter: counter(var(--name), upper-roman) }
             h1 { counter-increment: var(--name) }
             div::before { content: var(--counter) }`,
			`style="--counter: counter(var(--name), lower-roman)"`, []string{"I", "II", "iii"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			section := body(t, boxtest.ParseAll(t, `
              <style>`+tt.css+`</style>
              <section>
                <h1></h1>
                <div></div>
                <h1></h1>
                <div></div>`+third(tt.lastDiv, len(tt.want))+`
              </section>
            `))[0]
			children := boxes.ChildrenOf(section)
			if len(children) != 2*len(tt.want) {
				t.Fatalf("section has %d children, want %d", len(children), 2*len(tt.want))
			}
			for i, want := range tt.want {
				if got := textAt(t, children[2*i+1], 3); got != want {
					t.Errorf("div %d: %q, want %q", i+1, got, want)
				}
			}
		})
	}
}

// third returns the third heading and div of the documents that have them.
func third(divAttributes string, n int) string {
	if n < 3 {
		return ""
	}
	return `
                <h1></h1>
                <div ` + divAttributes + `></div>`
}

func TestMissingVariableInCounterFunction(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	section := body(t, boxtest.ParseAll(t, `
          <style>
            h1 { counter-increment: var(--var) }
            div::before { content: counter(var(--var)) }
          </style>
          <section>
            <h1></h1>
            <div></div>
            <h1></h1>
            <div></div>
          </section>
        `))[0]
	lines := logs.Lines()
	if len(lines) != 2 || !strings.Contains(lines[0], "no value") || !strings.Contains(lines[1], "invalid value") {
		t.Errorf("logs = %q", lines)
	}
	children := boxes.ChildrenOf(section)
	if len(children) != 4 {
		t.Fatalf("section has %d children, want 4", len(children))
	}
	for _, div := range []boxes.Box{children[1], children[3]} {
		if c := boxes.ChildrenOf(div); len(c) != 0 {
			t.Errorf("div has children %v", c)
		}
	}
}
