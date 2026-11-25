package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	revanode "github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/test-go/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func TestGatherDataRecheck(t *testing.T) {
	scanStart := time.Now()

	testcases := []struct {
		Name        string
		Listed      []string
		ExpectList  bool
		SkipRecheck bool
		Setup       func(t *testing.T, dir string) (events []any, expected *Consistency)
	}{
		{
			Name: "blob missing but exists now",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				bp := filepath.Join(dir, "blob")
				writeFile(t, bp, "data")
				return []any{
						nodeData(np, bp, false),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name: "blob missing and still absent",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				bp := filepath.Join(dir, "blob")
				return []any{
						nodeData(np, bp, false),
					},
					consistency(func(c *Consistency) {
						blobReference(c, bp, InconsistencyBlobMissing)
					})
			},
		},
		{
			Name:        "blob missing but exists now - no recheck",
			SkipRecheck: true,
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				bp := filepath.Join(dir, "blob")
				writeFile(t, bp, "data")
				return []any{
						nodeData(np, bp, false),
					},
					consistency(func(c *Consistency) {
						blobReference(c, bp, InconsistencyBlobMissing)
					})
			},
		},
		{
			Name:       "remote blob missing but listed now",
			Listed:     []string{"blob"},
			ExpectList: true,
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				return []any{
						nodeData(np, "bucket/blob", false),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name:       "remote blob missing and still absent",
			Listed:     []string{"other"},
			ExpectList: true,
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				return []any{
						nodeData(np, "bucket/blob", false),
					},
					consistency(func(c *Consistency) {
						blobReference(c, "bucket/blob", InconsistencyBlobMissing)
					})
			},
		},
		{
			Name:   "no remote findings - no listing",
			Listed: []string{"blob"},
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				bp := filepath.Join(dir, "blob")
				return []any{
						nodeData(np, bp, false),
					},
					consistency(func(c *Consistency) {
						blobReference(c, bp, InconsistencyBlobMissing)
					})
			},
		},
		{
			Name: "node missing but exists now",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				return []any{
						linkData("linkpath", np),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name: "node missing and still absent",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				return []any{
						linkData("linkpath", np),
					},
					consistency(func(c *Consistency) {
						inconsistentLink(c, np, InconsistencyNodeMissing)
					})
			},
		},
		{
			Name: "files missing but exist now",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				writeFile(t, np+".mpk", "")
				return []any{
						nodeData(np, "", false, InconsistencyFilesMissing),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name: "files missing and still missing",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				return []any{
						nodeData(np, "", false, InconsistencyFilesMissing),
					},
					consistency(func(c *Consistency) {
						inconsistentNode(c, np, InconsistencyFilesMissing)
					})
			},
		},
		{
			Name: "malformed file but valid now",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				b, err := msgpack.Marshal(map[string][]byte{"user.oc.blobid": []byte("blob")})
				require.NoError(t, err)
				writeFile(t, np+".mpk", string(b))
				return []any{
						nodeData(np, "", false, InconsistencyMalformedFile),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name: "malformed file and still malformed",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np+".mpk", "not msgpack")
				return []any{
						nodeData(np, "", false, InconsistencyMalformedFile),
					},
					consistency(func(c *Consistency) {
						inconsistentNode(c, np, InconsistencyMalformedFile)
					})
			},
		},
		{
			Name: "symlink missing but node modified during scan",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				chtimes(t, np, scanStart.Add(time.Minute))
				return []any{
						nodeData(np, "", true),
					},
					consistency(func(c *Consistency) {})
			},
		},
		{
			Name: "symlink missing and node modified before scan",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				chtimes(t, np, scanStart.Add(-time.Hour))
				return []any{
						nodeData(np, "", true),
					},
					consistency(func(c *Consistency) {
						inconsistentNode(c, np, InconsistencySymlinkMissing)
					})
			},
		},
		{
			Name: "orphaned blob untouched",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				bp := filepath.Join(dir, "blob")
				writeFile(t, bp, "data")
				return []any{
						blobData(bp),
					},
					consistency(func(c *Consistency) {
						orphanedBlob(c, bp, InconsistencyBlobOrphaned)
					})
			},
		},
		{
			Name: "real defect stays next to resolved one",
			Setup: func(t *testing.T, dir string) ([]any, *Consistency) {
				np := filepath.Join(dir, "node")
				writeFile(t, np, "")
				writeFile(t, np+".mpk", "")
				bp := filepath.Join(dir, "blob")
				return []any{
						nodeData(np, bp, false, InconsistencyFilesMissing, InconsistencyMetadataMissing),
					},
					consistency(func(c *Consistency) {
						inconsistentNode(c, np, InconsistencyMetadataMissing)
						blobReference(c, bp, InconsistencyBlobMissing)
					})
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			dir := t.TempDir()
			events, expected := tc.Setup(t, dir)
			lbs := &fakeBlobstore{listed: tc.Listed}

			c := NewConsistency()
			if !tc.SkipRecheck {
				c.recheck = newRechecker(dir, lbs, scanStart)
			}
			c.GatherData(eventChannel(events))

			require.Equal(t, expected.Nodes, c.Nodes)
			require.Equal(t, expected.LinkedNodes, c.LinkedNodes)
			require.Equal(t, expected.Blobs, c.Blobs)
			require.Equal(t, expected.BlobReferences, c.BlobReferences)
			require.Equal(t, tc.ExpectList, lbs.listCalled)
		})
	}
}

func eventChannel(events []any) <-chan any {
	ch := make(chan any, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch
}

type fakeBlobstore struct {
	listed     []string
	listCalled bool
}

func (f *fakeBlobstore) List() ([]*revanode.Node, error) {
	f.listCalled = true
	nodes := make([]*revanode.Node, 0, len(f.listed))
	for _, id := range f.listed {
		nodes = append(nodes, &revanode.Node{BlobID: id})
	}
	return nodes, nil
}

func (f *fakeBlobstore) Path(n *revanode.Node) string {
	return "bucket/" + n.BlobID
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}

func chtimes(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	require.NoError(t, os.Chtimes(path, mtime, mtime))
}

func nodeData(nodePath, blobPath string, requiresSymlink bool, incs ...Inconsistency) NodeData {
	return NodeData{
		NodePath:        nodePath,
		BlobPath:        blobPath,
		RequiresSymlink: requiresSymlink,
		Inconsistencies: incs,
	}
}

func linkData(linkPath, nodePath string) LinkData {
	return LinkData{
		LinkPath: linkPath,
		NodePath: nodePath,
	}
}

func blobData(blobPath string) BlobData {
	return BlobData{
		BlobPath: blobPath,
	}
}

func consistency(f func(*Consistency)) *Consistency {
	c := NewConsistency()
	f(c)
	return c
}

func inconsistentNode(c *Consistency, path string, inc ...Inconsistency) {
	c.Nodes[path] = inc
}

func inconsistentLink(c *Consistency, path string, inc ...Inconsistency) {
	c.LinkedNodes[path] = inc
}

func orphanedBlob(c *Consistency, path string, inc ...Inconsistency) {
	c.Blobs[path] = inc
}

func blobReference(c *Consistency, path string, inc ...Inconsistency) {
	c.BlobReferences[path] = inc
}
