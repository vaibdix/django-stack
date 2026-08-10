package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// joinPath is filepath.Join, named so plan.go can build paths without importing
// path/filepath for a single call.
var joinPath = filepath.Join

// Generate scaffolds the project described by the Env.
//
// The work itself is in plan(): this function validates the configuration,
// prepares the directory, then runs each applicable step in order. Keeping the
// driver this small is the point — the sequence lives in one readable list
// rather than being spread through a 250-line function.
func (e *Env) Generate(ctx context.Context) error {
	e.ctx = ctx

	if err := e.validate(); err != nil {
		return err
	}

	projectPath, err := e.prepareProjectDirectory()
	if err != nil {
		return err
	}

	for _, s := range plan() {
		if !s.applies(e.Cfg) {
			continue
		}

		// Exactly one progress update per step, emitted here and nowhere else.
		// The step functions used to call Progress themselves — fifteen calls
		// scattered across a dozen files, with Tailwind firing twice — so the
		// bar never matched the declared total. StepCount counts this same
		// list, so the two cannot drift.
		e.Progress(s.label)

		if err := s.run(e, projectPath); err != nil {
			// Naming the step turns a bare "permission denied" into something
			// the user can act on.
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}

	e.reportPerformanceTip()
	e.Step("Django project setup complete!")
	return nil
}

// validate checks the configuration before anything is written to disk.
func (e *Env) validate() error {
	if e.Cfg.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if err := e.Cfg.ValidateProjectName(); err != nil {
		return fmt.Errorf("invalid project name '%s': %w", e.Cfg.ProjectName, err)
	}
	if err := e.Cfg.ValidateAppName(); err != nil {
		return fmt.Errorf("invalid app name '%s': %w", e.Cfg.AppName, err)
	}
	return nil
}

// prepareProjectDirectory resolves the project path and creates it.
//
// An existing non-empty directory is refused rather than merged into: half of
// someone else's project mixed with half of a generated one is worse than a
// clear failure.
func (e *Env) prepareProjectDirectory() (string, error) {
	projectPath := e.Cfg.ProjectName
	if !filepath.IsAbs(projectPath) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		projectPath = filepath.Join(wd, e.Cfg.ProjectName)
	}

	if err := checkDirectoryIsUsable(projectPath); err != nil {
		return "", err
	}

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}
	e.Step(fmt.Sprintf("Project directory created: %s", projectPath))

	return projectPath, nil
}

// checkDirectoryIsUsable reports whether path is absent or an empty directory.
func checkDirectoryIsUsable(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("project path exists and is not a directory: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("project directory already exists and is not empty: %s", path)
	}
	return nil
}

// writeDevStartScript writes the platform-appropriate start-dev script.
//
// A failure here is reported but not fatal: the project is complete and
// runnable without the convenience script.
func (e *Env) writeDevStartScript(projectPath string) error {
	scriptPath := filepath.Join(projectPath, devStartScriptName())
	script := generateStartScript(projectPath, findVSCode(), e.Cfg.SetupTailwind)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		e.Step("Warning: Could not create development start script: " + err.Error())
		return nil
	}

	e.Step(fmt.Sprintf("Created: %s", filepath.Base(scriptPath)))
	return nil
}
