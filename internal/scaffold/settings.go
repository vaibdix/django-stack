package scaffold

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
)

const (
	managedEnvConfigStart = "# === django-stack env config start ==="
	managedEnvConfigEnd   = "# === django-stack env config end ==="
)

type SettingsEditor struct {
	path    string
	content string

	firstParty string
}

func NewSettingsEditor(path string) (*SettingsEditor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", SettingsFile, err)
	}
	return &SettingsEditor{path: path, content: string(data)}, nil
}

func (s *SettingsEditor) Contains(substr string) bool {
	return strings.Contains(s.content, substr)
}

func (s *SettingsEditor) AddToList(listName, itemToAdd string) error {
	quotedItem := fmt.Sprintf("'%s'", strings.Trim(itemToAdd, "'\""))

	if strings.Contains(s.content, quotedItem) {
		return nil
	}

	listMarker := fmt.Sprintf("%s = [", listName)
	listStartIndex := strings.Index(s.content, listMarker)
	if listStartIndex == -1 {
		listMarker = fmt.Sprintf("%s=[", listName)
		listStartIndex = strings.Index(s.content, listMarker)
		if listStartIndex == -1 {
			return fmt.Errorf("could not find list '%s' in settings", listName)
		}
	}

	actualListStartIndex := listStartIndex + strings.Index(s.content[listStartIndex:], "[")
	openBracketCount := 0
	listEndIndex := -1
	for i := actualListStartIndex; i < len(s.content); i++ {
		if s.content[i] == '[' {
			openBracketCount++
		} else if s.content[i] == ']' {
			openBracketCount--
			if openBracketCount == 0 {
				listEndIndex = i
				break
			}
		}
	}

	if listEndIndex == -1 {
		return fmt.Errorf("could not find closing bracket for list '%s'", listName)
	}

	lineStartForListMarker := 0
	if idx := strings.LastIndex(s.content[:listStartIndex], "\n"); idx != -1 {
		lineStartForListMarker = idx + 1
	}
	baseIndent := ""
	for _, r := range s.content[lineStartForListMarker:listStartIndex] {
		if r == ' ' || r == '\t' {
			baseIndent += string(r)
		} else {
			break
		}
	}
	itemIndent := baseIndent + "    "
	contentBeforeClosingBracket := strings.TrimSpace(s.content[actualListStartIndex+1 : listEndIndex])

	var newEntry string
	switch {
	case contentBeforeClosingBracket == "":

		newEntry = fmt.Sprintf("\n%s%s,\n%s", itemIndent, quotedItem, baseIndent)
	case strings.HasSuffix(contentBeforeClosingBracket, ","):
		newEntry = fmt.Sprintf("\n%s%s,", itemIndent, quotedItem)
	default:

		newEntry = fmt.Sprintf(",\n%s%s,", itemIndent, quotedItem)
	}

	s.content = s.content[:listEndIndex] + newEntry + s.content[listEndIndex:]
	return nil
}

func (s *SettingsEditor) AddToListAfter(listName, after, itemToAdd string) error {
	quotedItem := fmt.Sprintf("'%s'", strings.Trim(itemToAdd, "'\""))
	if strings.Contains(s.content, quotedItem) {
		return nil
	}

	anchor := fmt.Sprintf("'%s',", strings.Trim(after, "'\""))
	anchorIndex := strings.Index(s.content, anchor)
	if anchorIndex == -1 {
		return s.AddToList(listName, itemToAdd)
	}

	lineStart := 0
	if idx := strings.LastIndex(s.content[:anchorIndex], "\n"); idx != -1 {
		lineStart = idx + 1
	}
	indent := s.content[lineStart:anchorIndex]

	insertAt := anchorIndex + len(anchor)
	s.content = s.content[:insertAt] + "\n" + indent + quotedItem + "," + s.content[insertAt:]
	return nil
}

func (s *SettingsEditor) AppendBlock(block string) {
	s.content += block
}

func (s *SettingsEditor) AddImport(importLine string) {
	if s.hasImport(importLine) {
		return
	}

	start, ok := s.stdlibImportAnchor()
	if !ok {
		s.content = importLine + "\n" + s.content
		return
	}
	end := s.endOfImportRegion(start)

	block := renderImportBlock(append(collectImports(s.content[start:end]), importLine), s.firstParty)

	s.content = s.content[:start] + block + strings.TrimLeft(s.content[end:], "\n")
}

func (s *SettingsEditor) hasImport(importLine string) bool {
	for line := range strings.SplitSeq(s.content, "\n") {
		if strings.TrimSpace(line) == importLine {
			return true
		}
	}
	return false
}

func (s *SettingsEditor) stdlibImportAnchor() (int, bool) {
	for _, anchor := range []string{"from pathlib import Path", "import os"} {
		idx := strings.Index(s.content, anchor)
		if idx == -1 {
			continue
		}
		lineEnd := strings.Index(s.content[idx:], "\n")
		if lineEnd == -1 {
			return len(s.content), true
		}
		return idx + lineEnd + 1, true
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
		next := offset + lineEnd + 1

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

func collectImports(region string) []string {
	var found []string
	for line := range strings.SplitSeq(region, "\n") {
		if trimmed := strings.TrimSpace(line); isImportLine(trimmed) {
			found = append(found, trimmed)
		}
	}
	return found
}

func renderImportBlock(lines []string, firstParty string) string {
	var straight, from, own []string
	for _, line := range lines {
		switch {
		case firstParty != "" && strings.HasPrefix(line, "from "+firstParty+"."):
			own = append(own, line)
		case strings.HasPrefix(line, "import "):
			straight = append(straight, line)
		default:
			from = append(from, line)
		}
	}

	sort.Strings(straight)
	sort.Strings(from)
	sort.Strings(own)

	var b strings.Builder
	b.WriteString("\n")

	for _, group := range [][]string{append(straight, from...), own} {
		if len(group) == 0 {
			continue
		}
		if b.Len() > 1 {
			b.WriteString("\n")
		}
		for _, line := range group {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func (s *SettingsEditor) RemoveCoreSetting(settingName string) {
	lines := strings.Split(s.content, "\n")
	var result []string
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, settingName) && strings.Contains(trimmed, "=") {
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(prev, "#") || prev == "" {
					result = result[:len(result)-1]
				} else {
					break
				}
			}
			continue
		}
		result = append(result, lines[i])
	}
	s.content = strings.Join(result, "\n")
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
			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
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
	staticBlock := staticfilesDirsSetting + "\n\n" + staticRootSetting

	if !strings.Contains(s.content, "STATICFILES_DIRS") {
		staticURLMarker := "STATIC_URL = "
		idx := strings.Index(s.content, staticURLMarker)
		if idx != -1 {
			lineEndIdx := strings.Index(s.content[idx:], "\n")
			if lineEndIdx != -1 {
				insertPos := idx + lineEndIdx + 1
				s.content = s.content[:insertPos] + "\n" + staticBlock + "\n" + s.content[insertPos:]
			} else {
				s.content += "\n\n" + staticBlock + "\n"
			}
		} else {
			s.content += "\n\n" + staticBlock + "\n"
		}
	} else if !strings.Contains(s.content, "STATIC_ROOT") {

		staticfilesDirsEnd := strings.Index(s.content, "STATICFILES_DIRS")
		if staticfilesDirsEnd != -1 {

			closingBracket := strings.Index(s.content[staticfilesDirsEnd:], "]")
			if closingBracket != -1 {
				insertPos := staticfilesDirsEnd + closingBracket + 1

				if insertPos < len(s.content) && s.content[insertPos] == '\n' {
					insertPos++
				}
				s.content = s.content[:insertPos] + "\n" + staticRootSetting + "\n" + s.content[insertPos:]
			}
		}
	}

	return nil
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

func (s *SettingsEditor) replaceManagedBlock(block string) bool {
	original := s.content
	start := strings.Index(s.content, managedEnvConfigStart)
	if start == -1 {
		return false
	}
	end := strings.Index(s.content[start:], managedEnvConfigEnd)
	if end == -1 {
		return false
	}
	end += start + len(managedEnvConfigEnd)
	prefix := strings.TrimRight(original[:start], "\n")
	suffix := strings.TrimLeft(original[end:], "\n")
	s.content = prefix + "\n\n" + block + "\n"
	if suffix != "" {

		s.content += "\n" + suffix
	}
	return true
}

func (s *SettingsEditor) removeLegacyEnvConfigBlock() {
	lines := strings.Split(s.content, "\n")
	var result []string
	skipping := false
	braceDepth := 0
	foundDatabaseBlock := false

	for i := range lines {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if !skipping && trimmed == "# Load environment variables from .env file" {
			skipping = true
		}

		if skipping {
			if strings.HasPrefix(trimmed, "DATABASES") && strings.Contains(trimmed, "{") {
				foundDatabaseBlock = true
				braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
				if braceDepth <= 0 {
					braceDepth = 0
				}
				continue
			}

			if foundDatabaseBlock {
				braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
				if braceDepth <= 0 {
					skipping = false
					foundDatabaseBlock = false
					braceDepth = 0
				}
				continue
			}

			continue
		}

		result = append(result, line)
	}

	s.content = strings.Join(result, "\n")
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
`
}

func (s *SettingsEditor) managedEnvConfigBlock(cfg EnvConfig) string {
	r := cfg.reader()

	var databaseBlock string
	switch cfg.Database {
	case config.PostgresDocker, config.PostgresString:
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
}`, r.required("DB_NAME"), r.str("DB_USER", "postgres"), r.secret("DB_PASSWORD"),
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

	return managedEnvConfigStart + `
` + cfg.preamble() + `
ENVIRONMENT = ` + r.environment() + `

ADMIN_URL = ` + r.adminURL() + `

SECRET_KEY = ` + r.requiredSecret("SECRET_KEY") + `

DEBUG = ` + r.debug() + `

ALLOWED_HOSTS = ` + r.list("ALLOWED_HOSTS", "localhost", "127.0.0.1") + `

` + databaseBlock + `
` + managedEnvConfigEnd
}

func (s *SettingsEditor) InjectEnvConfig(cfg EnvConfig) {
	s.firstParty = cfg.ModuleName
	s.AddImport(cfg.importLine())

	managedBlock := s.managedEnvConfigBlock(cfg)
	if s.replaceManagedBlock(managedBlock) {
		return
	}

	s.removeLegacyEnvConfigBlock()
	s.RemoveDatabasesBlock()
	s.RemoveCoreSetting("SECRET_KEY")
	s.RemoveCoreSetting("DEBUG")
	s.RemoveCoreSetting("ALLOWED_HOSTS")

	s.content = strings.TrimRight(s.content, "\n") + "\n\n" + managedBlock + "\n"
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

			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
			}
			continue
		}

		result = append(result, lines[i])
		i++
	}
	s.content = strings.Join(result, "\n")
}

func (s *SettingsEditor) InjectSecuritySettings(nativeCSP bool, cfg EnvConfig) error {
	if s.Contains("PASSWORD_HASHERS") {
		return nil
	}

	s.RemovePasswordValidatorsBlock()

	r := cfg.reader()
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

if not DEBUG:
    SECURE_SSL_REDIRECT = ` + r.boolean("SECURE_SSL_REDIRECT", true) + `
    SECURE_HSTS_SECONDS = ` + r.integer("SECURE_HSTS_SECONDS", 31536000) + `
    SECURE_HSTS_INCLUDE_SUBDOMAINS = ` + r.boolean("SECURE_HSTS_INCLUDE_SUBDOMAINS", false) + `
    SECURE_HSTS_PRELOAD = ` + r.boolean("SECURE_HSTS_PRELOAD", false) + `
    if ` + r.boolean("TRUST_PROXY_SSL_HEADER", false) + `:
        SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')
    SESSION_COOKIE_SECURE = True
    CSRF_COOKIE_SECURE = True

SECURE_CONTENT_TYPE_NOSNIFF = True
SESSION_COOKIE_HTTPONLY = True
CSRF_COOKIE_HTTPONLY = True
X_FRAME_OPTIONS = 'DENY'

LOGIN_URL = 'admin:login'
`
	s.content += securityBlock

	if !nativeCSP {
		return nil
	}

	s.content += `
SECURE_CSP_REPORT_ONLY = {
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

	s.AddImport("from django.utils.csp import CSP")

	return s.AddToList(MiddlewareList, MiddlewareCSP)
}

func (s *SettingsEditor) Save() error {
	if err := os.WriteFile(s.path, []byte(s.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", SettingsFile, err)
	}
	return nil
}
