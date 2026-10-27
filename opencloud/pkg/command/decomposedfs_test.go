package command

import (
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/test-go/testify/require"
)

func TestBackend(t *testing.T) {
	tests := []struct {
		name string
		bod  string
		want metadata.Backend
	}{
		{
			name: "selects messagepack",
			bod:  "mpk",
			want: metadata.MessagePackBackend{},
		},
		{
			name: "xattrs is no longer supported",
			bod:  "xattrs",
			want: metadata.NullBackend{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.IsType(t, tt.want, backend(tt.bod))
		})
	}
}
