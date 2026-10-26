package scaffold

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"

	"github.com/vaibdix/django-stack/internal/blueprint"
	"github.com/vaibdix/django-stack/internal/config"
)

type fileSet map[string]blueprint.File

func (s fileSet) add(rel, content string, mode fs.FileMode) {
	s[filepath.ToSlash(rel)] = blueprint.File{Content: []byte(content), Mode: mode}
}

func (s fileSet) addAll(dir string, files map[string]string) {
	for name, content := range files {
		s.add(filepath.Join(dir, name), content, fileMode)
	}
}

func Build(spec config.Spec, resolved config.Resolved, tools Tools) (blueprint.Blueprint, error) {
	return project{cfg: spec.Choices(), resolved: resolved, toolchain: tools.Toolchain, node: tools.NodeManager}.blueprint()
}

func (p project) blueprint() (blueprint.Blueprint, error) {
	env, err := p.envFiles()
	if err != nil {
		return blueprint.Blueprint{}, err
	}
	compose, err := p.composeFiles()
	if err != nil {
		return blueprint.Blueprint{}, err
	}
	workspace, err := p.workspaceFiles()
	if err != nil {
		return blueprint.Blueprint{}, err
	}
	files := fileSet{}
	for _, set := range []fileSet{
		p.projectFiles(),
		p.settingsFiles(),
		env,
		p.envExampleFiles(),
		compose,
		p.gitignoreFiles(),
		p.urlFiles(),
		p.confFiles(),
		p.templateFiles(),
		p.tailwindFiles(),
		p.allAuthFiles(),
		p.sharedAppFiles(),
		p.appFiles(),
		p.apiFiles(),
		p.typecheckFiles(),
		p.deployedEnvFiles(),
		p.readmeFiles(),
		workspace,
		p.scriptFiles(),
	} {
		maps.Copy(files, set)
	}

	commands := make([]blueprint.Command, 0, len(p.verificationChecks()))
	for _, check := range p.verificationChecks() {
		commands = append(commands, blueprint.Command{Name: check.name, Args: check.args})
	}
	return blueprint.Blueprint{Files: files, Commands: commands}, nil
}

func (e *Env) emit(files fileSet) error {
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if err := e.write(filepath.FromSlash(rel), files[rel].Content, files[rel].Mode); err != nil {
			return err
		}
	}
	return nil
}
