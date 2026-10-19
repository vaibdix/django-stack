package scaffold

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
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

// Content is the file as edited so far.
func (s *SettingsEditor) Content() string {
	return s.content
}

func (s *SettingsEditor) Contains(substr string) bool {
	return strings.Contains(s.content, substr)
}

// listSpan finds the top-level assignment `name = [...]` and returns the
// offsets of its brackets. Only an assignment at the start of a line counts,
// so DEBUG_MIDDLEWARE never matches MIDDLEWARE and an indented
// `INSTALLED_APPS = [*INSTALLED_APPS, ...]` inside `if DEBUG:` is skipped.
// Brackets inside strings and comments do not count toward the nesting.
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

// listHas reports whether the list between open and closing already holds item
// in either quote style. Mentions elsewhere in the file (a comment, another
// list) do not count: they are why a file-wide search skipped real edits.
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
		// Order is the point of this edit (WhiteNoise must follow
		// SecurityMiddleware), so appending it anywhere else would be wrong.
		return &AnchorError{Edit: "'" + item + "' in " + listName, Anchor: anchor}
	}
	anchorIndex := open + offset

	lineStart := strings.LastIndex(s.content[:anchorIndex], "\n") + 1
	indent := s.content[lineStart:anchorIndex]

	insertAt := anchorIndex + len(anchor)
	s.content = s.content[:insertAt] + "\n" + indent + "'" + item + "'," + s.content[insertAt:]
	return nil
}

// StripSettingBlock removes the top-level `name = {...}` dict assignment and
// the comments right above it, if the file has one.
func (s *SettingsEditor) StripSettingBlock(name string) {
	s.content = stripSettingBlock(s.content, name)
}

func (s *SettingsEditor) AppendBlock(block string) {
	s.content += block
}

// AddImport adds importLine to the module's top-level import block and sorts
// the block. The block is found by its first top-level import, not by a
// particular line, because the pydantic settings drop `from pathlib import
// Path` once conf.py owns BASE_DIR.
func (s *SettingsEditor) AddImport(importLine string) error {
	if s.hasImport(importLine) {
		return nil
	}

	start, ok := s.firstTopLevelImport()
	if !ok {
		return &AnchorError{Edit: "the import " + importLine, Anchor: "a top-level import"}
	}
	end := s.endOfImportRegion(start)

	block := renderImportBlock(append(collectImports(s.content[start:end]), importLine), s.firstParty)
	s.content = s.content[:start] + strings.TrimPrefix(block, "\n") + strings.TrimLeft(s.content[end:], "\n")
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

func collectImports(region string) []string {
	var found []string
	for line := range strings.SplitSeq(region, "\n") {
		if trimmed := strings.TrimSpace(line); isImportLine(trimmed) {
			found = append(found, trimmed)
		}
	}
	return found
}

// renderImportBlock sorts the import lines into isort's sections: standard
// library, third party, then the project's own package, each separated by a
// blank line, with `import x` before `from x import y` inside a section.
func renderImportBlock(lines []string, firstParty string) string {
	var sections [3][]string
	for _, line := range lines {
		sections[importSection(line, firstParty)] = append(sections[importSection(line, firstParty)], line)
	}

	var b strings.Builder
	b.WriteString("\n")
	for _, section := range sections {
		if len(section) == 0 {
			continue
		}
		sort.SliceStable(section, func(i, j int) bool {
			left, right := strings.HasPrefix(section[i], "from "), strings.HasPrefix(section[j], "from ")
			if left != right {
				return !left
			}
			return section[i] < section[j]
		})
		if b.Len() > 1 {
			b.WriteString("\n")
		}
		for _, line := range section {
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

const (
	stdlibSection = iota
	thirdPartySection
	firstPartySection
)

func importSection(line, firstParty string) int {
	module := strings.Fields(strings.TrimPrefix(strings.TrimPrefix(line, "from "), "import "))[0]
	top, _, _ := strings.Cut(module, ".")
	switch {
	case firstParty != "" && top == firstParty:
		return firstPartySection
	case config.IsStdlibModule(top):
		return stdlibSection
	default:
		return thirdPartySection
	}
}

// RemoveCoreSetting deletes the one-line top-level assignment of name, with
// the comments and blank lines right above it. The name must match exactly:
// a prefix match also took DEBUG_PROPAGATE_EXCEPTIONS with DEBUG.
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

	switch {
	case !strings.Contains(s.content, "STATICFILES_DIRS"):
		return s.InsertAfterLine(staticURLPrefix, staticfilesDirsSetting+"\n\n"+staticRootSetting)
	case !strings.Contains(s.content, "STATIC_ROOT"):
		return s.InsertAfterLine("]", staticRootSetting, "STATICFILES_DIRS")
	}
	return nil
}

const staticURLPrefix = "STATIC_URL = "

// InsertAfterLine puts block, as its own paragraph, after the first line that
// starts with prefix. With after given, the search starts at the first
// occurrence of after, for an anchor such as the closing bracket of a list.
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
`
}

func (s *SettingsEditor) envConfigBlock(cfg EnvConfig) string {
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
	s.firstParty = cfg.ModuleName
	if err := s.AddImport(cfg.importLine()); err != nil {
		return err
	}
	if cfg.Pydantic {
		// conf.py already resolves the project root to find .env; one
		// definition keeps the two from drifting apart.
		if err := s.AddImport("from " + cfg.ModuleName + "." + strings.TrimSuffix(ConfFile, ".py") + " import BASE_DIR"); err != nil {
			return err
		}
		if err := s.RemoveCoreSetting("BASE_DIR"); err != nil {
			return err
		}
		if !strings.Contains(s.content, "Path(") {
			s.content = strings.Replace(s.content, "from pathlib import Path\n", "", 1)
		}
	}

	s.RemoveDatabasesBlock()
	for _, name := range []string{"SECRET_KEY", "DEBUG", "ALLOWED_HOSTS"} {
		if err := s.RemoveCoreSetting(name); err != nil {
			return err
		}
	}

	s.content = strings.TrimRight(s.content, "\n") + "\n\n" + s.envConfigBlock(cfg) + "\n"
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

	if err := s.AddImport("from django.utils.csp import CSP"); err != nil {
		return err
	}
	return s.AddToList(MiddlewareList, MiddlewareCSP)
}

// injectDjangoCSP gives Django versions without native CSP (before 6.0) the
// same report-only policy through django-csp 4.
func (s *SettingsEditor) injectDjangoCSP() error {
	s.content += `
CONTENT_SECURITY_POLICY_REPORT_ONLY = {
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

// SetTopLevelAfter is SetTopLevel for a setting that belongs with another: a
// new assignment goes on the line after anchor's, so related settings read
// together instead of accumulating at the end of the file. A missing anchor
// is an error, because the grouping is the point.
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
