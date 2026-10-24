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

package publicshare

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	// StorageIDFilterType defines a new filter type for storage id.
	// TODO: Remove this once the filter type is in the CS3 API.
	StorageIDFilterType link.ListPublicSharesRequest_Filter_Type = 4
)

// Manager manipulates public shares.
type Manager interface {
	CreatePublicShare(ctx context.Context, u *user.User, md *provider.ResourceInfo, g *link.Grant) (*link.PublicShare, error)
	UpdatePublicShare(ctx context.Context, u *user.User, req *link.UpdatePublicShareRequest) (*link.PublicShare, error)
	GetPublicShare(ctx context.Context, u *user.User, ref *link.PublicShareReference, sign bool) (*link.PublicShare, error)
	ListPublicShares(ctx context.Context, u *user.User, filters []*link.ListPublicSharesRequest_Filter, sign bool) ([]*link.PublicShare, error)
	RevokePublicShare(ctx context.Context, u *user.User, ref *link.PublicShareReference) error
	GetPublicShareByToken(ctx context.Context, token string, auth *link.PublicShareAuthentication, sign bool) (*link.PublicShare, error)
}

// WithPassword holds the relevant information for representing a public share
type WithPassword struct {
	Password    string `json:"password"`
	PublicShare link.PublicShare
}

// DumpableManager defines a share manager which supports dumping its contents
type DumpableManager interface {
	Dump(ctx context.Context, shareChan chan<- *WithPassword) error
}

// LoadableManager defines a share manager which supports loading contents from a dump
type LoadableManager interface {
	Load(ctx context.Context, shareChan <-chan *WithPassword) error
}

// CreateSignature calculates a signature for a public share.
func CreateSignature(token, pw string, expiration time.Time) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(pw))
	if err != nil {
		return "", err
	}

	key := make([]byte, 0, 32)
	key = h.Sum(key)

	mac := hmac.New(sha512.New512_256, key)
	_, err = mac.Write([]byte(token + "|" + expiration.Format(time.RFC3339)))
	if err != nil {
		return "", err
	}

	sig := make([]byte, 0, 32)
	sig = mac.Sum(sig)

	return hex.EncodeToString(sig), nil
}

// AddSignature augments a public share with a signature.
// The signature has a validity of 30 minutes.
func AddSignature(share *link.PublicShare, pw string) error {
	expiration := time.Now().Add(time.Minute * 30)
	sig, err := CreateSignature(share.Token, pw, expiration)
	if err != nil {
		return err
	}

	share.Signature = &link.ShareSignature{
		Signature: sig,
		SignatureExpiration: &typesv1beta1.Timestamp{
			Seconds: uint64(expiration.UnixNano() / 1000000000),
			Nanos:   uint32(expiration.UnixNano() % 1000000000),
		},
	}
	return nil
}

// ResourceIDFilter is an abstraction for creating filter by resource id.
func ResourceIDFilter(id *provider.ResourceId) *link.ListPublicSharesRequest_Filter {
	return &link.ListPublicSharesRequest_Filter{
		Type: link.ListPublicSharesRequest_Filter_TYPE_RESOURCE_ID,
		Term: &link.ListPublicSharesRequest_Filter_ResourceId{
			ResourceId: id,
		},
	}
}

// StorageIDFilter is an abstraction for creating filter by storage id.
func StorageIDFilter(id string) *link.ListPublicSharesRequest_Filter {
	return &link.ListPublicSharesRequest_Filter{
		Type: StorageIDFilterType,
		Term: &link.ListPublicSharesRequest_Filter_ResourceId{
			ResourceId: &provider.ResourceId{
				StorageId: id,
			},
		},
	}
}

// MatchesFilter tests if the share passes the filter.
func MatchesFilter(share *link.PublicShare, filter *link.ListPublicSharesRequest_Filter) bool {
	switch filter.Type {
	case link.ListPublicSharesRequest_Filter_TYPE_RESOURCE_ID:
		return utils.ResourceIDEqual(share.ResourceId, filter.GetResourceId())
	case StorageIDFilterType:
		return share.GetResourceId().GetStorageId() == filter.GetResourceId().GetStorageId()
	default:
		return false
	}
}

// MatchesAnyFilter checks if the share passes at least one of the given filters.
func MatchesAnyFilter(share *link.PublicShare, filters []*link.ListPublicSharesRequest_Filter) bool {
	for _, f := range filters {
		if MatchesFilter(share, f) {
			return true
		}
	}
	return false
}

// MatchesFilters checks if the share passes the given filters.
// Filters of the same type form a disjuntion, a logical OR. Filters of separate type form a conjunction, a logical AND.
// Here is an example:
// (resource_id=1 OR resource_id=2) AND (grantee_type=USER OR grantee_type=GROUP)
func MatchesFilters(share *link.PublicShare, filters []*link.ListPublicSharesRequest_Filter) bool {
	if len(filters) == 0 {
		return true
	}
	grouped := GroupFiltersByType(filters)
	for _, f := range grouped {
		if !MatchesAnyFilter(share, f) {
			return false
		}
	}
	return true
}

// GroupFiltersByType groups the given filters and returns a map using the filter type as the key.
func GroupFiltersByType(filters []*link.ListPublicSharesRequest_Filter) map[link.ListPublicSharesRequest_Filter_Type][]*link.ListPublicSharesRequest_Filter {
	grouped := make(map[link.ListPublicSharesRequest_Filter_Type][]*link.ListPublicSharesRequest_Filter)
	for _, f := range filters {
		grouped[f.Type] = append(grouped[f.Type], f)
	}
	return grouped
}

// IsExpired tests whether a public share is expired
func IsExpired(s *link.PublicShare) bool {
	expiration := time.Unix(int64(s.Expiration.GetSeconds()), int64(s.Expiration.GetNanos()))
	return s.Expiration != nil && expiration.Before(time.Now())
}

// Authenticate checks the signature or password authentication for a public share
func Authenticate(share *link.PublicShare, pw string, auth *link.PublicShareAuthentication) bool {
	switch {
	case auth.GetPassword() != "":
		if err := bcrypt.CompareHashAndPassword([]byte(pw), []byte(auth.GetPassword())); err == nil {
			return true
		}
	case auth.GetSignature() != nil:
		sig := auth.GetSignature()
		now := time.Now()
		expiration := time.Unix(int64(sig.GetSignatureExpiration().GetSeconds()), int64(sig.GetSignatureExpiration().GetNanos()))
		if now.After(expiration) {
			return false
		}
		s, err := CreateSignature(share.Token, pw, expiration)
		if err != nil {
			return false
		}
		return sig.GetSignature() == s
	}
	return false
}

// IsCreatedByUser checks if a share was created by the user.
func IsCreatedByUser(share *link.PublicShare, user *user.User) bool {
	return utils.UserEqual(user.Id, share.Owner) || utils.UserEqual(user.Id, share.Creator)
}

// IsWriteable checks if the grant for a publicshare allows writes or uploads.
func IsWriteable(perm *link.PublicSharePermissions) bool {
	p := perm.GetPermissions()
	return p != nil && (p.CreateContainer || p.Delete || p.InitiateFileUpload ||
		p.Move || p.AddGrant || p.PurgeRecycle || p.RestoreFileVersion || p.RestoreRecycleItem)
}
