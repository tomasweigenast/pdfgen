package html

import (
	"regexp"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// Metadata is the document information found in the head of a document
// (HTML Living Standard §4.2.5, standard metadata names).
type Metadata struct {
	// Title is the text of the first title element. It is nil when there
	// is none.
	Title *string
	// Description, Generator, Created and Modified come from the first meta
	// element of each name. Created and Modified hold W3C date-time
	// strings.
	Description *string
	Generator   *string
	Created     *string
	Modified    *string
	// Keywords lists the distinct keywords of all keywords meta elements.
	Keywords []string
	// Authors holds the content of every author meta element, in order.
	Authors []string
	// Attachments lists the link elements with rel=attachment.
	Attachments []Attachment
	// Lang is the lang attribute of the root element, as written.
	Lang *string
	// Custom holds the content of the first meta element of every other
	// name, which is lower-cased.
	Custom map[string]string
}

// Attachment is a file to embed in the output, announced by a link element
// with rel=attachment.
type Attachment struct {
	// URL is the absolute URL of the file.
	URL string
	// Description is the title attribute, or "".
	Description string
}

// Metadata extracts the metadata of the document. Invalid dates and
// attachments without a usable href are reported and skipped.
func (d *Document) Metadata() *Metadata {
	m := &Metadata{Custom: map[string]string{}}
	if v, ok := d.Root.Lookup("lang"); ok {
		m.Lang = &v
	}
	for _, e := range d.elements {
		if e.namespace != XHTMLNamespace {
			continue
		}
		switch e.node.Data {
		case "title":
			if m.Title == nil {
				title := e.ChildText()
				m.Title = &title
			}
		case "meta":
			m.addMeta(e)
		case "link":
			if e.HasLinkType("attachment") {
				m.addAttachment(d, e)
			}
		}
	}
	return m
}

func (m *Metadata) addMeta(e *Element) {
	name := syntax.AsciiLower(e.Get("name"))
	content := e.Get("content")
	switch name {
	case "keywords":
		for _, keyword := range strings.Split(content, ",") {
			keyword = strings.Trim(keyword, asciiWhitespace)
			if !contains(m.Keywords, keyword) {
				m.Keywords = append(m.Keywords, keyword)
			}
		}
	case "author":
		m.Authors = append(m.Authors, content)
	case "description":
		if m.Description == nil {
			m.Description = &content
		}
	case "generator":
		if m.Generator == nil {
			m.Generator = &content
		}
	case "dcterms.created":
		if m.Created == nil {
			m.Created = parseW3CDate(name, content)
		}
	case "dcterms.modified":
		if m.Modified == nil {
			m.Modified = parseW3CDate(name, content)
		}
	case "":
	default:
		if _, ok := m.Custom[name]; !ok {
			m.Custom[name] = content
		}
	}
}

func (m *Metadata) addAttachment(d *Document, e *Element) {
	url, ok := ParseURL(e.Get("href"), d.BaseURL, false)
	if !ok {
		logging.Errorf(`Missing href in <link rel="attachment">`)
		return
	}
	m.Attachments = append(m.Attachments, Attachment{URL: url, Description: e.Get("title")})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// w3cDate matches the date and time formats of the W3C profile of ISO 8601
// (https://www.w3.org/TR/NOTE-datetime): YYYY, YYYY-MM, YYYY-MM-DD and
// YYYY-MM-DDThh:mm[:ss[.s]] followed by Z or a numeric offset.
var w3cDate = regexp.MustCompile(`^[ \t\n\f\r]*` +
	`[0-9]{4}` +
	`(?:-(?:0[0-9]|1[012])` +
	`(?:-(?:[012][0-9]|3[01])` +
	`(?:T(?:[01][0-9]|2[0-3]):[0-5][0-9]` +
	`(?::[0-5][0-9](?:\.[0-9]+)?)?` +
	`(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])` +
	`)?)?)?` +
	`[ \t\n\f\r]*$`)

// parseW3CDate returns a pointer to the date string when it is valid, and
// nil, after a warning, otherwise.
func parseW3CDate(metaName, s string) *string {
	if w3cDate.MatchString(s) {
		return &s
	}
	logging.Warnf(`Invalid date in <meta name="%s"> '%s'`, metaName, s)
	return nil
}
