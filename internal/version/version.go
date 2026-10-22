package version

import (
	"fmt"
	"strconv"
	"strings"
)

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
