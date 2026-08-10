package assets

import _ "embed"

// The stylesheets are embedded directly into string variables rather than read
// out of an embed.FS at init.
//
// This matters: the previous version loaded them through embedded.ReadFile and
// discarded the error, so renaming or moving a stylesheet produced a binary
// that shipped an empty <style> block in every generated page — no build
// failure, no runtime error, just silently unstyled output. With a direct
// //go:embed on a string, a missing file is a compile error.

//go:embed templ/css/base.css
var baseCSS string

//go:embed templ/css/home.css
var homeCSS string

// BaseCSS returns the stylesheet shared by every generated page.
func BaseCSS() string { return baseCSS }

// HomeCSS returns the landing-page stylesheet.
func HomeCSS() string { return homeCSS }
