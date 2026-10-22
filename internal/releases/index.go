package releases

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
)

const probeTimeout = 5 * time.Second

func DetectToolchain(ctx context.Context, run command.Runner) pyenv.Toolchain {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	output, err := run.Run(ctx, "", "uv", "--version")
	if err != nil || len(bytes.TrimSpace(output)) == 0 {
		return pyenv.Pip
	}
	return pyenv.UV
}

func IndexFor(toolchain pyenv.Toolchain, getenv func(string) string) string {
	variables := []string{"PIP_INDEX_URL"}
	if toolchain == pyenv.UV {
		variables = []string{"UV_DEFAULT_INDEX", "UV_INDEX_URL"}
	}
	for _, name := range variables {
		if index := strings.TrimSpace(getenv(name)); index != "" {
			return strings.TrimRight(index, "/") + "/"
		}
	}
	return pypi.DefaultIndex
}
