package scaffold

import (
	"context"
	"strings"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) commandVersion(ctx context.Context, name string) string {
	output, err := pyenv.ProbeVersion(ctx, e.commandRunner(), name)
	if err != nil {
		return ""
	}
	for field := range strings.SplitSeq(strings.TrimSpace(output), " ") {
		if len(field) > 0 && field[0] >= '0' && field[0] <= '9' {
			return field
		}
	}
	return ""
}
