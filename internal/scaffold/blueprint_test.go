package scaffold

import (
	"context"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type refusingRunner struct{ t *testing.T }

func (r refusingRunner) Run(_ context.Context, _, name string, args ...string) ([]byte, error) {
	r.t.Errorf("building the blueprint ran %s %v", name, args)
	return nil, nil
}

func everyAddOnBut(skip config.AddOn) config.Set[config.AddOn] {
	set := config.NewSet[config.AddOn]()
	for _, addOn := range config.AddOns() {
		if addOn != skip {
			set = set.With(addOn)
		}
	}
	return set
}

func TestTheBlueprintBuildsForEveryStackWithoutDiskOrSubprocess(t *testing.T) {
	t.Parallel()

	addOnSets := map[string]config.Set[config.AddOn]{
		"none":           config.NewSet[config.AddOn](),
		"all with drf":   everyAddOnBut(config.Ninja),
		"all with ninja": everyAddOnBut(config.RestFramework),
	}
	for _, stack := range config.AllStacks() {
		for _, db := range config.Databases() {
			for name, addOns := range addOnSets {
				for _, app := range []string{"", "library"} {
					t.Run(string(stack)+"/"+string(db)+"/"+name+"/"+app, func(t *testing.T) {
						t.Parallel()

						resolved := credentialsFor(db, config.DatabaseCredentials{})
						resolved.PythonVersion = "Python 3.13.2"
						resolved.SecretKey = "a-test-secret-key"
						e := &Env{
							cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: db, Stack: stack, AddOns: addOns, App: app},
							resolved:  resolved,
							toolchain: pyenv.UV,
							node:      NPM,
							runner:    refusingRunner{t},
						}
						bp, err := e.blueprint()
						if err != nil {
							t.Fatalf("blueprint: %v", err)
						}
						for _, want := range []string{".env", ".env.example", ".gitignore", "README.md", "my_shop/urls.py", "accounts/models.py", ".vscode/tasks.json"} {
							if _, ok := bp.Files[want]; !ok {
								t.Errorf("the blueprint has no %s", want)
							}
						}
						for path, file := range bp.Files {
							if len(file.Content) == 0 && !strings.HasSuffix(path, "__init__.py") {
								t.Errorf("%s is empty", path)
							}
							if file.Mode == 0 {
								t.Errorf("%s has no mode", path)
							}
						}
						if !strings.Contains(string(bp.Files[".env"].Content), "SECRET_KEY=a-test-secret-key") {
							t.Errorf(".env does not carry the resolved secret key")
						}
						if len(bp.Commands) < 2 || bp.Commands[0].Name != "system check" {
							t.Errorf("commands = %v, want the system check first", bp.Commands)
						}
					})
				}
			}
		}
	}
}
