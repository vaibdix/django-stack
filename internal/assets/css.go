package assets

import _ "embed"

//go:embed templ/css/base.css
var baseCSS string

//go:embed templ/css/home.css
var homeCSS string

func BaseCSS() string { return baseCSS }

func HomeCSS() string { return homeCSS }
