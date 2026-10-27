package scaffold

import (
	"context"
	"os/exec"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) findVSCode() string {
	path, err := e.dependencies().lookPath("code")
	if err != nil {
		return ""
	}
	return path
}

func (e *Env) StartDevelopmentEnvironment(ctx context.Context) {
	projectPath, err := e.projectPath()
	if err != nil {
		e.Step("Could not open the project: " + err.Error())
		return
	}

	editor := e.findVSCode()
	if editor == "" {
		e.Step("VS Code was not found. Open " + projectPath + " in your editor, or run " + render.DevStartScriptName() + " there.")
		return
	}

	if err := e.dependencies().startDetached(ctx, projectPath, editor, "-n", projectPath); err != nil {
		e.Step("Failed to open VS Code: " + err.Error())
		e.Step("Run " + render.DevStartScriptName() + " in the project folder")
		return
	}
	e.Step("VS Code opened; it starts the dev task once you allow automatic tasks.")
}

func startDetached(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
