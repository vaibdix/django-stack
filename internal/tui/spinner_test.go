package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
)

type slowNodeRunner struct{}

func (slowNodeRunner) Run(ctx context.Context, _, name string, _ ...string) ([]byte, error) {
	if name == "pnpm" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return nil, errors.New(name + " is not available in this test")
}

func TestTheSpinnerKeepsMovingWhileNodeIsProbed(t *testing.T) {
	restore := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) { return []pyenv.Option{fakePython}, nil },
		func(pyenv.Option, time.Time) ([]releases.Option, string) { return nil, "" },
	)
	defer restore()

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Chdir(t.TempDir())

	m := questionModel(t, stepAppName)
	m.runner = slowNodeRunner{}
	m.project = config.Project{
		ProjectName: "shop", AppName: "catalog", DatabaseType: config.SQLite,
		SelectedPythonPath: fakePython.Path, SelectedPythonVersion: fakePython.Version, DjangoVersion: "6.1.2",
		CreateTemplates: true, SetupTailwind: true,
	}
	m.rebuildForm(stepAppName).Init()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 32))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		title := bytes.LastIndex(out, []byte(screenTitle(stepSetup)))
		if title < 0 {
			return false
		}
		seen := 0
		for _, frame := range spinner.Dot.Frames {
			if bytes.Contains(out[title:], []byte(frame)) {
				seen++
			}
		}
		return seen >= 3
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(50*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(10*time.Second))
}
