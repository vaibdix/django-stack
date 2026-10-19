package version

import "testing"

func TestReleaseVersionsOrderNumerically(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left, right string
		want        int
	}{
		{"6.1.2", "6.1.10", -1},
		{"3.12", "3.12.0", 0},
		{"1.44", "1.43", 1},
		{"10.0", "9.9.9", 1},
	}
	for _, tc := range cases {
		l, err := Parse(tc.left)
		if err != nil {
			t.Fatal(err)
		}
		r, err := Parse(tc.right)
		if err != nil {
			t.Fatal(err)
		}
		if got := Compare(l, r); got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestOnlyReleaseVersionsParse(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{"6.2a1", "", "3..12", "1.44-beta", "-1.0"} {
		if parts, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) = %v, want an error", bad, parts)
		}
	}
}

func TestHasPrefixPadsWithZeros(t *testing.T) {
	t.Parallel()

	if !HasPrefix([]int{3, 11, 4}, []int{3, 11}) || HasPrefix([]int{3, 12}, []int{3, 11}) || !HasPrefix([]int{3}, []int{3, 0}) {
		t.Error("HasPrefix does not match release prefixes as PEP 440 wildcards do")
	}
}
