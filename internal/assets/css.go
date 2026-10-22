package assets

import _ "embed"

//go:embed static/css/base.css
var baseCSS string

//go:embed static/css/home.css
var homeCSS string

//go:embed static/css/app.css
var appCSS string

//go:embed static/css/auth.css
var authCSS string

//go:embed static/css/auth-allauth.css
var allauthAuthCSS string

func BaseCSS() string { return baseCSS }

func HomeCSS() string { return homeCSS }

func AppCSS() string { return appCSS }

func AuthCSS(allauth bool) string {
	if allauth {
		return authCSS + allauthAuthCSS
	}
	return authCSS
}
