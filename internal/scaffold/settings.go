package scaffold

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
)

type SettingsEditor struct {
	path    string
	content string
}

func NewSettingsEditor(path string) (*SettingsEditor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", SettingsFile, err)
	}
	return &SettingsEditor{path: path, content: string(data)}, nil
}

func (s *SettingsEditor) Content() string {
	return s.content
}

func (s *SettingsEditor) Contains(substr string) bool {
	return strings.Contains(s.content, substr)
}

func (s *SettingsEditor) listSpan(name string) (open, closing int, err error) {
	assignment := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `\s*=\s*\[`)
	at := assignment.FindStringIndex(s.content)
	if at == nil {
		return 0, 0, &AnchorError{Edit: "the " + name + " list", Anchor: name + " = ["}
	}
	open = at[1] - 1

	depth := 0
	var quote byte
	for i := open; i < len(s.content); i++ {
		c := s.content[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '#':
			for i < len(s.content) && s.content[i] != '\n' {
				i++
			}
		case c == '[':
			depth++
		case c == ']':
			depth--
			if depth == 0 {
				return open, i, nil
			}
		}
	}
	return 0, 0, &AnchorError{Edit: "the " + name + " list", Anchor: "the closing ] of " + name}
}

func (s *SettingsEditor) listHas(open, closing int, item string) bool {
	body := s.content[open:closing]
	return strings.Contains(body, "'"+item+"'") || strings.Contains(body, `"`+item+`"`)
}

func (s *SettingsEditor) AddToList(listName, itemToAdd string) error {
	item := strings.Trim(itemToAdd, "'\"")
	open, closing, err := s.listSpan(listName)
	if err != nil {
		return err
	}
	if s.listHas(open, closing, item) {
		return nil
	}

	lineStart := strings.LastIndex(s.content[:open], "\n") + 1
	baseIndent := s.content[lineStart:open]
	baseIndent = baseIndent[:len(baseIndent)-len(strings.TrimLeft(baseIndent, " \t"))]
	itemIndent := baseIndent + "    "
	quotedItem := "'" + item + "'"

	body := strings.TrimRight(s.content[open+1:closing], " \t\n")
	var entry string
	switch {
	case strings.TrimSpace(body) == "":
		entry = "\n" + itemIndent + quotedItem + ",\n" + baseIndent
	case strings.HasSuffix(body, ","):
		entry = body + "\n" + itemIndent + quotedItem + ",\n" + baseIndent
	default:
		entry = body + ",\n" + itemIndent + quotedItem + ",\n" + baseIndent
	}
	s.content = s.content[:open+1] + entry + s.content[closing:]
	return nil
}

func (s *SettingsEditor) AddToListAfter(listName, after, itemToAdd string) error {
	item := strings.Trim(itemToAdd, "'\"")
	open, closing, err := s.listSpan(listName)
	if err != nil {
		return err
	}
	if s.listHas(open, closing, item) {
		return nil
	}

	anchor := "'" + strings.Trim(after, "'\"") + "',"
	offset := strings.Index(s.content[open:closing], anchor)
	if offset == -1 {
		return &AnchorError{Edit: "'" + item + "' in " + listName, Anchor: anchor}
	}
	anchorIndex := open + offset

	lineStart := strings.LastIndex(s.content[:anchorIndex], "\n") + 1
	indent := s.content[lineStart:anchorIndex]

	insertAt := anchorIndex + len(anchor)
	s.content = s.content[:insertAt] + "\n" + indent + "'" + item + "'," + s.content[insertAt:]
	return nil
}

func (s *SettingsEditor) StripSettingBlock(name string) {
	s.content = stripSettingBlock(s.content, name)
}

func (s *SettingsEditor) AppendBlock(block string) {
	s.content += block
}

func (s *SettingsEditor) AddImport(importLine string) error {
	if s.hasImport(importLine) {
		return nil
	}

	start, ok := s.firstTopLevelImport()
	if !ok {
		return &AnchorError{Edit: "the import " + importLine, Anchor: "a top-level import"}
	}
	end := s.endOfImportRegion(start)

	s.content = s.content[:end] + importLine + "\n\n" + strings.TrimLeft(s.content[end:], "\n")
	return nil
}

func (s *SettingsEditor) hasImport(importLine string) bool {
	for line := range strings.SplitSeq(s.content, "\n") {
		if strings.TrimSpace(line) == importLine {
			return true
		}
	}
	return false
}

func (s *SettingsEditor) firstTopLevelImport() (int, bool) {
	offset := 0
	for line := range strings.SplitSeq(s.content, "\n") {
		if isImportLine(line) {
			return offset, true
		}
		offset += len(line) + 1
	}
	return 0, false
}

func (s *SettingsEditor) endOfImportRegion(start int) int {
	offset, end := start, start

	for offset < len(s.content) {
		lineEnd := strings.Index(s.content[offset:], "\n")
		if lineEnd == -1 {
			lineEnd = len(s.content) - offset
		}
		line := strings.TrimSpace(s.content[offset : offset+lineEnd])
		next := min(offset+lineEnd+1, len(s.content))

		switch {
		case line == "":
			offset = next
		case isImportLine(line):
			offset, end = next, next
		default:
			return end
		}
	}
	return end
}

func isImportLine(line string) bool {
	return strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ")
}

func (s *SettingsEditor) RemoveCoreSetting(name string) error {
	assignment := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `\s*=`)
	lines := strings.Split(s.content, "\n")
	var result []string
	removed := false
	for _, line := range lines {
		if assignment.MatchString(line) {
			removed = true
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				if !strings.HasPrefix(prev, "#") && prev != "" {
					break
				}
				result = result[:len(result)-1]
			}
			continue
		}
		result = append(result, line)
	}
	if !removed {
		return &AnchorError{Edit: "removing " + name, Anchor: name + " = "}
	}
	s.content = strings.Join(result, "\n")
	return nil
}

func (s *SettingsEditor) RemoveDatabasesBlock() {
	lines := strings.Split(s.content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if strings.HasPrefix(trimmed, "DATABASES") && strings.Contains(trimmed, "=") && strings.Contains(trimmed, "{") {
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				isDatabaseComment := strings.HasPrefix(prev, "#") &&
					(strings.Contains(prev, "atabase") || strings.Contains(prev, "djangoproject.com"))
				if prev != "" && !isDatabaseComment {
					break
				}
				result = result[:len(result)-1]
			}

			depth := 0
			for i < len(lines) {
				depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
				i++
				if depth <= 0 {
					break
				}
			}
			continue
		}

		result = append(result, lines[i])
		i++
	}
	s.content = strings.Join(result, "\n")
}

type AnchorError struct {
	Edit string

	Anchor string
}

func (e *AnchorError) Error() string {
	return fmt.Sprintf(
		"could not apply %s: %s does not contain the expected anchor %q "+
			"(the file may have been generated by an unsupported Django version, or already edited by hand)",
		e.Edit, SettingsFile, e.Anchor,
	)
}

func (s *SettingsEditor) ConfigureTemplatesDirs() error {
	const templatesAnchor = "'DIRS': []"
	templatesDirsSetting := "'DIRS': [BASE_DIR / 'templates']"

	if !strings.Contains(s.content, templatesDirsSetting) {
		if !strings.Contains(s.content, templatesAnchor) {
			return &AnchorError{Edit: "the global templates directory", Anchor: templatesAnchor}
		}
		s.content = strings.Replace(s.content, templatesAnchor, templatesDirsSetting, 1)
	}
	staticfilesDirsSetting := "STATICFILES_DIRS = [\n    BASE_DIR / 'static',\n]"
	staticRootSetting := "STATIC_ROOT = BASE_DIR / 'staticfiles'"

	switch {
	case !strings.Contains(s.content, "STATICFILES_DIRS"):
		return s.InsertAfterLine(staticURLPrefix, staticfilesDirsSetting+"\n\n"+staticRootSetting)
	case !strings.Contains(s.content, "STATIC_ROOT"):
		return s.InsertAfterLine("]", staticRootSetting, "STATICFILES_DIRS")
	}
	return nil
}

const staticURLPrefix = "STATIC_URL = "

func (s *SettingsEditor) InsertAfterLine(prefix, block string, after ...string) error {
	from := 0
	if len(after) > 0 {
		at := strings.Index(s.content, after[0])
		if at == -1 {
			return &AnchorError{Edit: firstLine(block), Anchor: after[0]}
		}
		from = at
	}

	offset := from
	for offset < len(s.content) {
		lineEnd := strings.Index(s.content[offset:], "\n")
		if lineEnd == -1 {
			lineEnd = len(s.content) - offset
		}
		line := s.content[offset : offset+lineEnd]
		if strings.HasPrefix(line, prefix) {
			insertAt := min(offset+lineEnd+1, len(s.content))
			s.content = s.content[:insertAt] + "\n" + block + "\n" + s.content[insertAt:]
			return nil
		}
		offset += lineEnd + 1
	}
	return &AnchorError{Edit: firstLine(block), Anchor: prefix}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}

func (s *SettingsEditor) AddProjectContextProcessor(projectModuleName string) error {
	const messagesAnchor = "'django.contrib.messages.context_processors.messages',"

	contextProcessor := fmt.Sprintf("'%s.context_processors.project_context'", projectModuleName)
	if strings.Contains(s.content, contextProcessor) {
		return nil
	}

	if !strings.Contains(s.content, messagesAnchor) {
		return &AnchorError{Edit: "the project context processor", Anchor: messagesAnchor}
	}

	s.content = strings.Replace(
		s.content,
		messagesAnchor,
		messagesAnchor+"\n                "+contextProcessor+",",
		1,
	)
	return nil
}

type EnvConfig struct {
	Database config.Database

	Pydantic bool

	ModuleName string
}

func (c EnvConfig) reader() envReader { return envReader{pydantic: c.Pydantic} }

func (c EnvConfig) importLine() string {
	if c.Pydantic {
		return "from " + c.ModuleName + "." + strings.TrimSuffix(ConfFile, ".py") + " import env"
	}
	return "import environ"
}

func (c EnvConfig) preamble() string {
	if c.Pydantic {
		return ""
	}
	return `env = environ.Env()
environ.Env.read_env(BASE_DIR / '.env')


` + envBoolFunction + `
`
}

const envBoolFunction = `def env_bool(name: str, *, default: bool) -> bool:
    value = env.str(name, default=str(default)).strip().lower()
    if value in {'1', 'on', 't', 'true', 'y', 'yes'}:
        return True
    if value in {'0', 'off', 'f', 'false', 'n', 'no'}:
        return False
    message = f'{name} must be true or false, not {value!r}.'
    raise ImproperlyConfigured(message)
`

func envConfigBlock(cfg EnvConfig) string {
	r := cfg.reader()

	var databaseBlock string
	switch cfg.Database {
	case config.PostgresDocker:
		databaseBlock = fmt.Sprintf(`DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': %s,
        'USER': %s,
        'PASSWORD': %s,
        'HOST': %s,
        'PORT': %s,
        'CONN_MAX_AGE': %s,
        'CONN_HEALTH_CHECKS': True,
    }
}`, r.required("DB_NAME"), r.str("DB_USER", "django_user"), r.secret("DB_PASSWORD"),
			r.str("DB_HOST", "localhost"), r.str("DB_PORT", defaultPostgresPort),
			r.integer("DB_CONN_MAX_AGE", 60))
	case config.MySQLDocker:
		databaseBlock = fmt.Sprintf(`DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.mysql',
        'NAME': %s,
        'USER': %s,
        'PASSWORD': %s,
        'HOST': %s,
        'PORT': %s,
        'OPTIONS': {
            'charset': 'utf8mb4',
            'isolation_level': 'read committed',
        },
        'CONN_MAX_AGE': %s,
        'CONN_HEALTH_CHECKS': True,
    }
}`, r.required("DB_NAME"), r.str("DB_USER", "django_user"), r.secret("DB_PASSWORD"),
			r.str("DB_HOST", mysqlTCPHost), r.str("DB_PORT", defaultMySQLPort),
			r.integer("DB_CONN_MAX_AGE", 60))
	default:
		databaseBlock = `DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}`
	}

	return strings.TrimLeft(cfg.preamble()+`
ENVIRONMENT = `+r.environment()+`

ADMIN_URL = `+r.adminURL()+`

SECRET_KEY = `+r.secretKey()+`

DEBUG = `+r.debug()+`

ALLOWED_HOSTS = `+r.list("ALLOWED_HOSTS", "localhost", "127.0.0.1")+`

`+databaseBlock, "\n")
}

func (s *SettingsEditor) InjectEnvConfig(cfg EnvConfig) error {
	if err := s.AddImport(cfg.importLine()); err != nil {
		return err
	}
	if cfg.Pydantic {
		if err := s.AddImport("from " + cfg.ModuleName + "." + strings.TrimSuffix(ConfFile, ".py") + " import BASE_DIR"); err != nil {
			return err
		}
		if err := s.RemoveCoreSetting("BASE_DIR"); err != nil {
			return err
		}
		if !strings.Contains(s.content, "Path(") {
			s.content = strings.Replace(s.content, "from pathlib import Path\n", "", 1)
		}
	} else if err := s.AddImport("from django.core.exceptions import ImproperlyConfigured"); err != nil {
		return err
	}

	s.RemoveDatabasesBlock()
	for _, name := range []string{"SECRET_KEY", "DEBUG", "ALLOWED_HOSTS"} {
		if err := s.RemoveCoreSetting(name); err != nil {
			return err
		}
	}

	s.content = strings.TrimRight(s.content, "\n") + "\n\n" + envConfigBlock(cfg) + "\n"
	return nil
}

func (s *SettingsEditor) RemovePasswordValidatorsBlock() {
	lines := strings.Split(s.content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if strings.HasPrefix(trimmed, "AUTH_PASSWORD_VALIDATORS") && strings.Contains(trimmed, "=") && strings.Contains(trimmed, "[") {
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				isValidatorComment := strings.HasPrefix(prev, "#") &&
					(strings.Contains(prev, "assword") || strings.Contains(prev, "djangoproject.com"))
				if prev != "" && !isValidatorComment {
					break
				}
				result = result[:len(result)-1]
			}

			depth := 0
			for i < len(lines) {
				depth += strings.Count(lines[i], "[") - strings.Count(lines[i], "]")
				i++
				if depth <= 0 {
					break
				}
			}
			continue
		}

		result = append(result, lines[i])
		i++
	}
	s.content = strings.Join(result, "\n")
}

func enforceCSP(nativeCSP bool, indent string) string {
	enforced, reported := "CONTENT_SECURITY_POLICY", "CONTENT_SECURITY_POLICY_REPORT_ONLY"
	if nativeCSP {
		enforced, reported = "SECURE_CSP", "SECURE_CSP_REPORT_ONLY"
	}
	return indent + enforced + " = " + reported + "\n" +
		indent + reported + " = {}\n"
}

func (s *SettingsEditor) InjectSecuritySettings(nativeCSP bool) error {
	if s.Contains("PASSWORD_HASHERS") {
		return nil
	}

	s.RemovePasswordValidatorsBlock()

	securityBlock := `
PASSWORD_HASHERS = [
    'django.contrib.auth.hashers.Argon2PasswordHasher',
    'django.contrib.auth.hashers.PBKDF2PasswordHasher',
    'django.contrib.auth.hashers.PBKDF2SHA1PasswordHasher',
    'django.contrib.auth.hashers.BCryptSHA256PasswordHasher',
    'django.contrib.auth.hashers.ScryptPasswordHasher',
]

AUTH_PASSWORD_VALIDATORS = [
    {'NAME': 'django.contrib.auth.password_validation.UserAttributeSimilarityValidator'},
    {'NAME': 'django.contrib.auth.password_validation.MinimumLengthValidator'},
    {'NAME': 'django.contrib.auth.password_validation.CommonPasswordValidator'},
    {'NAME': 'django.contrib.auth.password_validation.NumericPasswordValidator'},
]

LOGIN_URL = 'admin:login'
`
	s.content += securityBlock

	if !nativeCSP {
		return s.injectDjangoCSP()
	}

	s.content += `
SECURE_CSP_REPORT_ONLY: dict[str, list[str]] = {
    'default-src': [CSP.NONE],
    'script-src': [CSP.SELF],
    'style-src': [CSP.SELF],
    'img-src': [CSP.SELF, 'data:'],
    'font-src': [CSP.SELF],
    'connect-src': [CSP.SELF],
    'form-action': [CSP.SELF],
    'base-uri': [CSP.SELF],
    'frame-ancestors': [CSP.NONE],
}
`

	if err := s.AddImport("from django.utils.csp import CSP"); err != nil {
		return err
	}
	return s.AddToList(MiddlewareList, MiddlewareCSP)
}

func (s *SettingsEditor) injectDjangoCSP() error {
	s.content += `
CONTENT_SECURITY_POLICY_REPORT_ONLY: dict[str, dict[str, list[str]]] = {
    'DIRECTIVES': {
        'default-src': [NONE],
        'script-src': [SELF],
        'style-src': [SELF],
        'img-src': [SELF, 'data:'],
        'font-src': [SELF],
        'connect-src': [SELF],
        'form-action': [SELF],
        'base-uri': [SELF],
        'frame-ancestors': [NONE],
    },
}
`
	for _, line := range []string{"from csp.constants import NONE", "from csp.constants import SELF"} {
		if err := s.AddImport(line); err != nil {
			return err
		}
	}

	if err := s.AddToList(InstalledApps, AppDjangoCSP); err != nil {
		return err
	}
	return s.AddToList(MiddlewareList, MiddlewareDjangoCSP)
}

func (s *SettingsEditor) SetTopLevel(name, value string) {
	lines := strings.Split(s.content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, name+" = ") || strings.HasPrefix(line, name+"=") {
			lines[i] = name + " = " + value
			s.content = strings.Join(lines, "\n")
			return
		}
	}
	s.content = strings.TrimRight(s.content, "\n") + "\n\n" + name + " = " + value + "\n"
}

func (s *SettingsEditor) SetTopLevelAfter(anchor, name, value string) error {
	lines := strings.Split(s.content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, name+" = ") || strings.HasPrefix(line, name+"=") {
			s.SetTopLevel(name, value)
			return nil
		}
	}
	for i, line := range lines {
		if strings.HasPrefix(line, anchor+" = ") {
			lines = slices.Insert(lines, i+1, name+" = "+value)
			s.content = strings.Join(lines, "\n")
			return nil
		}
	}
	return &AnchorError{Edit: name, Anchor: anchor + " = "}
}

func (s *SettingsEditor) Save() error {
	if err := os.WriteFile(s.path, []byte(s.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", SettingsFile, err)
	}
	return nil
}
