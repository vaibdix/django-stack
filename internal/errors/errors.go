package errors

import (
	"fmt"
	"os"
)

type CLIError struct {
	Code    string
	Message string
	Hint    string
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CLIError) WithHint(hint string) *CLIError {
	return &CLIError{
		Code:    e.Code,
		Message: e.Message,
		Hint:    hint,
	}
}

func (e *CLIError) Print() {
	fmt.Fprintf(os.Stderr, "✗ %s\n", e.Message)
	if e.Hint != "" {
		fmt.Fprintf(os.Stderr, "  Hint: %s\n", e.Hint)
	}
}

func (e *CLIError) PrintWithContext(context string) {
	fmt.Fprintf(os.Stderr, "✗ %s: %s\n", context, e.Message)
	if e.Hint != "" {
		fmt.Fprintf(os.Stderr, "  Hint: %s\n", e.Hint)
	}
}

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

func NewCLIError(code, message, hint string) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Hint:    hint,
	}
}
