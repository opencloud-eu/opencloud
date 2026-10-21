package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/test-go/testify/require"
)

func TestIsPosixStorage(t *testing.T) {
	tests := []struct {
		name       string
		dirs       []string
		configured string
		want       bool
	}{
		{"decomposed layout, driver not configured", []string{"spaces/ab/cdef/nodes"}, "posix", false},
		{"decomposed layout, driver configured", []string{"spaces/ab/cdef/nodes"}, "decomposed", false},
		{"posix layout, driver not configured", []string{"users/alice/.oc-nodes"}, "decomposed", true},
		{"posix layout, driver configured", []string{"projects/space/.oc-nodes"}, "posix", true},
		{"empty storage falls back to posix", nil, "posix", true},
		{"empty storage falls back to decomposed", nil, "decomposed", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, d := range tt.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(root, d), os.ModePerm))
			}
			require.Equal(t, tt.want, isPosixStorage(root, tt.configured))
		})
	}
}
