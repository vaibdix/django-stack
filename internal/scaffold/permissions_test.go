package scaffold

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestSecretFilesAreNotWorldReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes are not meaningful on Windows")
	}

	projectPath := t.TempDir()

	e := &Env{Cfg: withCredentials(config.Project{
		ProjectName:            "my-shop",
		DatabaseType:           config.PostgresDocker,
		PostgresDockerPassword: "hunter2",
	})}

	if err := e.writeEnvFile(projectPath); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}

	info, err := os.Stat(filepath.Join(projectPath, EnvFile))
	if err != nil {
		t.Fatalf("stat %s: %v", EnvFile, err)
	}

	const ownerOnly = 0o600
	if got := info.Mode().Perm(); got != ownerOnly {
		t.Errorf("%s has mode %#o, want %#o — it contains the SECRET_KEY and the database password",
			EnvFile, got, ownerOnly)
	}
}

func TestEnvExampleIsCommittableAndCarriesNoSecret(t *testing.T) {
	projectPath := t.TempDir()

	e := &Env{Cfg: withCredentials(config.Project{
		ProjectName:            "my-shop",
		DatabaseType:           config.PostgresDocker,
		PostgresDockerPassword: "hunter2",
	})}

	if err := e.writeEnvExampleFile(projectPath); err != nil {
		t.Fatalf("writeEnvExampleFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectPath, EnvExampleFile))
	if err != nil {
		t.Fatalf("reading %s: %v", EnvExampleFile, err)
	}

	if string(data) == "" {
		t.Fatal("the example file is empty")
	}
	if contains := string(data); len(contains) > 0 {
		for _, secret := range []string{"hunter2"} {
			if idx := indexOf(contains, secret); idx >= 0 {
				t.Errorf("%s leaks the real password %q at byte %d", EnvExampleFile, secret, idx)
			}
		}
	}
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
