package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/render"
)

func TestTailwindInputIsNotAStaticFile(t *testing.T) {
	t.Parallel()

	if render.TailwindSourceDir == "static" || strings.HasPrefix(render.TailwindSourceDir, "static/") {
		t.Fatalf("the Tailwind input sits in %s, inside STATICFILES_DIRS: collectstatic copies it and "+
			"the manifest storage aborts on its @import \"tailwindcss\"", render.TailwindSourceDir)
	}

	projectPath := t.TempDir()
	e := &Env{Cfg: mustSpec(projectForStack("tailwind", ""))}
	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.at(projectPath).updatePackageJSONForTailwind(); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	want := "-i ./" + render.TailwindSourceDir + "/" + render.TailwindSourceFile + " -o ./static/dist/styles.css"
	if !strings.Contains(string(manifest), want) {
		t.Errorf("package.json does not build %q:\n%s", want, manifest)
	}
}
