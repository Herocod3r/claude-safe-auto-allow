package version

import (
	"strconv"
	"strings"
)

// Gt reports whether semver a is greater than semver b.
func Gt(a, b string) bool {
	pa, ok := parse(a)
	if !ok {
		return false
	}
	pb, ok := parse(b)
	if !ok {
		return false
	}

	for i := range pa {
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}

	return false
}

func parse(s string) ([3]int, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}

	var out [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}

	return out, true
}
