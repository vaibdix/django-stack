//go:build !never

package sample

import _ "embed"

//go:embed sample.go
var source string

// an explanation // want "comments are not allowed"
func Explained() string { return source }

func Directive() int { //nolint:gocritic
	return 1 /* a block comment */ // want "comments are not allowed" "comments are not allowed"
}
