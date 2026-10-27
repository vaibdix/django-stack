package scaffold

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/livetest"
)

func randomAppNames(seed uint64, lengths ...int) []string {
	r := rand.New(rand.NewPCG(seed, seed))
	const first, rest = "abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz0123456789_"
	names := make([]string, 0, len(lengths))
	for _, n := range lengths {
		name := []byte{first[r.IntN(len(first))]}
		for len(name) < n {
			name = append(name, rest[r.IntN(len(rest))])
		}
		names = append(names, string(name))
	}
	return names
}

func TestTemplatesNeedNoDjlintReformatForAnyAppName(t *testing.T) {
	t.Parallel()
	livetest.Require(t, "run djlint's formatter over templates rendered for random app names")

	cmd, ok := djlintCommand(t, "--version")
	if !ok || cmd.Run() != nil {
		livetest.MissingTool(t, "djlint is not runnable")
	}
	root := t.TempDir()
	for _, name := range randomAppNames(7, 1, 2, 5, 9, 14, 20, 27, 35, 44, 60) {
		for variant, cfg := range appTemplateVariants(name) {
			for file, content := range everyGeneratedTemplate(cfg) {
				path := filepath.Join(root, name, variant, file)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	check, _ := djlintCommand(t, "--check", "--profile=django", ".")
	check.Dir = root
	if out, err := check.CombinedOutput(); err != nil {
		t.Errorf("djlint would reformat templates rendered for some app name, so the tag wrapping "+
			"the scaffold reimplements has drifted from djlint's:\n%s", lastLines(string(out), 60))
	}
}
