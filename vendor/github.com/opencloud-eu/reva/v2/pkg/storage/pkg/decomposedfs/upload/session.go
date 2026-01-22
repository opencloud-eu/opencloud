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

package upload

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/renameio/v2"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// DecomposedFsSession extends tus upload lifecycle with postprocessing steps.
type DecomposedFsSession struct {
	store DecomposedFsStore
	// for now, we keep the json files in the uploads folder
	info tusd.FileInfo
}

// Context returns a context with the user, logger and lockid used when initiating the upload session
func (session *DecomposedFsSession) Context(ctx context.Context) context.Context { // restore logger from file info
	sub := session.store.log.With().Int("pid", os.Getpid()).Logger()
	ctx = appctx.WithLogger(ctx, &sub)
	ctx = ctxpkg.ContextSetLockID(ctx, session.lockID())
	ctx = ctxpkg.ContextSetUser(ctx, session.executantUser())
	return ctxpkg.ContextSetInitiator(ctx, session.InitiatorID())
}

func (session *DecomposedFsSession) lockID() string {
	return session.info.MetaData["lockid"]
}
func (session *DecomposedFsSession) executantUser() *userpb.User {
	var o *typespb.Opaque
	_ = json.Unmarshal([]byte(session.info.Storage["UserOpaque"]), &o)
	return &userpb.User{
		Id: &userpb.UserId{
			Type:     userpb.UserType(userpb.UserType_value[session.info.Storage["UserType"]]),
			Idp:      session.info.Storage["Idp"],
			OpaqueId: session.info.Storage["UserId"],
		},
		Username:    session.info.Storage["UserName"],
		DisplayName: session.info.Storage["UserDisplayName"],
		Opaque:      o,
	}
}

// Purge deletes the upload session metadata and written binary data
func (session *DecomposedFsSession) Purge(ctx context.Context) error {
	_, span := tracer.Start(ctx, "Purge")
	defer span.End()
	sessionPath := sessionPath(session.store.root, session.info.ID)
	if err := os.Remove(sessionPath); err != nil {
		return err
	}
	if err := os.Remove(session.binPath()); err != nil {
		return err
	}
	return nil
}

// TouchBin creates a file to contain the binary data. It's size will be used to keep track of the tus upload offset.
func (session *DecomposedFsSession) TouchBin() error {
	file, err := os.OpenFile(session.binPath(), os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return err
	}
	return file.Close()
}

// Persist writes the upload session metadata to disk
// events can update the scan outcome and the finished event might read an empty file because of race conditions
// so we need to lock the file while writing and use atomic writes
func (session *DecomposedFsSession) Persist(ctx context.Context) error {
	_, span := tracer.Start(ctx, "Persist")
	defer span.End()
	sessionPath := sessionPath(session.store.root, session.info.ID)
	// create folder structure (if needed)
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0700); err != nil {
		return err
	}

	var d []byte
	d, err := json.Marshal(session.info)
	if err != nil {
		return err
	}
	return renameio.WriteFile(sessionPath, d, 0600)
}

// ToFileInfo returns tus compatible FileInfo so the tus handler can access the upload offset
func (session *DecomposedFsSession) ToFileInfo() tusd.FileInfo {
	return session.info
}

// ProviderID returns the provider id
func (session *DecomposedFsSession) ProviderID() string {
	return session.info.MetaData["providerID"]
}

// SpaceID returns the space id
func (session *DecomposedFsSession) SpaceID() string {
	return session.info.Storage["SpaceRoot"]
}

// NodeID returns the node id
func (session *DecomposedFsSession) NodeID() string {
	return session.info.Storage["NodeId"]
}

// NodeParentID returns the nodes parent id
func (session *DecomposedFsSession) NodeParentID() string {
	return session.info.Storage["NodeParentId"]
}

// NodeExists returns wether or not the node existed during InitiateUpload.
// FIXME If two requests try to write the same file they both will store a new
// random node id in the session and try to initialize a new node when
// finishing the upload. The second request will fail with an already exists
// error when trying to create the symlink for the node in the parent directory.
// A node should be created as part of InitiateUpload. When listing a directory
// we can decide if we want to skip the entry, or expose uploed progress
// information. But that is a bigger change and might involve client work.
func (session *DecomposedFsSession) NodeExists() bool {
	return session.info.Storage["NodeExists"] == "true"
}

// HeaderIfMatch returns the if-match header for the upload session
func (session *DecomposedFsSession) HeaderIfMatch() string {
	return session.info.MetaData["if-match"]
}

// HeaderIfNoneMatch returns the if-none-match header for the upload session
func (session *DecomposedFsSession) HeaderIfNoneMatch() string {
	return session.info.MetaData["if-none-match"]
}

// HeaderIfUnmodifiedSince returns the if-unmodified-since header for the upload session
func (session *DecomposedFsSession) HeaderIfUnmodifiedSince() string {
	return session.info.MetaData["if-unmodified-since"]
}

// Node returns the node for the session
func (session *DecomposedFsSession) Node(ctx context.Context) (*node.Node, error) {
	return node.ReadNode(ctx, session.store.lu, session.SpaceID(), session.info.Storage["NodeId"], "", false, nil, true)
}

// ID returns the upload session id
func (session *DecomposedFsSession) ID() string {
	return session.info.ID
}

// Filename returns the name of the node which is not the same as the name af the file being uploaded for legacy chunked uploads
func (session *DecomposedFsSession) Filename() string {
	return session.info.Storage["NodeName"]
}

// Chunk returns the chunk name when a legacy chunked upload was started
func (session *DecomposedFsSession) Chunk() string {
	return session.info.Storage["Chunk"]
}

// SetMetadata is used to fill the upload metadata that will be exposed to the end user
func (session *DecomposedFsSession) SetMetadata(key, value string) {
	session.info.MetaData[key] = value
}

// SetStorageValue is used to set metadata only relevant for the upload session implementation
func (session *DecomposedFsSession) SetStorageValue(key, value string) {
	session.info.Storage[key] = value
}

// SetSize will set the upload size of the underlying tus info.
func (session *DecomposedFsSession) SetSize(size int64) {
	session.info.Size = size
}

// SetSizeIsDeferred is uset to change the SizeIsDeferred property of the underlying tus info.
func (session *DecomposedFsSession) SetSizeIsDeferred(value bool) {
	session.info.SizeIsDeferred = value
}

// Dir returns the directory to which the upload is made
// TODO get rid of Dir(), whoever consumes the reference should be able to deal
// with a relative reference.
// Dir is only used to:
//   - fill the Path property when emitting the UploadReady event after
//     postprocessing finished. I wonder why the UploadReady contains a finished
//     flag ... maybe multiple distinct events would make more sense.
//   - build the reference that is passed to the FileUploaded event in the
//     UploadFinishedFunc callback passed to the Upload call used for simple
//     datatx put requests
//
// AFAICT only search and audit services consume the path.
//   - search needs to index from the root anyway. And it only needs the most
//     recent path to put it in the index. So it should already be able to deal
//     with an id based reference.
//   - audit on the other hand needs to log events with the path at the state of
//     the event ... so it does need the full path.
//
// I think we can safely determine the path later, right before emitting the
// event. And maybe make it configurable, because only audit needs it, anyway.
func (session *DecomposedFsSession) Dir() string {
	return session.info.Storage["Dir"]
}

// Size returns the upload size
func (session *DecomposedFsSession) Size() int64 {
	return session.info.Size
}

// SizeDiff returns the size diff that was calculated after postprocessing
func (session *DecomposedFsSession) SizeDiff() int64 {
	sizeDiff, _ := strconv.ParseInt(session.info.MetaData["sizeDiff"], 10, 64)
	return sizeDiff
}

// Reference returns a reference that can be used to access the uploaded resource
func (session *DecomposedFsSession) Reference() provider.Reference {
	return provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: session.info.MetaData["providerID"],
			SpaceId:   session.info.Storage["SpaceRoot"],
			OpaqueId:  session.info.Storage["NodeId"],
		},
		// Path is not used
	}
}

// Executant returns the id of the user that initiated the upload session
func (session *DecomposedFsSession) Executant() userpb.UserId {
	return userpb.UserId{
		Type:     userpb.UserType(userpb.UserType_value[session.info.Storage["UserType"]]),
		Idp:      session.info.Storage["Idp"],
		OpaqueId: session.info.Storage["UserId"],
	}
}

// SetExecutant is used to remember the user that initiated the upload session
func (session *DecomposedFsSession) SetExecutant(u *userpb.User) {
	session.info.Storage["Idp"] = u.GetId().GetIdp()
	session.info.Storage["UserId"] = u.GetId().GetOpaqueId()
	session.info.Storage["UserType"] = utils.UserTypeToString(u.GetId().Type)
	session.info.Storage["UserName"] = u.GetUsername()
	session.info.Storage["UserDisplayName"] = u.GetDisplayName()

	b, _ := json.Marshal(u.GetOpaque())
	session.info.Storage["UserOpaque"] = string(b)
}

// Offset returns the current upload offset
func (session *DecomposedFsSession) Offset() int64 {
	return session.info.Offset
}

// SpaceOwner returns the id of the space owner
func (session *DecomposedFsSession) SpaceOwner() *userpb.UserId {
	return &userpb.UserId{
		// idp and type do not seem to be consumed and the node currently only stores the user id anyway
		OpaqueId: session.info.Storage["SpaceOwnerOrManager"],
	}
}

// Expires returns the time the upload session expires
func (session *DecomposedFsSession) Expires() time.Time {
	var t time.Time
	if value, ok := session.info.MetaData["expires"]; ok {
		t, _ = utils.MTimeToTime(value)
	}
	return t
}

// MTime returns the mtime to use for the uploaded file
func (session *DecomposedFsSession) MTime() time.Time {
	var t time.Time
	if value, ok := session.info.MetaData["mtime"]; ok {
		t, _ = utils.MTimeToTime(value)
	}
	return t
}

// IsProcessing returns true if all bytes have been received. The session then has entered postprocessing state.
func (session *DecomposedFsSession) IsProcessing() bool {
	// We might need a more sophisticated way to determine processing status soon
	return session.info.Size == session.info.Offset && session.info.MetaData["scanResult"] == ""
}

// binPath returns the path to the file storing the binary data.
func (session *DecomposedFsSession) binPath() string {
	return filepath.Join(session.store.root, "uploads", session.info.ID)
}

// InitiatorID returns the id of the initiating client
func (session *DecomposedFsSession) InitiatorID() string {
	return session.info.MetaData["initiatorid"]
}

// SetScanData sets virus scan data to the upload session
func (session *DecomposedFsSession) SetScanData(result string, date time.Time) {
	session.info.MetaData["scanResult"] = result
	session.info.MetaData["scanDate"] = date.Format(time.RFC3339)
}

// ScanData returns the virus scan data
func (session *DecomposedFsSession) ScanData() (string, time.Time) {
	date := session.info.MetaData["scanDate"]
	if date == "" {
		return "", time.Time{}
	}
	d, _ := time.Parse(time.RFC3339, date)
	return session.info.MetaData["scanResult"], d
}

// sessionPath returns the path to the .info file storing the file's info.
func sessionPath(root, id string) string {
	return filepath.Join(root, "uploads", id+".info")
}
