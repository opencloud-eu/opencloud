// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"encoding/json"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
)

// TokenOptions carries the JWT-signing configuration needed to produce transfer
// URLs for the postprocessing service.
type TokenOptions struct {
	DownloadEndpoint     string
	DataGatewayEndpoint  string
	TransferSharedSecret string
	TransferExpires      int64
}

// SessionStore abstracts upload-session persistence for the Coordinator.
type SessionStore interface {
	New(ctx context.Context) Session
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	// FromInfo wraps an upload's tus FileInfo without reading the store, so it also
	// works after a synchronous upload has cleaned up its session.
	FromInfo(info tusd.FileInfo) Session
}

// FileStore is a filesystem-backed SessionStore. Sessions are stored as a pair
// of files in the upload directory:
//
//   - <id>.info  — JSON-encoded tusd.FileInfo
//   - <id>       — staged binary bytes
//
// This is the same on-disk format used by OcisStore so existing sessions
// survive a rolling deploy that switches to FileStore.
type FileStore struct {
	uploadDir string
	opts      TokenOptions
	log       *zerolog.Logger
}

// FileStoreFromDriverConf builds a FileStore from a reva driver config map.
// Returns nil if the config carries no upload path (driver does not support
// coordinated uploads). Each service that mounts the same driver calls this
// independently.
func FileStoreFromDriverConf(driverConf map[string]interface{}, log *zerolog.Logger) *FileStore {
	if driverConf == nil {
		return nil
	}

	// storage_root is the ocm driver's spelling of root; it stages uploads under
	// the same <root>/uploads/ layout the decomposedfs family uses.
	type driverRootConf struct {
		Root            string `mapstructure:"root"`
		UploadDirectory string `mapstructure:"upload_directory"`
		StorageRoot     string `mapstructure:"storage_root"`
	}
	var rc driverRootConf
	_ = mapstructure.Decode(driverConf, &rc)

	// upload_directory already names the upload directory itself, the way
	// decomposedfs reads it (options.go:172, posix tree.go:168). The other two are
	// storage roots that stage uploads in a subdirectory, so only they get joined.
	if rc.UploadDirectory != "" {
		return newFileStoreWithTokens(rc.UploadDirectory, driverConf, log)
	}

	root := rc.Root
	if root == "" {
		root = rc.StorageRoot
	}
	if root == "" {
		return nil
	}

	return newFileStoreWithTokens(filepath.Join(root, "uploads"), driverConf, log)
}

// NewFileStoreFromConfig builds a FileStore staging uploads in uploadDir when
// set, falling back to the active driver config. This allows drivers that have
// no local root (e.g. KW) to still get a coordinator by setting
// upload_directory at the service level rather than inside the driver.
// Returns nil only when neither source resolves to a non-empty path.
func NewFileStoreFromConfig(uploadDir string, driverConf map[string]interface{}, log *zerolog.Logger) *FileStore {
	if uploadDir != "" {
		// Still take the tokens from the driver config: they sign the transfer URL
		// that postprocessing downloads the staged bytes from, and a service-level
		// upload directory says nothing about them.
		return newFileStoreWithTokens(uploadDir, driverConf, log)
	}
	return FileStoreFromDriverConf(driverConf, log)
}

func newFileStoreWithTokens(uploadDir string, driverConf map[string]interface{}, log *zerolog.Logger) *FileStore {
	type tokenConf struct {
		DownloadEndpoint     string `mapstructure:"download_endpoint"`
		DataGatewayEndpoint  string `mapstructure:"datagateway_endpoint"`
		TransferSharedSecret string `mapstructure:"transfer_shared_secret"`
		TransferExpires      int64  `mapstructure:"transfer_expires"`
	}
	var tc tokenConf
	if tokens, ok := driverConf["tokens"]; ok {
		_ = mapstructure.Decode(tokens, &tc)
	}
	return NewFileStore(uploadDir, TokenOptions(tc), log)
}

// NewFileStore creates a FileStore staging uploads in uploadDir.
// uploadDir must be on a shared filesystem when multiple pods handle the same space.
func NewFileStore(uploadDir string, opts TokenOptions, log *zerolog.Logger) *FileStore {
	return &FileStore{uploadDir: uploadDir, opts: opts, log: log}
}

// UploadDir returns the directory this FileStore stages uploads in.
func (fs *FileStore) UploadDir() string {
	return fs.uploadDir
}

// Setup creates the upload directory eagerly so permission problems are caught
// at startup rather than on the first upload.
func (fs *FileStore) Setup() error {
	return os.MkdirAll(fs.uploadDir, 0700)
}

// New allocates a fresh session with a new UUID.
func (fs *FileStore) New(_ context.Context) Session {
	return &FileSession{
		store: fs,
		info: tusd.FileInfo{
			ID: uuid.New().String(),
			Storage: map[string]string{
				"Type": "OCISStore",
			},
			MetaData: tusd.MetaData{
				"status": SessionStatusUploading,
			},
		},
	}
}

// FromInfo wraps an upload's tus FileInfo into a session without reading the store.
func (fs *FileStore) FromInfo(info tusd.FileInfo) Session {
	return &FileSession{store: fs, info: info}
}

// Get loads the session with the given id from disk.
func (fs *FileStore) Get(ctx context.Context, id string) (Session, error) {
	infoPath := fileSessionPath(fs.uploadDir, id)

	data, err := os.ReadFile(infoPath)
	if err != nil {
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ESTALE {
			return nil, tusd.ErrNotFound
		}
		if errors.Is(err, iofs.ErrNotExist) {
			return nil, tusd.ErrNotFound
		}
		return nil, err
	}

	var info tusd.FileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	session := &FileSession{store: fs, info: info}

	stat, err := os.Stat(session.binPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, tusd.ErrNotFound
		}
		return nil, err
	}
	session.info.Offset = stat.Size()

	return session, nil
}

// List returns all sessions found in the upload directory.
func (fs *FileStore) List(ctx context.Context) ([]Session, error) {
	infoFiles, err := filepath.Glob(filepath.Join(fs.uploadDir, "*.info"))
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(infoFiles))
	for _, path := range infoFiles {
		id := strings.TrimSuffix(filepath.Base(path), ".info")
		session, err := fs.Get(ctx, id)
		if err != nil {
			appctx.GetLogger(ctx).Error().Str("path", path).Err(err).Msg("filestore: could not load session")
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}
