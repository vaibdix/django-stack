package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

func TestAStackTheMachineCannotBuildIsGreyedAndRefused(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepStack)
	m.applyHost(scaffold.Host{"npm|pnpm": errors.New("needs npm or pnpm"), "node>=20": nil, "git": nil})
	m.stackForm = buildStackForm(&m.choices.Stack, m.host)
	m.syncFormWidths()
	m.stackForm.Init()

	if view := m.stackForm.View(); !strings.Contains(view, "needs npm or pnpm") {
		t.Errorf("the Tailwind stacks are not marked unavailable:\n%s", view)
	}
	if err := availableAddOns(m.host)([]config.AddOn{config.Git}); err != nil {
		t.Errorf("git is installed, but the add-on was refused: %v", err)
	}
	refused := availableAddOns(scaffold.Host{"git": errors.New("needs git")})([]config.AddOn{config.PreCommit})
	if refused == nil || !strings.Contains(refused.Error(), "Git") {
		t.Errorf("pre-commit adds git, so it must be refused without git: %v", refused)
	}
	if err := availableAddOns(scaffold.Host{"git": errors.New("needs git")})([]config.AddOn{config.Git}); err == nil {
		t.Error("the git add-on was accepted on a machine without git")
	}
}
