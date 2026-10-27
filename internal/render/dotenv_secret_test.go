package render

import (
	"crypto/rand"
	"strings"
	"testing"
)

const urlSafeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func TestGeneratedSecretKeysSurviveADotEnvFileAndTheDeployCheck(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for range 500 {
		key, err := GenerateSecretKey(rand.Reader)
		if err != nil {
			t.Fatalf("generateSecretKey: %v", err)
		}
		if len(key) < 50 {
			t.Fatalf("Django's deploy check wants at least 50 characters, got %d: %q", len(key), key)
		}
		if strings.Trim(key, urlSafeAlphabet) != "" {
			t.Fatalf("%q has characters outside the URL-safe alphabet, so it cannot sit unquoted in .env", key)
		}
		seen[key] = true
	}
	if len(seen) < 500 {
		t.Errorf("500 keys had only %d distinct values", len(seen))
	}
}
