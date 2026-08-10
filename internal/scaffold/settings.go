package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
)

const (
	managedEnvConfigStart = "# === django-stack env config start ==="
	managedEnvConfigEnd   = "# === django-stack env config end ==="
)

// SettingsEditor provides methods to manipulate a Django settings.py file.
type SettingsEditor struct {
	path    string
	content string
}

// NewSettingsEditor reads the settings file at path and returns an editor.
func NewSettingsEditor(path string) (*SettingsEditor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", SettingsFile, err)
	}
	return &SettingsEditor{path: path, content: string(data)}, nil
}

// Content returns the current settings content.
func (s *SettingsEditor) Content() string {
	return s.content
}

// SetContent replaces the full content of the editor.
func (s *SettingsEditor) SetContent(content string) {
	s.content = content
}

// Contains checks if the settings content contains the given substring.
func (s *SettingsEditor) Contains(substr string) bool {
	return strings.Contains(s.content, substr)
}

// AddToList adds an item to a Python list (e.g. INSTALLED_APPS, MIDDLEWARE) in settings.py.
// It is a no-op if the item is already present.
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
		// Empty list: the closing bracket needs its own indented line.
		newEntry = fmt.Sprintf("\n%s%s,\n%s", itemIndent, quotedItem, baseIndent)
	case strings.HasSuffix(contentBeforeClosingBracket, ","):
		newEntry = fmt.Sprintf("\n%s%s,", itemIndent, quotedItem)
	default:
		// The last entry has no trailing comma; add one before appending.
		newEntry = fmt.Sprintf(",\n%s%s,", itemIndent, quotedItem)
	}

	s.content = s.content[:listEndIndex] + newEntry + s.content[listEndIndex:]
	return nil
}

// AppendBlock appends a block of text to the end of the settings content.
func (s *SettingsEditor) AppendBlock(block string) {
	s.content += block
}

// RemoveCoreSetting strips a single-line setting assignment (e.g. SECRET_KEY = ...)
// and any preceding comment/blank lines from settings.py.
func (s *SettingsEditor) RemoveCoreSetting(settingName string) {
	lines := strings.Split(s.content, "\n")
	var result []string
	for i := 0; i < len(lines); i++ {
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

// RemoveDatabasesBlock strips the original DATABASES = { ... } block from settings.py.
// It uses brace-counting so it works regardless of Django version, quote style, or comments.
func (s *SettingsEditor) RemoveDatabasesBlock() {
	lines := strings.Split(s.content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if strings.HasPrefix(trimmed, "DATABASES") && strings.Contains(trimmed, "=") && strings.Contains(trimmed, "{") {
			// Walk back over the comment header and blank lines Django emits
			// above DATABASES, so removing the block does not leave them behind.
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				isDatabaseComment := strings.HasPrefix(prev, "#") &&
					(strings.Contains(prev, "atabase") || strings.Contains(prev, "djangoproject.com"))
				if prev != "" && !isDatabaseComment {
					break // real code
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

// AnchorError reports that an edit could not be applied because the literal
// text it keys on was not found in settings.py.
//
// This is the failure mode the whole type is exposed to: the editor rewrites
// Django's generated file by matching exact strings ('DIRS': [], the messages
// context-processor line). When Django reflows settings.py those matches stop
// firing. Previously the strings.Replace simply did nothing and the caller
// reported success, producing a project that looks scaffolded and does not
// work. Returning an error converts that into a failure the user can act on.
type AnchorError struct {
	// Edit names what was being attempted, in the user's terms.
	Edit string
	// Anchor is the literal text that was expected and not found.
	Anchor string
}

func (e *AnchorError) Error() string {
	return fmt.Sprintf(
		"could not apply %s: %s does not contain the expected anchor %q "+
			"(the file may have been generated by an unsupported Django version, or already edited by hand)",
		e.Edit, SettingsFile, e.Anchor,
	)
}

// ConfigureTemplatesDirs points TEMPLATES at the project-level templates/
// directory and adds STATICFILES_DIRS / STATIC_ROOT.
//
// It returns an AnchorError if Django's 'DIRS': [] line is absent, rather than
// leaving templates unconfigured and reporting success.
func (s *SettingsEditor) ConfigureTemplatesDirs() error {
	const templatesAnchor = "'DIRS': []"
	templatesDirsSetting := "'DIRS': [BASE_DIR / 'templates']"

	// Already applied — a previous pass configured it. Not an error: several
	// features re-open settings.py after it has been edited once.
	if !strings.Contains(s.content, templatesDirsSetting) {
		if !strings.Contains(s.content, templatesAnchor) {
			return &AnchorError{Edit: "the global templates directory", Anchor: templatesAnchor}
		}
		s.content = strings.Replace(s.content, templatesAnchor, templatesDirsSetting, 1)
	}
	staticfilesDirsSetting := "STATICFILES_DIRS = [\n    BASE_DIR / 'static',\n]"
	staticRootSetting := "STATIC_ROOT = BASE_DIR / 'staticfiles'"
	staticBlock := staticfilesDirsSetting + "\n\n" + staticRootSetting
	// The static block has a real fallback — appending it to the end of the
	// file produces a working configuration — so a missing STATIC_URL anchor is
	// not a silent failure and does not error.
	if !strings.Contains(s.content, "STATICFILES_DIRS") {
		staticUrlMarker := "STATIC_URL = "
		idx := strings.Index(s.content, staticUrlMarker)
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
		// STATICFILES_DIRS exists but STATIC_ROOT doesn't — add it after STATICFILES_DIRS
		staticfilesDirsEnd := strings.Index(s.content, "STATICFILES_DIRS")
		if staticfilesDirsEnd != -1 {
			// Find the closing bracket of STATICFILES_DIRS
			closingBracket := strings.Index(s.content[staticfilesDirsEnd:], "]")
			if closingBracket != -1 {
				insertPos := staticfilesDirsEnd + closingBracket + 1
				// Skip past the newline after ]
				if insertPos < len(s.content) && s.content[insertPos] == '\n' {
					insertPos++
				}
				s.content = s.content[:insertPos] + "\n" + staticRootSetting + "\n" + s.content[insertPos:]
			}
		}
	}

	return nil
}

// AddProjectContextProcessor registers the project's context processor so
// templates can read project_name and app_name.
//
// It returns an AnchorError if Django's messages context-processor line is
// absent: without it the processor is never registered and every template
// renders with those variables empty.
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
		// Restore the blank line that separates the managed block from whatever
		// follows it. Without this the replace path emits one fewer newline than
		// the append path in InjectEnvConfig, so re-running the pipeline shifts
		// the rest of the file up a line each time.
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

	for i := 0; i < len(lines); i++ {
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

func (s *SettingsEditor) managedEnvConfigBlock(databaseType config.Database) string {
	var databaseBlock string
	switch databaseType {
	case config.PostgresDocker, config.PostgresString:
		databaseBlock = `DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': env('DB_NAME'),
        'USER': env('DB_USER', default='postgres'),
        'PASSWORD': env('DB_PASSWORD'),
        'HOST': env('DB_HOST', default='localhost'),
        'PORT': env('DB_PORT', default='5432'),
    }
}`
	case config.MySQLDocker:
		databaseBlock = `DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.mysql',
        'NAME': env('DB_NAME'),
        'USER': env('DB_USER', default='django_user'),
        'PASSWORD': env('DB_PASSWORD'),
        'HOST': env('DB_HOST', default='localhost'),
        'PORT': env('DB_PORT', default='3306'),
        'OPTIONS': {
            'charset': 'utf8mb4',
            'isolation_level': 'read committed',
        },
    }
}`
	default:
		databaseBlock = `DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}`
	}

	return managedEnvConfigStart + `
# Load environment variables from .env file
load_dotenv(BASE_DIR / '.env')

# Initialize django-environ
env = environ.Env()

# =============================================================================
# Core Settings (configured via environment variables)
# =============================================================================
SECRET_KEY = env('SECRET_KEY', default='django-insecure-dev-key-change-in-production')

DEBUG = env.bool('DEBUG', default=True)

ALLOWED_HOSTS = env.list('ALLOWED_HOSTS', default=['localhost', '127.0.0.1'])

# =============================================================================
# Database
# =============================================================================
` + databaseBlock + `
` + managedEnvConfigEnd
}

// InjectEnvConfig adds django-environ imports, env configuration, and database settings.
func (s *SettingsEditor) InjectEnvConfig(databaseType config.Database) {
	var newImports []string
	if !strings.Contains(s.content, "import environ") {
		newImports = append(newImports, "import environ")
	}
	if !strings.Contains(s.content, "from dotenv") {
		newImports = append(newImports, "from dotenv import load_dotenv")
	}

	if len(newImports) > 0 {
		importsBlock := "\n\n" + strings.Join(newImports, "\n") + "\n"

		// Anchor the new imports after an existing one so they land in the
		// import section. Unlike the edits guarded by AnchorError, this has a
		// working fallback — prepending to the file is still valid Python — so
		// a miss is not a silent failure.
		switch {
		case strings.Contains(s.content, "from pathlib import Path"):
			s.content = strings.Replace(s.content, "from pathlib import Path", "from pathlib import Path"+importsBlock, 1)
		case strings.Contains(s.content, "import os"):
			s.content = strings.Replace(s.content, "import os", "import os"+importsBlock, 1)
		default:
			s.content = importsBlock + s.content
		}
	}

	managedBlock := s.managedEnvConfigBlock(databaseType)
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

// RemovePasswordValidatorsBlock strips the default AUTH_PASSWORD_VALIDATORS block
// generated by Django's startproject, so we can replace it with an enhanced version.
func (s *SettingsEditor) RemovePasswordValidatorsBlock() {
	lines := strings.Split(s.content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if strings.HasPrefix(trimmed, "AUTH_PASSWORD_VALIDATORS") && strings.Contains(trimmed, "=") && strings.Contains(trimmed, "[") {
			// Walk back over the comment header and blank lines above the block,
			// so removing it does not leave an orphaned heading behind.
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				isValidatorComment := strings.HasPrefix(prev, "#") &&
					(strings.Contains(prev, "assword") || strings.Contains(prev, "djangoproject.com"))
				if prev != "" && !isValidatorComment {
					break // real code
				}
				result = result[:len(result)-1]
			}

			// Skip past the closing bracket using bracket counting
			depth := 0
			for i < len(lines) {
				depth += strings.Count(lines[i], "[") - strings.Count(lines[i], "]")
				i++
				if depth <= 0 {
					break
				}
			}
			// Skip trailing blank lines
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

// InjectSecuritySettings appends production-ready security settings to settings.py.
// It replaces the default AUTH_PASSWORD_VALIDATORS with an enhanced version and adds
// security headers/cookies that are safe in dev (via DEBUG check) and production-ready.
// It is safe to call more than once: the security block is appended only if it
// is not already present. Callers used to have to guard this themselves, which
// meant every new call site could reintroduce a duplicate block.
func (s *SettingsEditor) InjectSecuritySettings() {
	if s.Contains("PASSWORD_HASHERS") {
		return
	}

	s.RemovePasswordValidatorsBlock()

	securityBlock := `
# =============================================================================
# Security Settings
# =============================================================================

# Password hashers - Argon2 is the most secure algorithm
PASSWORD_HASHERS = [
    'django.contrib.auth.hashers.Argon2PasswordHasher',
    'django.contrib.auth.hashers.PBKDF2PasswordHasher',
    'django.contrib.auth.hashers.PBKDF2SHA1PasswordHasher',
    'django.contrib.auth.hashers.BCryptSHA256PasswordHasher',
    'django.contrib.auth.hashers.ScryptPasswordHasher',
]

# Password validation
AUTH_PASSWORD_VALIDATORS = [
    {'NAME': 'django.contrib.auth.password_validation.UserAttributeSimilarityValidator'},
    {'NAME': 'django.contrib.auth.password_validation.MinimumLengthValidator'},
    {'NAME': 'django.contrib.auth.password_validation.CommonPasswordValidator'},
    {'NAME': 'django.contrib.auth.password_validation.NumericPasswordValidator'},
]

# Security headers and cookies (production-safe via DEBUG check)
if not DEBUG:
    SECURE_SSL_REDIRECT = env.bool('SECURE_SSL_REDIRECT', default=True)
    SECURE_HSTS_SECONDS = env.int('SECURE_HSTS_SECONDS', default=31536000)  # 1 year
    SECURE_HSTS_INCLUDE_SUBDOMAINS = True
    SECURE_HSTS_PRELOAD = True
    SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')
    SESSION_COOKIE_SECURE = True
    CSRF_COOKIE_SECURE = True

SECURE_CONTENT_TYPE_NOSNIFF = True
SESSION_COOKIE_HTTPONLY = True
CSRF_COOKIE_HTTPONLY = True
X_FRAME_OPTIONS = 'DENY'
`
	s.content += securityBlock
}

// Save writes the current content back to disk.
func (s *SettingsEditor) Save() error {
	if err := os.WriteFile(s.path, []byte(s.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", SettingsFile, err)
	}
	return nil
}
