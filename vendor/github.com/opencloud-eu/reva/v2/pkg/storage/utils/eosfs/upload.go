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

package eosfs

import (
	"context"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
)

func (fs *eosfs) MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error {
	return errtypes.NotSupported("op not supported")
}

func (fs *eosfs) CommitUpload(_ context.Context, _ *provider.Reference, _ string, _ storage.UploadSource) error {
	return errtypes.NotSupported("op not supported")
}

func (fs *eosfs) PrepareUpload(_ context.Context, _ *provider.Reference, _ string, info storage.UploadInfo) (*storage.PrepareUploadResult, error) {
	return &storage.PrepareUploadResult{VersionCreated: info.NodeExisted}, nil
}

func (fs *eosfs) RollbackUpload(_ context.Context, _ *provider.Reference, _ string, _ storage.RollbackInfo) error {
	return nil
}
