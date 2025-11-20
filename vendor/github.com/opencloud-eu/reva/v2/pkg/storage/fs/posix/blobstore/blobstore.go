// Copyright 2018-2021 CERN
// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
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

package blobstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/pkg/xattr"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
)

const (
	TMPDir = ".oc-tmp"
)

// Blobstore provides an interface to an filesystem based blobstore
type Blobstore struct {
	root string
}

// New returns a new Blobstore
func New(root string) (*Blobstore, error) {
	return &Blobstore{
		root: root,
	}, nil
}

// Upload is responsible for transferring data from a source file (upload) to its final location;
// the file operation is done atomically using a temporary file followed by a rename
func (bs *Blobstore) Upload(n *node.Node, source, copyTarget string) error {
	tempName := filepath.Join(n.SpaceRoot.InternalPath(), TMPDir, filepath.Base(source))

	// there is no guarantee that the space root TMPDir exists at this point, so we create the directory if needed
	if err := os.MkdirAll(filepath.Dir(tempName), 0700); err != nil {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %v", source, err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	tempFile, err := os.OpenFile(tempName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to create temp file '%s': %v", tempName, err)
	}

	if _, err := tempFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to write data from source file '%s' to temp file '%s' - %v", source, tempName, err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file '%s' - %v", tempName, err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file '%s' - %v", tempName, err)
	}

	nodeAttributes, err := n.Xattrs(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get xattrs for node '%s': %v", n.InternalPath(), err)
	}

	var mtime *time.Time
	for k, v := range nodeAttributes {
		if err := xattr.Set(tempName, k, v); err != nil {
			return fmt.Errorf("failed to set xattr '%s' on temp file '%s' - %v", k, tempName, err)
		}

		if k == "user.oc.mtime" {
			tv, err := time.Parse(time.RFC3339Nano, string(v))
			if err == nil {
				mtime = &tv
			}
		}
	}

	// the extended attributes should always contain a mtime, but in case they don't, we fetch it from the node
	if mtime == nil {
		switch nodeMtime, err := n.GetMTime(context.Background()); {
		case err != nil:
			return fmt.Errorf("failed to get mtime for node '%s' - %v", n.InternalPath(), err)
		default:
			mtime = &nodeMtime
		}

	}

	// etags rely on the id and the mtime, so we need to ensure the mtime is set correctly
	if err := os.Chtimes(tempName, *mtime, *mtime); err != nil {
		return fmt.Errorf("failed to set mtime on temp file '%s' - %v", tempName, err)
	}

	// atomically move the file to its final location,
	// on Windows systems (unsupported oc os) os.Rename is not atomic
	if err := os.Rename(tempName, n.InternalPath()); err != nil {
		return fmt.Errorf("failed to move temp file '%s' to node '%s' - %v", tempName, n.InternalPath(), err)
	}

	// upload successfully, now handle the copy target if set
	if copyTarget == "" {
		return nil
	}

	// also "upload" the file to a local path, e.g., for keeping the "current" version of the file
	if err := os.MkdirAll(filepath.Dir(copyTarget), 0700); err != nil {
		return err
	}

	if _, err := sourceFile.Seek(0, 0); err != nil {
		return err
	}

	copyFile, err := os.OpenFile(copyTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrapf(err, "could not open copy target '%s' for writing", copyTarget)
	}
	defer func() {
		_ = copyFile.Close()
	}()

	if _, err := copyFile.ReadFrom(sourceFile); err != nil {
		return errors.Wrapf(err, "could not write blob copy of '%s' to '%s'", n.InternalPath(), copyTarget)
	}

	return nil
}

// Download retrieves a blob from the blobstore for reading
func (bs *Blobstore) Download(node *node.Node) (io.ReadCloser, error) {
	file, err := os.Open(node.InternalPath())
	if err != nil {
		return nil, errors.Wrapf(err, "could not read blob '%s'", node.InternalPath())
	}
	return file, nil
}

// Delete deletes a blob from the blobstore
func (bs *Blobstore) Delete(node *node.Node) error {
	return nil
}
