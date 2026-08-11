package clierr_test

import (
	"bytes"
	stderrors "errors"
	"strings"
	"testing"

	clierrors "github.com/vaibdix/django-cli/internal/clierr"
)

func TestCLIErrorErrorIncludesCodeAndMessage(t *testing.T) {
	t.Parallel()

	err := clierrors.NewCLIError("E999", "something broke", "try again")

	if got, want := err.Error(), "E999: something broke"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestCLIErrorSatisfiesErrorsIs(t *testing.T) {
	t.Parallel()

	wrapped := stderrors.Join(clierrors.ErrProjectExists, stderrors.New("context"))

	if !stderrors.Is(wrapped, clierrors.ErrProjectExists) {
		t.Fatal("errors.Is could not unwrap to ErrProjectExists; sentinel errors must stay comparable")
	}
}

func TestWithHintReturnsCopyAndLeavesReceiverUnchanged(t *testing.T) {
	t.Parallel()

	original := clierrors.ErrPythonNotFound
	originalHint := original.Hint

	derived := original.WithHint("install python 3.12")

	if derived.Hint != "install python 3.12" {
		t.Errorf("derived.Hint = %q, want the new hint", derived.Hint)
	}
	if original.Hint != originalHint {
		t.Errorf("WithHint mutated the shared sentinel: Hint = %q, want %q", original.Hint, originalHint)
	}
	if derived == original {
		t.Error("WithHint returned the same pointer; mutating a package-level sentinel would leak across callers")
	}
	if derived.Code != original.Code || derived.Message != original.Message {
		t.Errorf("WithHint changed code/message: got %q/%q, want %q/%q",
			derived.Code, derived.Message, original.Code, original.Message)
	}
}

func TestDerivedErrorsStillMatchTheirSentinel(t *testing.T) {
	t.Parallel()

	withHint := clierrors.ErrVenvCreationFailed.WithHint("custom")
	if !stderrors.Is(withHint, clierrors.ErrVenvCreationFailed) {
		t.Error("WithHint result no longer matches its sentinel")
	}

	wrapped := clierrors.ErrVenvCreationFailed.Wrap(stderrors.New("exit status 1"))
	if !stderrors.Is(wrapped, clierrors.ErrVenvCreationFailed) {
		t.Error("Wrap result no longer matches its sentinel")
	}
	if !strings.Contains(wrapped.Error(), "exit status 1") {
		t.Errorf("Wrap lost the cause: %q", wrapped.Error())
	}

	if stderrors.Is(wrapped, clierrors.ErrPythonNotFound) {
		t.Error("a wrapped ErrVenvCreationFailed matched an unrelated sentinel")
	}
}

func TestFprintWritesMessageAndHintToTheGivenWriter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	clierrors.ErrProjectExists.Fprint(&buf)

	out := buf.String()
	if !strings.Contains(out, clierrors.ErrProjectExists.Message) {
		t.Errorf("output %q is missing the message", out)
	}
	if !strings.Contains(out, clierrors.ErrProjectExists.Hint) {
		t.Errorf("output %q is missing the hint", out)
	}
}

func TestFprintOmitsHintLineWhenThereIsNoHint(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	clierrors.NewCLIError("E900", "something failed", "").Fprint(&buf)

	if strings.Contains(buf.String(), "Hint:") {
		t.Errorf("output %q includes a Hint line for an error with no hint", buf.String())
	}
}

func TestFprintWithContextPrefixesTheContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	clierrors.ErrGitInitFailed.FprintWithContext(&buf, "while scaffolding")

	if !strings.Contains(buf.String(), "while scaffolding") {
		t.Errorf("output %q is missing the context label", buf.String())
	}
}

func TestSentinelErrorCodesAreUniqueAndPopulated(t *testing.T) {
	t.Parallel()

	sentinels := []*clierrors.CLIError{
		clierrors.ErrProjectExists,
		clierrors.ErrInvalidProjectName,
		clierrors.ErrReservedWord,
		clierrors.ErrInvalidDjangoVersion,
		clierrors.ErrPythonNotFound,
		clierrors.ErrVenvCreationFailed,
		clierrors.ErrDjangoInstallFailed,
		clierrors.ErrProjectCreationFailed,
		clierrors.ErrAppCreationFailed,
		clierrors.ErrGitInitFailed,
		clierrors.ErrTailwindSetupFailed,
		clierrors.ErrConfigReadFailed,
		clierrors.ErrConfigWriteFailed,
		clierrors.ErrUnknownConfigKey,
	}

	seen := make(map[string]string, len(sentinels))
	for _, s := range sentinels {
		if s.Code == "" {
			t.Errorf("sentinel %q has an empty code", s.Message)
		}
		if s.Message == "" {
			t.Errorf("sentinel %q has an empty message", s.Code)
		}
		if s.Hint == "" {
			t.Errorf("sentinel %q has no hint; hints are the whole point of CLIError", s.Code)
		}
		if strings.HasSuffix(s.Message, ".") {
			t.Errorf("sentinel %q message %q ends in a period; error strings are fragments, not sentences", s.Code, s.Message)
		}
		if prev, dup := seen[s.Code]; dup {
			t.Errorf("duplicate error code %q used by both %q and %q", s.Code, prev, s.Message)
		}
		seen[s.Code] = s.Message
	}
}
