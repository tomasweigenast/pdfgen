package metadata

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// Version is the version written in the Producer entry. Release builds set it
// with -ldflags.
var Version = "dev"

// SetInfo fills the document information dictionary info from the metadata
// of the source document, and the language of the catalog. A nil m only sets
// the producer.
func SetInfo(info, catalog *writer.Dict, m *html.Metadata) {
	info.Set("Producer", writer.Text("pdfgen "+Version))
	if m == nil {
		return
	}
	if m.Title != nil && *m.Title != "" {
		info.Set("Title", writer.Text(*m.Title))
	}
	if len(m.Authors) > 0 {
		info.Set("Author", writer.Text(strings.Join(m.Authors, ", ")))
	}
	if m.Description != nil && *m.Description != "" {
		info.Set("Subject", writer.Text(*m.Description))
	}
	if len(m.Keywords) > 0 {
		info.Set("Keywords", writer.Text(strings.Join(m.Keywords, ", ")))
	}
	if m.Generator != nil && *m.Generator != "" {
		info.Set("Creator", writer.Text(*m.Generator))
	}
	if date, ok := w3cDateToPDF(m.Created, "created"); ok {
		info.Set("CreationDate", writer.Text(date))
	}
	if date, ok := w3cDateToPDF(m.Modified, "modified"); ok {
		info.Set("ModDate", writer.Text(date))
	}
	if m.Lang != nil && *m.Lang != "" {
		catalog.Set("Lang", writer.Text(*m.Lang))
	}
}

var w3cDate = regexp.MustCompile(`^[ \t\n\f\r]*` +
	`(?P<year>[0-9]{4})` +
	`(?:-(?P<month>0[0-9]|1[012])` +
	`(?:-(?P<day>[012][0-9]|3[01])` +
	`(?:T(?P<hour>[01][0-9]|2[0-3]):(?P<minute>[0-5][0-9])` +
	`(?::(?P<second>[0-5][0-9])(?:\.[0-9]+)?)?` +
	`(?:Z|(?P<tzhour>[+-](?:[01][0-9]|2[0-3])):(?P<tzminute>[0-5][0-9]))` +
	`)?)?)?` +
	`[ \t\n\f\r]*$`)

// w3cDateToPDF converts a W3C date-time to the PDF date format (ISO 32000-1
// §7.9.4): D:YYYYMMDDHHmmSSZ or D:YYYYMMDDHHmmSS+HH'mm. The seconds that a
// date-time may omit are filled in, so that "2011-04-21T23:00Z" becomes
// "D:20110421230000Z"; a date has no time zone. ok is false, after a warning, for an invalid date; nil is not an error.
func w3cDateToPDF(s *string, attr string) (date string, ok bool) {
	if s == nil {
		return "", false
	}
	match := w3cDate.FindStringSubmatch(*s)
	if match == nil {
		logging.Warnf("Invalid %s date: %q", attr, *s)
		return "", false
	}
	group := func(name string) string { return match[w3cDate.SubexpIndex(name)] }

	var b strings.Builder
	b.WriteString("D:")
	for _, name := range []string{"year", "month", "day", "hour", "minute", "second"} {
		if v := group(name); v != "" {
			b.WriteString(v)
		} else if group("hour") != "" {
			// Only the seconds can be absent from a date-time.
			b.WriteString("00")
		}
	}
	if group("hour") != "" {
		if tz := group("tzhour"); tz != "" {
			hour, _ := strconv.Atoi(tz)
			minute, _ := strconv.Atoi(group("tzminute"))
			fmt.Fprintf(&b, "%+03d'%02d", hour, minute)
		} else {
			b.WriteByte('Z')
		}
	}
	return b.String(), true
}
