package scaffold

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) applyTailwind(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if err := e.ensurePackageJSON(); err != nil {
		return fmt.Errorf("create package.json: %w", err)
	}

	name, args := e.Node.Install(render.TailwindPackages()...)
	if output, err := e.runIn(ctx, e.dir, name, args...); err != nil {
		return fmt.Errorf("install Tailwind CSS with %s: %w\nOutput: %s", e.Node.Name, err, string(output))
	}
	e.Step("Tailwind CSS " + render.TailwindVersion() + " installed with " + e.Node.Name + ".")

	for _, dir := range []string{render.TailwindSourceDir, filepath.Join("static", "dist")} {
		if err := e.machine().FS.MkdirAll(filepath.Join(e.dir, dir)); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	e.Step("Tailwind directory structure created.")

	if err := e.emitGroup(render.GroupTailwind); err != nil {
		return err
	}
	e.Step("Tailwind source CSS created.")

	return e.updatePackageJSONForTailwind()
}

func (e *Env) ensurePackageJSON() error {
	switch _, err := e.machine().FS.Stat(filepath.Join(e.dir, render.PackageJSONFile)); {
	case err == nil:
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	manifest := map[string]any{
		"name":    strings.ToLower(filepath.Base(e.dir)),
		"version": "1.0.0",
		"private": true,
	}
	content, err := render.MarshalPackageJSON(manifest)
	if err != nil {
		return err
	}
	if err := e.write(render.PackageJSONFile, content, render.FileMode); err != nil {
		return err
	}
	e.Step("Created: package.json")
	return nil
}

func (e *Env) buildTailwindCSS(ctx context.Context) error {
	manager := e.Node
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	name, args := manager.Run("build:css")
	if output, err := e.runIn(ctx, e.dir, name, args...); err != nil {
		return fmt.Errorf("build Tailwind CSS, so every page would reference a stylesheet that does not exist: %w\nOutput: %s", err, string(output))
	}

	e.Step("Tailwind CSS compiled successfully.")
	e.Step("Run '" + manager.RunCommand("watch:css") + "' for development or '" +
		manager.RunCommand("build:css") + "' for production.")
	return nil
}

func (e *Env) updatePackageJSONForTailwind() error {
	packageData, err := e.read(render.PackageJSONFile)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}
	var packageJSON map[string]any
	if err := json.Unmarshal(packageData, &packageJSON); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	tailwindArgs := " -i ./" + render.TailwindSourceDir + "/" + render.TailwindSourceFile + " -o ./" + render.TailwindOutput
	scripts := map[string]string{
		"build:css": render.TailwindBinary + tailwindArgs,
		"watch:css": render.TailwindBinary + tailwindArgs + " --watch",
	}
	packageJSON["engines"] = map[string]string{"node": ">=" + strconv.Itoa(render.TailwindMinNode())}
	if existingScripts, ok := packageJSON["scripts"].(map[string]any); ok {
		for key, value := range scripts {
			existingScripts[key] = value
		}
		packageJSON["scripts"] = existingScripts
	} else {
		packageJSON["scripts"] = scripts
	}
	updatedData, err := render.MarshalPackageJSON(packageJSON)
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}
	if err := e.write(render.PackageJSONFile, updatedData, render.FileMode); err != nil {
		return err
	}
	e.Step("package.json updated with Tailwind scripts.")
	return nil
}
