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

package storage

import (
	"context"
	"io"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

// UploadFinishedFunc is a callback function used in storage drivers to indicate that an upload has finished
type UploadFinishedFunc func(spaceOwner, executant *userpb.UserId, ref *provider.Reference)

// UploadRequest us used in FS.Upload() to carry required upload metadata
type UploadRequest struct {
	Ref    *provider.Reference
	Body   io.ReadCloser
	Length int64
}

// UploadsManager defines the interface for storage drivers that allow for managing uploads
//
// Deprecated: No longer used. Storage drivers should implement the UploadSessionLister.
type UploadsManager interface {
	ListUploads() ([]tusd.FileInfo, error)
	PurgeExpiredUploads(chan<- tusd.FileInfo) error
}

// UploadSessionLister defines the interface for FS implementations that allow listing and purging upload sessions
type UploadSessionLister interface {
	// ListUploadSessions returns the upload sessions matching the given filter
	ListUploadSessions(ctx context.Context, filter UploadSessionFilter) ([]UploadSession, error)
}

// UploadSession is the interface that storage drivers need to return whan listing upload sessions.
type UploadSession interface {
	// ID returns the upload id
	ID() string
	// Filename returns the filename of the file
	Filename() string
	// Size returns the size of the upload
	Size() int64
	// Offset returns the current offset
	Offset() int64
	// Reference returns a reference for the file being uploaded. May be absolute id based or relative to e.g. a space root
	Reference() provider.Reference
	// Executant returns the userid of the user that created the upload
	Executant() userpb.UserId
	// SpaceOwner returns the owner of a space if set. optional
	SpaceOwner() *userpb.UserId
	// Expires returns the time when the upload can no longer be used
	Expires() time.Time

	// IsProcessing returns true if postprocessing has not finished, yet
	// The actual postprocessing state is tracked in the postprocessing service.
	IsProcessing() bool

	// Purge allows completely removing an upload. Should emit a PostprocessingFinished event with a Delete outcome
	Purge(ctx context.Context) error

	// ScanData returns the scan data for the UploadSession
	ScanData() (string, time.Time)
}

// UploadSessionFilter can be used to filter upload sessions
type UploadSessionFilter struct {
	ID         *string
	Processing *bool
	Expired    *bool
	HasVirus   *bool
}
