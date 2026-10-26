package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

type everyTemplateField struct {
	Generated string
	IndexName string

	AppName                 string
	AppNameTitle            string
	HTMX                    bool
	Tailwind                bool
	InputClass              string
	RuffEnabled             bool
	MypyEnabled             bool
	UVAvailable             bool
	DjangoTrack             string
	DjLintVersion           string
	MypyEntry               string
	RuffCheckEntry          string
	RuffFormatEntry         string
	DjLintEntry             string
	PreCommitHooksRev       string
	DjangoUpgradeRev        string
	ProjectDirName          string
	AllAuth                 bool
	Mailers                 bool
	NativeCSP               bool
	Database                bool
	DBUser                  string
	DBHost                  string
	UpsertTarget            bool
	DBPort                  string
	Sentry                  bool
	Structlog               bool
	ModuleName              string
	PythonVersion           string
	DatabaseLabel           string
	TestCommand             string
	AdminURL                string
	Docker                  bool
	MultiEnv                bool
	REST                    bool
	Ruff                    bool
	Mypy                    bool
	PreCommit               bool
	Templates               bool
	UV                      bool
	Python                  string
	Tool                    string
	ProjectName             string
	Pydantic                bool
	RuntimeGenerics         bool
	HTMLViews               bool
	Postgres                bool
	TestSecretKey           string
	PlaceholderSecretKey    string
	DeployPython            string
	DeployTool              string
	DRF                     bool
	Ninja                   bool
	DjangoAuth              bool
	Node                    string
	Env                     productionEnvExpressions
	AccountEmail            bool
	AllAuthPages            bool
	FromEmailGuard          string
	BrowserReloadApp        string
	BrowserReloadMiddleware string
	ManifestStaticStorage   string
	Environ                 bool
	EnvBool                 string
	SecretKey               string
	AllowedHosts            string
	Engine                  string
	User                    string
	Password                string
	Host                    string
	Port                    string
	Options                 []string
	ConnMaxAge              string
	Modules                 string
	Bases                   string
	Decorators              string
	LazyAnnotations         bool
	OS                      string
	Arch                    string
	Stubs52                 bool
	Language                string
	RuffDeps                []string
	DjLintDeps              []string
	MypyDeps                []string
	Module                  string
	Series                  string
	Settings                string
	Django                  string
	Imports                 []string
	BaseDir                 bool
	Apps                    []string
	Middleware              []string
	DefaultAutoField        bool
	EnvBlock                string
	SettingsPackage         bool
	Tail                    string
	Target                  string
	FirstParty              string
	RuntimeEvaluated        string
	Plugins                 string
	SettingsModule          string
	Untyped                 string
	Debug                   string
	Level                   string
	Name                    string
	DSN                     string
	Environment             string
	Release                 string
	SendPII                 string
	Traces                  string
	Profiles                string
	Lazy                    bool
}

func fullTemplateData(htmx bool) everyTemplateField {
	return everyTemplateField{
		AppName:                 "catalog",
		Generated:               "2026-01-01 00:00",
		IndexName:               "catalog_boo_created_46e110_idx",
		AppNameTitle:            "Catalog",
		HTMX:                    htmx,
		Tailwind:                true,
		InputClass:              "cls",
		RuffEnabled:             true,
		MypyEnabled:             true,
		UVAvailable:             true,
		DjangoTrack:             "5.2",
		ProjectDirName:          "my-shop",
		AllAuth:                 true,
		Mailers:                 true,
		Database:                true,
		DBUser:                  "postgres",
		DBHost:                  "localhost",
		UpsertTarget:            true,
		DBPort:                  "5432",
		Sentry:                  true,
		Structlog:               true,
		ModuleName:              "my_shop",
		PythonVersion:           "3.13",
		DatabaseLabel:           "PostgreSQL",
		TestCommand:             "python manage.py test",
		AdminURL:                "admin/",
		Docker:                  true,
		MultiEnv:                true,
		REST:                    true,
		Ruff:                    true,
		Mypy:                    true,
		PreCommit:               true,
		Templates:               true,
		ProjectName:             "my-shop",
		Pydantic:                true,
		RuntimeGenerics:         true,
		HTMLViews:               true,
		Postgres:                true,
		DRF:                     true,
		Ninja:                   true,
		DjangoAuth:              true,
		Node:                    "pnpm",
		TestSecretKey:           testSecretKey,
		PlaceholderSecretKey:    placeholderSecretKey,
		BrowserReloadApp:        appBrowserReload,
		BrowserReloadMiddleware: middlewareBrowserReload,
		ManifestStaticStorage:   manifestStaticStorage,
		Environ:                 true,
		EnvBool:                 "def env_bool(): ...\n",
		SecretKey:               `env("SECRET_KEY")`,
		AllowedHosts:            "[]",
		Engine:                  "django.db.backends.postgresql",
		User:                    `env("DB_USER")`,
		Password:                `env("DB_PASSWORD")`,
		Host:                    `env("DB_HOST")`,
		Port:                    `env("DB_PORT")`,
		Options:                 []string{`"charset": "utf8mb4"`},
		ConnMaxAge:              "60",
		Modules:                 `"environ"`,
		Bases:                   `["pydantic.BaseModel"]`,
		Decorators:              `["ninja.Router.get"]`,
		LazyAnnotations:         true,
		OS:                      "linux",
		Arch:                    "amd64",
		Env:                     (&Env{}).productionEnvExpressions(),
		AccountEmail:            true,
		AllAuthPages:            true,
		Stubs52:                 true,
		Language:                "python",
		RuffDeps:                []string{"ruff==0.17.1"},
		DjLintDeps:              []string{"djlint==1.46.6"},
		MypyDeps:                []string{"mypy==2.4.0"},
		FromEmailGuard:          (&Env{cfg: mustSpec(config.Choices{AddOns: config.NewSet(config.AllAuth)})}).fromEmailGuard(""),
		Module:                  "my_shop",
		Series:                  "6.1",
		Settings:                "my_shop.settings",
		Django:                  "6.1.2",
		Imports:                 []string{"from pathlib import Path"},
		BaseDir:                 true,
		Apps:                    []string{"django.contrib.admin"},
		Middleware:              []string{middlewareSecurity},
		DefaultAutoField:        true,
		EnvBlock:                "DEBUG = True\n",
		SettingsPackage:         true,
		Tail:                    "\nAUTH_USER_MODEL = \"accounts.User\"\n",
		Target:                  "py313",
		FirstParty:              `"my_shop"`,
		Plugins:                 `"mypy_django_plugin.main"`,
		SettingsModule:          "my_shop.typecheck_settings",
		Debug:                   "DEBUG",
		Level:                   `"INFO"`,
		Name:                    "my-shop",
		DSN:                     `env("SENTRY_DSN", default="")`,
		Environment:             "ENVIRONMENT",
		Release:                 `""`,
		SendPII:                 "False",
		Traces:                  "0.0",
		Profiles:                "0.0",
	}
}

func TestEveryTemplateParsesAndRenders(t *testing.T) {
	t.Parallel()

	names, err := templateNames()
	if err != nil {
		t.Fatalf("walking embedded templates: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded templates found; the //go:embed pattern is not matching")
	}

	for _, name := range names {
		for _, htmx := range []bool{false, true} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("template %s panicked with HTMX=%v: %v", name, htmx, r)
					}
				}()

				got := render(name, fullTemplateData(htmx))
				if strings.TrimSpace(got) == "" {
					t.Errorf("template %s rendered empty output", name)
				}
			})
		}
	}
}

func TestRenderedTemplatesLeaveNoUnexpandedActions(t *testing.T) {
	t.Parallel()

	names, err := templateNames()
	if err != nil {
		t.Fatalf("walking embedded templates: %v", err)
	}

	for _, name := range names {
		got := render(name, fullTemplateData(true))

		if strings.Contains(strings.ReplaceAll(got, assets.LeftDelim+"tool.", ""), assets.LeftDelim) {
			t.Errorf("template %s rendered output still containing %q — a mistyped action", name, assets.LeftDelim)
		}
	}
}

func templateNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(assets.TemplateFS(), "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !pageTemplate(filepath.Base(path)) {
			names = append(names, filepath.Base(path))
		}
		return nil
	})
	return names, err
}

func pageTemplate(name string) bool {
	for _, prefix := range []string{"page_", "auth_", "error_", "partials"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
