// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package localfs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

var defaultFilePerm = os.FileMode(0664)

func (fs *localfs) Upload(ctx context.Context, req storage.UploadRequest, uff storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	upload, err := fs.GetUpload(ctx, req.Ref.GetPath())
	if err != nil {
		return &provider.ResourceInfo{}, errors.Wrap(err, "localfs: error retrieving upload")
	}

	uploadInfo := upload.(*fileUpload)

	p := uploadInfo.info.Storage["InternalDestination"]
	if chunking.IsChunked(p) {
		var assembledFile string
		p, assembledFile, err = fs.chunkHandler.WriteChunk(p, req.Body)
		if err != nil {
			return &provider.ResourceInfo{}, err
		}
		if p == "" {
			if err = uploadInfo.Terminate(ctx); err != nil {
				return &provider.ResourceInfo{}, errors.Wrap(err, "localfs: error removing auxiliary files")
			}
			return &provider.ResourceInfo{}, errtypes.PartialContent(req.Ref.String())
		}
		uploadInfo.info.Storage["InternalDestination"] = p
		fd, err := os.Open(assembledFile)
		if err != nil {
			return &provider.ResourceInfo{}, errors.Wrap(err, "localfs: error opening assembled file")
		}
		defer fd.Close()
		defer os.RemoveAll(assembledFile)
		req.Body = fd
	}

	if _, err := uploadInfo.WriteChunk(ctx, 0, req.Body); err != nil {
		return &provider.ResourceInfo{}, errors.Wrap(err, "localfs: error writing to binary file")
	}

	if err := uploadInfo.FinishUpload(ctx); err != nil {
		return &provider.ResourceInfo{}, err
	}

	if uff != nil {
		info := uploadInfo.info
		uploadRef := &provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: info.MetaData["providerID"],
				SpaceId:   info.Storage["SpaceRoot"],
				OpaqueId:  info.Storage["SpaceRoot"],
			},
			Path: utils.MakeRelativePath(filepath.Join(info.MetaData["dir"], info.MetaData["filename"])),
		}
		owner, ok := ctxpkg.ContextGetUser(uploadInfo.ctx)
		if !ok {
			return &provider.ResourceInfo{}, errtypes.PreconditionFailed("error getting user from uploadinfo context")
		}
		// spaces support in localfs needs to be revisited:
		// * info.Storage["SpaceRoot"] is never set
		// * there is no space owner or manager that could be passed to the UploadFinishedFunc
		uff(owner.Id, owner.Id, uploadRef)
	}

	// return id, etag and mtime
	ri, err := fs.GetMD(ctx, req.Ref, []string{}, []string{"id", "etag", "mtime"})
	if err != nil {
		return &provider.ResourceInfo{}, err
	}

	return ri, nil
}

// InitiateUpload returns upload ids corresponding to different protocols it supports
// It resolves the resource and then reuses the NewUpload function
// Currently requires the uploadLength to be set
// TODO to implement LengthDeferrerDataStore make size optional
// TODO read optional content for small files in this request
func (fs *localfs) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {

	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving reference")
	}

	info := tusd.FileInfo{
		MetaData: tusd.MetaData{
			"filename": filepath.Base(np),
			"dir":      filepath.Dir(np),
		},
		Size: uploadLength,
	}

	if metadata != nil {
		info.MetaData["providerID"] = metadata["providerID"]
		if metadata["mtime"] != "" {
			info.MetaData["mtime"] = metadata["mtime"]
		}
		if _, ok := metadata["sizedeferred"]; ok {
			info.SizeIsDeferred = true
		}
	}

	upload, err := fs.NewUpload(ctx, info)
	if err != nil {
		return nil, err
	}

	info, _ = upload.GetInfo(ctx)

	return map[string]string{
		"simple": info.ID,
		"tus":    info.ID,
	}, nil
}

func (fs *localfs) MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error {
	return errtypes.NotSupported("op not supported")
}

func (fs *localfs) CommitUpload(_ context.Context, _ *provider.Reference, _ string, _ storage.UploadSource) error {
	return errtypes.NotSupported("op not supported")
}

// UseIn tells the tus upload middleware which extensions it supports.
func (fs *localfs) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(fs)
	composer.UseTerminater(fs)
	// TODO composer.UseConcater(fs)
	// TODO composer.UseLengthDeferrer(fs)
}

// NewUpload creates a new upload using the size as the file's length. To determine where to write the binary data
// the Fileinfo metadata must contain a dir and a filename.
// returns a unique id which is used to identify the upload. The properties Size and MetaData will be filled.
func (fs *localfs) NewUpload(ctx context.Context, info tusd.FileInfo) (upload tusd.Upload, err error) {

	log := appctx.GetLogger(ctx)
	log.Debug().Interface("info", info).Msg("localfs: NewUpload")

	fn := info.MetaData["filename"]
	if fn == "" {
		return nil, errors.New("localfs: missing filename in metadata")
	}
	info.MetaData["filename"] = filepath.Clean(info.MetaData["filename"])

	dir := info.MetaData["dir"]
	if dir == "" {
		return nil, errors.New("localfs: missing dir in metadata")
	}
	info.MetaData["dir"] = filepath.Clean(info.MetaData["dir"])

	np := fs.wrap(ctx, filepath.Join(info.MetaData["dir"], info.MetaData["filename"]))

	log.Debug().Interface("info", info).Msg("localfs: resolved filename")

	info.ID = uuid.New().String()

	binPath, err := fs.getUploadPath(ctx, info.ID)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving upload path")
	}
	usr := ctxpkg.ContextMustGetUser(ctx)
	info.Storage = map[string]string{
		"Type":                "LocalStore",
		"BinPath":             binPath,
		"InternalDestination": np,

		"Idp":      usr.Id.Idp,
		"UserId":   usr.Id.OpaqueId,
		"UserName": usr.Username,
		"UserType": utils.UserTypeToString(usr.Id.Type),

		"LogLevel": log.GetLevel().String(),
	}
	// Create binary file with no content
	file, err := os.OpenFile(binPath, os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	u := &fileUpload{
		info:     info,
		binPath:  binPath,
		infoPath: binPath + ".info",
		fs:       fs,
		ctx:      ctx,
	}

	// writeInfo creates the file by itself if necessary
	err = u.writeInfo()
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (fs *localfs) getUploadPath(ctx context.Context, uploadID string) (string, error) {
	return filepath.Join(fs.conf.Uploads, uploadID), nil
}

// GetUpload returns the Upload for the given upload id
func (fs *localfs) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	binPath, err := fs.getUploadPath(ctx, id)
	if err != nil {
		return nil, err
	}
	infoPath := binPath + ".info"
	info := tusd.FileInfo{}
	data, err := os.ReadFile(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Interpret os.ErrNotExist as 404 Not Found
			err = tusd.ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	stat, err := os.Stat(binPath)
	if err != nil {
		return nil, err
	}

	info.Offset = stat.Size()

	u := &userpb.User{
		Id: &userpb.UserId{
			Idp:      info.Storage["Idp"],
			OpaqueId: info.Storage["UserId"],
			Type:     utils.UserTypeMap(info.Storage["UserType"]),
		},
		Username: info.Storage["UserName"],
	}

	ctx = ctxpkg.ContextSetUser(ctx, u)

	return &fileUpload{
		info:     info,
		binPath:  binPath,
		infoPath: infoPath,
		fs:       fs,
		ctx:      ctx,
	}, nil
}

type fileUpload struct {
	// info stores the current information about the upload
	info tusd.FileInfo
	// infoPath is the path to the .info file
	infoPath string
	// binPath is the path to the binary file (which has no extension)
	binPath string
	// only fs knows how to handle metadata and versions
	fs *localfs
	// a context with a user
	ctx context.Context
}

// GetInfo returns the FileInfo
func (upload *fileUpload) GetInfo(ctx context.Context) (tusd.FileInfo, error) {
	return upload.info, nil
}

// GetReader returns an io.Reader for the upload
func (upload *fileUpload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(upload.binPath)
}

// WriteChunk writes the stream from the reader to the given offset of the upload
func (upload *fileUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, src)

	// If the HTTP PATCH request gets interrupted in the middle (e.g. because
	// the user wants to pause the upload), Go's net/http returns an io.ErrUnexpectedEOF.
	// However, for the driver it's not important whether the stream has ended
	// on purpose or accidentally.
	if err != nil {
		if err != io.ErrUnexpectedEOF {
			return n, err
		}
	}

	upload.info.Offset += n
	err = upload.writeInfo()

	return n, err
}

// writeInfo updates the entire information. Everything will be overwritten.
func (upload *fileUpload) writeInfo() error {
	data, err := json.Marshal(upload.info)
	if err != nil {
		return err
	}
	return os.WriteFile(upload.infoPath, data, defaultFilePerm)
}

// FinishUpload finishes an upload and moves the file to the internal destination
func (upload *fileUpload) FinishUpload(ctx context.Context) error {

	np := upload.info.Storage["InternalDestination"]

	// TODO check etag with If-Match header
	// if destination exists
	// if _, err := os.Stat(np); err == nil {
	// the local storage does not store metadata
	// the fileid is based on the path, so no we do not need to copy it to the new file
	// the local storage does not track revisions
	//}

	// if destination exists
	if _, err := os.Stat(np); err == nil {
		// create revision
		if err := upload.fs.archiveRevision(upload.ctx, np); err != nil {
			return err
		}
	}

	err := os.Rename(upload.binPath, np)
	if err != nil {
		return err
	}

	// only delete the upload if it was successfully written to the fs
	if err := os.Remove(upload.infoPath); err != nil {
		if !os.IsNotExist(err) {
			log := appctx.GetLogger(ctx)
			log.Err(err).Interface("info", upload.info).Msg("localfs: could not delete upload info")
		}
	}

	// TODO: set mtime if specified in metadata

	// metadata propagation is left to the storage implementation
	return err
}

// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// - the storage needs to implement AsTerminatableUpload
// - the upload needs to implement Terminate

// AsTerminatableUpload returns a a TerminatableUpload
func (fs *localfs) AsTerminatableUpload(upload tusd.Upload) tusd.TerminatableUpload {
	return upload.(*fileUpload)
}

// Terminate terminates the upload
func (upload *fileUpload) Terminate(ctx context.Context) error {
	if err := os.Remove(upload.infoPath); err != nil {
		return err
	}
	if err := os.Remove(upload.binPath); err != nil {
		return err
	}
	return nil
}
