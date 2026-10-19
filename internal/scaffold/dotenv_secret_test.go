package scaffold

import (
	"strings"
	"testing"
)

func TestGeneratedSecretKeysSurviveADotEnvFile(t *testing.T) {
	t.Parallel()

	for range 500 {
		key := generateSecretKey()
		if len(key) < 50 {
			t.Fatalf("Django's deploy check wants at least 50 characters, got %d: %q", len(key), key)
		}
		// "$" makes django-environ treat the value as a reference to another
		// variable; "#" can start a comment; quotes and spaces need quoting.
		if strings.ContainsAny(key, "$#'\" \\") {
			t.Fatalf("%q cannot be written unquoted into .env and read back unchanged", key)
		}
	}
}
