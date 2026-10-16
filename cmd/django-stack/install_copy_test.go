package main

import "testing"

func TestMergePathListAddsOnlyWhatIsMissing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, existing, dir, want string
		changed                   bool
	}{
		{"empty", "", `C:\Users\me\AppData\Local\DjangoStack`, `C:\Users\me\AppData\Local\DjangoStack`, true},
		{"appends", `%USERPROFILE%\bin;C:\tools`, `C:\DjangoStack`, `%USERPROFILE%\bin;C:\tools;C:\DjangoStack`, true},
		{"already there, different case", `C:\tools;c:\djangostack\`, `C:\DjangoStack`, `C:\tools;c:\djangostack\`, false},
		{"a longer path is not a match", `C:\DjangoStack\old`, `C:\DjangoStack`, `C:\DjangoStack\old;C:\DjangoStack`, true},
		{"trailing separator", `C:\tools;`, `C:\DjangoStack`, `C:\tools;C:\DjangoStack`, true},
	}
	for _, tc := range cases {
		got, changed := mergePathList(tc.existing, tc.dir, ";")
		if got != tc.want || changed != tc.changed {
			t.Errorf("%s: mergePathList(%q, %q) = (%q, %v), want (%q, %v)",
				tc.name, tc.existing, tc.dir, got, changed, tc.want, tc.changed)
		}
	}
}
