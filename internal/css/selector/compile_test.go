package selector

import (
	"reflect"
	"testing"
)

const compileDoc = `<root id="root" xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:other">
  <section id="s1" class="a b" data-x="Alpha-beta gamma" title="">
    <p id="p1" lang="en-US">one</p>
    <p id="p2" class="a">two</p>
    <div id="d1"><span id="sp1"/></div>
    <p id="p3"/>
    <em id="e1"/>
    <p id="p4" o:flag="yes"/>
  </section>
  <section id="s2">
    <p id="p5" xml:lang="fr"/>
    <div id="d2"><div id="d3"/></div>
  </section>
</root>`

func TestCompiledSelectorsMatch(t *testing.T) {
	root := parseXML(t, compileDoc, false)
	tests := []struct {
		selector string
		want     []string
	}{
		{"p", []string{"p1", "p2", "p3", "p4", "p5"}},
		{"section p", []string{"p1", "p2", "p3", "p4", "p5"}},
		{"section > div", []string{"d1", "d2"}},
		{"div div", []string{"d3"}},
		{"div > div", []string{"d3"}},
		{"p + p", []string{"p2"}},
		{"p + div", []string{"d1", "d2"}},
		{"p + em", []string{"e1"}},
		{"div + p", []string{"p3"}},
		{"p ~ em", []string{"e1"}},
		{"p ~ p", []string{"p2", "p3", "p4"}},
		{"* > em", []string{"e1"}},
		{"* + em", []string{"e1"}},
		{"* em", []string{"e1"}},
		{"* ~ p", []string{"p2", "p3", "p4"}},
		{"#root > section.a.b", []string{"s1"}},
		{"section:not(.a)", []string{"s2"}},
		{"p:first-child", []string{"p1", "p5"}},
		{"p:last-child", []string{"p4"}},
		{"section :last-child", []string{"sp1", "p4", "d2", "d3"}},
		{"p:first-of-type", []string{"p1", "p5"}},
		{"p:last-of-type", []string{"p4", "p5"}},
		{"em:only-of-type", []string{"e1"}},
		{"div:only-child", []string{"d3"}},
		{"p:nth-child(2)", []string{"p2"}},
		{"p:nth-child(-n+2)", []string{"p1", "p2", "p5"}},
		{"p:nth-last-child(-n+2)", []string{"p4", "p5"}},
		{"*:nth-of-type(2)", []string{"p2", "s2"}},
		{"section > :nth-child(odd of p)", []string{"p1", "p3", "p5"}},
		{"section > :nth-last-child(1 of p)", []string{"p4", "p5"}},
		{"p:nth-of-type(2n+1)", []string{"p1", "p3", "p5"}},
		{"p:nth-child(0n+3)", []string{}},
		{":is(#p1, #e1)", []string{"p1", "e1"}},
		{"section:has(> div)", []string{"s1", "s2"}},
		{"section:has(div div)", []string{"s2"}},
		{"p:has(+ em)", []string{"p3"}},
		{"div:has(~ p)", []string{"d1"}},
		{":lang(en)", []string{"p1"}},
		{":lang(fr)", []string{"p5"}},
		{":empty", []string{"sp1", "p3", "e1", "p4", "p5", "d3"}},
		{"[title]", []string{"s1"}},
		{"[title=\"\"]", []string{"s1"}},
		{"[data-x|=Alpha]", []string{"s1"}},
		{"[data-x~=gamma]", []string{"s1"}},
		{"[data-x^=Alpha]", []string{"s1"}},
		{"[data-x$=gamma]", []string{"s1"}},
		{"[data-x*=\"a-B\" i]", []string{"s1"}},
		{"[data-x*=\"a-B\"]", []string{}},
		{"[data-x^=\"\"]", []string{}},
		{"[data-x~=\"a b\"]", []string{}},
		{"[data-x~=\"\"]", []string{}},
		{"[flag]", []string{}},
		{":hover, :focus, :visited, :target", []string{}},
		{":root", []string{"root"}},
		{"p::before", []string{}},
		{"p:hover::before", []string{}},
	}
	for _, tt := range tests {
		if got := idsOf(queryAll(t, root, tt.selector)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: got %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestCompiledNamespaces(t *testing.T) {
	root := parseXML(t, compileDoc, false)
	ns := map[string]string{"h": XHTMLNamespace, "o": "urn:other", "none": ""}
	tests := []struct {
		selector string
		want     []string
	}{
		{"h|em", []string{"e1"}},
		{"none|em", []string{}},
		{"*|em", []string{"e1"}},
		{"|em", []string{}},
		{"[o|flag]", []string{"p4"}},
		{"[o|flag=yes]", []string{"p4"}},
		{"[|id=p1]", []string{"p1"}},
		{"[none|id=p1]", []string{"p1"}},
		{"[h|id=p1]", []string{}},
	}
	for _, tt := range tests {
		sels, err := CompileString(tt.selector, ns)
		if err != nil {
			t.Errorf("%q: %v", tt.selector, err)
			continue
		}
		if got := idsOf(queryCompiled(root, sels)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: got %v, want %v", tt.selector, got, tt.want)
		}
	}

	sels, err := CompileString("em", map[string]string{"": "urn:nowhere"})
	if err != nil || len(queryCompiled(root, sels)) != 0 {
		t.Errorf("default namespace should apply to type selectors: %v", err)
	}
	sels, err = CompileString("em", map[string]string{"": XHTMLNamespace})
	if err != nil || len(queryCompiled(root, sels)) != 1 {
		t.Errorf("default namespace should match its elements: %v", err)
	}
}

func TestCompileErrors(t *testing.T) {
	for _, css := range []string{
		"a:unknown", ":is(:unknown)", ":not(a:unknown)", "a:nth-child()",
		"a:nth-child(foo)", "a:nth-child(2 of)", "a:nth-foo(2)", "a:foo(2)",
		":lang()", ":lang(1)", ":lang(en 1)", ":nth-child(1 of x|y)",
		"[*|a]", ":marker",
	} {
		if _, err := CompileString(css, nil); err == nil {
			t.Errorf("%q should fail to compile", css)
		}
	}
}

func TestSelectorNeverMatches(t *testing.T) {
	tests := []struct {
		css   string
		never bool
	}{
		{":hover", true},
		{"a:hover", true},
		{":hover a", true},
		{"a :hover", true},
		{"[a^=\"\"]", true},
		{"[a~=\"x y\"]", true},
		{":not(*)", true},
		{":is(:hover)", true},
		{":where(:hover, :focus)", true},
		{":has(:hover)", true},
		{":is(:hover, a)", false},
		{"*", false},
		{"a", false},
		{":not(:hover)", false},
	}
	for _, tt := range tests {
		sels, err := CompileString(tt.css, nil)
		if err != nil {
			t.Errorf("%q: %v", tt.css, err)
			continue
		}
		if got := sels[0].NeverMatches(); got != tt.never {
			t.Errorf("%q: NeverMatches = %v, want %v", tt.css, got, tt.never)
		}
	}
}

func TestCompileSkipsRightSideOfNeverMatchingSelector(t *testing.T) {
	// The right-hand compound of an impossible selector is not compiled, so
	// its unknown pseudo-class goes unnoticed.
	if _, err := CompileString(":hover :unknown", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := CompileString(":unknown :hover", nil); err == nil {
		t.Error("unknown pseudo-class on the left should fail")
	}
}

func TestStatefulFormPseudoClasses(t *testing.T) {
	const form = `<html xmlns="http://www.w3.org/1999/xhtml"><body>
	<a id="link" href="x"/><a id="local" href="/rel"/><a id="abs" href="http://e/"/><a id="mail" href="mailto:x"/><a id="anchor"/>
	<input id="in1"/><input id="in2" disabled=""/>
	<fieldset id="fs" disabled="">
	  <legend id="lg1"><input id="in3"/></legend>
	  <legend id="lg2"><input id="in4"/></legend>
	  <input id="in5"/>
	  <div><select id="sel"/></div>
	</fieldset>
	<input id="cb" type="CheckBox" checked=""/><input id="tx" type="text" checked=""/>
	<select id="sel2"><option id="op1" selected=""/><option id="op2"/></select>
	</body></html>`
	for _, html := range []bool{false, true} {
		root := parseXML(t, form, html)
		tests := []struct {
			selector string
			want     []string
		}{
			{":link", []string{"link", "local", "abs", "mail"}},
			{":local-link", []string{"link", "local"}},
			{":enabled", []string{"link", "local", "abs", "mail", "in1", "in3", "cb", "tx", "sel2", "op1", "op2"}},
			{":disabled", []string{"in2", "fs", "in4", "in5", "sel"}},
			{":checked", []string{"cb", "op1"}},
		}
		for _, tt := range tests {
			got := idsOf(queryAll(t, root, tt.selector))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("html=%v %q: got %v, want %v", html, tt.selector, got, tt.want)
			}
		}
	}
}

func TestLanguageInheritance(t *testing.T) {
	tests := []struct {
		name      string
		doc       string
		html      bool
		transport string
		lang      string
		want      bool
	}{
		{"no language", `<html xmlns="http://www.w3.org/1999/xhtml"></html>`, false, "", "fr", false},
		{"meta wins over transport", `<html xmlns="http://www.w3.org/1999/xhtml"><meta http-equiv="Content-Language" content=" fr &#9;"/></html>`, false, "en", "fr", true},
		{"meta ignored outside XHTML", `<html><meta http-equiv="Content-Language" content=" fr "/></html>`, false, "en", "en", true},
		{"transport only", `<html></html>`, false, "en", "en", true},
		{"several transport languages", `<html></html>`, false, "en, es", "en", false},
		{"lang attribute in HTML documents", `<html lang="eN"></html>`, true, "", "en", true},
		{"lang attribute ignored in XML", `<html lang="eN"></html>`, false, "", "en", false},
		{"xml:lang in XML", `<html xml:lang="EN"></html>`, false, "", "en", true},
		{"subtag", `<html lang="en-GB"></html>`, true, "", "en", true},
		{"prefix is not a subtag", `<html lang="english"></html>`, true, "", "en", false},
	}
	for _, tt := range tests {
		root := parseXML(t, tt.doc, tt.html)
		root.transportLang = tt.transport
		sels, err := CompileString(":lang("+tt.lang+")", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := sels[0].Match(root); got != tt.want {
			t.Errorf("%s: matches = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestHTMLDocumentsIgnoreNameCase(t *testing.T) {
	const doc = `<html xmlns="http://www.w3.org/1999/xhtml"><a id="x" name="n" Title="t"/></html>`
	for _, tt := range []struct {
		selector string
		xml, htm int
	}{
		{"A", 0, 1},
		{"a", 1, 1},
		{"a[NAME]", 0, 1},
		{"a[name]", 1, 1},
		{"a[title]", 0, 0},
		{"a[Title]", 1, 0},
	} {
		for html, want := range map[bool]int{false: tt.xml, true: tt.htm} {
			root := parseXML(t, doc, html)
			if got := len(queryAll(t, root, tt.selector)); got != want {
				t.Errorf("%q html=%v: %d matches, want %d", tt.selector, html, got, want)
			}
		}
	}
}
