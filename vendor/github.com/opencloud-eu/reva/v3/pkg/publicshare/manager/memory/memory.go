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

package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/publicshare"
	"github.com/opencloud-eu/reva/v3/pkg/publicshare/manager/registry"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

func init() {
	registry.Register("memory", New)
}

// New returns a new memory manager.
func New(c map[string]interface{}) (publicshare.Manager, error) {
	return &manager{
		shares: sync.Map{},
	}, nil
}

type manager struct {
	shares sync.Map
}

var (
	passwordProtected bool
)

// CreatePublicShare adds a new entry to manager.shares
func (m *manager) CreatePublicShare(ctx context.Context, u *user.User, rInfo *provider.ResourceInfo, g *link.Grant) (*link.PublicShare, error) {
	id := &link.PublicShareId{
		OpaqueId: randString(15),
	}

	tkn := randString(15)
	now := uint64(time.Now().Unix())

	displayName, ok := rInfo.ArbitraryMetadata.Metadata["name"]
	if !ok {
		displayName = tkn
	}

	if g.Password != "" {
		passwordProtected = true
	}

	createdAt := &typespb.Timestamp{
		Seconds: now,
		Nanos:   uint32(now % 1000000000),
	}

	modifiedAt := &typespb.Timestamp{
		Seconds: now,
		Nanos:   uint32(now % 1000000000),
	}

	s := link.PublicShare{
		Id:                id,
		Owner:             rInfo.GetOwner(),
		Creator:           u.Id,
		ResourceId:        rInfo.Id,
		Token:             tkn,
		Permissions:       g.Permissions,
		Ctime:             createdAt,
		Mtime:             modifiedAt,
		PasswordProtected: passwordProtected,
		Expiration:        g.Expiration,
		DisplayName:       displayName,
	}

	m.shares.Store(s.Token, &s)
	return &s, nil
}

// UpdatePublicShare updates the expiration date, permissions and Mtime
func (m *manager) UpdatePublicShare(ctx context.Context, u *user.User, req *link.UpdatePublicShareRequest) (*link.PublicShare, error) {
	log := appctx.GetLogger(ctx)
	share, err := m.GetPublicShare(ctx, u, req.Ref, false)
	if err != nil {
		return nil, errors.New("ref does not exist")
	}

	token := share.GetToken()

	switch req.GetUpdate().GetType() {
	case link.UpdatePublicShareRequest_Update_TYPE_DISPLAYNAME:
		log.Debug().Str("memory", "update display name").Msgf("from: `%v` to `%v`", share.DisplayName, req.Update.GetDisplayName())
		share.DisplayName = req.Update.GetDisplayName()
	case link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS:
		old, _ := json.Marshal(share.Permissions)
		new, _ := json.Marshal(req.Update.GetGrant().Permissions)
		log.Debug().Str("memory", "update grants").Msgf("from: `%v`\nto\n`%v`", old, new)
		share.Permissions = req.Update.GetGrant().GetPermissions()
	case link.UpdatePublicShareRequest_Update_TYPE_EXPIRATION:
		old, _ := json.Marshal(share.Expiration)
		new, _ := json.Marshal(req.Update.GetGrant().Expiration)
		log.Debug().Str("memory", "update expiration").Msgf("from: `%v`\nto\n`%v`", old, new)
		share.Expiration = req.Update.GetGrant().Expiration
	case link.UpdatePublicShareRequest_Update_TYPE_PASSWORD:
		// TODO(refs) Do public shares need Grants? Struct is defined, just not used. Fill this once it's done.
		fallthrough
	default:
		return nil, fmt.Errorf("invalid update type: %v", req.GetUpdate().GetType())
	}

	// share.Expiration = g.Expiration
	share.Mtime = &typespb.Timestamp{
		Seconds: uint64(time.Now().Unix()),
		Nanos:   uint32(time.Now().Unix() % 1000000000),
	}

	m.shares.Store(token, share)

	return share, nil
}

func (m *manager) GetPublicShare(ctx context.Context, u *user.User, ref *link.PublicShareReference, sign bool) (share *link.PublicShare, err error) {
	// TODO(refs) return an error if the share is expired.

	// Attempt to fetch public share by token
	if ref.GetToken() != "" {
		share, err = m.GetPublicShareByToken(ctx, ref.GetToken(), &link.PublicShareAuthentication{}, sign)
		if err != nil {
			return nil, errors.New("no shares found by token")
		}
	}

	// Attempt to fetch public share by Id
	if ref.GetId() != nil {
		share, err = m.getPublicShareByTokenID(ctx, ref.GetId())
		if err != nil {
			return nil, errors.New("no shares found by id")
		}
	}

	return
}

func (m *manager) ListPublicShares(ctx context.Context, u *user.User, filters []*link.ListPublicSharesRequest_Filter, sign bool) ([]*link.PublicShare, error) {
	// TODO(refs) filter out expired shares
	shares := []*link.PublicShare{}
	m.shares.Range(func(k, v interface{}) bool {
		s := v.(*link.PublicShare)
		if len(filters) == 0 {
			shares = append(shares, s)
		} else {
			for _, f := range filters {
				if f.Type == link.ListPublicSharesRequest_Filter_TYPE_RESOURCE_ID {
					if utils.ResourceIDEqual(s.ResourceId, f.GetResourceId()) {
						shares = append(shares, s)
					}
				}
			}
		}
		return true
	})

	return shares, nil
}

func (m *manager) RevokePublicShare(ctx context.Context, u *user.User, ref *link.PublicShareReference) error {
	// check whether the reference exists
	switch {
	case ref.GetId() != nil && ref.GetId().OpaqueId != "":
		s, err := m.getPublicShareByTokenID(ctx, ref.GetId())
		if err != nil {
			return errors.New("reference does not exist")
		}
		m.shares.Delete(s.Token)
	case ref.GetToken() != "":
		if _, err := m.GetPublicShareByToken(ctx, ref.GetToken(), &link.PublicShareAuthentication{}, false); err != nil {
			return errors.New("reference does not exist")
		}
		m.shares.Delete(ref.GetToken())
	default:
		return errors.New("reference does not exist")
	}
	return nil
}

func (m *manager) GetPublicShareByToken(ctx context.Context, token string, auth *link.PublicShareAuthentication, sign bool) (*link.PublicShare, error) {
	if ps, ok := m.shares.Load(token); ok {
		return ps.(*link.PublicShare), nil
	}
	return nil, errtypes.NotFound("invalid token")
}

func randString(n int) string {
	var l = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = l[rand.Intn(len(l))]
	}
	return string(b)
}

func (m *manager) getPublicShareByTokenID(ctx context.Context, targetID *link.PublicShareId) (*link.PublicShare, error) {
	var found *link.PublicShare
	m.shares.Range(func(k, v interface{}) bool {
		id := v.(*link.PublicShare).GetId()
		if targetID.String() == id.String() {
			found = v.(*link.PublicShare)
		}
		return true
	})

	if found != nil {
		return found, nil
	}
	return nil, errors.New("resource not found")
}
