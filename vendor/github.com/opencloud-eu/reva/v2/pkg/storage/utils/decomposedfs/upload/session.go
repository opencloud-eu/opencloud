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
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// OcisSession extends tus upload lifecycle with postprocessing steps.
type OcisSession struct {
	store OcisStore
	// for now, we keep the json files in the uploads folder
	info tusd.FileInfo
}

// Context returns a context with the user, logger and lockid used when initiating the upload session
func (s *OcisSession) Context(ctx context.Context) context.Context { // restore logger from file info
	sub := s.store.log.With().Int("pid", os.Getpid()).Logger()
	ctx = appctx.WithLogger(ctx, &sub)
	ctx = ctxpkg.ContextSetLockID(ctx, s.lockID())
	ctx = ctxpkg.ContextSetUser(ctx, s.executantUser())
	return ctxpkg.ContextSetInitiator(ctx, s.InitiatorID())
}

func (s *OcisSession) lockID() string {
	return s.info.MetaData["lockid"]
}
func (s *OcisSession) executantUser() *userpb.User {
	var o *typespb.Opaque
	_ = json.Unmarshal([]byte(s.info.Storage["UserOpaque"]), &o)
	return &userpb.User{
		Id: &userpb.UserId{
			Type:     userpb.UserType(userpb.UserType_value[s.info.Storage["UserType"]]),
			Idp:      s.info.Storage["Idp"],
			OpaqueId: s.info.Storage["UserId"],
		},
		Username:    s.info.Storage["UserName"],
		DisplayName: s.info.Storage["UserDisplayName"],
		Opaque:      o,
	}
}

// Purge deletes the upload session metadata and written binary data
func (s *OcisSession) Purge(ctx context.Context) error {
	_, span := tracer.Start(ctx, "Purge")
	defer span.End()
	sessionPath := sessionPath(s.store.root, s.info.ID)
	if err := os.Remove(sessionPath); err != nil {
		return err
	}
	if err := os.Remove(s.binPath()); err != nil {
		return err
	}
	return nil
}

// TouchBin creates a file to contain the binary data. It's size will be used to keep track of the tus upload offset.
func (s *OcisSession) TouchBin() error {
	file, err := os.OpenFile(s.binPath(), os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return err
	}
	return file.Close()
}

// Persist writes the upload session metadata to disk
// events can update the scan outcome and the finished event might read an empty file because of race conditions
// so we need to lock the file while writing and use atomic writes
func (s *OcisSession) Persist(ctx context.Context) error {
	_, span := tracer.Start(ctx, "Persist")
	defer span.End()
	sessionPath := sessionPath(s.store.root, s.info.ID)
	// create folder structure (if needed)
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0700); err != nil {
		return err
	}

	var d []byte
	d, err := json.Marshal(s.info)
	if err != nil {
		return err
	}
	return renameio.WriteFile(sessionPath, d, 0600)
}

// ToFileInfo returns tus compatible FileInfo so the tus handler can access the upload offset
func (s *OcisSession) ToFileInfo() tusd.FileInfo {
	return s.info
}

// ProviderID returns the provider id
func (s *OcisSession) ProviderID() string {
	return s.info.MetaData["providerID"]
}

// SpaceID returns the space id
func (s *OcisSession) SpaceID() string {
	return s.info.Storage["SpaceRoot"]
}

// NodeID returns the node id
func (s *OcisSession) NodeID() string {
	return s.info.Storage["NodeId"]
}

// NodeParentID returns the nodes parent id
func (s *OcisSession) NodeParentID() string {
	return s.info.Storage["NodeParentId"]
}

// NodeExists returns wether or not the node existed during InitiateUpload.
// FIXME If two requests try to write the same file they both will store a new
// random node id in the session and try to initialize a new node when
// finishing the upload. The second request will fail with an already exists
// error when trying to create the symlink for the node in the parent directory.
// A node should be created as part of InitiateUpload. When listing a directory
// we can decide if we want to skip the entry, or expose uploed progress
// information. But that is a bigger change and might involve client work.
func (s *OcisSession) NodeExists() bool {
	return s.info.Storage["NodeExists"] == "true"
}

// HeaderIfMatch returns the if-match header for the upload session
func (s *OcisSession) HeaderIfMatch() string {
	return s.info.MetaData["if-match"]
}

// HeaderIfNoneMatch returns the if-none-match header for the upload session
func (s *OcisSession) HeaderIfNoneMatch() string {
	return s.info.MetaData["if-none-match"]
}

// HeaderIfUnmodifiedSince returns the if-unmodified-since header for the upload session
func (s *OcisSession) HeaderIfUnmodifiedSince() string {
	return s.info.MetaData["if-unmodified-since"]
}

// Node returns the node for the session
func (s *OcisSession) Node(ctx context.Context) (*node.Node, error) {
	return node.ReadNode(ctx, s.store.lu, s.SpaceID(), s.info.Storage["NodeId"], false, nil, true)
}

// ID returns the upload session id
func (s *OcisSession) ID() string {
	return s.info.ID
}

// Filename returns the name of the node which is not the same as the name af the file being uploaded for legacy chunked uploads
func (s *OcisSession) Filename() string {
	return s.info.Storage["NodeName"]
}

// Chunk returns the chunk name when a legacy chunked upload was started
func (s *OcisSession) Chunk() string {
	return s.info.Storage["Chunk"]
}

// SetMetadata is used to fill the upload metadata that will be exposed to the end user
func (s *OcisSession) SetMetadata(key, value string) {
	s.info.MetaData[key] = value
}

// SetStorageValue is used to set metadata only relevant for the upload session implementation
func (s *OcisSession) SetStorageValue(key, value string) {
	s.info.Storage[key] = value
}

// SetSize will set the upload size of the underlying tus info.
func (s *OcisSession) SetSize(size int64) {
	s.info.Size = size
}

// SetSizeIsDeferred is uset to change the SizeIsDeferred property of the underlying tus info.
func (s *OcisSession) SetSizeIsDeferred(value bool) {
	s.info.SizeIsDeferred = value
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
func (s *OcisSession) Dir() string {
	return s.info.Storage["Dir"]
}

// Size returns the upload size
func (s *OcisSession) Size() int64 {
	return s.info.Size
}

// SizeDiff returns the size diff that was calculated after postprocessing
func (s *OcisSession) SizeDiff() int64 {
	sizeDiff, _ := strconv.ParseInt(s.info.MetaData["sizeDiff"], 10, 64)
	return sizeDiff
}

// Reference returns a reference that can be used to access the uploaded resource
func (s *OcisSession) Reference() provider.Reference {
	return provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: s.info.MetaData["providerID"],
			SpaceId:   s.info.Storage["SpaceRoot"],
			OpaqueId:  s.info.Storage["NodeId"],
		},
		// Path is not used
	}
}

// Executant returns the id of the user that initiated the upload session
func (s *OcisSession) Executant() userpb.UserId {
	return userpb.UserId{
		Type:     userpb.UserType(userpb.UserType_value[s.info.Storage["UserType"]]),
		Idp:      s.info.Storage["Idp"],
		OpaqueId: s.info.Storage["UserId"],
	}
}

// SetExecutant is used to remember the user that initiated the upload session
func (s *OcisSession) SetExecutant(u *userpb.User) {
	s.info.Storage["Idp"] = u.GetId().GetIdp()
	s.info.Storage["UserId"] = u.GetId().GetOpaqueId()
	s.info.Storage["UserType"] = utils.UserTypeToString(u.GetId().Type)
	s.info.Storage["UserName"] = u.GetUsername()
	s.info.Storage["UserDisplayName"] = u.GetDisplayName()

	b, _ := json.Marshal(u.GetOpaque())
	s.info.Storage["UserOpaque"] = string(b)
}

// Offset returns the current upload offset
func (s *OcisSession) Offset() int64 {
	return s.info.Offset
}

// SpaceOwner returns the id of the space owner
func (s *OcisSession) SpaceOwner() *userpb.UserId {
	return &userpb.UserId{
		// idp and type do not seem to be consumed and the node currently only stores the user id anyway
		OpaqueId: s.info.Storage["SpaceOwnerOrManager"],
	}
}

// Expires returns the time the upload session expires
func (s *OcisSession) Expires() time.Time {
	var t time.Time
	if value, ok := s.info.MetaData["expires"]; ok {
		t, _ = utils.MTimeToTime(value)
	}
	return t
}

// MTime returns the mtime to use for the uploaded file
func (s *OcisSession) MTime() time.Time {
	var t time.Time
	if value, ok := s.info.MetaData["mtime"]; ok {
		t, _ = utils.MTimeToTime(value)
	}
	return t
}

// IsProcessing returns true if all bytes have been received. The session then has entered postprocessing state.
func (s *OcisSession) IsProcessing() bool {
	// We might need a more sophisticated way to determine processing status soon
	return s.info.Size == s.info.Offset && s.info.MetaData["scanResult"] == ""
}

// binPath returns the path to the file storing the binary data.
func (s *OcisSession) binPath() string {
	return filepath.Join(s.store.root, "uploads", s.info.ID)
}

// InitiatorID returns the id of the initiating client
func (s *OcisSession) InitiatorID() string {
	return s.info.MetaData["initiatorid"]
}

// SetScanData sets virus scan data to the upload session
func (s *OcisSession) SetScanData(result string, date time.Time) {
	s.info.MetaData["scanResult"] = result
	s.info.MetaData["scanDate"] = date.Format(time.RFC3339)
}

// ScanData returns the virus scan data
func (s *OcisSession) ScanData() (string, time.Time) {
	date := s.info.MetaData["scanDate"]
	if date == "" {
		return "", time.Time{}
	}
	d, _ := time.Parse(time.RFC3339, date)
	return s.info.MetaData["scanResult"], d
}

func (s *OcisSession) SetStatus(status string) {
	s.info.MetaData["status"] = status
}

func (s *OcisSession) Status() string {
	return s.info.MetaData["status"]
}

func (s *OcisSession) SetStatusMessage(message string) {
	s.info.MetaData["status_message"] = message
}

func (s *OcisSession) StatusMessage() string {
	return s.info.MetaData["status_message"]
}

// sessionPath returns the path to the .info file storing the file's info.
func sessionPath(root, id string) string {
	return filepath.Join(root, "uploads", id+".info")
}
