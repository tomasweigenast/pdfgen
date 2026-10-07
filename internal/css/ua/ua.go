package ua

import _ "embed"

var (
	// HTML is the default user agent stylesheet.
	//
	//go:embed html5_ua.css
	HTML string

	// Form holds the default styles of form controls.
	//
	//go:embed html5_ua_form.css
	Form string

	// PresentationalHints translates presentational HTML attributes into CSS.
	//
	//go:embed html5_ph.css
	PresentationalHints string
)
