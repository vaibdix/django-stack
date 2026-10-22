package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func TestAFailedTailwindInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith("pnpm add --ignore-scripts --save-exact tailwindcss@"+TailwindVersion+" @tailwindcss/cli@"+TailwindVersion, "ERR_PNPM_FETCH_404")
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: PNPM}

	err := e.applyTailwind(root)
	if err == nil || !strings.Contains(err.Error(), "install Tailwind CSS") || !strings.Contains(err.Error(), "ERR_PNPM_FETCH_404") {
		t.Fatalf("applyTailwind = %v, want the install failure and its cause", err)
	}
}

func TestTailwindRunsNodeWithTheManagerValidationFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: NPM}

	if err := e.applyTailwind(root); err != nil {
		t.Fatalf("applyTailwind: %v", err)
	}
	if err := e.buildTailwindCSS(root); err != nil {
		t.Fatalf("buildTailwindCSS: %v", err)
	}
	for _, want := range []string{"npm install --ignore-scripts --save-exact tailwindcss@" + TailwindVersion + " @tailwindcss/cli@" + TailwindVersion, "npm run build:css"} {
		if !fake.Ran(want) {
			t.Errorf("%q never ran; calls: %v", want, fake.CommandLines())
		}
	}
}
