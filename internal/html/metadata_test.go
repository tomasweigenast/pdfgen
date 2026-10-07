package html

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func strp(s string) *string { return &s }

func metadataOf(t *testing.T, markup string) *Metadata {
	t.Helper()
	doc, err := ParseString(markup, "http://example.org/dir/page.html")
	if err != nil {
		t.Fatal(err)
	}
	return doc.Metadata()
}

func TestMetadataEmptyDocument(t *testing.T) {
	testutil.AssertNoLogs(t)
	got := metadataOf(t, "<body>")
	want := &Metadata{Custom: map[string]string{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMetadataStandardNames(t *testing.T) {
	testutil.AssertNoLogs(t)
	got := metadataOf(t, `
		<html lang="en"><head>
		<meta name=author content="I Me &amp; Myself">
		<meta name=author content="Smith, John">
		<title>Test document</title>
		<h1>Another title</h1>
		<meta name=generator content="Human after all">
		<meta name=generator content="Human">
		<meta name=dummy content=ignored>
		<meta name=dummy>
		<meta content=ignored>
		<meta>
		<meta name=keywords content="html ,	css,
		                             pdf,css">
		<meta name=dcterms.created content=2011-04>
		<meta name=dcterms.created content=2011-05>
		<meta name=dcterms.modified content=2013>
		<meta name=keywords content="Python; pydyf">
		<meta name=description content="Blah… ">
		<meta name=description content="*Oh-no/">
		<meta name=dcterms.modified content=2012>
		</head></html>`)
	want := &Metadata{
		Authors:     []string{"I Me & Myself", "Smith, John"},
		Title:       strp("Test document"),
		Generator:   strp("Human after all"),
		Keywords:    []string{"html", "css", "pdf", "Python; pydyf"},
		Description: strp("Blah… "),
		Created:     strp("2011-04"),
		Modified:    strp("2013"),
		Lang:        strp("en"),
		Custom:      map[string]string{"dummy": "ignored"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestMetadataFirstTitleAndEmptyAuthor(t *testing.T) {
	testutil.AssertNoLogs(t)
	got := metadataOf(t, `
		<title>One</title>
		<meta name=Author>
		<title>Two</title>
		<title>Three</title>
		<meta name=author content=Me>`)
	if got.Title == nil || *got.Title != "One" {
		t.Errorf("Title = %v", got.Title)
	}
	if want := []string{"", "Me"}; !reflect.DeepEqual(got.Authors, want) {
		t.Errorf("Authors = %q, want %q", got.Authors, want)
	}
}

func TestMetadataInvalidDateIsSkipped(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	got := metadataOf(t, `
		<meta name=dcterms.created content=wrong>
		<meta name=author content=Me>
		<title>Title</title>
		<meta name=dcterms.created content=2020-01-02>`)
	if lines := logs.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "Invalid date") {
		t.Errorf("logs: %q", lines)
	}
	if got.Created == nil || *got.Created != "2020-01-02" {
		t.Errorf("Created = %v, want the next valid date", got.Created)
	}
	if got.Title == nil || *got.Title != "Title" || len(got.Authors) != 1 {
		t.Errorf("other fields lost: %+v", got)
	}
}

func TestMetadataEmptyValuesAreSet(t *testing.T) {
	got := metadataOf(t, `<title></title><meta name=description content=""><meta name=description content=later><meta name=generator content="">`)
	if got.Title == nil || *got.Title != "" {
		t.Errorf("Title = %v, want an empty title", got.Title)
	}
	if got.Description == nil || *got.Description != "" {
		t.Errorf("Description = %v, want the first, empty, one", got.Description)
	}
	if got.Generator == nil {
		t.Error("Generator not set")
	}
}

func TestMetadataNamesAreCaseInsensitive(t *testing.T) {
	got := metadataOf(t, `<meta name=KeyWords content="a,b"><meta name=Dcterms.Modified content=2001-02-03T04:05+01:00><meta name=Custom-Name content=v>`)
	if !reflect.DeepEqual(got.Keywords, []string{"a", "b"}) || got.Modified == nil || got.Custom["custom-name"] != "v" {
		t.Errorf("got %+v", got)
	}
}

func TestMetadataTitleIgnoresForeignTitles(t *testing.T) {
	got := metadataOf(t, `<body><svg><title>drawing</title></svg><title>doc &amp; ument</title>`)
	if got.Title == nil || *got.Title != "doc & ument" {
		t.Errorf("Title = %v", got.Title)
	}
}

func TestMetadataAttachments(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	got := metadataOf(t, `
		<link rel=attachment href="data.csv" title="The data">
		<link rel="stylesheet Attachment" href="http://other.org/a b.txt">
		<link rel=attachment title="no href">
		<link rel=attachment href="  ">
		<link rel=stylesheet href="style.css">`)
	want := []Attachment{
		{URL: "http://example.org/dir/data.csv", Description: "The data"},
		{URL: "http://other.org/a%20b.txt"},
	}
	if !reflect.DeepEqual(got.Attachments, want) {
		t.Errorf("Attachments = %+v, want %+v", got.Attachments, want)
	}
	lines := logs.Lines()
	if len(lines) != 2 || !strings.Contains(lines[0], "Missing href") {
		t.Errorf("logs: %q", lines)
	}
}

func TestMetadataAttachmentUsesBaseElement(t *testing.T) {
	got := metadataOf(t, `<base href="/files/"><link rel=attachment href="x.bin">`)
	if len(got.Attachments) != 1 || got.Attachments[0].URL != "http://example.org/files/x.bin" {
		t.Errorf("Attachments = %+v", got.Attachments)
	}
}

func TestParseW3CDate(t *testing.T) {
	valid := []string{
		"1997", "1997-07", "1997-07-16", "1997-07-16T19:20+01:00", "1997-07-16T19:20:30+01:00",
		"1997-07-16T19:20:30.45+01:00", "1997-07-16T19:20Z", "  2011-04\n", "2000-12-31T23:59:59Z",
	}
	invalid := []string{
		"", "wrong", "97", "1997-13", "1997-00-1", "1997-07-32", "1997-07-16T19:20", "1997-07-16T25:00Z",
		"1997-07-16T19:60Z", "1997-07-16T19:20+24:00", "1997-07-16T19:20+0100", "1997-07-16 19:20Z", "19970716",
		"1997-07-16T19Z",
	}
	logs := testutil.CaptureLogs(t)
	for _, s := range valid {
		if got := parseW3CDate("dcterms.created", s); got == nil || *got != s {
			t.Errorf("%q rejected or altered: %v", s, got)
		}
	}
	if len(logs.Lines()) != 0 {
		t.Errorf("valid dates logged: %q", logs.Lines())
	}
	for _, s := range invalid {
		if got := parseW3CDate("dcterms.created", s); got != nil {
			t.Errorf("%q accepted", s)
		}
	}
	if len(logs.Lines()) != len(invalid) {
		t.Errorf("%d warnings for %d invalid dates", len(logs.Lines()), len(invalid))
	}
}

func TestMetadataLangAsWritten(t *testing.T) {
	if got := metadataOf(t, `<html lang="EN-us">`); got.Lang == nil || *got.Lang != "EN-us" {
		t.Errorf("Lang = %v", got.Lang)
	}
	if got := metadataOf(t, `<html lang="">`); got.Lang == nil || *got.Lang != "" {
		t.Errorf("empty Lang = %v", got.Lang)
	}
	if got := metadataOf(t, `<body lang=fr>`); got.Lang != nil {
		t.Errorf("Lang = %v taken from body", *got.Lang)
	}
}
