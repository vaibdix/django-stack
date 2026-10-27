package scaffold

import (
	"fmt"
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
	return project{cfg: spec, resolved: resolved, toolchain: tools.Toolchain, node: tools.NodeManager}.blueprint()
}

type group string

const (
	groupProject    group = "project"
	groupSettings   group = "settings"
	groupEnv        group = "env"
	groupEnvExample group = "env-example"
	groupCompose    group = "compose"
	groupGitignore  group = "gitignore"
	groupURLs       group = "urls"
	groupTemplates  group = "templates"
	groupTailwind   group = "tailwind"
	groupAllAuth    group = "allauth"
	groupSharedApps group = "shared-apps"
	groupApp        group = "app"
	groupAPI        group = "api"
	groupReadme     group = "readme"
	groupWorkspace  group = "workspace"
	groupScripts    group = "scripts"
)

func groups() []group {
	return []group{
		groupProject, groupSettings, groupEnv, groupEnvExample, groupCompose, groupGitignore, groupURLs, groupTemplates,
		groupTailwind, groupAllAuth, groupSharedApps, groupApp, groupAPI, groupReadme, groupWorkspace, groupScripts,
	}
}

func (p project) files(g group) (fileSet, error) {
	switch g {
	case groupProject:
		return p.projectFiles(), nil
	case groupSettings:
		settings, err := p.settingsFiles()
		return union(p.confFiles(), settings, p.typecheckFiles()), err
	case groupEnv:
		return p.envFiles()
	case groupEnvExample:
		return union(p.envExampleFiles(), p.deployedEnvFiles()), nil
	case groupCompose:
		return p.composeFiles()
	case groupGitignore:
		return p.gitignoreFiles(), nil
	case groupURLs:
		return p.urlFiles()
	case groupTemplates:
		return p.templateFiles(), nil
	case groupTailwind:
		return p.tailwindFiles(), nil
	case groupAllAuth:
		return p.allAuthFiles(), nil
	case groupSharedApps:
		return p.sharedAppFiles(), nil
	case groupApp:
		return p.appFiles(), nil
	case groupAPI:
		return p.apiFiles(), nil
	case groupReadme:
		return p.readmeFiles(), nil
	case groupWorkspace:
		return p.workspaceFiles()
	case groupScripts:
		return p.scriptFiles(), nil
	default:
		return nil, fmt.Errorf("no file group %q", g)
	}
}

func union(sets ...fileSet) fileSet {
	all := fileSet{}
	for _, set := range sets {
		maps.Copy(all, set)
	}
	return all
}

func (p project) blueprint() (blueprint.Blueprint, error) {
	files := fileSet{}
	for _, g := range groups() {
		set, err := p.files(g)
		if err != nil {
			return blueprint.Blueprint{}, err
		}
		for rel, file := range set {
			if _, twice := files[rel]; twice {
				return blueprint.Blueprint{}, fmt.Errorf("%s is written by two file groups", rel)
			}
			files[rel] = file
		}
	}

	commands := make([]blueprint.Command, 0, len(p.verificationChecks()))
	for _, check := range p.verificationChecks() {
		commands = append(commands, blueprint.Command{Name: check.name, Args: check.args})
	}
	return blueprint.Blueprint{Files: files, Commands: commands}, nil
}

func (e *Env) emitGroup(g group) error {
	files, err := e.files(g)
	if err != nil {
		return err
	}
	return e.emit(files)
}

func (e *Env) emit(files fileSet) error {
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if err := e.write(filepath.FromSlash(rel), files[rel].Content, files[rel].Mode); err != nil {
			return err
		}
	}
	return nil
}
