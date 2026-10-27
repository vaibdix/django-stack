package render

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

var machinePackages = []string{
	"os",
	"os/exec",
	"crypto/rand",
	"math/rand/v2",
	"net",
	"net/http",
	"github.com/vaibdix/django-stack/internal/adapter",
	"github.com/vaibdix/django-stack/internal/command",
	"github.com/vaibdix/django-stack/internal/dbdocker",
	"github.com/vaibdix/django-stack/internal/pypi",
	"github.com/vaibdix/django-stack/internal/releases",
}

func TestNothingInRenderTouchesTheMachine(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo}, ".")
	if err != nil || len(pkgs) != 1 || len(pkgs[0].Errors) > 0 {
		t.Fatalf("load the render package: %v %v", err, pkgs)
	}
	if found := machineUses(pkgs[0]); len(found) > 0 {
		t.Errorf("rendering a project must not read or change the machine, but:\n%s", strings.Join(found, "\n"))
	}
}

func machineUses(pkg *packages.Package) []string {
	bodies := map[types.Object]*ast.FuncDecl{}
	reached := map[types.Object]string{}
	var queue []types.Object
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				obj := pkg.TypesInfo.Defs[fn.Name]
				bodies[obj] = fn
				reached[obj] = obj.Name()
				queue = append(queue, obj)
			}
		}
	}
	var found []string
	for len(queue) > 0 {
		fn := queue[0]
		queue = queue[1:]
		ast.Inspect(bodies[fn], func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := pkg.TypesInfo.Uses[id]
			switch {
			case obj == nil || obj.Pkg() == nil:
			case touchesMachine(obj):
				found = append(found, reached[fn]+" uses "+obj.Pkg().Name()+"."+obj.Name())
			case obj.Pkg() == pkg.Types && bodies[obj] != nil && reached[obj] == "":
				reached[obj] = reached[fn] + " → " + obj.Name()
				queue = append(queue, obj)
			}
			return true
		})
	}
	slices.Sort(found)
	return slices.Compact(found)
}

func touchesMachine(obj types.Object) bool {
	if obj.Pkg().Path() == "time" && obj.Name() == "Now" {
		return true
	}
	return slices.Contains(machinePackages, obj.Pkg().Path()) && !isPureMachineName(obj)
}

func isPureMachineName(obj types.Object) bool {
	if _, ok := obj.(*types.TypeName); ok {
		return true
	}
	if _, ok := obj.(*types.Const); ok {
		return true
	}
	return slices.Contains([]string{"dbdocker.Compose", "dbdocker.DatabaseName", "pypi.NormalizeName"}, obj.Pkg().Name()+"."+obj.Name())
}
