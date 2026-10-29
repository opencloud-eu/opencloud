package command

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
	decomposedbs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposed/blobstore"
	s3bs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposeds3/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/spf13/cobra"
)

// Blobstore is the entry point for the blobstore command group.
func Blobstore(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{Use: "blobstore", Short: "manage the blobstore"}
	cmd.AddCommand(BlobstoreCheck(cfg), BlobstoreGet(cfg))
	return cmd
}

// BlobstoreCheck uploads random bytes to a random path, downloads and
// verifies them, then deletes them again.  All three steps must succeed.
func BlobstoreCheck(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "check blobstore connectivity via an upload/download/delete round-trip",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bs, err := initBlobstore(cfg)
			if err != nil {
				return err
			}
			blobSize, err := cmd.Flags().GetString("blob-size")
			if err != nil {
				return err
			}
			size, err := bytesize.Parse(blobSize)
			if err != nil {
				return fmt.Errorf("invalid --blob-size %q: %w", blobSize, err)
			}
			if size > bytesize.ByteSize(int(^uint(0)>>1)) {
				return fmt.Errorf("--blob-size exceeds the maximum supported size")
			}
			return runBlobstoreRoundTrip(bs, int(size))
		},
	}
	cmd.Flags().String("blob-size", "64", "size of the random blob to upload, e.g. 64, 1KB, 1MB, 4MiB")
	return cmd
}

// BlobstoreGet downloads a specific blob by its ID to verify it is readable.
func BlobstoreGet(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a blob from the blobstore by ID",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bs, err := initBlobstore(cfg)
			if err != nil {
				return err
			}
			blobID, err := cmd.Flags().GetString("blob-id")
			if err != nil {
				return err
			}
			spaceID, err := cmd.Flags().GetString("space-id")
			if err != nil {
				return err
			}
			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("path") {
				spaceID, blobID, err = parseBlobPath(path)
				if err != nil {
					return err
				}
			}
			if blobID == "" || spaceID == "" {
				return fmt.Errorf("either --path or both --blob-id and --space-id must be set")
			}
			size, err := cmd.Flags().GetInt64("blob-size")
			if err != nil {
				return err
			}
			if size < 0 {
				return fmt.Errorf("--blob-size must not be negative")
			}
			return downloadBlob(bs, blobID, spaceID, size)
		},
	}
	cmd.Flags().String("path", "", "blobstore path from a log line; extracts --blob-id and --space-id automatically")
	cmd.Flags().String("blob-id", "", "blob ID to download (required when --path is not set)")
	cmd.Flags().String("space-id", "", "space ID the blob belongs to (required when --path is not set)")
	cmd.Flags().Int64("blob-size", 0, "expected blob size in bytes; only needed for s3ng when known upfront")
	return cmd
}

// parseBlobPath extracts spaceID and blobID from a blobstore path as it
// appears in log lines.  Two formats are supported:
//
//   - S3NG:  "<spaceID>/<pathified_blobID>"
//     e.g.  "b19ec764-5398-458a-8ff1-1925bd906999/61/03/ab/c3/-b08a-4556-9937-2bf3065c1202"
//
//   - OCIS:  any path containing "/spaces/<pathified_spaceID>/blobs/<pathified_blobID>"
//     e.g.  "/var/lib/ocis/storage/users/spaces/b1/9ec764-.../blobs/61/03/ab/c3/-b08a-..."
func parseBlobPath(path string) (spaceID, blobID string, err error) {
	// OCIS paths contain both /spaces/ and /blobs/ as structural markers.
	if _, rest, ok := strings.Cut(path, "/spaces/"); ok {
		pathifiedSpaceID, pathifiedBlobID, ok := strings.Cut(rest, "/blobs/")
		if !ok {
			return "", "", fmt.Errorf("cannot parse ocis blob path %q: missing /blobs/ segment", path)
		}
		spaceID = depathify(pathifiedSpaceID, 1)
		blobID = depathify(pathifiedBlobID, 4)
		return
	}

	// S3NG paths: <spaceID>/<pathified_blobID>
	var pathifiedBlobID string
	spaceID, pathifiedBlobID, _ = strings.Cut(path, "/")
	if pathifiedBlobID == "" {
		return "", "", fmt.Errorf("cannot parse s3ng blob path %q: expected <spaceID>/<pathified_blobID>", path)
	}
	blobID = depathify(pathifiedBlobID, 4)
	return
}

// depathify reverses the effect of lookup.Pathify(id, depth, 2):
// it strips the directory separators that were inserted every two characters
// up to the given depth.
// e.g. depathify("61/03/ab/c3/-b08a-4556-9937-2bf3065c1202", 4)
//
//	→ "6103abc3-b08a-4556-9937-2bf3065c1202"
func depathify(path string, depth int) string {
	parts := strings.SplitN(path, "/", depth+1)
	return strings.Join(parts, "")
}

// initBlobstore builds a node.Blobstore from the service configuration.
func initBlobstore(cfg *config.Config) (node.Blobstore, error) {
	switch cfg.Driver {
	case "ocis", "decomposed":
		return decomposedbs.New(cfg.Drivers.Decomposed.Root)
	case "s3ng", "decomposeds3":
		return s3bs.New(
			cfg.Drivers.DecomposedS3.Endpoint,
			cfg.Drivers.DecomposedS3.Region,
			cfg.Drivers.DecomposedS3.Bucket,
			cfg.Drivers.DecomposedS3.AccessKey,
			cfg.Drivers.DecomposedS3.SecretKey,
			s3bs.Options{
				DisableContentSha256:  cfg.Drivers.DecomposedS3.DisableContentSha256,
				DisableMultipart:      cfg.Drivers.DecomposedS3.DisableMultipart,
				SendContentMd5:        cfg.Drivers.DecomposedS3.SendContentMd5,
				ConcurrentStreamParts: cfg.Drivers.DecomposedS3.ConcurrentStreamParts,
				NumThreads:            cfg.Drivers.DecomposedS3.NumThreads,
				PartSize:              cfg.Drivers.DecomposedS3.PartSize,
			},
		)
	default:
		return nil, fmt.Errorf("blobstore operations are not supported for driver %q", cfg.Driver)
	}
}

// blobSizeMismatchRe matches the error produced by the s3ng blobstore when the
// retrieved object size does not match node.Blobsize, and captures the actual size.
var blobSizeMismatchRe = regexp.MustCompile(`blob has unexpected size\. \d+ bytes expected, got (\d+) bytes`)

// downloadBlob downloads a single blob identified by blobID, drains the reader
// to surface any streaming errors, then closes it.
// If the s3ng blobstore rejects the download due to a size mismatch, the actual
// size is extracted from the error and the download is retried with the correct value.
func downloadBlob(bs node.Blobstore, blobID, spaceID string, blobSize int64) error {
	n := &node.Node{
		BaseNode: node.BaseNode{SpaceID: spaceID},
		BlobID:   blobID,
		Blobsize: blobSize,
	}
	rc, err := bs.Download(n)
	if err != nil {
		if m := blobSizeMismatchRe.FindStringSubmatch(err.Error()); m != nil {
			fmt.Printf("blob size mismatch, retrying with actual size %s bytes\n", m[1])
			if _, err := fmt.Sscan(m[1], &n.Blobsize); err != nil {
				return fmt.Errorf("download failed: could not parse actual blob size %q: %w", m[1], err)
			}
			rc, err = bs.Download(n)
		}
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	}
	if _, err := io.Copy(io.Discard, rc); err != nil {
		_ = rc.Close()
		return fmt.Errorf("download failed while reading: %w", err)
	}
	if err := rc.Close(); err != nil {
		return fmt.Errorf("download failed while closing blob reader: %w", err)
	}
	fmt.Println("Download: OK")
	return nil
}

// runBlobstoreRoundTrip uploads random data, verifies it can be retrieved, and
// then removes it again.  All three steps must succeed for the check to pass.
func runBlobstoreRoundTrip(bs node.Blobstore, blobSize int) error {
	// 1. Generate random bytes that serve as the test payload.
	data := make([]byte, blobSize)
	if _, err := rand.Read(data); err != nil {
		return fmt.Errorf("failed to generate random data: %w", err)
	}

	// Write the payload to a temporary file. The decomposed blobstore may rename
	// (move) this file into the blobstore, so os.Remove in the defer is a
	// no-op for that driver – which is fine.
	tmpFile, err := os.CreateTemp("", "decomposed-blobstore-check-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // best-effort cleanup

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// 2. Build a test node with random identifiers.
	testNode := &node.Node{
		BaseNode: node.BaseNode{SpaceID: uuid.New().String()},
		BlobID:   uuid.New().String(),
		Blobsize: int64(len(data)),
	}
	fmt.Printf("Uploading test blob: spaceID=%s blobID=%s\n", testNode.SpaceID, testNode.BlobID)

	// 3. Upload.
	if err := bs.Upload(testNode, tmpPath, ""); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	fmt.Println("Upload: OK")

	// 4. Download and verify.
	rc, err := bs.Download(testNode)
	if err != nil {
		return roundTripFailure(bs, testNode, fmt.Errorf("download failed: %w", err))
	}
	downloaded, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil {
		if closeErr != nil {
			return roundTripFailure(bs, testNode, fmt.Errorf("failed to read downloaded data: %w (also failed to close blob: %v)", readErr, closeErr))
		}
		return roundTripFailure(bs, testNode, fmt.Errorf("failed to read downloaded data: %w", readErr))
	}
	if closeErr != nil {
		return roundTripFailure(bs, testNode, fmt.Errorf("failed to close downloaded blob: %w", closeErr))
	}
	if !bytes.Equal(data, downloaded) {
		return roundTripFailure(bs, testNode, fmt.Errorf("data integrity check failed: downloaded content does not match uploaded content"))
	}
	fmt.Println("Download and verify: OK")

	// 5. Delete.
	if err := bs.Delete(testNode); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	fmt.Println("Delete: OK")
	fmt.Println("Blobstore check successful.")
	return nil
}

func roundTripFailure(bs node.Blobstore, n *node.Node, cause error) error {
	if err := bs.Delete(n); err != nil {
		return fmt.Errorf("%w (also failed to clean up test blob: %v)", cause, err)
	}
	return cause
}
