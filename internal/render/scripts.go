package render

import (
	"runtime"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func generateStartScript(goos string, useTailwind bool) string {
	return Render(DevStartScriptNameFor(goos)+".tmpl", struct{ Tailwind bool }{Tailwind: useTailwind})
}

func DevStartScriptName() string { return DevStartScriptNameFor(runtime.GOOS) }

func DevStartScriptNameFor(goos string) string { return "start-dev" + pyenv.ScriptExt(goos) }

func devServerCommandFor(goos string) string {
	return pyenv.InterpreterCommand(goos) + " manage.py runserver"
}
