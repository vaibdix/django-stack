package config_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestValidateProjectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		projectName string
		wantErr     string
	}{
		{name: "simple lowercase", projectName: "myproject"},
		{name: "hyphenated becomes valid module", projectName: "my-project"},
		{name: "underscored", projectName: "my_project"},
		{name: "trailing digits", projectName: "project2"},
		{name: "mixed case", projectName: "MyProject"},

		{name: "empty", projectName: "", wantErr: "cannot be empty"},
		{name: "leading digit", projectName: "2project", wantErr: "must start with a letter"},
		{name: "leading hyphen", projectName: "-project", wantErr: "must start with a letter"},
		{name: "space", projectName: "my project", wantErr: "may only contain"},
		{name: "dot", projectName: "my.project", wantErr: "may only contain"},
		{name: "slash traversal", projectName: "../escape", wantErr: "may only contain"},
		{name: "reserved word", projectName: "class", wantErr: "Python keyword"},
		{name: "reserved word via hyphen", projectName: "import", wantErr: "Python keyword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := config.Project{ProjectName: tt.projectName}
			err := p.ValidateProjectName(py313)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateProjectName(%q) = %v, want nil", tt.projectName, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateProjectName(%q) = nil, want error containing %q", tt.projectName, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateProjectName(%q) = %q, want error containing %q", tt.projectName, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectNameRejectsNamesWhoseModuleWouldBeReserved(t *testing.T) {
	t.Parallel()

	p := config.Project{ProjectName: "lambda"}

	if err := p.ValidateProjectName(py313); err == nil {
		t.Fatal("ValidateProjectName(\"lambda\") = nil, want a keyword error")
	}
}

func TestValidateAppName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appName string
		wantErr string
	}{
		{name: "empty is allowed because the app is optional", appName: ""},
		{name: "simple", appName: "blog"},
		{name: "underscored", appName: "my_app"},
		{name: "leading underscore", appName: "_internal"},

		{name: "hyphen is not a python identifier", appName: "my-app", wantErr: "not a valid Python identifier"},
		{name: "leading digit", appName: "2app", wantErr: "not a valid Python identifier"},
		{name: "reserved word", appName: "return", wantErr: "Python keyword"},
		{name: "space", appName: "my app", wantErr: "not a valid Python identifier"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := config.Project{AppName: tt.appName}
			err := p.ValidateAppName(py313)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateAppName(%q) = %v, want nil", tt.appName, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateAppName(%q) = nil, want error containing %q", tt.appName, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateAppName(%q) = %q, want error containing %q", tt.appName, err, tt.wantErr)
			}
		})
	}
}

func TestModuleNameReplacesHyphens(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"myproject":     "myproject",
		"my-project":    "my_project",
		"my-long-name":  "my_long_name",
		"already_snake": "already_snake",
		"":              "",
	}

	for projectName, want := range tests {
		p := config.Project{ProjectName: projectName}
		if got := p.ModuleName(); got != want {
			t.Errorf("ModuleName() with ProjectName=%q = %q, want %q", projectName, got, want)
		}
	}
}

func TestApplyFeaturesMapsSelectionsToFlags(t *testing.T) {
	t.Parallel()

	var p config.Project
	selected := []config.Feature{
		config.FeatureStandardProject,
		config.FeatureTailwind,
		config.FeatureRestFramework,
		config.FeatureHTMX,
		config.FeatureAllAuth,
		config.FeatureRuff,
		config.FeatureMypy,
		config.FeatureMultiEnv,
	}

	p.ApplyFeatures(selected)

	checks := map[string]bool{
		"CreateTemplates":    p.CreateTemplates,
		"SetupTailwind":      p.SetupTailwind,
		"SetupRestFramework": p.SetupRestFramework,
		"SetupHTMX":          p.SetupHTMX,
		"SetupAllAuth":       p.SetupAllAuth,
		"SetupRuff":          p.SetupRuff,
		"SetupMypy":          p.SetupMypy,
		"SetupMultiEnv":      p.SetupMultiEnv,
	}
	for field, enabled := range checks {
		if !enabled {
			t.Errorf("%s = false, want true", field)
		}
	}
	if p.SetupPreCommit {
		t.Error("SetupPreCommit = true, want false (it was not selected)")
	}
	if p.InitializeGit {
		t.Error("InitializeGit = true, want false (it was not selected and pre-commit is off)")
	}
}

func TestApplyFeaturesAutoEnablesGitForPreCommit(t *testing.T) {
	t.Parallel()

	var p config.Project
	selected := []config.Feature{config.FeaturePreCommit}

	p.ApplyFeatures(selected)

	if !p.SetupPreCommit {
		t.Fatal("SetupPreCommit = false, want true")
	}
	if !p.InitializeGit {
		t.Error("InitializeGit = false, want true: pre-commit hooks require a git repository")
	}
}

func TestApplyFeaturesClearsStaleFlagsOnReRun(t *testing.T) {
	t.Parallel()

	var p config.Project
	p.ApplyFeatures([]config.Feature{config.FeatureTailwind, config.FeatureRuff, config.FeatureAllAuth})
	p.ApplyFeatures([]config.Feature{config.FeatureMypy})

	if p.SetupTailwind {
		t.Error("SetupTailwind = true after deselection, want false")
	}
	if p.SetupRuff {
		t.Error("SetupRuff = true after deselection, want false")
	}
	if p.SetupAllAuth {
		t.Error("SetupAllAuth = true after deselection, want false")
	}
	if !p.SetupMypy {
		t.Error("SetupMypy = false, want true")
	}
}

func TestApplyFeaturesIgnoresUnknownFeatures(t *testing.T) {
	t.Parallel()

	var p config.Project
	selected := []config.Feature{config.Feature("Not A Real Option")}

	p.ApplyFeatures(selected)

	if p.CreateTemplates || p.SetupTailwind || p.SetupRuff || p.SetupMypy || p.SetupMultiEnv {
		t.Error("an unrecognised option enabled a feature flag")
	}
}

func TestNamesThatWouldCollideAreRejectedBeforeAnythingIsBuilt(t *testing.T) {
	t.Parallel()

	appCases := map[string]string{
		"accounts":  "already used by the generated project",
		"core":      "already used by the generated project",
		"templates": "already used by the generated project",
		"test":      "standard library",
		"email":     "standard library",
		"logging":   "standard library",
		"ninja":     "django-ninja",
		"pydantic":  "pydantic",
		"structlog": "structlog",
		"shop":      "project's own package",
	}
	for app, want := range appCases {
		err := config.Project{ProjectName: "shop", AppName: app}.ValidateAppName(py313)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("app %q: err = %v, want it to mention %q", app, err, want)
		}
	}

	for _, project := range []string{"core", "accounts", "email", "sentry_sdk", "static"} {
		if err := (config.Project{ProjectName: project}).ValidateProjectName(py313); err == nil {
			t.Errorf("project %q was accepted although its package would collide", project)
		}
	}

	for _, fine := range []string{"blog", "catalog", "shop_core", "emails"} {
		if err := (config.Project{ProjectName: "shop", AppName: fine}).ValidateAppName(py313); err != nil {
			t.Errorf("app %q was rejected: %v", fine, err)
		}
	}
}

var py313 = config.PythonNames{
	Keywords: []string{
		"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class", "continue",
		"def", "del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in",
		"is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield",
	},
	Stdlib: []string{"email", "json", "logging", "os", "sys", "test"},
}

func TestAModuleTheInterpreterShipsIsRefusedOnlyWhereItShips(t *testing.T) {
	t.Parallel()

	py310 := config.PythonNames{Stdlib: []string{"os", "telnetlib"}}
	py313 := config.PythonNames{Stdlib: []string{"os"}}
	p := config.Project{ProjectName: "telnetlib"}
	if err := p.ValidateProjectName(py310); err == nil || !strings.Contains(err.Error(), "standard library") {
		t.Errorf("telnetlib on Python 3.10: err = %v, want the standard-library refusal", err)
	}
	if err := p.ValidateProjectName(py313); err != nil {
		t.Errorf("telnetlib on Python 3.13, which removed it: err = %v, want it accepted", err)
	}
}
