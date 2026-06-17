package store

import (
	"fmt"
	"strconv"
	"strings"
)

func ParsePrefix(name string) (int, string, bool) {
	idx := strings.IndexByte(name, '-')
	if idx < 3 {
		return 0, "", false
	}
	prefixStr := name[:idx]
	if len(prefixStr) < 3 {
		return 0, "", false
	}
	n, err := strconv.Atoi(prefixStr)
	if err != nil {
		return 0, "", false
	}
	return n, name[idx+1:], true
}

func FormatPrefix(prefix int, slug string) string {
	return fmt.Sprintf("%03d-%s", prefix, slug)
}

func PrefixBetween(a, b int) (int, bool) {
	if a > b {
		a, b = b, a
	}
	if b-a < 2 {
		return 0, false
	}
	return a + (b-a)/2, true
}

func RenumberGapOfTen(count int) []int {
	out := make([]int, count)
	for i := 0; i < count; i++ {
		out[i] = (i + 1) * 10
	}
	return out
}
