package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func TestBeforePseudoElementWithAttr(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 10, `
		<style>
			@page { size: 300px 30px }
			body { margin: 0 }
			a[href]:before { content: '[' attr(href) '] ' }
		</style>
		<p><a href="some url">some content</a></p>
	`, `
		<style>
			@page { size: 300px 30PX }
			body { margin: 0 }
		</style>
		<p><a href="another url"><span>[some url] </span>some content</p>
	`)
}

func TestNestedQuotesPseudoElements(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 10, `
		<style>
			@page { size: 500px 30px }
			body { margin: 0; quotes: '«' '»' '“' '”' }
			q:before { content: open-quote ' '}
			q:after { content: ' ' close-quote }
		</style>
		<p><q>Lorem ipsum <q>dolor</q> sit amet</q></p>
	`, `
		<style>
			@page { size: 500px 30Px }
			body { margin: 0 }
			q:before, q:after { content: none }
		</style>
		<p><span><span>« </span>Lorem ipsum
			<span><span>“ </span>dolor<span> ”</span></span>
			sit amet<span> »</span></span></p>
	`)
}

func TestBeforeWithImageContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 10, `
		<style>
			@page { size: 100px 30px }
			body { margin: 0; }
			p:before { content: 'a' url(pattern.png) 'b'}
		</style>
		<p>c</p>
	`, `
		<style>
			@page { size: 100PX 30px }
			body { margin: 0 }
		</style>
		<p><span>a<img src="pattern.png" alt="Missing image">b</span>c</p>
	`)
}
