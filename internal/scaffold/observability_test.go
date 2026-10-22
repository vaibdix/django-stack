package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func settingsWithObservability(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: commandtest.NewFake()}

	dirs := []string{filepath.Join(root, cfg.ModuleName())}
	if cfg.AppName != "" {
		dirs = append(dirs, filepath.Join(root, cfg.AppName))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}
	settingsPath := filepath.Join(root, cfg.ModuleName(), SettingsFile)
	if err := os.WriteFile(settingsPath, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}

	if err := e.configureProjectSettings(root); err != nil {
		t.Fatalf("configuring settings: %v", err)
	}
	if err := e.applyStructlog(root); err != nil {
		t.Fatalf("applying structlog: %v", err)
	}
	if err := e.applySentry(root); err != nil {
		t.Fatalf("applying sentry: %v", err)
	}
	if err := e.applyRESTFramework(root); err != nil {
		t.Fatalf("applying rest framework: %v", err)
	}

	written, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading the generated settings: %v", err)
	}
	return string(written)
}

func observabilityConfig() config.Project {
	return config.Project{
		ProjectName:    "demo",
		DjangoVersion:  "6.1",
		DatabaseType:   config.SQLite,
		SetupStructlog: true,
		SetupSentry:    true,
	}
}

func TestStructlogMiddlewareFollowsAuthentication(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())

	auth := strings.Index(settings, "django.contrib.auth.middleware.AuthenticationMiddleware")
	request := strings.Index(settings, MiddlewareStructlog)

	if auth == -1 || request == -1 {
		t.Fatalf("expected both middlewares in MIDDLEWARE:\n%s", settings)
	}
	if request < auth {
		t.Errorf("RequestMiddleware is listed before AuthenticationMiddleware, so user_id binds as None:\n%s", settings)
	}
}

func TestStructlogMergesContextvarsForStdlibLoggers(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())

	if !strings.Contains(settings, "'foreign_pre_chain': _FOREIGN_PRE_CHAIN") {
		t.Errorf("formatters do not set foreign_pre_chain, so stdlib records lose request_id:\n%s", settings)
	}
	if !strings.Contains(settings, "structlog.contextvars.merge_contextvars") {
		t.Errorf("merge_contextvars is missing, so nothing binds request_id at all:\n%s", settings)
	}
	if strings.Count(settings, "'foreign_pre_chain': _FOREIGN_PRE_CHAIN") != 2 {
		t.Errorf("both the console and JSON formatters need foreign_pre_chain:\n%s", settings)
	}
}

func TestStructlogWritesToStdoutOnly(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())

	for _, unwanted := range []string{"WatchedFileHandler", "logs/json.log", "logs/flat_line.log", "filename"} {
		if strings.Contains(settings, unwanted) {
			t.Errorf("logging configuration writes to a file (%q); it should use stdout:\n%s", unwanted, settings)
		}
	}
	if !strings.Contains(settings, "'class': 'logging.StreamHandler'") {
		t.Errorf("no stream handler is configured:\n%s", settings)
	}
}

func TestStructlogAppAndMiddlewareAreRegistered(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())

	for _, want := range []string{"'" + AppStructlog + "'", MiddlewareStructlog, "import structlog"} {
		if !strings.Contains(settings, want) {
			t.Errorf("settings are missing %q:\n%s", want, settings)
		}
	}

	if !strings.Contains(settings, "structlog.configure(") {
		t.Errorf("nothing configures structlog:\n%s", settings)
	}
}

func observabilityModule(t *testing.T) string {
	t.Helper()
	return settingsWithObservability(t, observabilityConfig())
}

func TestSentryInitIsGuardedByADSN(t *testing.T) {
	t.Parallel()

	module := observabilityModule(t)

	guard := strings.Index(module, "if SENTRY_DSN:")
	init := strings.Index(module, "sentry_sdk.init(")

	if guard == -1 {
		t.Fatalf("sentry_sdk.init is not guarded by a DSN check:\n%s", module)
	}
	if init < guard {
		t.Errorf("sentry_sdk.init runs before the DSN guard:\n%s", module)
	}
}

func TestSentryReliesOnTheAutoEnabledDjangoIntegration(t *testing.T) {
	t.Parallel()

	module := observabilityModule(t)

	for _, unwanted := range []string{"DjangoIntegration(", "sentry_sdk.integrations.django", "integrations="} {
		if strings.Contains(module, unwanted) {
			t.Errorf("the module adds %q by hand; the SDK enables it already:\n%s", unwanted, module)
		}
	}
}

func TestSentryDoesNotSendPersonalDataByDefault(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())

	if !strings.Contains(settings, "SENTRY_SEND_PII = env_bool('SENTRY_SEND_PII', default=False)") {
		t.Errorf("send_default_pii should default to False and come from the environment:\n%s", settings)
	}
	if strings.Contains(observabilityModule(t), "send_default_pii=True") {
		t.Errorf("send_default_pii is hardcoded to True:\n%s", observabilityModule(t))
	}
}

func TestSentryUsesTheCurrentProfilingOptions(t *testing.T) {
	t.Parallel()

	module := observabilityModule(t)

	if !strings.Contains(module, "profile_session_sample_rate=") || !strings.Contains(module, "profile_lifecycle='trace'") {
		t.Errorf("the module does not use the current profiling API:\n%s", module)
	}
	if strings.Contains(module, "profiles_sample_rate") {
		t.Errorf("profiles_sample_rate is the superseded option:\n%s", module)
	}
	if !strings.Contains(module, "enable_logs=True") {
		t.Errorf("logs are not forwarded to Sentry:\n%s", module)
	}
}

func TestObservabilitySettingsReadTypedConfigWhenItIsOn(t *testing.T) {
	t.Parallel()

	cfg := observabilityConfig()
	cfg.SetupPydanticConf = true
	settings := settingsWithObservability(t, cfg)

	for _, want := range []string{"SENTRY_DSN = env.SENTRY_DSN.get_secret_value()", "SENTRY_SEND_PII = env.SENTRY_SEND_PII"} {
		if !strings.Contains(settings, want) {
			t.Errorf("Sentry did not read the typed settings object, expected %q:\n%s", want, settings)
		}
	}
	if strings.Contains(settings, "env.bool('SENTRY_SEND_PII'") {
		t.Errorf("Sentry still uses the django-environ form under typed settings:\n%s", settings)
	}
}

func TestSentryIsDocumentedInTheReadmeAndNeverSetInEnvExample(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: observabilityConfig(), Reporter: NopReporter{}}
	if strings.Contains(e.renderEnvExampleContent(), "SENTRY") {
		t.Errorf(".env.example sets a Sentry variable:\n%s", e.renderEnvExampleContent())
	}
	if readme := e.renderReadme(true); !strings.Contains(readme, "`SENTRY_DSN`") {
		t.Errorf("the README does not document SENTRY_DSN:\n%s", readme)
	}

	off := &Env{Cfg: config.Project{ProjectName: "demo", DatabaseType: config.SQLite}, Reporter: NopReporter{}}
	if strings.Contains(off.renderReadme(true), "SENTRY") {
		t.Error("a project without the Sentry add-on should not mention it in the README")
	}
}

const djangoAdminSettingsFixture = `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc'

DEBUG = True

ALLOWED_HOSTS = []

INSTALLED_APPS = [
    'django.contrib.admin',
    'django.contrib.auth',
    'django.contrib.contenttypes',
    'django.contrib.sessions',
    'django.contrib.messages',
    'django.contrib.staticfiles',
]

MIDDLEWARE = [
    'django.middleware.security.SecurityMiddleware',
    'django.contrib.sessions.middleware.SessionMiddleware',
    'django.middleware.common.CommonMiddleware',
    'django.middleware.csrf.CsrfViewMiddleware',
    'django.contrib.auth.middleware.AuthenticationMiddleware',
    'django.contrib.messages.middleware.MessageMiddleware',
    'django.middleware.clickjacking.XFrameOptionsMiddleware',
]

TEMPLATES = [
    {
        'BACKEND': 'django.template.backends.django.DjangoTemplates',
        'DIRS': [],
        'APP_DIRS': True,
        'OPTIONS': {
            'context_processors': [
                'django.template.context_processors.request',
                'django.contrib.auth.context_processors.auth',
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}

STATIC_URL = 'static/'
`
