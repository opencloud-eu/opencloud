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

//go:build windows
// +build windows

package localfs

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"golang.org/x/sys/windows"
)

// calcEtag will create an etag based on the md5 of
// - mtime,
// - inode (if available),
// - device (if available) and
// - size.
// errors are logged, but an etag will still be returned
func calcEtag(ctx context.Context, fi os.FileInfo) string {
	log := appctx.GetLogger(ctx)
	h := md5.New()
	err := binary.Write(h, binary.BigEndian, fi.ModTime().Unix())
	if err != nil {
		log.Error().Err(err).Msg("error writing mtime")
	}
	// device and inode have no meaning on windows
	err = binary.Write(h, binary.BigEndian, fi.Size())
	if err != nil {
		log.Error().Err(err).Msg("error writing size")
	}
	etag := fmt.Sprintf(`"%x"`, h.Sum(nil))
	return fmt.Sprintf("\"%s\"", strings.Trim(etag, "\""))
}

func (fs *localfs) GetQuota(ctx context.Context, ref *provider.Reference) (uint64, uint64, uint64, error) {
	// TODO quota of which storage space?
	// we could use the logged in user, but when a user has access to multiple storages this falls short
	// for now return quota of root
	var free, total, avail uint64

	pathPtr, err := windows.UTF16PtrFromString(fs.wrap(ctx, "/"))
	if err != nil {
		return 0, 0, 0, err
	}
	err = windows.GetDiskFreeSpaceEx(pathPtr, &avail, &total, &free)
	if err != nil {
		return 0, 0, 0, err
	}

	used := total - free
	return total, used, free, nil
}
