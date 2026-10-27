package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestAFailedTailwindInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith("pnpm add --ignore-scripts --save-exact tailwindcss@"+render.TailwindVersion()+" @tailwindcss/cli@"+render.TailwindVersion(), "ERR_PNPM_FETCH_404")
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo", Stack: config.StackTailwind}), reporter: report.Nop{}, Node: render.Pnpm, ports: port.Ports{Run: fake}}

	e.dir = root
	err := e.applyTailwind(t.Context())
	if err == nil || !strings.Contains(err.Error(), "install Tailwind CSS") || !strings.Contains(err.Error(), "ERR_PNPM_FETCH_404") {
		t.Fatalf("applyTailwind = %v, want the install failure and its cause", err)
	}
}

func TestTailwindRunsNodeWithTheManagerValidationFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo", Stack: config.StackTailwind}), reporter: report.Nop{}, Node: render.NPM, ports: port.Ports{Run: fake}}

	e.dir = root
	if err := e.applyTailwind(t.Context()); err != nil {
		t.Fatalf("applyTailwind: %v", err)
	}
	if err := e.buildTailwindCSS(t.Context()); err != nil {
		t.Fatalf("buildTailwindCSS: %v", err)
	}
	for _, want := range []string{"npm install --ignore-scripts --save-exact tailwindcss@" + render.TailwindVersion() + " @tailwindcss/cli@" + render.TailwindVersion(), "npm run build:css"} {
		if !fake.Ran(want) {
			t.Errorf("%q never ran; calls: %v", want, fake.CommandLines())
		}
	}
}
