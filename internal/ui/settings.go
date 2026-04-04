package ui

import (
	"fmt"
	"os"
	"strings"
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
	if contentBeforeClosingBracket == "" {
		newEntry = fmt.Sprintf("\n%s%s,\n%s", itemIndent, quotedItem, baseIndent)
	} else if strings.HasSuffix(contentBeforeClosingBracket, ",") {
		newEntry = fmt.Sprintf("\n%s%s,", itemIndent, quotedItem)
	} else {
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
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(prev, "#") && (strings.Contains(prev, "atabase") || strings.Contains(prev, "djangoproject.com")) {
					result = result[:len(result)-1]
				} else if prev == "" {
					result = result[:len(result)-1]
				} else {
					break
				}
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

// ConfigureTemplatesDirs updates TEMPLATES DIRS and adds STATICFILES_DIRS.
func (s *SettingsEditor) ConfigureTemplatesDirs() {
	templatesDirsSetting := "'DIRS': [BASE_DIR / 'templates']"
	if !strings.Contains(s.content, templatesDirsSetting) {
		s.content = strings.Replace(s.content, "'DIRS': []", templatesDirsSetting, 1)
	}
	staticfilesDirsSetting := "STATICFILES_DIRS = [\n    BASE_DIR / 'static',\n]"
	staticRootSetting := "STATIC_ROOT = BASE_DIR / 'staticfiles'"
	staticBlock := staticfilesDirsSetting + "\n\n" + staticRootSetting
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
}

// AddProjectContextProcessor adds the project-level context processor.
func (s *SettingsEditor) AddProjectContextProcessor(projectModuleName string) {
	contextProcessor := fmt.Sprintf("'%s.context_processors.project_context'", projectModuleName)
	if strings.Contains(s.content, contextProcessor) {
		return
	}

	s.content = strings.Replace(
		s.content,
		"'django.contrib.messages.context_processors.messages',",
		"'django.contrib.messages.context_processors.messages',\n                "+contextProcessor+",",
		1,
	)
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
		s.content += suffix
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

func (s *SettingsEditor) managedEnvConfigBlock(databaseType string) string {
	var databaseBlock string
	switch databaseType {
	case "postgres-docker", "postgres-string":
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
	case "mysql-docker":
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
func (s *SettingsEditor) InjectEnvConfig(databaseType string) {
	var newImports []string
	if !strings.Contains(s.content, "import environ") {
		newImports = append(newImports, "import environ")
	}
	if !strings.Contains(s.content, "from dotenv") {
		newImports = append(newImports, "from dotenv import load_dotenv")
	}

	if len(newImports) > 0 {
		importsBlock := "\n\n" + strings.Join(newImports, "\n") + "\n"
		if strings.Contains(s.content, "from pathlib import Path") {
			s.content = strings.Replace(s.content, "from pathlib import Path", "from pathlib import Path"+importsBlock, 1)
		} else if strings.Contains(s.content, "import os") {
			s.content = strings.Replace(s.content, "import os", "import os"+importsBlock, 1)
		} else {
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
			// Remove preceding comments and blank lines related to password validation
			for len(result) > 0 {
				prev := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(prev, "#") && (strings.Contains(prev, "assword") || strings.Contains(prev, "djangoproject.com")) {
					result = result[:len(result)-1]
				} else if prev == "" {
					result = result[:len(result)-1]
				} else {
					break
				}
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
func (s *SettingsEditor) InjectSecuritySettings() {
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
