package generator

import (
	"fmt"
	"strconv"
	"strings"
)

type sizeUnit struct {
	suffix string
	len    int
	mul    uint64
}

var sizeUnits = []sizeUnit{
	{"GIB", 3, 1 << 30}, {"GB", 2, 1 << 30},
	{"MIB", 3, 1 << 20}, {"MB", 2, 1 << 20},
	{"KIB", 3, 1 << 10}, {"KB", 2, 1 << 10},
}

// ParseMaxInputFileSize parses a size string like "50MB" into bytes.
// Empty string returns 0 (no limit).
func ParseMaxInputFileSize(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.TrimSpace(s)
	var unitMultiplier uint64 = 1
	upper := strings.ToUpper(s)

	for _, u := range sizeUnits {
		if strings.HasSuffix(upper, u.suffix) {
			unitMultiplier = u.mul
			s = s[:len(s)-u.len]
			break
		}
	}

	size, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid file size: %w", err)
	}

	return size * unitMultiplier, nil
}
