package scaffold

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const tailwindBinary = "tailwindcss"

const tailwindVersion = "4.3.3"

const tailwindMinNode = 20

func tailwindPackages() []string {
	return []string{"tailwindcss@" + tailwindVersion, "@tailwindcss/cli@" + tailwindVersion}
}

const (
	tailwindSourceDir  = "assets"
	tailwindSourceFile = "tailwind.css"
	tailwindOutput     = "static/dist/styles.css"
)

func (e *Env) applyTailwind(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if err := e.ensurePackageJSON(); err != nil {
		return fmt.Errorf("create package.json: %w", err)
	}

	name, args := e.node.install(tailwindPackages()...)
	if output, err := e.runIn(ctx, e.dir, name, args...); err != nil {
		return fmt.Errorf("install Tailwind CSS with %s: %w\nOutput: %s", e.node.Name, err, string(output))
	}
	e.Step("Tailwind CSS " + tailwindVersion + " installed with " + e.node.Name + ".")

	for _, dir := range []string{tailwindSourceDir, filepath.Join("static", "dist")} {
		if err := os.MkdirAll(filepath.Join(e.dir, dir), dirMode); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	e.Step("Tailwind directory structure created.")

	if err := e.write(filepath.Join(tailwindSourceDir, tailwindSourceFile), []byte(renderStatic("tailwind-source.css.tmpl")), fileMode); err != nil {
		return err
	}
	e.Step("Tailwind source CSS created.")

	return e.updatePackageJSONForTailwind()
}

func (e *Env) ensurePackageJSON() error {
	switch _, err := os.Stat(filepath.Join(e.dir, packageJSONFile)); {
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
	content, err := marshalPackageJSON(manifest)
	if err != nil {
		return err
	}
	if err := e.write(packageJSONFile, content, fileMode); err != nil {
		return err
	}
	e.Step("Created: package.json")
	return nil
}

func (e *Env) buildTailwindCSS(ctx context.Context) error {
	manager := e.node
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	name, args := manager.run("build:css")
	if output, err := e.runIn(ctx, e.dir, name, args...); err != nil {
		return fmt.Errorf("build Tailwind CSS, so every page would reference a stylesheet that does not exist: %w\nOutput: %s", err, string(output))
	}

	e.Step("Tailwind CSS compiled successfully.")
	e.Step("Run '" + manager.RunCommand("watch:css") + "' for development or '" +
		manager.RunCommand("build:css") + "' for production.")
	return nil
}

func (e *Env) updatePackageJSONForTailwind() error {
	packageData, err := e.read(packageJSONFile)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}
	var packageJSON map[string]any
	if err := json.Unmarshal(packageData, &packageJSON); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	tailwindArgs := " -i ./" + tailwindSourceDir + "/" + tailwindSourceFile + " -o ./" + tailwindOutput
	scripts := map[string]string{
		"build:css": tailwindBinary + tailwindArgs,
		"watch:css": tailwindBinary + tailwindArgs + " --watch",
	}
	packageJSON["engines"] = map[string]string{"node": ">=" + strconv.Itoa(tailwindMinNode)}
	if existingScripts, ok := packageJSON["scripts"].(map[string]any); ok {
		for key, value := range scripts {
			existingScripts[key] = value
		}
		packageJSON["scripts"] = existingScripts
	} else {
		packageJSON["scripts"] = scripts
	}
	updatedData, err := marshalPackageJSON(packageJSON)
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}
	if err := e.write(packageJSONFile, updatedData, fileMode); err != nil {
		return err
	}
	e.Step("package.json updated with Tailwind scripts.")
	return nil
}

func marshalPackageJSON(manifest any) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
