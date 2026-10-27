package scaffold

import (
	"context"
	"fmt"
	"strings"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) freezeVersions(ctx context.Context, projectPath string) (map[string]string, error) {
	if e.frozen != nil {
		return e.frozen, nil
	}

	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	name, args := pyenv.FreezeCommand(e.Toolchain, projectPath)
	output, err := e.runIn(ctx, projectPath, name, args...)
	if err != nil {
		return nil, fmt.Errorf("read the installed package versions: %w", err)
	}

	versions := make(map[string]string)
	for line := range strings.SplitSeq(string(output), "\n") {
		distribution, version, ok := strings.Cut(strings.TrimSpace(line), "==")
		if !ok {
			continue
		}
		versions[pypi.NormalizeName(distribution)] = version
	}
	e.frozen = versions
	return versions, nil
}

func (e *Env) writeDependencyManifest(ctx context.Context) error {
	e.packages.AddDev(e.DeclaredFormattingPackages()...)

	versions, err := e.freezeVersions(ctx, e.dir)
	if err != nil {
		return err
	}
	runtime, err := render.PinSpecs(e.packages.Runtime, versions)
	if err != nil {
		return err
	}
	dev, err := render.PinSpecs(e.packages.Dev, versions)
	if err != nil {
		return err
	}

	if err := e.write(render.PyprojectFile, []byte(e.RenderPyproject(runtime, dev)), render.FileMode); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Declared %d runtime and %d dev dependencies in %s", len(runtime), len(dev), render.PyprojectFile))

	if err := e.writePythonVersionFile(); err != nil {
		return err
	}

	if e.Toolchain == pyenv.Pip {
		return e.writeRequirementsFiles(ctx, versions)
	}
	return e.lockDependencies(ctx, e.dir)
}

func (e *Env) writePythonVersionFile() error {
	_, _, _, version, ok := pyenv.ParseDetailedVersionOutput(e.Resolved.PythonVersion)
	if !ok {
		return fmt.Errorf("cannot write %s: the selected interpreter's version %q is not major.minor.micro",
			render.PythonVersionFile, e.Resolved.PythonVersion)
	}

	if err := e.write(render.PythonVersionFile, []byte(version+"\n"), render.FileMode); err != nil {
		return err
	}
	e.Step("Created: " + render.PythonVersionFile)
	return nil
}

func (e *Env) lockDependencies(ctx context.Context, projectPath string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if dated := e.resolveAtArgs(ctx); len(dated) > 0 {
		if output, err := e.runIn(ctx, projectPath, "uv", append([]string{"lock"}, dated...)...); err != nil {
			return fmt.Errorf("uv lock failed, so %s could not be written: %w\nOutput: %s", render.LockFile, err, string(output))
		}
	}
	if output, err := e.runIn(ctx, projectPath, "uv", "lock"); err != nil {
		return fmt.Errorf("uv lock failed, so %s could not be written: %w\nOutput: %s", render.LockFile, err, string(output))
	}

	e.Step("Created: " + render.LockFile + " (run 'uv sync --frozen' to reproduce this environment)")
	return nil
}
