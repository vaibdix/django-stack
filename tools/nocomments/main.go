package main

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/unitchecker"
)

var directives = []string{"//go:", "//nolint", "//lint:"}

var analyzer = &analysis.Analyzer{
	Name: "nocomments",
	Doc:  "reports every comment that is not a machine directive (//go:, //nolint, //lint:) or the package doc in doc.go",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		packageDoc := filepath.Base(pass.Fset.File(file.Pos()).Name()) == "doc.go"
		for _, group := range file.Comments {
			if packageDoc && group == file.Doc {
				continue
			}
			for _, comment := range group.List {
				if !isDirective(comment.Text) {
					pass.Reportf(comment.Pos(), "comments are not allowed in Go code; explain it in the README or the PR instead")
				}
			}
		}
	}
	return nil, nil
}

func isDirective(text string) bool {
	for _, prefix := range directives {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func main() { unitchecker.Main(analyzer) }
