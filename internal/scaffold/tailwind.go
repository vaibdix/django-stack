package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const tailwindBinary = "tailwindcss"

// The Tailwind input lives outside STATICFILES_DIRS on purpose: collectstatic
// would copy it, and the manifest storage aborts on its @import "tailwindcss".
// Only the compiled static/dist/styles.css is a static file.
const (
	tailwindSourceDir  = "assets"
	tailwindSourceFile = "tailwind.css"
	tailwindOutput     = "static/dist/styles.css"
)

func (e *Env) applyTailwind(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if err := e.ensurePackageJSON(projectPath); err != nil {
		return fmt.Errorf("failed to create package.json: %w", err)
	}

	name, args := e.node.Install("tailwindcss", "@tailwindcss/cli")
	if output, err := e.runIn(ctx, projectPath, name, args...); err != nil {
		return fmt.Errorf("failed to install Tailwind CSS with %s: %w\nOutput: %s", e.node.Name, err, string(output))
	}
	e.Step("Tailwind CSS v4 installed with " + e.node.Name + ".")

	for _, dir := range []string{tailwindSourceDir, filepath.Join("static", "dist")} {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	e.Step("Tailwind directory structure created.")

	source := filepath.Join(projectPath, tailwindSourceDir, tailwindSourceFile)
	if err := os.WriteFile(source, []byte(renderStatic("tailwind-source.css.tmpl")), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", tailwindSourceFile, err)
	}
	e.Step("Tailwind source CSS created.")

	return e.updatePackageJSONForTailwind(projectPath)
}

func (e *Env) ensurePackageJSON(projectPath string) error {
	path := filepath.Join(projectPath, "package.json")
	switch _, err := os.Stat(path); {
	case err == nil:
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return err
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
	manager := e.node
	ctx, cancel := e.timeoutContext()
	defer cancel()

	name, args := manager.Run("build:css")
	if output, err := e.runIn(ctx, projectPath, name, args...); err != nil {
		return fmt.Errorf("failed to build Tailwind CSS, so every page would reference a stylesheet that does not exist: %w\nOutput: %s", err, string(output))
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

	tailwindArgs := " -i ./" + tailwindSourceDir + "/" + tailwindSourceFile + " -o ./" + tailwindOutput
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
