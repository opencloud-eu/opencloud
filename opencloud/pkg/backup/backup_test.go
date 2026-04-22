package backup_test

import (
	"testing"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/backup"
	"github.com/test-go/testify/require"
)

func TestGatherData(t *testing.T) {
	testcases := []struct {
		Name     string
		Events   []any
		Expected *backup.Consistency
	}{
		{
			Name: "no symlinks - no blobs",
			Events: []any{
				nodeData("nodepath", "blobpath", true),
			},
			Expected: consistency(func(c *backup.Consistency) {
				node(c, "nodepath", backup.InconsistencySymlinkMissing)
				blobReference(c, "blobpath", backup.InconsistencyBlobMissing)
			}),
		},
		{
			Name: "symlink not required - no blobs",
			Events: []any{
				nodeData("nodepath", "blobpath", false),
			},
			Expected: consistency(func(c *backup.Consistency) {
				blobReference(c, "blobpath", backup.InconsistencyBlobMissing)
			}),
		},
		{
			Name: "no inconsistencies",
			Events: []any{
				nodeData("nodepath", "blobpath", true),
				linkData("linkpath", "nodepath"),
				blobData("blobpath"),
			},
			Expected: consistency(func(c *backup.Consistency) {
			}),
		},
		{
			Name: "orphaned blob",
			Events: []any{
				nodeData("nodepath", "blobpath", true),
				linkData("linkpath", "nodepath"),
				blobData("blobpath"),
				blobData("anotherpath"),
			},
			Expected: consistency(func(c *backup.Consistency) {
				blob(c, "anotherpath", backup.InconsistencyBlobOrphaned)
			}),
		},
		{
			Name: "missing node",
			Events: []any{
				linkData("linkpath", "nodepath"),
				blobData("blobpath"),
			},
			Expected: consistency(func(c *backup.Consistency) {
				linkedNode(c, "nodepath", backup.InconsistencyNodeMissing)
				blob(c, "blobpath", backup.InconsistencyBlobOrphaned)
			}),
		},
		{
			Name: "corrupt metadata",
			Events: []any{
				nodeData("nodepath", "blobpath", true, backup.InconsistencyMetadataMissing),
				linkData("linkpath", "nodepath"),
				blobData("blobpath"),
			},
			Expected: consistency(func(c *backup.Consistency) {
				node(c, "nodepath", backup.InconsistencyMetadataMissing)
			}),
		},
		{
			Name: "corrupt metadata, no blob",
			Events: []any{
				nodeData("nodepath", "blobpath", true, backup.InconsistencyMetadataMissing),
				linkData("linkpath", "nodepath"),
			},
			Expected: consistency(func(c *backup.Consistency) {
				node(c, "nodepath", backup.InconsistencyMetadataMissing)
				blobReference(c, "blobpath", backup.InconsistencyBlobMissing)
			}),
		},
	}

	for _, tc := range testcases {
		events := make(chan any)

		go func() {
			for _, ev := range tc.Events {
				switch e := ev.(type) {
				case backup.NodeData:
					events <- e
				case backup.LinkData:
					events <- e
				case backup.BlobData:
					events <- e
				}
			}
			close(events)
		}()

		c := backup.NewConsistency()
		c.GatherData(events)

		require.Equal(t, tc.Expected.Nodes, c.Nodes)
		require.Equal(t, tc.Expected.LinkedNodes, c.LinkedNodes)
		require.Equal(t, tc.Expected.Blobs, c.Blobs)
		require.Equal(t, tc.Expected.BlobReferences, c.BlobReferences)
	}

}

func nodeData(nodePath, blobPath string, requiresSymlink bool, incs ...backup.Inconsistency) backup.NodeData {
	return backup.NodeData{
		NodePath:        nodePath,
		BlobPath:        blobPath,
		RequiresSymlink: requiresSymlink,
		Inconsistencies: incs,
	}
}

func linkData(linkPath, nodePath string) backup.LinkData {
	return backup.LinkData{
		LinkPath: linkPath,
		NodePath: nodePath,
	}
}

func blobData(blobPath string) backup.BlobData {
	return backup.BlobData{
		BlobPath: blobPath,
	}
}

func consistency(f func(*backup.Consistency)) *backup.Consistency {
	c := backup.NewConsistency()
	f(c)
	return c
}

func node(c *backup.Consistency, path string, inc ...backup.Inconsistency) {
	c.Nodes[path] = inc
}

func linkedNode(c *backup.Consistency, path string, inc ...backup.Inconsistency) {
	c.LinkedNodes[path] = inc
}

func blob(c *backup.Consistency, path string, inc ...backup.Inconsistency) {
	c.Blobs[path] = inc
}

func blobReference(c *backup.Consistency, path string, inc ...backup.Inconsistency) {
	c.BlobReferences[path] = inc
}
