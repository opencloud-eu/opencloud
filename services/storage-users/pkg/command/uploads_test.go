package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/test-go/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	opencloudstorage "github.com/opencloud-eu/opencloud/pkg/storage"
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

func TestDeleteStaleNodesHonoursMetadataPrefix(t *testing.T) {
	root := t.TempDir()
	nodePath := filepath.Join(root, "spaces", "ab", "cdef", "nodes", "12", "34", "56", "78", "-9abc")
	require.NoError(t, os.MkdirAll(filepath.Dir(nodePath), 0700))

	b, err := msgpack.Marshal(map[string][]byte{"user.foreign.nodestatus": []byte("processing:upload-1")})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(nodePath+".mpk", b, 0600))

	// the prefix is process wide, this test fixes it for the whole package
	cfg := &config.Config{Driver: "decomposed", Drivers: config.Drivers{Decomposed: config.DecomposedDriver{Root: root, MetadataPrefix: "user.foreign."}}}
	cmd := DeleteStaleProcessingNodes(cfg)
	require.NoError(t, cmd.RunE(cmd, nil))
	require.Equal(t, 1, deleteStaleNode(root, nodePath+".mpk", true, false, nil))
}

func TestStaleNodesStorage(t *testing.T) {
	cfg := &config.Config{Drivers: config.Drivers{
		Decomposed:   config.DecomposedDriver{Root: "/local", MetadataPrefix: "user.local."},
		DecomposedS3: config.DecomposedS3Driver{Root: "/s3", MetadataPrefix: "user.s3."},
	}}
	for _, tc := range []struct{ driver, root, prefix string }{
		{"decomposed", "/local", "user.local."},
		{"decomposeds3", "/s3", "user.s3."},
		{"ocis", "/local", opencloudstorage.LegacyMetadataPrefix()},
		{"s3ng", "/s3", opencloudstorage.LegacyMetadataPrefix()},
	} {
		t.Run(tc.driver, func(t *testing.T) {
			cfg.Driver = tc.driver
			root, prefix, err := staleNodesStorage(cfg)
			require.NoError(t, err)
			require.Equal(t, tc.root, root)
			require.Equal(t, tc.prefix, prefix)
		})
	}
	cfg.Driver = "posix"
	_, _, err := staleNodesStorage(cfg)
	require.Error(t, err)
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
