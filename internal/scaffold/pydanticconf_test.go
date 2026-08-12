package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

func environEnv() *Env {
	return &Env{Cfg: config.Project{ProjectName: "demo", DatabaseType: config.SQLite}, Reporter: NopReporter{}}
}

func pydanticEnv() *Env {
	return &Env{
		Cfg: config.Project{
			ProjectName:       "demo",
			DatabaseType:      config.PostgresDocker,
			SetupPydanticConf: true,
		},
		Reporter: NopReporter{},
	}
}

func environSettingsData() multiEnvSettingsData {
	return multiEnvSettingsData{AllAuth: true, Mailers: true, Env: environEnv().productionEnvExpressions()}
}

func pydanticSettingsData() multiEnvSettingsData {
	return multiEnvSettingsData{AllAuth: true, Mailers: true, Env: pydanticEnv().productionEnvExpressions()}
}

func pydanticConfData() confTemplateData {
	e := pydanticEnv()
	e.Cfg.SetupSentry = true
	return e.confTemplateData()
}

func TestPydanticSettingsReplacesDjangoEnviron(t *testing.T) {
	t.Parallel()

	e := pydanticEnv()

	if got := e.configPackages(); len(got) != 2 || got[0] != PkgPydantic || got[1] != PkgPydanticSettings {
		t.Fatalf("configPackages() = %v, want pydantic and pydantic-settings", got)
	}
	if got := environEnv().configPackages(); len(got) != 1 || got[0] != PkgEnviron {
		t.Fatalf("configPackages() without the add-on = %v, want only %s", got, PkgEnviron)
	}

	settings := injectEnvConfigInto(baseSettingsFixture, e)

	for _, forbidden := range []string{"import environ", "environ.Env()", "env('", "env.bool(", "env.list("} {
		if strings.Contains(settings, forbidden) {
			t.Errorf("typed settings still carry the django-environ form %q:\n%s", forbidden, settings)
		}
	}
	for _, want := range []string{
		"from demo.conf import env",
		"SECRET_KEY = env.SECRET_KEY",
		"DEBUG = env.DEBUG",
		"ALLOWED_HOSTS = env.ALLOWED_HOSTS",
		"'NAME': env.DB_NAME",
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("typed settings are missing %q:\n%s", want, settings)
		}
	}
}

func TestDjangoEnvironRemainsTheDefault(t *testing.T) {
	t.Parallel()

	settings := injectEnvConfigInto(baseSettingsFixture, environEnv())

	if !strings.Contains(settings, "import environ") {
		t.Errorf("a project without the add-on should still use django-environ:\n%s", settings)
	}
	if strings.Contains(settings, "conf import env") {
		t.Errorf("a project without the add-on should not import a conf module:\n%s", settings)
	}
}

// The conf module has to stay importable after the multi-env split turns
// settings.py into a package. An absolute import is what makes that true; a
// relative one would silently need an extra dot and nothing would catch it
// until the generated project failed to start.
func TestMultiEnvKeepsTheTypedSettingsImportResolvable(t *testing.T) {
	t.Parallel()

	e := pydanticEnv()
	root := t.TempDir()
	moduleDir := filepath.Join(root, e.Cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}

	if err := e.writeConfModule(root); err != nil {
		t.Fatalf("writing the conf module: %v", err)
	}

	settings := injectEnvConfigInto(baseSettingsFixture, e)
	base := e.generateBasePy(settings)

	if strings.Contains(base, "from .conf import env") || strings.Contains(base, "from ..conf import env") {
		t.Errorf("base.py uses a relative import for the conf module, which breaks under the split:\n%s", base)
	}
	if !strings.Contains(base, "from demo.conf import env") {
		t.Errorf("base.py lost the absolute conf import:\n%s", base)
	}

	if _, err := os.Stat(filepath.Join(moduleDir, ConfFile)); err != nil {
		t.Errorf("conf.py is not beside settings, so the split would move it: %v", err)
	}
}

// pydantic-settings decodes complex types as JSON, so a bare list[str] would be
// handed 'localhost,127.0.0.1' and raise before Django ever starts. NoDecode
// plus a BeforeValidator is what keeps the .env format this scaffold writes
// readable by the settings it generates.
func TestTypedSettingsReadCommaSeparatedListsFromTheDotEnv(t *testing.T) {
	t.Parallel()

	conf := render("conf.py.tmpl", pydanticConfData())

	for _, want := range []string{
		"NoDecode",
		"BeforeValidator(_split_comma)",
		"CommaSeparated = Annotated[list[str], NoDecode, BeforeValidator(_split_comma)]",
		"ALLOWED_HOSTS: CommaSeparated",
	} {
		if !strings.Contains(conf, want) {
			t.Errorf("conf.py is missing %q, so a comma-separated ALLOWED_HOSTS would be parsed as JSON:\n%s", want, conf)
		}
	}

	if strings.Contains(conf, "ALLOWED_HOSTS: list[str]") {
		t.Error("ALLOWED_HOSTS is annotated as a plain list, which pydantic-settings decodes as JSON")
	}
}

func TestTypedSettingsRequireASecretKey(t *testing.T) {
	t.Parallel()

	conf := render("conf.py.tmpl", pydanticConfData())

	if !strings.Contains(conf, "SECRET_KEY: SecretKey\n") {
		t.Errorf("SECRET_KEY should have no default, so a missing one fails at startup:\n%s", conf)
	}
	if !strings.Contains(conf, "SecretKey = Annotated[str, Field(min_length=50)]") {
		t.Errorf("SECRET_KEY is typed but unbounded, so the empty value .env.example ships would pass:\n%s", conf)
	}
	if strings.Contains(conf, "django-insecure") {
		t.Errorf("typed settings must not carry the insecure development fallback:\n%s", conf)
	}
}

func TestTypedSettingsDeclareOnlyTheDatabaseFieldsTheProjectUses(t *testing.T) {
	t.Parallel()

	sqlite := environEnv()
	sqlite.Cfg.SetupPydanticConf = true

	if got := render("conf.py.tmpl", sqlite.confTemplateData()); strings.Contains(got, "DB_NAME") {
		t.Errorf("a SQLite project should declare no DB_ fields:\n%s", got)
	}

	mysql := pydanticEnv()
	mysql.Cfg.DatabaseType = config.MySQLDocker
	got := render("conf.py.tmpl", mysql.confTemplateData())
	if !strings.Contains(got, "DB_USER: str = 'django_user'") || !strings.Contains(got, "DB_PORT: Port = 3306") {
		t.Errorf("MySQL defaults did not reach conf.py:\n%s", got)
	}
}

func TestMypyLoadsThePydanticPluginOnlyWhenTypedSettingsAreOn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		pydantic bool
		want     bool
	}{
		{name: "typed settings on", pydantic: true, want: true},
		{name: "typed settings off", pydantic: false, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			e := &Env{
				Cfg: config.Project{
					ProjectName:           "demo",
					DjangoVersion:         "6.1",
					SelectedPythonVersion: "Python 3.13.2",
					SetupMypy:             true,
					SetupPydanticConf:     tc.pydantic,
				},
				Reporter: NopReporter{},
				Runner:   command.NewFake(),
			}

			if err := e.applyMypy(root); err != nil {
				t.Fatalf("applyMypy: %v", err)
			}

			written, err := os.ReadFile(filepath.Join(root, PyprojectFile))
			if err != nil {
				t.Fatalf("reading pyproject.toml: %v", err)
			}
			if got := strings.Contains(string(written), `"pydantic.mypy"`); got != tc.want {
				t.Errorf("pydantic.mypy plugin present = %v, want %v:\n%s", got, tc.want, written)
			}
		})
	}
}

const baseSettingsFixture = `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc'

DEBUG = True

ALLOWED_HOSTS = []

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}
`
