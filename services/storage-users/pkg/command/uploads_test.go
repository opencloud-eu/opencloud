package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/test-go/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
)

func TestBuildInfo(t *testing.T) {
	testCases := []struct {
		alias        string
		filter       storage.UploadSessionFilter
		expectedInfo string
	}{
		{
			alias:        "empty filter",
			filter:       storage.UploadSessionFilter{},
			expectedInfo: "Sessions:",
		},
		{
			alias:        "processing",
			filter:       storage.UploadSessionFilter{Processing: boolPtr(true)},
			expectedInfo: "Processing sessions:",
		},
		{
			alias:        "processing and not expired",
			filter:       storage.UploadSessionFilter{Processing: boolPtr(true), Expired: boolPtr(false)},
			expectedInfo: "Processing, not expired sessions:",
		},
		{
			alias:        "processing and expired",
			filter:       storage.UploadSessionFilter{Processing: boolPtr(true), Expired: boolPtr(true)},
			expectedInfo: "Processing, expired sessions:",
		},
		{
			alias:        "with id",
			filter:       storage.UploadSessionFilter{ID: strPtr("123")},
			expectedInfo: "Session with id '123':",
		},
		{
			alias:        "processing, not expired and not virus infected",
			filter:       storage.UploadSessionFilter{Processing: boolPtr(true), Expired: boolPtr(false), HasVirus: boolPtr(false)},
			expectedInfo: "Processing, not expired, not virusinfected sessions:",
		},
		{
			alias:        "not virusinfected",
			filter:       storage.UploadSessionFilter{HasVirus: boolPtr(false)},
			expectedInfo: "Not virusinfected sessions:",
		},
		{
			alias:        "expired and virusinfected",
			filter:       storage.UploadSessionFilter{Expired: boolPtr(true), HasVirus: boolPtr(true)},
			expectedInfo: "Expired, virusinfected sessions:",
		},
		{
			alias:        "expired and not virus infected",
			filter:       storage.UploadSessionFilter{Expired: boolPtr(true), HasVirus: boolPtr(false)},
			expectedInfo: "Expired, not virusinfected sessions:",
		},
		{
			alias:        "processing, not expired, virus infected and with id (note: this makes no sense)",
			filter:       storage.UploadSessionFilter{Processing: boolPtr(true), Expired: boolPtr(false), HasVirus: boolPtr(true), ID: strPtr("123")},
			expectedInfo: "Processing, not expired, virusinfected session with id '123':",
		},
	}

	for _, tc := range testCases {
		alias := tc.alias
		filter := tc.filter
		expectedInfo := tc.expectedInfo

		t.Run(alias, func(t *testing.T) {
			require.Equal(t, expectedInfo, buildInfo(filter))
		})
	}
}

func TestDeleteStaleNodeMetadataPrefix(t *testing.T) {
	testCases := []struct {
		alias   string
		prefix  string
		diskKey string
	}{
		{alias: "native prefix", prefix: "", diskKey: "user.oc.nodestatus"},
		{alias: "foreign prefix", prefix: "user.foreign.", diskKey: "user.foreign.nodestatus"},
	}

	for _, tc := range testCases {
		t.Run(tc.alias, func(t *testing.T) {
			root := t.TempDir()
			nodePath := filepath.Join(root, "spaces", "ab", "cdef", "nodes", "12", "34", "56", "78", "-9abc")
			require.NoError(t, os.MkdirAll(filepath.Dir(nodePath), 0700))

			b, err := msgpack.Marshal(map[string][]byte{tc.diskKey: []byte("processing:upload-1")})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(nodePath+".mpk", b, 0600))

			cfg := &config.Config{Drivers: config.Drivers{Decomposed: config.DecomposedDriver{Root: root, MetadataPrefix: tc.prefix}}}
			require.Equal(t, 1, deleteStaleNode(cfg, nodePath+".mpk", true, false, nil))
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
