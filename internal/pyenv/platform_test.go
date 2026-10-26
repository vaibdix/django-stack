package pyenv

import "testing"

func TestPlatformPathsForEachOS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos, bin, exe, script, rel, command, finder string
	}{
		{"linux", "bin", "", ".sh", ".venv/bin/python", "./.venv/bin/python", "which"},
		{"darwin", "bin", "", ".sh", ".venv/bin/python", "./.venv/bin/python", "which"},
		{Windows, "Scripts", ".exe", ".bat", `.venv\Scripts\python.exe`, `.venv\Scripts\python.exe`, "where"},
	}
	for _, c := range cases {
		finder, _ := pathFinder(c.goos)
		got := []string{VenvBin(c.goos), ExeSuffix(c.goos), ScriptExt(c.goos), InterpreterRel(c.goos), InterpreterCommand(c.goos), finder}
		want := []string{c.bin, c.exe, c.script, c.rel, c.command, c.finder}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %q, want %q", c.goos, got, want)
				break
			}
		}
		if len(Host{GOOS: c.goos, Getenv: func(string) string { return "" }}.knownPythonPaths()) == 0 && !IsWindows(c.goos) {
			t.Errorf("%s: no known interpreter locations", c.goos)
		}
	}
}
