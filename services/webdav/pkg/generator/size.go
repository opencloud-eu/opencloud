package generator

import (
	"fmt"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
)

// ParseMaxInputFileSize parses a size string like "50MB" into bytes, the same
// way the thumbnails service did before the workflow moved here. An empty
// string means no limit, and a negative value is refused because bytesize
// reads it as a very large unsigned one.
func ParseMaxInputFileSize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("invalid file size: %s", s)
	}

	size, err := bytesize.Parse(s)
	if err != nil {
		return 0, fmt.Errorf("invalid file size: %w", err)
	}

	return size.Bytes(), nil
}
