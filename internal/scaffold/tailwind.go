package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const tailwindBinary = "tailwindcss"

func (e *Env) applyTailwind(projectPath string) {
	if !e.Cfg.SetupTailwind {
		return
	}

	manager, ok := detectNodeManager()
	if !ok {
		e.Step("Warning: neither pnpm nor npm was found. Install Node.js to use Tailwind CSS.")
		return
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	if err := e.ensurePackageJSON(projectPath); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to create package.json: %v", err))
		return
	}

	name, args := manager.Install("tailwindcss", "@tailwindcss/cli")
	if output, err := e.runIn(ctx, projectPath, name, args...); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to install Tailwind CSS: %v\nOutput: %s", err, string(output)))
		return
	}
	e.Step("Tailwind CSS v4 installed with " + manager.Name + ".")

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

func (e *Env) ensurePackageJSON(projectPath string) error {
	path := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	manifest := map[string]any{
		"name":    strings.ToLower(filepath.Base(projectPath)),
		"version": "1.0.0",
		"private": true,
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(content, '\n'), 0644); err != nil {
		return err
	}
	e.Step("Created: package.json")
	return nil
}

func (e *Env) buildTailwindCSS(projectPath string) error {
	manager, ok := detectNodeManager()
	if !ok {
		return nil
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	name, args := manager.Run("build:css")
	if output, err := e.runIn(ctx, projectPath, name, args...); err != nil {
		e.Step(fmt.Sprintf("Warning: Failed to build Tailwind CSS: %v\nOutput: %s", err, string(output)))
		return nil
	}

	e.Step("Tailwind CSS compiled successfully.")
	e.Step("Run '" + manager.RunCommand("watch:css") + "' for development or '" +
		manager.RunCommand("build:css") + "' for production.")
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

	const tailwindArgs = " -i ./static/src/styles.css -o ./static/dist/styles.css"
	scripts := map[string]string{
		"build:css": tailwindBinary + tailwindArgs,
		"watch:css": tailwindBinary + tailwindArgs + " --watch",
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
	if err := os.WriteFile(packageJSONPath, append(updatedData, '\n'), 0644); err != nil {
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
