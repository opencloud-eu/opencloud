// Copyright 2018-2023 CERN
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

package ocm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/adler32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var defaultFilePerm = os.FileMode(0664)

func (d *driver) ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error) {
	return []storage.UploadSession{}, nil
}
func (d *driver) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	shareID, rel := shareInfoFromReference(ref)
	p := getPathFromShareIDAndRelPath(shareID, rel)

	info := tusd.FileInfo{
		MetaData: tusd.MetaData{
			"filename": filepath.Base(p),
			"dir":      filepath.Dir(p),
		},
		Size: uploadLength,
	}

	upload, err := d.NewUpload(ctx, info)
	if err != nil {
		return nil, err
	}

	info, _ = upload.GetInfo(ctx)

	return map[string]string{
		"simple": info.ID,
		"tus":    info.ID,
	}, nil
}

func (d *driver) Upload(ctx context.Context, req storage.UploadRequest, _ storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	shareID, _ := shareInfoFromReference(req.Ref)
	u, err := d.GetUpload(ctx, shareID.OpaqueId)
	if err != nil {
		return &provider.ResourceInfo{}, err
	}

	info, err := u.GetInfo(ctx)
	if err != nil {
		return &provider.ResourceInfo{}, err
	}

	defer cleanup(&upload{Info: info})

	client, _, rel, err := d.webdavClient(ctx, nil, &provider.Reference{
		Path: filepath.Join(info.MetaData["dir"], info.MetaData["filename"]),
	})
	if err != nil {
		return &provider.ResourceInfo{}, err
	}
	client.SetInterceptor(func(method string, rq *http.Request) {
		// Set the content length on the request struct directly instead of the header.
		// The content-length header gets reset by the golang http library before
		// sendind out the request, resulting in chunked encoding to be used which
		// breaks the quota checks in ocdav.
		if method == "PUT" {
			rq.ContentLength = req.Length
		}
	})

	locktoken, _ := ctxpkg.ContextGetLockID(ctx)
	return &provider.ResourceInfo{}, client.WriteStream(rel, req.Body, 0, locktoken)
}

// UseIn tells the tus upload middleware which extensions it supports.
func (d *driver) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(d)
	composer.UseTerminater(d)
	composer.UseConcater(d)
	composer.UseLengthDeferrer(d)
}

// AsTerminatableUpload returns a TerminatableUpload
// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// the storage needs to implement AsTerminatableUpload
func (d *driver) AsTerminatableUpload(up tusd.Upload) tusd.TerminatableUpload {
	return up.(*upload)
}

// AsLengthDeclarableUpload returns a LengthDeclarableUpload
// To implement the creation-defer-length extension as specified in https://tus.io/protocols/resumable-upload.html#creation
// the storage needs to implement AsLengthDeclarableUpload
func (d *driver) AsLengthDeclarableUpload(up tusd.Upload) tusd.LengthDeclarableUpload {
	return up.(*upload)
}

// AsConcatableUpload returns a ConcatableUpload
// To implement the concatenation extension as specified in https://tus.io/protocols/resumable-upload.html#concatenation
// the storage needs to implement AsConcatableUpload
func (d *driver) AsConcatableUpload(up tusd.Upload) tusd.ConcatableUpload {
	return up.(*upload)
}

// To implement the core tus.io protocol as specified in https://tus.io/protocols/resumable-upload.html#core-protocol
// - the storage needs to implement NewUpload and GetUpload
// - the upload needs to implement the tusd.Upload interface: WriteChunk, GetInfo, GetReader and FinishUpload

// NewUpload returns a new tus Upload instance
func (d *driver) NewUpload(ctx context.Context, info tusd.FileInfo) (tusd.Upload, error) {
	return NewUpload(ctx, d, d.c.StorageRoot, info)
}

// GetUpload returns the Upload for the given upload id
func (d *driver) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	return GetUpload(ctx, d, d.c.StorageRoot, id)
}
func NewUpload(ctx context.Context, d *driver, storageRoot string, info tusd.FileInfo) (tusd.Upload, error) {
	if info.MetaData["filename"] == "" {
		return nil, errors.New("decomposedfs: missing filename in metadata")
	}
	if info.MetaData["dir"] == "" {
		return nil, errors.New("decomposedfs: missing dir in metadata")
	}

	uploadRoot := filepath.Join(storageRoot, "uploads")
	info.ID = uuid.New().String()

	user, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return nil, errors.New("no user in context")
	}
	info.MetaData["user"] = user.GetId().GetOpaqueId()
	info.MetaData["idp"] = user.GetId().GetIdp()

	info.Storage = map[string]string{
		"Type": "OCM",
		"Path": uploadRoot,
	}

	u := &upload{
		Info: info,
		Ctx:  ctx,
		d:    d,
	}

	err := os.MkdirAll(uploadRoot, 0755)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(u.BinPath(), os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = u.Persist()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func GetUpload(ctx context.Context, d *driver, storageRoot string, id string) (tusd.Upload, error) {
	info := tusd.FileInfo{}
	data, err := os.ReadFile(filepath.Join(storageRoot, "uploads", id+".info"))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	upload := &upload{
		Info: info,
		Ctx:  ctx,
		d:    d,
	}
	return upload, nil
}

type upload struct {
	Info tusd.FileInfo
	Ctx  context.Context

	d *driver
}

func (u *upload) InfoPath() string {
	return filepath.Join(u.Info.Storage["Path"], u.Info.ID+".info")
}

func (u *upload) BinPath() string {
	return filepath.Join(u.Info.Storage["Path"], u.Info.ID)
}

func (u *upload) Persist() error {
	data, err := json.Marshal(u.Info)
	if err != nil {
		return err
	}
	return os.WriteFile(u.InfoPath(), data, defaultFilePerm)
}

func (u *upload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(u.BinPath(), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// calculate cheksum here? needed for the TUS checksum extension. https://tus.io/protocols/resumable-upload.html#checksum
	// TODO but how do we get the `Upload-Checksum`? WriteChunk() only has a context, offset and the reader ...
	// It is sent with the PATCH request, well or in the POST when the creation-with-upload extension is used
	// but the tus handler uses a context.Background() so we cannot really check the header and put it in the context ...
	n, err := io.Copy(file, src)

	// If the HTTP PATCH request gets interrupted in the middle (e.g. because
	// the user wants to pause the upload), Go's net/http returns an io.ErrUnexpectedEOF.
	// However, for the ocm driver it's not important whether the stream has ended
	// on purpose or accidentally.
	if err != nil && err != io.ErrUnexpectedEOF {
		return n, err
	}

	u.Info.Offset += n
	return n, u.Persist()
}

func (u *upload) GetInfo(ctx context.Context) (tusd.FileInfo, error) {
	return u.Info, nil
}

func (u *upload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(u.BinPath())
}

func (u *upload) FinishUpload(ctx context.Context) error {
	log := appctx.GetLogger(u.Ctx)

	// calculate the checksum of the written bytes
	// they will all be written to the metadata later, so we cannot omit any of them
	// TODO only calculate the checksum in sync that was requested to match, the rest could be async ... but the tests currently expect all to be present
	// TODO the hashes all implement BinaryMarshaler so we could try to persist the state for resumable upload. we would neet do keep track of the copied bytes ...
	sha1h := sha1.New()
	md5h := md5.New()
	adler32h := adler32.New()
	{
		f, err := os.Open(u.BinPath())
		if err != nil {
			// we can continue if no oc checksum header is set
			log.Info().Err(err).Str("binPath", u.BinPath()).Msg("error opening binPath")
		}
		defer f.Close()

		r1 := io.TeeReader(f, sha1h)
		r2 := io.TeeReader(r1, md5h)

		_, err = io.Copy(adler32h, r2)
		if err != nil {
			log.Info().Err(err).Msg("error copying checksums")
		}
	}

	defer cleanup(u)
	// compare if they match the sent checksum
	// TODO the tus checksum extension would do this on every chunk, but I currently don't see an easy way to pass in the requested checksum. for now we do it in FinishUpload which is also called for chunked uploads
	if u.Info.MetaData["checksum"] != "" {
		var err error
		parts := strings.SplitN(u.Info.MetaData["checksum"], " ", 2)
		if len(parts) != 2 {
			return errtypes.BadRequest("invalid checksum format. must be '[algorithm] [checksum]'")
		}
		switch parts[0] {
		case "sha1":
			err = u.checkHash(parts[1], sha1h)
		case "md5":
			err = u.checkHash(parts[1], md5h)
		case "adler32":
			err = u.checkHash(parts[1], adler32h)
		default:
			err = errtypes.BadRequest("unsupported checksum algorithm: " + parts[0])
		}
		if err != nil {
			return err
		}
	}

	// send to the remote storage via webdav
	// shareID, rel := shareInfoFromReference(u.Info.MetaData["ref"])
	// p := getPathFromShareIDAndRelPath(shareID, rel)

	serviceUserCtx, err := utils.GetServiceUserContextWithContext(context.Background(), u.d.gateway, u.d.c.ServiceAccountID, u.d.c.ServiceAccountSecret)
	if err != nil {
		return err
	}
	client, _, rel, err := u.d.webdavClient(serviceUserCtx, &userpb.UserId{
		OpaqueId: u.Info.MetaData["user"],
		Idp:      u.Info.MetaData["idp"],
	}, &provider.Reference{
		Path: filepath.Join(u.Info.MetaData["dir"], u.Info.MetaData["filename"]),
	})
	if err != nil {
		return err
	}

	client.SetInterceptor(func(method string, rq *http.Request) {
		// Set the content length on the request struct directly instead of the header.
		// The content-length header gets reset by the golang http library before
		// sendind out the request, resulting in chunked encoding to be used which
		// breaks the quota checks in ocdav.
		if method == "PUT" {
			rq.ContentLength = u.Info.Size
		}
	})

	f, err := os.Open(u.BinPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return client.WriteStream(rel, f, 0, "")
}

func (u *upload) Terminate(ctx context.Context) error {
	cleanup(u)
	return nil
}

func (u *upload) ConcatUploads(_ context.Context, uploads []tusd.Upload) error {
	file, err := os.OpenFile(u.BinPath(), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, partialUpload := range uploads {
		fileUpload := partialUpload.(*upload)

		src, err := os.Open(fileUpload.BinPath())
		if err != nil {
			return err
		}
		defer src.Close()

		if _, err := io.Copy(file, src); err != nil {
			return err
		}
	}
	return nil
}

func (u *upload) DeclareLength(ctx context.Context, length int64) error {
	u.Info.Size = length
	u.Info.SizeIsDeferred = false
	return nil
}

func (u *upload) checkHash(expected string, h hash.Hash) error {
	if expected != hex.EncodeToString(h.Sum(nil)) {
		return errtypes.ChecksumMismatch(fmt.Sprintf("invalid checksum: expected %s got %x", u.Info.MetaData["checksum"], h.Sum(nil)))
	}
	return nil
}

func cleanup(u *upload) {
	if u == nil {
		return
	}
	_ = os.Remove(u.BinPath())
	_ = os.Remove(u.InfoPath())
}
