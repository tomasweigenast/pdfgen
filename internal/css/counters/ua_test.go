package counters_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func render(t *testing.T, style string, values []int, want string) {
	t.Helper()
	r := cascade.UACounterStyle()
	expected := strings.Fields(want)
	if len(expected) != len(values) {
		t.Fatalf("%d values, %d expected", len(values), len(expected))
	}
	for i, v := range values {
		if got := r.RenderValueOf(v, style); got != expected[i] {
			t.Errorf("%s: %d renders %q, want %q", style, v, got, expected[i])
		}
	}
}

func TestDecimalLeadingZero(t *testing.T) {
	render(t, "decimal-leading-zero",
		[]int{-1986, -1985, -11, -10, -9, -8, -1, 0, 1, 2, 9, 10, 11, 99, 100, 101, 4135, 4136},
		"-1986 -1985 -11 -10 -9 -8 -1 00 01 02 09 10 11 99 100 101 4135 4136")
}

func TestRoman(t *testing.T) {
	values := []int{-1986, -1985, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 49, 50, 389, 390, 3489, 3490, 3491, 4999, 5000, 5001}
	render(t, "lower-roman", values,
		"-1986 -1985 -1 0 i ii iii iv v vi vii viii ix x xi xii xlix l ccclxxxix cccxc mmmcdlxxxix mmmcdxc mmmcdxci 4999 5000 5001")
	render(t, "upper-roman", values,
		"-1986 -1985 -1 0 I II III IV V VI VII VIII IX X XI XII XLIX L CCCLXXXIX CCCXC MMMCDLXXXIX MMMCDXC MMMCDXCI 4999 5000 5001")
}

func TestAlphabetic(t *testing.T) {
	values := []int{-1986, -1985, -1, 0, 1, 2, 3, 4, 25, 26, 27, 28, 29, 2002, 2003}
	lower := "-1986 -1985 -1 0 a b c d y z aa ab ac bxz bya"
	upper := "-1986 -1985 -1 0 A B C D Y Z AA AB AC BXZ BYA"
	render(t, "lower-alpha", values, lower)
	render(t, "upper-alpha", values, upper)
	render(t, "lower-latin", values, lower)
	render(t, "upper-latin", values, upper)
}

func TestGeorgian(t *testing.T) {
	render(t, "georgian", []int{
		-1986, -1985, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		20, 30, 40, 50, 60, 70, 80, 90, 100,
		200, 300, 400, 500, 600, 700, 800, 900, 1000,
		2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000,
		19999, 20000, 20001,
	}, `-1986 -1985 -1 0 ა ბ გ დ ე ვ ზ ჱ თ ი ია იბ კ ლ მ ნ ჲ ო პ ჟ რ
		ს ტ ჳ ფ ქ ღ ყ შ ჩ ც ძ წ ჭ ხ ჴ ჯ ჰ ჵ ჵჰშჟთ 20000 20001`)
}

func TestArmenian(t *testing.T) {
	render(t, "armenian", []int{
		-1986, -1985, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		20, 30, 40, 50, 60, 70, 80, 90, 100,
		200, 300, 400, 500, 600, 700, 800, 900, 1000,
		2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000,
		9999, 10000, 10001,
	}, `-1986 -1985 -1 0 Ա Բ Գ Դ Ե Զ Է Ը Թ Ժ ԺԱ ԺԲ Ի Լ Խ Ծ Կ Հ Ձ Ղ Ճ
		Մ Յ Ն Շ Ո Չ Պ Ջ Ռ Ս Վ Տ Ր Ց Ւ Փ Ք ՔՋՂԹ 10000 10001`)
}

func TestPredefinedSymbols(t *testing.T) {
	r := cascade.UACounterStyle()
	tests := []struct {
		style string
		value int
		want  string
	}{
		{"disc", -11, "•"},
		{"circle", -10, "◦"},
		{"square", -9, "▪"},
		{"decimal", -8, "-8"},
	}
	for _, tt := range tests {
		if got := r.RenderValueOf(tt.value, tt.style); got != tt.want {
			t.Errorf("%s: %d renders %q, want %q", tt.style, tt.value, got, tt.want)
		}
	}
}

func TestCounterStylesWithMultipleExtends(t *testing.T) {
	env := &cascade.Environment{CounterStyle: cascade.UACounterStyle()}
	cascade.ParseStylesheet(`
        @counter-style a { system: extends b; prefix: a; }
        @counter-style b { system: extends c; suffix: b; }
        @counter-style c { system: extends b; pad: 2 c; }
        @counter-style d { system: extends d; prefix: d; }
        @counter-style e { system: extends unknown; prefix: e; }
        @counter-style f { system: extends decimal; symbols: a; }
        @counter-style g { system: extends decimal; additive-symbols: 1 a; }`, "", env)
	want := []string{"a1b", "2b", "c3. ", "d4. ", "e5. ", "6. ", "7. ", "8. "}
	for i, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if got := env.CounterStyle.RenderMarker(counters.Named(name), i+1); got != want[i] {
			t.Errorf("marker of %s = %q, want %q", name, got, want[i])
		}
	}
}

func TestListStyleTypeMarkers(t *testing.T) {
	r := cascade.UACounterStyle()
	sym := func(kind string, s ...string) props.ListStyleType {
		return props.ListStyleType{Kind: props.ListStyleSymbols, Name: kind, Symbols: s}
	}
	tests := []struct {
		t      props.ListStyleType
		values [4]string
	}{
		{sym("cyclic", "a", "b", "c"), [4]string{"a ", "b ", "c ", "a "}},
		{sym("symbolic", "a", "b"), [4]string{"a ", "b ", "aa ", "bb "}},
		{sym("alphabetic", "a", "b"), [4]string{"a ", "b ", "aa ", "ab "}},
		{sym("fixed", "a", "b"), [4]string{"a ", "b ", "3 ", "4 "}},
		{sym("numeric", "0", "1", "2"), [4]string{"1 ", "2 ", "10 ", "11 "}},
		{counters.Named("decimal"), [4]string{"1. ", "2. ", "3. ", "4. "}},
		{props.ListStyleType{Kind: props.ListStyleString, Name: "/"}, [4]string{"/", "/", "/", "/"}},
	}
	for _, tt := range tests {
		for i, want := range tt.values {
			if got := r.RenderMarker(tt.t, i+1); got != want {
				t.Errorf("%+v: marker %d = %q, want %q", tt.t, i+1, got, want)
			}
		}
	}
	if got := r.RenderMarker(props.ListStyleType{Kind: props.ListStyleString}, 1); got != "" {
		t.Errorf("empty string marker = %q", got)
	}
}
