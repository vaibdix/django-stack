package render

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
)

const mypyTimeout = 3 * time.Minute

func TestTheHTMXAppSurvivesStrictMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	writeTypedHTMXApp(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "mypy", "--with", "django", "--with", "django-stubs",
		"--with", "django-htmx", "--with", "pydantic", "--with", "pydantic-settings",
		"mypy", "--config-file", "mypy.ini", "--ignore-missing-imports",
		"--no-error-summary", "--no-incremental", "catalog")
	cmd.Dir = root

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("mypy rejects the htmx code this scaffold generates, which is the gap that let "+
			"request.htmx ship as an attr-defined error:\n%s", out)
	}
}

const htmxMypyConfig = `[mypy]
strict = True
disallow_any_explicit = True
follow_imports = silent
plugins = pydantic.mypy

[pydantic-mypy]
init_typed = True
init_forbid_extra = True
`

func writeTypedHTMXApp(t *testing.T, root string) {
	t.Helper()

	appDir := filepath.Join(root, "catalog")
	for _, dir := range []string{appDir, filepath.Join(root, "core")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "mypy.ini"):            htmxMypyConfig,
		filepath.Join(root, "core", "__init__.py"): "",
		filepath.Join(root, "core", "models.py"):   RenderStatic("core_models.py.tmpl"),
		filepath.Join(appDir, "__init__.py"):       "",
		filepath.Join(appDir, "htmx.py"):           generateHtmxSupport(),
		filepath.Join(appDir, "fields.py"):         generateBookFields(typingData{RuntimeGenerics: true}),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	htmx := files[filepath.Join(appDir, "htmx.py")]
	if !strings.Contains(htmx, "HtmxDetails") {
		t.Fatal("htmx.py does not reference django-htmx, so this test would prove nothing")
	}
}

func TestTheNinjaSchemasSurviveStrictMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	apiDir := filepath.Join(root, "catalog", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", apiDir, err)
	}
	files := map[string]string{
		filepath.Join(root, "mypy.ini"):               htmxMypyConfig,
		filepath.Join(root, "catalog", "__init__.py"): "",
		filepath.Join(apiDir, "__init__.py"):          "",
		filepath.Join(apiDir, "schemas.py"):           generateNinjaPackage("catalog", "demo")["schemas.py"],
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "mypy", "--with", "django", "--with", "django-stubs", "--with", "django-ninja",
		"mypy", "--config-file", "mypy.ini", "--ignore-missing-imports",
		"--no-error-summary", "--no-incremental", "catalog")
	cmd.Dir = root

	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Errorf("mypy rejects the Ninja schemas this scaffold generates:\n%s", out)
	}
}

func TestMissingImportsAreIgnoredOnlyForPackagesWithoutTypes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		pydantic, allauth bool
		want              string
	}{
		{pydantic: false, allauth: false, want: `module = ["environ"]`},
		{pydantic: true, allauth: true, want: `module = ["allauth.*"]`},
		{pydantic: false, allauth: true, want: `module = ["environ", "allauth.*"]`},
		{pydantic: true, allauth: false, want: ""},
	} {
		e := &Project{Cfg: mustSpec(config.Choices{AddOns: addOnsWhere(map[config.AddOn]bool{config.Pydantic: tc.pydantic, config.AllAuth: tc.allauth})})}
		got := e.untypedImportsOverride()
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("pydantic=%v allauth=%v: override = %q, want one naming %q", tc.pydantic, tc.allauth, got, tc.want)
		}
	}
}
