package scaffold

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

var terminalWriters = map[string]map[string]bool{
	"os":  {"Stdout": true, "Stderr": true},
	"fmt": {"Print": true, "Printf": true, "Println": true},
}

func TestTheEngineNeverWritesToTheTerminalBehindTheTUI(t *testing.T) {
	t.Parallel()

	roots := []string{".", "../config", "../assets", "../pypi"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if ok && terminalWriters[pkg.Name][sel.Sel.Name] {
					t.Errorf("%s uses %s.%s; Bubble Tea owns the terminal, so report through a Reporter",
						path, pkg.Name, sel.Sel.Name)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}
