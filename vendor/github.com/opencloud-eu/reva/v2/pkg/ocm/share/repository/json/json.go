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

package json

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share/repository/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/cfg"
)

func init() {
	registry.Register("json", New)
}

// New returns a new authorizer object.
func New(m map[string]interface{}) (share.Repository, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	// load or create file
	model, err := loadOrCreate(c.File)
	if err != nil {
		err = errors.Wrap(err, "error loading the file containing the shares")
		return nil, err
	}

	mgr := &mgr{
		c:     &c,
		model: model,
	}

	return mgr, nil
}

func loadOrCreate(file string) (*shareModel, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return nil, errors.Wrap(err, "error creating the base directory: "+filepath.Dir(file))
		}
		if err := os.WriteFile(file, []byte("{}"), 0700); err != nil {
			return nil, errors.Wrap(err, "error creating the file: "+file)
		}
	}

	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		err = errors.Wrap(err, "error opening the file: "+file)
		return nil, err
	}
	defer f.Close()

	var m shareModel
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		if err != io.EOF {
			return nil, errors.Wrap(err, "error decoding data to json")
		}
	}

	if m.Shares == nil {
		m.Shares = map[string]*ocm.Share{}
	}
	if m.ReceivedShares == nil {
		m.ReceivedShares = map[string]*ocm.ReceivedShare{}
	}

	return &m, nil
}

type shareModel struct {
	Shares         map[string]*ocm.Share         `json:"shares"`          // share_id -> share
	ReceivedShares map[string]*ocm.ReceivedShare `json:"received_shares"` // share_id -> share
}

func (s *shareModel) UnmarshalJSON(d []byte) error {
	m := struct {
		Shares         map[string]json.RawMessage `json:"shares"`
		ReceivedShares map[string]json.RawMessage `json:"received_shares"`
	}{}

	if err := json.Unmarshal(d, &m); err != nil {
		return err
	}

	share := map[string]*ocm.Share{}
	for k, v := range m.Shares {
		var s ocm.Share
		if err := utils.UnmarshalJSONToProtoV1(v, &s); err != nil {
			return err
		}
		share[k] = &s
	}

	received := map[string]*ocm.ReceivedShare{}
	for k, v := range m.ReceivedShares {
		var s ocm.ReceivedShare
		if err := utils.UnmarshalJSONToProtoV1(v, &s); err != nil {
			return err
		}
		received[k] = &s
	}

	*s = shareModel{
		Shares:         share,
		ReceivedShares: received,
	}

	return nil
}

func (s *shareModel) MarshalJSON() ([]byte, error) {
	shares := map[string]json.RawMessage{}
	for k, v := range s.Shares {
		d, err := utils.MarshalProtoV1ToJSON(v)
		if err != nil {
			return nil, err
		}
		shares[k] = d
	}

	received := map[string]json.RawMessage{}
	for k, v := range s.ReceivedShares {
		d, err := utils.MarshalProtoV1ToJSON(v)
		if err != nil {
			return nil, err
		}
		received[k] = d
	}

	return json.Marshal(map[string]any{
		"shares":          shares,
		"received_shares": received,
	})
}

type config struct {
	File string `mapstructure:"file"`
}

func (c *config) ApplyDefaults() {
	if c.File == "" {
		c.File = "/var/tmp/reva/ocm-shares.json"
	}
}

type mgr struct {
	c          *config
	sync.Mutex // concurrent access to the file
	model      *shareModel
}

func (m *mgr) save() error {
	f, err := os.OpenFile(m.c.File, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "error opening file "+m.c.File)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(m.model); err != nil {
		return errors.Wrap(err, "error encoding to json")
	}

	return f.Sync()
}

func (m *mgr) load() error {
	f, err := os.OpenFile(m.c.File, os.O_RDONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "error opening file "+m.c.File)
	}
	defer f.Close()

	d, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var model shareModel
	if err := json.Unmarshal(d, &model); err != nil {
		return err
	}

	m.model = &model
	return nil
}

func genID() string {
	return uuid.New().String()
}

func (m *mgr) StoreShare(ctx context.Context, ocmshare *ocm.Share) (*ocm.Share, error) {
	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	if _, err := m.getByKey(ctx, &ocm.ShareKey{
		Owner:      ocmshare.Owner,
		ResourceId: ocmshare.ResourceId,
		Grantee:    ocmshare.Grantee,
	}); err == nil {
		return nil, share.ErrShareAlreadyExisting
	}

	ocmshare.Id = &ocm.ShareId{OpaqueId: genID()}
	clone, err := cloneShare(ocmshare)
	if err != nil {
		return nil, err
	}
	m.model.Shares[ocmshare.Id.OpaqueId] = clone

	if err := m.save(); err != nil {
		return nil, errors.Wrap(err, "error saving share")
	}

	return ocmshare, nil
}

func cloneShare(s *ocm.Share) (*ocm.Share, error) {
	d, err := utils.MarshalProtoV1ToJSON(s)
	if err != nil {
		return nil, errtypes.InternalError("failed to marshal ocm share")
	}
	var cloned ocm.Share
	if err := utils.UnmarshalJSONToProtoV1(d, &cloned); err != nil {
		return nil, errtypes.InternalError("failed to unmarshal ocm share")
	}
	return &cloned, nil
}

func cloneReceivedShare(s *ocm.ReceivedShare) (*ocm.ReceivedShare, error) {
	d, err := utils.MarshalProtoV1ToJSON(s)
	if err != nil {
		return nil, errtypes.InternalError("failed to marshal ocm received share")
	}
	var cloned ocm.ReceivedShare
	if err := utils.UnmarshalJSONToProtoV1(d, &cloned); err != nil {
		return nil, errtypes.InternalError("failed to unmarshal ocm received share")
	}
	return &cloned, nil
}

func (m *mgr) GetShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.Share, error) {
	m.Lock()
	defer m.Unlock()

	var (
		s   *ocm.Share
		err error
	)

	if err := m.load(); err != nil {
		return nil, err
	}

	switch {
	case ref.GetId() != nil:
		s, err = m.getByID(ctx, ref.GetId())
	case ref.GetKey() != nil:
		s, err = m.getByKey(ctx, ref.GetKey())
	case ref.GetToken() != "":
		return m.getByToken(ctx, ref.GetToken())
	default:
		err = errtypes.NotFound(ref.String())
	}

	if err != nil {
		return nil, err
	}

	// check if we are the owner
	if utils.UserEqual(user.Id, s.Owner) || utils.UserEqual(user.Id, s.Creator) {
		return s, nil
	}

	return nil, share.ErrShareNotFound
}

func (m *mgr) getByToken(ctx context.Context, token string) (*ocm.Share, error) {
	for _, share := range m.model.Shares {
		if share.Token == token {
			return share, nil
		}
	}
	return nil, errtypes.NotFound(token)
}

func (m *mgr) getByID(ctx context.Context, id *ocm.ShareId) (*ocm.Share, error) {
	if share, ok := m.model.Shares[id.OpaqueId]; ok {
		return share, nil
	}
	return nil, errtypes.NotFound(id.String())
}

func (m *mgr) getByKey(ctx context.Context, key *ocm.ShareKey) (*ocm.Share, error) {
	for _, share := range m.model.Shares {
		if (utils.UserEqual(key.Owner, share.Owner) || utils.UserEqual(key.Owner, share.Creator)) &&
			utils.ResourceIDEqual(key.ResourceId, share.ResourceId) && utils.GranteeEqual(key.Grantee, share.Grantee) {
			return share, nil
		}
	}
	return nil, share.ErrShareNotFound
}

func (m *mgr) DeleteShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) error {
	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return err
	}

	for id, share := range m.model.Shares {
		if sharesEqual(ref, share) {
			if utils.UserEqual(user.Id, share.Owner) || utils.UserEqual(user.Id, share.Creator) {
				delete(m.model.Shares, id)
				return m.save()
			}
		}
	}
	return errtypes.NotFound(ref.String())
}

func sharesEqual(ref *ocm.ShareReference, s *ocm.Share) bool {
	if ref.GetId() != nil && s.Id != nil {
		if ref.GetId().OpaqueId == s.Id.OpaqueId {
			return true
		}
	} else if ref.GetKey() != nil {
		if (utils.UserEqual(ref.GetKey().Owner, s.Owner) || utils.UserEqual(ref.GetKey().Owner, s.Creator)) &&
			utils.ResourceIDEqual(ref.GetKey().ResourceId, s.ResourceId) && utils.GranteeEqual(ref.GetKey().Grantee, s.Grantee) {
			return true
		}
	}
	return false
}

func receivedShareEqual(ref *ocm.ShareReference, s *ocm.ReceivedShare) bool {
	if ref.GetId() != nil && s.Id != nil {
		if ref.GetId().OpaqueId == s.Id.OpaqueId {
			return true
		}
	}
	return false
}

// UpdateShare updates the share with the given fields.
func (m *mgr) UpdateShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference, fields ...*ocm.UpdateOCMShareRequest_UpdateField) (*ocm.Share, error) {
	m.Lock()
	defer m.Unlock()
	if err := m.load(); err != nil {
		return nil, err
	}
	for _, s := range m.model.Shares {
		if sharesEqual(ref, s) {
			if utils.UserEqual(user.Id, s.Owner) || utils.UserEqual(user.Id, s.Creator) {

				for _, f := range fields {
					if exp := f.GetExpiration(); exp != nil {
						s.Expiration = exp
					}
					if am := f.GetAccessMethods(); am != nil {
						var (
							webdavOptions *ocm.WebDAVAccessMethod
							webappOptions *ocm.WebappAccessMethod
							// TODO: *AccessMethod_GenericOptions

							newWebdavOptions *ocm.WebDAVAccessMethod
							newWebappOptions *ocm.WebappAccessMethod
							// TODO: *AccessMethod_GenericOptions
						)

						for _, sm := range s.GetAccessMethods() {
							webdavOptions = sm.GetWebdavOptions()
							webappOptions = sm.GetWebappOptions()
						}

						newWebdavOptions = am.GetWebdavOptions()
						newWebappOptions = am.GetWebappOptions()

						newAccesMethods := []*ocm.AccessMethod{}

						if newWebdavOptions != nil {
							newAccesMethods = append(newAccesMethods, &ocm.AccessMethod{
								Term: &ocm.AccessMethod_WebdavOptions{
									WebdavOptions: newWebdavOptions,
								},
							})
						} else if webdavOptions != nil {
							newAccesMethods = append(newAccesMethods, &ocm.AccessMethod{
								Term: &ocm.AccessMethod_WebdavOptions{
									WebdavOptions: webdavOptions,
								},
							})
						}

						if newWebappOptions != nil {
							newAccesMethods = append(newAccesMethods, &ocm.AccessMethod{
								Term: &ocm.AccessMethod_WebappOptions{
									WebappOptions: newWebappOptions,
								},
							})
						} else if webappOptions != nil {
							newAccesMethods = append(newAccesMethods, &ocm.AccessMethod{
								Term: &ocm.AccessMethod_WebappOptions{
									WebappOptions: webappOptions,
								},
							})
						}

						s.AccessMethods = newAccesMethods
					}
				}

				clone, err := cloneShare(s)
				if err != nil {
					return nil, err
				}
				m.model.Shares[s.Id.OpaqueId] = clone

				if err := m.save(); err != nil {
					return nil, errors.Wrap(err, "error saving share")
				}

				return clone, nil
			}
		}
	}

	return nil, errtypes.NotFound(ref.String())
}

func (m *mgr) ListShares(ctx context.Context, user *userpb.User, filters []*ocm.ListOCMSharesRequest_Filter) ([]*ocm.Share, error) {
	var ss []*ocm.Share

	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	for _, share := range m.model.Shares {
		if utils.UserEqual(user.Id, share.Owner) || utils.UserEqual(user.Id, share.Creator) || utils.UserEqual(user.Id, share.GetGrantee().GetUserId()) {
			// no filter we return earlier
			if len(filters) == 0 {
				ss = append(ss, share)
			} else {
				// check filters
				// TODO(labkode): add the rest of filters.
				for _, f := range filters {
					if f.Type == ocm.ListOCMSharesRequest_Filter_TYPE_RESOURCE_ID {
						if utils.ResourceIDEqual(share.ResourceId, f.GetResourceId()) {
							ss = append(ss, share)
						}
					}
				}
			}
		}
	}
	return ss, nil
}

func (m *mgr) StoreReceivedShare(ctx context.Context, share *ocm.ReceivedShare) (*ocm.ReceivedShare, error) {
	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	ts := &typespb.Timestamp{
		Seconds: uint64(now / 1000000000),
		Nanos:   uint32(now % 1000000000),
	}

	share.Id = &ocm.ShareId{
		OpaqueId: genID(),
	}
	share.Ctime = ts
	share.Mtime = ts

	clone, err := cloneReceivedShare(share)
	if err != nil {
		return nil, err
	}

	m.model.ReceivedShares[share.Id.OpaqueId] = clone
	if err := m.save(); err != nil {
		return nil, err
	}

	return share, nil
}

func (m *mgr) ListReceivedShares(ctx context.Context, user *userpb.User) ([]*ocm.ReceivedShare, error) {
	var rss []*ocm.ReceivedShare
	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	for _, share := range m.model.ReceivedShares {
		if utils.UserEqual(user.Id, share.Owner) || utils.UserEqual(user.Id, share.Creator) {
			// omit shares created by me
			continue
		}

		if share.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER && utils.UserEqual(user.Id, share.Grantee.GetUserId()) {
			rss = append(rss, share)
		}
	}
	return rss, nil
}

func (m *mgr) GetReceivedShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.ReceivedShare, error) {
	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	for _, share := range m.model.ReceivedShares {
		if receivedShareEqual(ref, share) {
			if share.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER && utils.UserEqual(user.Id, share.Grantee.GetUserId()) {
				return share, nil
			}
		}
	}
	return nil, errtypes.NotFound(ref.String())
}

func (m *mgr) UpdateReceivedShare(ctx context.Context, user *userpb.User, share *ocm.ReceivedShare, fieldMask *field_mask.FieldMask) (*ocm.ReceivedShare, error) {
	rs, err := m.GetReceivedShare(ctx, user, &ocm.ShareReference{Spec: &ocm.ShareReference_Id{Id: share.Id}})
	if err != nil {
		return nil, err
	}

	m.Lock()
	defer m.Unlock()

	if err := m.load(); err != nil {
		return nil, err
	}

	for _, mask := range fieldMask.Paths {
		switch mask {
		case "state":
			rs.State = share.State
			m.model.ReceivedShares[share.Id.OpaqueId].State = share.State
		// TODO case "mount_point":
		default:
			return nil, errtypes.NotSupported("updating " + mask + " is not supported")
		}
	}

	if err := m.save(); err != nil {
		return nil, errors.Wrap(err, "error saving model")
	}

	return rs, nil
}
