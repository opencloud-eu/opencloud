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

package owncloudsql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

var defaultFilePerm = os.FileMode(0664)

func (fs *owncloudsqlfs) Upload(ctx context.Context, req storage.UploadRequest, uff storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	upload, err := fs.GetUpload(ctx, req.Ref.GetPath())
	if err != nil {
		return &provider.ResourceInfo{}, errors.Wrap(err, "owncloudsql: error retrieving upload")
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
				return &provider.ResourceInfo{}, errors.Wrap(err, "owncloudsql: error removing auxiliary files")
			}
			return &provider.ResourceInfo{}, errtypes.PartialContent(req.Ref.String())
		}
		uploadInfo.info.Storage["InternalDestination"] = p
		fd, err := os.Open(assembledFile)
		if err != nil {
			return &provider.ResourceInfo{}, errors.Wrap(err, "owncloudsql: error opening assembled file")
		}
		defer fd.Close()
		defer os.RemoveAll(assembledFile)
		req.Body = fd
	}

	if _, err := uploadInfo.WriteChunk(ctx, 0, req.Body); err != nil {
		return &provider.ResourceInfo{}, errors.Wrap(err, "owncloudsql: error writing to binary file")
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

	ri := &provider.ResourceInfo{
		// fill with at least fileid, mtime and etag
		Id: &provider.ResourceId{
			StorageId: uploadInfo.info.MetaData["providerID"],
			SpaceId:   uploadInfo.info.Storage["StorageId"],
			OpaqueId:  uploadInfo.info.Storage["fileid"],
		},
		Etag: uploadInfo.info.MetaData["etag"],
	}

	if mtime, err := utils.MTimeToTS(uploadInfo.info.MetaData["mtime"]); err == nil {
		ri.Mtime = &mtime
	}

	return ri, nil
}

// InitiateUpload returns upload ids corresponding to different protocols it supports
// TODO read optional content for small files in this request
func (fs *owncloudsqlfs) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// permissions are checked in NewUpload below

	p := fs.toStoragePath(ctx, ip)

	info := tusd.FileInfo{
		MetaData: tusd.MetaData{
			"filename": filepath.Base(p),
			"dir":      filepath.Dir(p),
			"mtime":    strconv.FormatInt(time.Now().Unix(), 10),
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

// UseIn tells the tus upload middleware which extensions it supports.
func (fs *owncloudsqlfs) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(fs)
	composer.UseTerminater(fs)
	composer.UseConcater(fs)
	composer.UseLengthDeferrer(fs)
}

// To implement the core tus.io protocol as specified in https://tus.io/protocols/resumable-upload.html#core-protocol
// - the storage needs to implement NewUpload and GetUpload
// - the upload needs to implement the tusd.Upload interface: WriteChunk, GetInfo, GetReader and FinishUpload

func (fs *owncloudsqlfs) NewUpload(ctx context.Context, info tusd.FileInfo) (upload tusd.Upload, err error) {

	log := appctx.GetLogger(ctx)
	log.Debug().Interface("info", info).Msg("owncloudsql: NewUpload")

	if info.MetaData["filename"] == "" {
		return nil, errors.New("owncloudsql: missing filename in metadata")
	}
	info.MetaData["filename"] = filepath.Clean(info.MetaData["filename"])

	dir := info.MetaData["dir"]
	if dir == "" {
		return nil, errors.New("owncloudsql: missing dir in metadata")
	}
	info.MetaData["dir"] = filepath.Clean(info.MetaData["dir"])

	ip := fs.toInternalPath(ctx, filepath.Join(info.MetaData["dir"], info.MetaData["filename"]))

	// check permissions
	var perm *provider.ResourcePermissions
	var perr error
	var fsInfo iofs.FileInfo
	// if destination exists
	if fsInfo, err = os.Stat(ip); err == nil {
		// check permissions of file to be overwritten
		perm, perr = fs.readPermissions(ctx, ip)
	} else {
		// check permissions of parent folder
		perm, perr = fs.readPermissions(ctx, filepath.Dir(ip))
	}
	if perr == nil {
		if !perm.InitiateFileUpload {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if os.IsNotExist(err) {
			return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	// if we are trying to overwriting a folder with a file
	if fsInfo != nil && fsInfo.IsDir() {
		return nil, errtypes.PreconditionFailed("resource is not a file")
	}

	log.Debug().Interface("info", info).Msg("owncloudsql: resolved filename")

	info.ID = uuid.New().String()

	binPath, err := fs.getUploadPath(ctx, info.ID)
	if err != nil {
		return nil, errors.Wrap(err, "owncloudsql: error resolving upload path")
	}
	usr := ctxpkg.ContextMustGetUser(ctx)
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return nil, err
	}
	info.Storage = map[string]string{
		"Type":                "OwnCloudStore",
		"BinPath":             binPath,
		"InternalDestination": ip,
		"Permissions":         strconv.Itoa((int)(conversions.RoleFromResourcePermissions(perm, false).OCSPermissions())),

		"Idp":      usr.Id.Idp,
		"UserId":   usr.Id.OpaqueId,
		"UserName": usr.Username,

		"LogLevel": log.GetLevel().String(),

		"StorageId": strconv.Itoa(storageID),
	}
	// Create binary file in the upload folder with no content
	log.Debug().Interface("info", info).Msg("owncloudsql: built storage info")
	file, err := os.OpenFile(binPath, os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	u := &fileUpload{
		info:     info,
		binPath:  binPath,
		infoPath: filepath.Join(fs.c.UploadInfoDir, info.ID+".info"),
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

func (fs *owncloudsqlfs) getUploadPath(ctx context.Context, uploadID string) (string, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx")
		return "", err
	}
	layout := templates.WithUser(u, fs.c.UserLayout)
	return filepath.Join(fs.c.DataDirectory, layout, "uploads", uploadID), nil
}

// GetUpload returns the Upload for the given upload id
func (fs *owncloudsqlfs) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	infoPath := filepath.Join(fs.c.UploadInfoDir, id+".info")

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

	stat, err := os.Stat(info.Storage["BinPath"])
	if err != nil {
		return nil, err
	}

	info.Offset = stat.Size()

	u := &userpb.User{
		Id: &userpb.UserId{
			Idp:      info.Storage["Idp"],
			OpaqueId: info.Storage["UserId"],
		},
		Username: info.Storage["UserName"],
	}

	ctx = ctxpkg.ContextSetUser(ctx, u)
	// TODO configure the logger the same way ... store and add traceid in file info

	opts := []logger.Option{
		logger.WithLevel(info.Storage["LogLevel"]),
		logger.WithWriter(os.Stderr, logger.ConsoleMode),
	}
	l := logger.New(opts...)

	sub := l.With().Int("pid", os.Getpid()).Logger()

	ctx = appctx.WithLogger(ctx, &sub)

	return &fileUpload{
		info:     info,
		binPath:  info.Storage["BinPath"],
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
	fs *owncloudsqlfs
	// a context with a user
	// TODO add logger as well?
	ctx context.Context
}

// GetInfo returns the FileInfo
func (upload *fileUpload) GetInfo(ctx context.Context) (tusd.FileInfo, error) {
	return upload.info, nil
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
	// However, for OwnCloudStore it's not important whether the stream has ended
	// on purpose or accidentally.
	if err != nil {
		if err != io.ErrUnexpectedEOF {
			return n, err
		}
	}

	upload.info.Offset += n
	err = upload.writeInfo() // TODO info is written here ... we need to truncate in DiscardChunk

	return n, err
}

// GetReader returns an io.Reader for the upload
func (upload *fileUpload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(upload.binPath)
}

// writeInfo updates the entire information. Everything will be overwritten.
func (upload *fileUpload) writeInfo() error {
	log.Debug().Str("path", upload.infoPath).Msg("Writing info file")
	data, err := json.Marshal(upload.info)
	if err != nil {
		return err
	}
	return os.WriteFile(upload.infoPath, data, defaultFilePerm)
}

// FinishUpload finishes an upload and moves the file to the internal destination
func (upload *fileUpload) FinishUpload(ctx context.Context) error {

	ip := upload.info.Storage["InternalDestination"]

	// if destination exists
	// TODO check etag with If-Match header
	if _, err := os.Stat(ip); err == nil {
		// create revision
		if err := upload.fs.archiveRevision(upload.ctx, upload.fs.getVersionsPath(upload.ctx, ip), ip); err != nil {
			return err
		}
	}

	sha1h, md5h, adler32h, err := upload.fs.HashFile(upload.binPath)
	if err != nil {
		log.Err(err).Msg("owncloudsql: could not open file for checksumming")
	}

	err = os.Rename(upload.binPath, ip)
	if err != nil {
		log.Err(err).Interface("info", upload.info).
			Str("binPath", upload.binPath).
			Str("ipath", ip).
			Msg("owncloudsql: could not rename")
		return err
	}

	var fi os.FileInfo
	fi, err = os.Stat(ip)
	if err != nil {
		return err
	}

	perms, err := strconv.Atoi(upload.info.Storage["Permissions"])
	if err != nil {
		return err
	}

	if upload.info.MetaData["mtime"] == "" {
		upload.info.MetaData["mtime"] = fmt.Sprintf("%d", fi.ModTime().Unix())
	}
	if upload.info.MetaData["etag"] == "" {
		upload.info.MetaData["etag"] = calcEtag(upload.ctx, fi)
	}

	data := map[string]interface{}{
		"path":          upload.fs.toDatabasePath(ip),
		"checksum":      fmt.Sprintf("SHA1:%032x MD5:%032x ADLER32:%032x", sha1h, md5h, adler32h),
		"etag":          upload.info.MetaData["etag"],
		"size":          upload.info.Size,
		"mimetype":      mime.Detect(false, ip),
		"permissions":   perms,
		"mtime":         upload.info.MetaData["mtime"],
		"storage_mtime": upload.info.MetaData["mtime"],
	}
	var fileid int
	fileid, err = upload.fs.filecache.InsertOrUpdate(ctx, upload.info.Storage["StorageId"], data, false)
	if err != nil {
		return err
	}
	upload.info.Storage["fileid"] = fmt.Sprintf("%d", fileid)

	// only delete the upload if it was successfully written to the storage
	if err := os.Remove(upload.infoPath); err != nil {
		if !os.IsNotExist(err) {
			log.Err(err).Interface("info", upload.info).Msg("owncloudsql: could not delete upload info")
			return err
		}
	}

	return upload.fs.propagate(upload.ctx, ip)
}

// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// - the storage needs to implement AsTerminatableUpload
// - the upload needs to implement Terminate

// AsTerminatableUpload returns a TerminatableUpload
func (fs *owncloudsqlfs) AsTerminatableUpload(upload tusd.Upload) tusd.TerminatableUpload {
	return upload.(*fileUpload)
}

// Terminate terminates the upload
func (upload *fileUpload) Terminate(ctx context.Context) error {
	if err := os.Remove(upload.infoPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Remove(upload.binPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// To implement the creation-defer-length extension as specified in https://tus.io/protocols/resumable-upload.html#creation
// - the storage needs to implement AsLengthDeclarableUpload
// - the upload needs to implement DeclareLength

// AsLengthDeclarableUpload returns a LengthDeclarableUpload
func (fs *owncloudsqlfs) AsLengthDeclarableUpload(upload tusd.Upload) tusd.LengthDeclarableUpload {
	return upload.(*fileUpload)
}

// DeclareLength updates the upload length information
func (upload *fileUpload) DeclareLength(ctx context.Context, length int64) error {
	upload.info.Size = length
	upload.info.SizeIsDeferred = false
	return upload.writeInfo()
}

// To implement the concatenation extension as specified in https://tus.io/protocols/resumable-upload.html#concatenation
// - the storage needs to implement AsConcatableUpload
// - the upload needs to implement ConcatUploads

// AsConcatableUpload returns a ConcatableUpload
func (fs *owncloudsqlfs) AsConcatableUpload(upload tusd.Upload) tusd.ConcatableUpload {
	return upload.(*fileUpload)
}

// ConcatUploads concatenates multiple uploads
func (upload *fileUpload) ConcatUploads(ctx context.Context, uploads []tusd.Upload) (err error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, partialUpload := range uploads {
		fileUpload := partialUpload.(*fileUpload)

		src, err := os.Open(fileUpload.binPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, src); err != nil {
			return err
		}
	}

	return
}
