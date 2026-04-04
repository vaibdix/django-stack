package ui

import (
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type formattingCommand struct {
	Stage  string
	Module string
	Args   []string
}

type scaffoldFormattingPlan struct {
	Packages      []string
	Commands      []formattingCommand
	TemplateFiles []string
}

var installFormattingPackages = func(m *Model, projectPath string, packages ...string) error {
	return m.installAndReport(projectPath, "scaffold formatters (ruff + djlint)", packages...)
}

var runPythonModuleCommand = func(projectPath, module string, args ...string) ([]byte, error) {
	pythonPath := getPythonPath(projectPath)
	commandArgs := append([]string{"-m", module}, args...)
	cmd := exec.Command(pythonPath, commandArgs...)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

func scaffoldFormattingPackages() []string {
	return []string{PkgRuff, PkgDjLint}
}

func buildScaffoldFormattingPlan(projectPath string) (scaffoldFormattingPlan, error) {
	templateFiles, err := findTemplateFilesForFormatting(projectPath)
	if err != nil {
		return scaffoldFormattingPlan{}, err
	}

	return scaffoldFormattingPlan{
		Packages: scaffoldFormattingPackages(),
		Commands: []formattingCommand{
			{
				Stage:  "ruff autofix",
				Module: "ruff",
				Args:   []string{"check", "--fix", "."},
			},
			{
				Stage:  "ruff format",
				Module: "ruff",
				Args:   []string{"format", "."},
			},
		},
		TemplateFiles: templateFiles,
	}, nil
}

func findTemplateFilesForFormatting(projectPath string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(projectPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case VenvDir, ".git", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		templateSegment := string(filepath.Separator) + TemplatesDir + string(filepath.Separator)
		if strings.Contains(path, templateSegment) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func (m *Model) RunMandatoryScaffoldFormatting(projectPath string) error {
	m.UpdateProgress("Formatting generated code and templates...")

	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		return fmt.Errorf("failed to discover generated templates for formatting: %w", err)
	}

	if err := installFormattingPackages(m, projectPath, plan.Packages...); err != nil {
		return err
	}

	for _, command := range plan.Commands {
		output, err := runPythonModuleCommand(projectPath, command.Module, command.Args...)
		if err != nil {
			return fmt.Errorf("failed during %s: %w\nOutput: %s", command.Stage, err, string(output))
		}
	}

	if len(plan.TemplateFiles) == 0 {
		m.AppendStepMessage("No HTML templates found for djlint reformat.")
		m.AppendStepMessage("Generated code and templates formatted.")
		return nil
	}

	templateArgs := append([]string{"--reformat", "--profile=django"}, plan.TemplateFiles...)
	output, err := runPythonModuleCommand(projectPath, "djlint", templateArgs...)
	if err != nil {
		return fmt.Errorf("failed during template formatting: %w\nOutput: %s", err, string(output))
	}

	m.AppendStepMessage("Generated code and templates formatted.")
	return nil
}
