package cascade

import (
	"sync"

	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/ua"
)

var (
	uaOnce         sync.Once
	uaSheet        *Stylesheet
	uaFormSheet    *Stylesheet
	phSheet        *Stylesheet
	uaCounterStyle counters.Registry
)

// loadUA parses the user agent stylesheets once. Their @counter-style rules
// define the predefined counter styles.
func loadUA() {
	uaOnce.Do(func() {
		uaCounterStyle = counters.Registry{}
		env := &Environment{CounterStyle: uaCounterStyle}
		uaSheet = ParseStylesheet(ua.HTML, "", env)
		uaFormSheet = ParseStylesheet(ua.Form, "", env)
		phSheet = ParseStylesheet(ua.PresentationalHints, "", nil)
	})
}

// UAStylesheets returns the default user agent stylesheets, with the styles
// of form controls when forms is set.
func UAStylesheets(forms bool) []*Stylesheet {
	loadUA()
	if forms {
		return []*Stylesheet{uaSheet, uaFormSheet}
	}
	return []*Stylesheet{uaSheet}
}

// PresentationalHintsStylesheet returns the stylesheet that maps
// presentational HTML attributes to CSS.
func PresentationalHintsStylesheet() *Stylesheet {
	loadUA()
	return phSheet
}

// UACounterStyle returns a copy of the counter styles predefined by the user
// agent stylesheet (CSS Counter Styles 3 §7).
func UACounterStyle() counters.Registry {
	loadUA()
	return uaCounterStyle.Copy()
}
