package scaffold

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
)

// anyFinder walks each source's syntax tree for the name Any: a reference
// (Any, typing.Any, t.Any), an import of it, or a string annotation naming
// it. Docstrings and comments are not code, so they never match.
const anyFinder = `
import ast, json, sys

def names_any(node):
    for sub in ast.walk(node):
        if isinstance(sub, ast.Name) and sub.id == "Any":
            return True
        if isinstance(sub, ast.Attribute) and sub.attr == "Any":
            return True
        if isinstance(sub, ast.alias) and sub.name == "Any":
            return True
    return False

def annotations(node):
    if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
        args = node.args.posonlyargs + node.args.args + node.args.kwonlyargs
        args += [a for a in (node.args.vararg, node.args.kwarg) if a is not None]
        return [a.annotation for a in args] + [node.returns]
    if isinstance(node, ast.AnnAssign):
        return [node.annotation]
    return []

found = {}
for name, source in json.load(sys.stdin).items():
    lines = set()
    for node in ast.walk(ast.parse(source, name)):
        if isinstance(node, (ast.Name, ast.Attribute, ast.Import, ast.ImportFrom)) and names_any(node):
            lines.add(node.lineno)
        for annotation in annotations(node):
            if isinstance(annotation, ast.Constant) and isinstance(annotation.value, str):
                if names_any(ast.parse(annotation.value, mode="eval")):
                    lines.add(annotation.lineno)
    if lines:
        found[name] = sorted(lines)
json.dump(found, sys.stdout)
`

// anyUsages parses every source with Python 3.12 and returns, by source name,
// the lines that use Any. A source that does not parse fails the test.
func anyUsages(t *testing.T, sources map[string]string) map[string][]int {
	t.Helper()

	uv, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("uv is not installed")
	}
	input, err := json.Marshal(sources)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), uv, "run", "--no-project", "--python", "3.12", "python", "-I", "-c", anyFinder)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("parsing the generated Python failed:\n%s", exitErr.Stderr)
		}
		t.Fatalf("running the Any finder: %v", err)
	}
	var found map[string][]int
	if err := json.Unmarshal(out, &found); err != nil {
		t.Fatalf("reading the Any finder's answer %q: %v", out, err)
	}
	return found
}

func TestGeneratedPythonUsesNoAnyAccordingToItsSyntaxTree(t *testing.T) {
	t.Parallel()

	for name, lines := range anyUsages(t, everyGeneratedPythonSource(t)) {
		t.Errorf("%s uses Any on lines %v; object, a TypedDict or a pydantic model says more", name, lines)
	}
}

func TestTheAnyFinderCatchesEverySpelling(t *testing.T) {
	t.Parallel()

	spellings := map[string]string{
		"annotation": "from typing import Any\ndef f(v: Any) -> None: ...\n",
		"cast":       "from typing import cast, Any\nx = cast(Any, 1)\n",
		"module":     "import typing as t\ndef f(v: t.Any) -> None: ...\n",
		"string":     "def f(v: 'dict[str, Any]') -> None: ...\n",
		"prose":      "def f(v: object) -> object:\n    \"\"\"dict[str, Any] is why this exists.\"\"\"\n    return v  # Any too\n",
	}
	found := anyUsages(t, spellings)
	for _, name := range []string{"annotation", "cast", "module", "string"} {
		if len(found[name]) == 0 {
			t.Errorf("the finder missed Any spelled as %s, so the guard proves nothing", name)
		}
	}
	if len(found["prose"]) != 0 {
		t.Errorf("a docstring and a comment mentioning Any were read as code: lines %v", found["prose"])
	}
}
