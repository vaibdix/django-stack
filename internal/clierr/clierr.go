// Package clierr defines the CLI's user-facing error taxonomy.
//
// It is named clierr rather than errors so that callers can use it alongside
// the standard library's errors package without aliasing either one.
//
// Each sentinel pairs a stable code with a message and an actionable hint. The
// hint is the point: a scaffolding failure should tell the user what to do
// next, not just what went wrong.
package clierr

import (
	"errors"
	"fmt"
	"io"
)

// CLIError is an error with a stable code and a remediation hint.
type CLIError struct {
	Code    string
	Message string
	Hint    string

	// cause is the underlying error, if this was wrapped around one.
	cause error
}

func (e *CLIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped cause to errors.Is and errors.As.
func (e *CLIError) Unwrap() error { return e.cause }

// Is matches any CLIError carrying the same code, so a wrapped sentinel still
// compares equal to the sentinel it was derived from.
func (e *CLIError) Is(target error) bool {
	var t *CLIError
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}

// WithHint returns a copy carrying a different hint, leaving the receiver
// untouched so package-level sentinels stay immutable.
func (e *CLIError) WithHint(hint string) *CLIError {
	clone := *e
	clone.Hint = hint
	return &clone
}

// Wrap returns a copy of the sentinel carrying cause as its underlying error.
// The result still satisfies errors.Is against the original sentinel.
func (e *CLIError) Wrap(cause error) *CLIError {
	clone := *e
	clone.cause = cause
	return &clone
}

// Fprint writes the message and hint to w.
//
// It takes an io.Writer rather than hardcoding os.Stderr so the TUI can render
// failures into its own surface — writing to stderr mid-render corrupts the
// alternate screen.
func (e *CLIError) Fprint(w io.Writer) {
	_, _ = fmt.Fprintf(w, "✗ %s\n", e.Message)
	if e.Hint != "" {
		_, _ = fmt.Fprintf(w, "  Hint: %s\n", e.Hint)
	}
}

// FprintWithContext is Fprint with a leading context label.
func (e *CLIError) FprintWithContext(w io.Writer, context string) {
	_, _ = fmt.Fprintf(w, "✗ %s: %s\n", context, e.Message)
	if e.Hint != "" {
		_, _ = fmt.Fprintf(w, "  Hint: %s\n", e.Hint)
	}
}

// The CLI's error taxonomy. Each sentinel pairs a stable code with a message
// and an actionable hint; codes are part of the user-facing contract and must
// not be renumbered.
var (
	ErrProjectExists = &CLIError{
		Code:    "E001",
		Message: "project directory already exists",
		Hint:    "Choose a different project name or remove the existing directory",
	}

	ErrInvalidProjectName = &CLIError{
		Code:    "E002",
		Message: "invalid project name",
		Hint:    "Use only letters, numbers, hyphens, and underscores (must start with a letter)",
	}

	ErrReservedWord = &CLIError{
		Code:    "E003",
		Message: "project name is a Python reserved word",
		Hint:    "Choose a different name that is not a Python keyword",
	}

	ErrInvalidDjangoVersion = &CLIError{
		Code:    "E004",
		Message: "invalid Django version format",
		Hint:    "Use format like '5.2', '4.2.1', or 'latest'",
	}

	ErrPythonNotFound = &CLIError{
		Code:    "E005",
		Message: "Python not found",
		Hint:    "Install Python 3.8+ and ensure it is in your PATH",
	}

	ErrVenvCreationFailed = &CLIError{
		Code:    "E006",
		Message: "failed to create virtual environment",
		Hint:    "Ensure virtualenv or venv is available",
	}

	ErrDjangoInstallFailed = &CLIError{
		Code:    "E007",
		Message: "failed to install Django",
		Hint:    "Check your internet connection and Python environment",
	}

	ErrProjectCreationFailed = &CLIError{
		Code:    "E008",
		Message: "failed to create Django project",
		Hint:    "Check Django installation and project name",
	}

	ErrAppCreationFailed = &CLIError{
		Code:    "E009",
		Message: "failed to create Django app",
		Hint:    "Ensure the app name is valid and doesn't conflict with existing apps",
	}

	ErrGitInitFailed = &CLIError{
		Code:    "E010",
		Message: "failed to initialize git repository",
		Hint:    "Ensure git is installed and configured",
	}

	ErrTailwindSetupFailed = &CLIError{
		Code:    "E011",
		Message: "failed to set up Tailwind CSS",
		Hint:    "Ensure Node.js and npm are installed",
	}

	ErrConfigReadFailed = &CLIError{
		Code:    "E012",
		Message: "failed to read configuration",
		Hint:    "Check config file permissions and format",
	}

	ErrConfigWriteFailed = &CLIError{
		Code:    "E013",
		Message: "failed to write configuration",
		Hint:    "Check config directory permissions",
	}

	ErrUnknownConfigKey = &CLIError{
		Code:    "E014",
		Message: "unknown configuration key",
		Hint:    "This build no longer supports config subcommands; use the interactive setup instead",
	}
)

// NewCLIError builds a one-off error for a case with no sentinel.
func NewCLIError(code, message, hint string) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Hint:    hint,
	}
}
