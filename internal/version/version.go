// Package version reads and orders dotted release versions such as "6.1.2",
// "3.13" or a Docker engine API version "1.44". Pre-releases and local
// segments are not release versions and are rejected.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse reads a release version into its numeric components.
func Parse(v string) ([]int, error) {
	fields := strings.Split(strings.TrimSpace(v), ".")
	parts := make([]int, len(fields))
	for i, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("%q is not a release version", v)
		}
		parts[i] = n
	}
	return parts, nil
}

// Compare orders two release versions, padding the shorter with zeros, so
// 3.12 and 3.12.0 are equal. It returns -1, 0 or 1.
func Compare(left, right []int) int {
	for i := 0; i < len(left) || i < len(right); i++ {
		l, r := at(left, i), at(right, i)
		switch {
		case l < r:
			return -1
		case l > r:
			return 1
		}
	}
	return 0
}

// HasPrefix reports whether v starts with prefix, padding v with zeros, which
// is how a "3.11.*" wildcard matches.
func HasPrefix(v, prefix []int) bool {
	for i, part := range prefix {
		if at(v, i) != part {
			return false
		}
	}
	return true
}

func at(parts []int, i int) int {
	if i < len(parts) {
		return parts[i]
	}
	return 0
}
