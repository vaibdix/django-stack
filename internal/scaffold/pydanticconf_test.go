package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestMypyLoadsThePydanticPluginOnlyWhenTypedSettingsAreOn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		pydantic bool
		want     bool
	}{
		{name: "typed settings on", pydantic: true, want: true},
		{name: "typed settings off", pydantic: false, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			written := mypyConfigFor(t, config.Choices{
				Name:   "demo",
				Django: "6.1",
				AddOns: addOnsWhere(map[config.AddOn]bool{config.Pydantic: tc.pydantic}),
			})
			if got := strings.Contains(written, `"pydantic.mypy"`); got != tc.want {
				t.Errorf("pydantic.mypy plugin present = %v, want %v:\n%s", got, tc.want, written)
			}
			if got := strings.Contains(written, "init_forbid_extra = true"); got != tc.want {
				t.Errorf("pydantic plugin Any-suppression present = %v, want %v:\n%s", got, tc.want, written)
			}
		})
	}
}
