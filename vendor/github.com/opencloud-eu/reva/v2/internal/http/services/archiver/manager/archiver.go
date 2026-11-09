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

package manager

import (
	"archive/tar"
	"archive/zip"
	"context"
	"io"
	"path/filepath"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/downloader"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/walker"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Config is the config for the Archiver
type Config struct {
	MaxNumFiles int64
	MaxSize     int64
}

// Archiver is the struct able to create an archive
type Archiver struct {
	resources  []*provider.ResourceId
	walker     walker.Walker
	downloader downloader.Downloader
	config     Config
}

// NewArchiver creates a new archiver able to create an archive containing the files in the list
func NewArchiver(r []*provider.ResourceId, w walker.Walker, d downloader.Downloader, config Config) (*Archiver, error) {
	if len(r) == 0 {
		return nil, ErrEmptyList{}
	}

	arc := &Archiver{
		resources:  r,
		walker:     w,
		downloader: d,
		config:     config,
	}
	return arc, nil
}

// CreateTar creates a tar and write it into the dst Writer
func (a *Archiver) CreateTar(ctx context.Context, dst io.Writer) (func(), error) {
	w := tar.NewWriter(dst)
	closer := func() {
		_ = w.Close()
	}

	var filesCount, sizeFiles int64

	for _, root := range a.resources {

		err := a.walker.Walk(auth.NewStaticSession(ctx), root, func(wd string, info *provider.ResourceInfo, err error) error {
			if err != nil {
				return err
			}

			// when archiving a space we can omit the spaceroot
			if utils.IsSpaceRoot(info) {
				return nil
			}

			isDir := info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER

			filesCount++
			if filesCount > a.config.MaxNumFiles {
				return ErrMaxFileCount{}
			}

			if !isDir {
				// only add the size if the resource is not a directory
				// as its size could be resursive-computed, and we would
				// count the files not only once
				sizeFiles += int64(info.Size)
				if sizeFiles > a.config.MaxSize {
					return ErrMaxSize{}
				}
			}

			header := tar.Header{
				Name:    filepath.Join(wd, info.Path),
				ModTime: time.Unix(int64(info.Mtime.Seconds), 0),
			}

			if isDir {
				// the resource is a folder
				header.Mode = 0755
				header.Typeflag = tar.TypeDir
			} else {
				header.Mode = 0644
				header.Typeflag = tar.TypeReg
				header.Size = int64(info.Size)
			}

			err = w.WriteHeader(&header)
			if err != nil {
				return err
			}

			if !isDir {
				err = a.downloader.Download(ctx, info.Id, w)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return closer, err
		}

	}
	return closer, nil
}

// CreateZip creates a zip and write it into the dst Writer
func (a *Archiver) CreateZip(ctx context.Context, dst io.Writer) (func(), error) {
	w := zip.NewWriter(dst)
	closer := func() {
		_ = w.Close()
	}

	var filesCount, sizeFiles int64

	for _, root := range a.resources {

		err := a.walker.Walk(auth.NewStaticSession(ctx), root, func(wd string, info *provider.ResourceInfo, err error) error {
			if err != nil {
				return err
			}

			// when archiving a space we can omit the spaceroot
			if utils.IsSpaceRoot(info) {
				return nil
			}

			isDir := info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER

			filesCount++
			if filesCount > a.config.MaxNumFiles {
				return ErrMaxFileCount{}
			}

			if !isDir {
				// only add the size if the resource is not a directory
				// as its size could be resursive-computed, and we would
				// count the files not only once
				sizeFiles += int64(info.Size)
				if sizeFiles > a.config.MaxSize {
					return ErrMaxSize{}
				}
			}

			header := zip.FileHeader{
				Name:     filepath.Join(wd, info.Path),
				Modified: time.Unix(int64(info.Mtime.Seconds), 0),
			}

			if isDir {
				header.Name += "/"
			} else {
				header.UncompressedSize64 = info.Size
			}

			dst, err := w.CreateHeader(&header)
			if err != nil {
				return err
			}

			if !isDir {
				err = a.downloader.Download(ctx, info.Id, dst)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return closer, err
		}

	}
	return closer, nil
}
