package assets

import _ "embed"

//go:embed templ/css/base.css
var baseCSS string

//go:embed templ/css/home.css
var homeCSS string

//go:embed templ/css/app.css
var appCSS string

//go:embed templ/css/auth.css
var authCSS string

//go:embed templ/css/auth-allauth.css
var allauthAuthCSS string

func BaseCSS() string { return baseCSS }

func HomeCSS() string { return homeCSS }

func AppCSS() string { return appCSS }

// AuthCSS is the sign-in pages' stylesheet, with the rules for the pages only
// allauth ships when the project installs it.
func AuthCSS(allauth bool) string {
	if allauth {
		return authCSS + allauthAuthCSS
	}
	return authCSS
}
