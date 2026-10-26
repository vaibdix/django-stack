package scaffold

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/vaibdix/django-stack/internal/config"
	"go.yaml.in/yaml/v3"
)

func parseConfigFile(t *testing.T, name string, content []byte) {
	t.Helper()
	var doc any
	var err error
	switch path.Ext(name) {
	case ".json":
		err = json.Unmarshal(content, &doc)
	case ".yml", ".yaml":
		err = yaml.Unmarshal(content, &doc)
	case ".toml":
		err = toml.Unmarshal(content, &doc)
	default:
		return
	}
	if err != nil {
		t.Errorf("%s does not parse: %v\n%s", name, err, content)
	}
}

func TestEveryGeneratedConfigFileParses(t *testing.T) {
	t.Parallel()

	for name, c := range treeCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resolved := credentialsFor(c.cfg.Database, config.DatabaseCredentials{})
			resolved.PythonVersion = "Python 3.13.2"
			resolved.SecretKey = "parse-check"
			bp, err := Build(specOf(t, c.cfg), resolved, Tools{Toolchain: c.toolchain, NodeManager: c.node})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			for _, file := range bp.Paths() {
				parseConfigFile(t, file, bp.Files[file].Content)
			}

			e := &Env{cfg: mustSpec(c.cfg), resolved: resolved, toolchain: c.toolchain, node: c.node}
			e.packages.addRuntime(e.packagePlan().runtime...)
			e.packages.addDev(e.packagePlan().dev...)
			parseConfigFile(t, pyprojectFile, []byte(e.renderPyproject(e.packages.runtime, e.packages.dev)))

			if c.cfg.UsesTailwind() {
				root := t.TempDir()
				e.dir = root
				if err := e.ensurePackageJSON(); err != nil {
					t.Fatal(err)
				}
				if err := e.updatePackageJSONForTailwind(); err != nil {
					t.Fatal(err)
				}
				content, err := os.ReadFile(filepath.Join(root, packageJSONFile))
				if err != nil {
					t.Fatal(err)
				}
				parseConfigFile(t, packageJSONFile, content)
			}
		})
	}
}
