package config

import (
	"strings"
	"testing"
)

// ValidateProjectName accepts hyphens, ModuleName rewrites them to underscores,
// and the result is then used as a Python package name. That is a transform
// feeding a validator feeding a code generator — three rules that have to agree
// about the same string, with no single place that states the invariant.
//
// These fuzz targets pin the invariant: whatever ValidateProjectName accepts,
// ModuleName must turn into a legal Python identifier. A counterexample means a
// project name the tool accepts but Django cannot import.

// FuzzValidateProjectNameImpliesValidModule is the central claim: accepted
// project name in, importable Python module out.
func FuzzValidateProjectNameImpliesValidModule(f *testing.F) {
	seeds := []string{
		"myproject", "my-project", "my_project", "a", "A1",
		"my--project", "project-", "-project", "", "1project",
		"class", "import-", "for", "None",
		"my-class", "my-import", // hyphenated forms of reserved words
		"σigma", "my project", "my.project", "my/project",
		strings.Repeat("a", 300),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		p := Project{ProjectName: name}
		if p.ValidateProjectName() != nil {
			return // rejected, nothing more to prove
		}

		module := p.ModuleName()

		if err := ValidatePythonIdentifier(module); err != nil {
			t.Errorf("ValidateProjectName accepted %q, but its module name %q is not a valid Python identifier: %v",
				name, module, err)
		}
		if strings.Contains(module, "-") {
			t.Errorf("module name %q derived from %q still contains a hyphen", module, name)
		}
		if module == "" {
			t.Errorf("project name %q produced an empty module name", name)
		}
	})
}

// ModuleName must be stable: deriving a module name from an already-derived one
// changes nothing. Several call sites apply it more than once.
func FuzzModuleNameIsIdempotent(f *testing.F) {
	for _, s := range []string{"my-project", "a-b-c", "already_fine", "-", "--"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		once := Project{ProjectName: name}.ModuleName()
		twice := Project{ProjectName: once}.ModuleName()
		if once != twice {
			t.Errorf("ModuleName is not idempotent for %q: %q then %q", name, once, twice)
		}
	})
}

// ValidateAppName feeds the same Python-identifier rule, and an empty app name
// is explicitly valid because the app is optional.
func FuzzValidateAppName(f *testing.F) {
	for _, s := range []string{"", "catalog", "my-app", "class", "1app", "_private"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		err := Project{AppName: name}.ValidateAppName()

		if name == "" {
			if err != nil {
				t.Errorf("an empty app name must be valid, got %v", err)
			}
			return
		}
		if err == nil {
			if idErr := ValidatePythonIdentifier(name); idErr != nil {
				t.Errorf("ValidateAppName accepted %q which is not a Python identifier: %v", name, idErr)
			}
		}
	})
}
