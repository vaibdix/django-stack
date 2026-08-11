package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// applyTailwind configures Tailwind CSS v4.
//
// Every failure here is non-fatal by design: the generated project is fully
// functional without Tailwind, so problems are reported as warnings and
// scaffolding continues. It therefore returns nothing — the previous signature
// returned an error that was always nil, which read as though the caller had a
// decision to make.
func (e *Env) applyTailwind(projectPath string) {
	if !e.Cfg.SetupTailwind {
		return
	}

	if !pyenv.CommandAvailable("npm") {
		e.Step("Warning: npm not found. Please install Node.js to use Tailwind CSS.")
		return
	}

	// npm, not pip — these go through the shared runner so they honour the run
	// context. Previously they were bare exec.Command calls, so quitting the TUI
	// left `npm install` downloading into a half-built project.
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if output, err := e.runIn(ctx, projectPath, "npm", "init", "-y"); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to initialize npm: %v\nOutput: %s", err, string(output)))
		return
	}
	e.Step("npm initialized.")

	if output, err := e.runIn(ctx, projectPath, "npm", "install", "tailwindcss", "@tailwindcss/cli"); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to install Tailwind CSS: %v\nOutput: %s", err, string(output)))
		return
	}
	e.Step("Tailwind CSS v4 installed.")

	staticSrcPath := filepath.Join(projectPath, "static", "src")
	staticDistPath := filepath.Join(projectPath, "static", "dist")
	if err := os.MkdirAll(staticSrcPath, 0755); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to create static/src directory: %v", err))
		return
	}
	if err := os.MkdirAll(staticDistPath, 0755); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to create static/dist directory: %v", err))
		return
	}
	e.Step("Tailwind directory structure created.")
	tailwindCSS := renderStatic("tailwind-source.css.tmpl")
	if err := os.WriteFile(filepath.Join(staticSrcPath, "styles.css"), []byte(tailwindCSS), 0644); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to create styles.css: %v", err))
		return
	}
	e.Step("Tailwind source CSS created.")
	if err := e.updatePackageJSONForTailwind(projectPath); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to update package.json: %v", err))
		return
	}
	if e.Cfg.CreateTemplates {
		if err := e.updateBaseTemplateForTailwind(projectPath); err != nil {
			e.Step(fmt.Sprintf("Warning: Failed to update base.html: %v", err))
			return
		}
	}
}

// buildTailwindCSS compiles static/dist/styles.css.
//
// This is a separate step from applyTailwind, and it runs near the end of the
// plan, because Tailwind v4 emits only the utilities it finds in the source
// files that exist when it runs. Compiling during applyTailwind produced a
// stylesheet built six steps before the Django app, the REST example and the
// allauth pages were written, so every utility used only by those templates was
// missing: the home page rendered correctly and every other page rendered
// unstyled.
func (e *Env) buildTailwindCSS(projectPath string) error {
	if !pyenv.CommandAvailable("npm") {
		return nil
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	if output, err := e.runIn(ctx, projectPath, "npm", "run", "build:css"); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to build Tailwind CSS: %v\nOutput: %s", err, string(output)))
		return nil
	}

	e.Step("Tailwind CSS compiled successfully.")
	e.Step("Run 'npm run watch:css' for development or 'npm run build:css' for production.")
	return nil
}

func (e *Env) updatePackageJSONForTailwind(projectPath string) error {
	packageJSONPath := filepath.Join(projectPath, "package.json")
	packageData, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}
	var packageJSON map[string]any
	if err := json.Unmarshal(packageData, &packageJSON); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}
	scripts := map[string]string{
		"build:css": "npx tailwindcss -i ./static/src/styles.css -o ./static/dist/styles.css",
		"watch:css": "npx tailwindcss -i ./static/src/styles.css -o ./static/dist/styles.css --watch",
	}
	if existingScripts, ok := packageJSON["scripts"].(map[string]any); ok {
		for key, value := range scripts {
			existingScripts[key] = value
		}
		packageJSON["scripts"] = existingScripts
	} else {
		packageJSON["scripts"] = scripts
	}
	updatedData, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}
	if err := os.WriteFile(packageJSONPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}
	e.Step("package.json updated with Tailwind scripts.")
	return nil
}

func (e *Env) updateBaseTemplateForTailwind(projectPath string) error {
	baseTemplatePath := filepath.Join(projectPath, "templates", "base.html")
	baseContent, err := os.ReadFile(baseTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read base.html: %w", err)
	}
	updatedBaseContent := strings.Replace(string(baseContent),
		`<link rel="stylesheet" href="{% static 'css/style.css' %}">`,
		`<link rel="stylesheet" href="{% static 'dist/styles.css' %}">`, 1)
	if err := os.WriteFile(baseTemplatePath, []byte(updatedBaseContent), 0644); err != nil {
		return fmt.Errorf("failed to update base.html: %w", err)
	}

	e.Step("base.html updated to use Tailwind CSS.")
	return nil
}
