package render

import (
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/blueprint"
	"github.com/vaibdix/django-stack/internal/config"
)

type FileSet map[string]blueprint.File

func (s FileSet) add(rel, content string, mode fs.FileMode) {
	s[filepath.ToSlash(rel)] = blueprint.File{Content: []byte(content), Mode: mode}
}

func (s FileSet) addAll(dir string, files map[string]string) {
	for name, content := range files {
		s.add(filepath.Join(dir, name), content, FileMode)
	}
}

func Build(spec config.Spec, resolved config.Resolved, tools Tools) (blueprint.Blueprint, error) {
	return Project{Cfg: spec, Resolved: resolved, Toolchain: tools.Toolchain, Node: tools.NodeManager}.blueprint()
}

type Group string

const (
	GroupProject    Group = "project"
	GroupSettings   Group = "settings"
	GroupEnv        Group = "env"
	GroupEnvExample Group = "env-example"
	GroupCompose    Group = "compose"
	GroupGitignore  Group = "gitignore"
	GroupURLs       Group = "urls"
	GroupTemplates  Group = "templates"
	GroupTailwind   Group = "tailwind"
	GroupAllAuth    Group = "allauth"
	GroupSharedApps Group = "shared-apps"
	GroupApp        Group = "app"
	GroupAPI        Group = "api"
	GroupReadme     Group = "readme"
	GroupWorkspace  Group = "workspace"
	GroupScripts    Group = "scripts"
)

func groups() []Group {
	return []Group{
		GroupProject, GroupSettings, GroupEnv, GroupEnvExample, GroupCompose, GroupGitignore, GroupURLs, GroupTemplates,
		GroupTailwind, GroupAllAuth, GroupSharedApps, GroupApp, GroupAPI, GroupReadme, GroupWorkspace, GroupScripts,
	}
}

func (p Project) Files(g Group) (FileSet, error) {
	switch g {
	case GroupProject:
		return p.projectFiles(), nil
	case GroupSettings:
		settings, err := p.SettingsFiles()
		return union(p.ConfFiles(), settings, p.typecheckFiles()), err
	case GroupEnv:
		return p.envFiles()
	case GroupEnvExample:
		return union(p.envExampleFiles(), p.deployedEnvFiles()), nil
	case GroupCompose:
		return p.composeFiles()
	case GroupGitignore:
		return p.gitignoreFiles(), nil
	case GroupURLs:
		return p.urlFiles()
	case GroupTemplates:
		return p.templateFiles(), nil
	case GroupTailwind:
		return p.tailwindFiles(), nil
	case GroupAllAuth:
		return p.allAuthFiles(), nil
	case GroupSharedApps:
		return p.sharedAppFiles(), nil
	case GroupApp:
		return p.appFiles(), nil
	case GroupAPI:
		return p.apiFiles(), nil
	case GroupReadme:
		return p.readmeFiles(), nil
	case GroupWorkspace:
		return p.workspaceFiles()
	case GroupScripts:
		return p.scriptFiles(), nil
	default:
		return nil, fmt.Errorf("no file group %q", g)
	}
}

func union(sets ...FileSet) FileSet {
	all := FileSet{}
	for _, set := range sets {
		maps.Copy(all, set)
	}
	return all
}

func (p Project) blueprint() (blueprint.Blueprint, error) {
	files := FileSet{}
	for _, g := range groups() {
		set, err := p.Files(g)
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

	commands := make([]blueprint.Command, 0, len(p.VerificationChecks()))
	for _, check := range p.VerificationChecks() {
		commands = append(commands, blueprint.Command{Name: check.Name, Args: check.Args})
	}
	return blueprint.Blueprint{Files: files, Commands: commands}, nil
}
