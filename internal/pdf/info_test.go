package pdf_test

import (
	"bytes"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// nbsp is a no-break space, which the PDF text string must keep.
const nbsp = "\u00a0"

func TestDocumentInfo(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<meta name=author content="I Me &amp; Myself">
		<title>Test document</title>
		<h1>Another title</h1>
		<meta name=generator content="Human`+nbsp+`after`+nbsp+`all">
		<meta name=keywords content="html ,	css,
			pdf,css">
		<meta name=description content="Blah… ">
		<meta name=dcterms.created content=2011-04-21T23:00:00Z>
		<meta name=dcterms.modified content=2013-07-21T23:46+01:00>`, pdf.Options{})
	for _, want := range []string{
		`/Author (I Me & Myself)`,
		`/Title (Test document)`,
		`/Creator <feff00480075006d0061006e00a00061006600740065007200a00061006c006c>`,
		`/Keywords (html, css, pdf)`,
		`/Subject <feff0042006c0061006820260020>`,
		`/CreationDate (D:20110421230000Z)`,
		`/ModDate (D:20130721234600+01'00)`,
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("PDF has no %q", want)
		}
	}
}

func TestDocumentInfoNamesTheProducer(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<body>`, pdf.Options{})
	if !bytes.Contains(out, []byte("/Producer (pdfgen ")) {
		t.Error("PDF has no /Producer entry")
	}
}

func TestDocumentLanguage(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<html lang="fr"><body>`, pdf.Options{})
	if !bytes.Contains(out, []byte("/Lang (fr)")) {
		t.Error("PDF has no /Lang (fr)")
	}
}

func TestFontDescentAndAscent(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<html style="font-family: testfont">abc`, pdf.Options{})
	for _, want := range []string{"/Descent -200", "/Ascent 800"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("PDF has no %q", want)
		}
	}
}
