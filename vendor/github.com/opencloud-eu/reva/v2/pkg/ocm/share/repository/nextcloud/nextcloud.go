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

// Package nextcloud verifies a clientID and clientSecret against a Nextcloud backend.
package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"

	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share/repository/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/cfg"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
)

func init() {
	registry.Register("nextcloud", New)
}

// Manager is the Nextcloud-based implementation of the share.Repository interface
// see https://github.com/cs3org/reva/blob/v1.13.0/pkg/ocm/share/share.go#L30-L57
type Manager struct {
	client       *http.Client
	sharedSecret string
	webDAVHost   string
	endPoint     string
}

// ShareManagerConfig contains config for a Nextcloud-based ShareManager.
type ShareManagerConfig struct {
	EndPoint     string `mapstructure:"endpoint" docs:";The Nextcloud backend endpoint for user check"`
	SharedSecret string `mapstructure:"shared_secret"`
	WebDAVHost   string `mapstructure:"webdav_host"`
	MockHTTP     bool   `mapstructure:"mock_http"`
}

// Action describes a REST request to forward to the Nextcloud backend.
type Action struct {
	verb string
	argS string
}

// GranteeAltMap is an alternative map to JSON-unmarshal a Grantee
// Grantees are hard to unmarshal, so unmarshalling into a map[string]interface{} first,
// see also https://github.com/pondersource/sciencemesh-nextcloud/issues/27
type GranteeAltMap struct {
	ID *provider.Grantee_UserId `json:"id"`
}

// ShareAltMap is an alternative map to JSON-unmarshal a Share.
type ShareAltMap struct {
	ID            *ocm.ShareId          `json:"id"`
	RemoteShareID string                `json:"remote_share_id"`
	Permissions   *ocm.SharePermissions `json:"permissions"`
	Grantee       *GranteeAltMap        `json:"grantee"`
	Owner         *userpb.UserId        `json:"owner"`
	Creator       *userpb.UserId        `json:"creator"`
	Ctime         *typespb.Timestamp    `json:"ctime"`
	Mtime         *typespb.Timestamp    `json:"mtime"`
}

// ReceivedShareAltMap is an alternative map to JSON-unmarshal a ReceivedShare.
type ReceivedShareAltMap struct {
	Share *ShareAltMap   `json:"share"`
	State ocm.ShareState `json:"state"`
}

// New returns a share manager implementation that verifies against a Nextcloud backend.
func New(m map[string]interface{}) (share.Repository, error) {
	var c ShareManagerConfig
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	return NewShareManager(&c)
}

// NewShareManager returns a new Nextcloud-based ShareManager.
func NewShareManager(c *ShareManagerConfig) (*Manager, error) {
	var client *http.Client
	if c.MockHTTP {
		// called := make([]string, 0)
		// nextcloudServerMock := GetNextcloudServerMock(&called)
		// client, _ = TestingHTTPClient(nextcloudServerMock)

		// Wait for SetHTTPClient to be called later
		client = nil
	} else {
		if len(c.EndPoint) == 0 {
			return nil, errors.New("Please specify 'endpoint' in '[grpc.services.ocmshareprovider.drivers.nextcloud]' and  '[grpc.services.ocmcore.drivers.nextcloud]'")
		}
		client = &http.Client{}
	}

	return &Manager{
		endPoint:     c.EndPoint, // e.g. "http://nc/apps/sciencemesh/"
		sharedSecret: c.SharedSecret,
		client:       client,
		webDAVHost:   c.WebDAVHost,
	}, nil
}

// SetHTTPClient sets the HTTP client.
func (sm *Manager) SetHTTPClient(c *http.Client) {
	sm.client = c
}

// StoreShare stores a share.
func (sm *Manager) StoreShare(ctx context.Context, share *ocm.Share) (*ocm.Share, error) {
	encShare, err := utils.MarshalProtoV1ToJSON(share)
	if err != nil {
		return nil, err
	}
	_, body, err := sm.do(ctx, Action{"addSentShare", string(encShare)}, getUsername(&userpb.User{Id: share.Creator}))
	if err != nil {
		return nil, err
	}
	share.Id = &ocm.ShareId{
		OpaqueId: string(body),
	}
	return share, nil
}

// GetShare gets the information for a share by the given ref.
func (sm *Manager) GetShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.Share, error) {
	data, err := json.Marshal(ref)
	if err != nil {
		return nil, err
	}
	_, body, err := sm.do(ctx, Action{"GetShare", string(data)}, getUsername(user))
	if err != nil {
		return nil, err
	}

	altResult := &ShareAltMap{}
	if err := json.Unmarshal(body, &altResult); err != nil {
		return nil, err
	}
	return &ocm.Share{
		Id: altResult.ID,
		Grantee: &provider.Grantee{
			Id: altResult.Grantee.ID,
		},
		Owner:   altResult.Owner,
		Creator: altResult.Creator,
		Ctime:   altResult.Ctime,
		Mtime:   altResult.Mtime,
	}, nil
}

// DeleteShare deletes the share pointed by ref.
func (sm *Manager) DeleteShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) error {
	bodyStr, err := json.Marshal(ref)
	if err != nil {
		return err
	}

	_, _, err = sm.do(ctx, Action{"Unshare", string(bodyStr)}, getUsername(user))
	return err
}

// UpdateShare updates the mode of the given share.
func (sm *Manager) UpdateShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference, f ...*ocm.UpdateOCMShareRequest_UpdateField) (*ocm.Share, error) {
	type paramsObj struct {
		Ref *ocm.ShareReference   `json:"ref"`
		P   *ocm.SharePermissions `json:"p"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
	}
	data, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, err
	}

	_, body, err := sm.do(ctx, Action{"UpdateShare", string(data)}, getUsername(user))
	if err != nil {
		return nil, err
	}

	altResult := &ShareAltMap{}
	if err := json.Unmarshal(body, &altResult); err != nil {
		return nil, err
	}
	return &ocm.Share{
		Id: altResult.ID,
		Grantee: &provider.Grantee{
			Id: altResult.Grantee.ID,
		},
		Owner:   altResult.Owner,
		Creator: altResult.Creator,
		Ctime:   altResult.Ctime,
		Mtime:   altResult.Mtime,
	}, nil
}

// ListShares returns the shares created by the user. If md is provided is not nil,
// it returns only shares attached to the given resource.
func (sm *Manager) ListShares(ctx context.Context, user *userpb.User, filters []*ocm.ListOCMSharesRequest_Filter) ([]*ocm.Share, error) {
	data, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	_, respBody, err := sm.do(ctx, Action{"ListShares", string(data)}, getUsername(user))
	if err != nil {
		return nil, err
	}

	var respArr []ShareAltMap
	if err := json.Unmarshal(respBody, &respArr); err != nil {
		return nil, err
	}

	var lst = make([]*ocm.Share, 0, len(respArr))
	for _, altResult := range respArr {
		lst = append(lst, &ocm.Share{
			Id: altResult.ID,
			Grantee: &provider.Grantee{
				Id: altResult.Grantee.ID,
			},
			Owner:   altResult.Owner,
			Creator: altResult.Creator,
			Ctime:   altResult.Ctime,
			Mtime:   altResult.Mtime,
		})
	}
	return lst, nil
}

// StoreReceivedShare stores a received share.
func (sm *Manager) StoreReceivedShare(ctx context.Context, share *ocm.ReceivedShare) (*ocm.ReceivedShare, error) {
	data, err := utils.MarshalProtoV1ToJSON(share)
	if err != nil {
		return nil, err
	}
	_, body, err := sm.do(ctx, Action{"addReceivedShare", string(data)}, getUsername(&userpb.User{Id: share.Grantee.GetUserId()}))
	if err != nil {
		return nil, err
	}
	share.Id = &ocm.ShareId{
		OpaqueId: string(body),
	}

	return share, nil
}

// ListReceivedShares returns the list of shares the user has access.
func (sm *Manager) ListReceivedShares(ctx context.Context, user *userpb.User) ([]*ocm.ReceivedShare, error) {
	log := appctx.GetLogger(ctx)
	_, respBody, err := sm.do(ctx, Action{"ListReceivedShares", ""}, getUsername(user))
	if err != nil {
		return nil, err
	}

	var respArr []ReceivedShareAltMap
	if err := json.Unmarshal(respBody, &respArr); err != nil {
		return nil, err
	}

	res := make([]*ocm.ReceivedShare, 0, len(respArr))
	for _, share := range respArr {
		altResultShare := share.Share
		log.Info().Msgf("Unpacking share object %+v\n", altResultShare)
		if altResultShare == nil {
			continue
		}
		res = append(res, &ocm.ReceivedShare{
			Id:            altResultShare.ID,
			RemoteShareId: altResultShare.RemoteShareID, // sic, see https://github.com/cs3org/reva/pull/3852#discussion_r1189681465
			Grantee: &provider.Grantee{
				Id: altResultShare.Grantee.ID,
			},
			Owner:   altResultShare.Owner,
			Creator: altResultShare.Creator,
			Ctime:   altResultShare.Ctime,
			Mtime:   altResultShare.Mtime,
			State:   share.State,
		})
	}
	return res, nil
}

// GetReceivedShare returns the information for a received share the user has access.
func (sm *Manager) GetReceivedShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.ReceivedShare, error) {
	data, err := json.Marshal(ref)
	if err != nil {
		return nil, err
	}

	_, respBody, err := sm.do(ctx, Action{"GetReceivedShare", string(data)}, getUsername(user))
	if err != nil {
		return nil, err
	}

	var altResult ReceivedShareAltMap
	if err := json.Unmarshal(respBody, &altResult); err != nil {
		return nil, err
	}
	altResultShare := altResult.Share
	if altResultShare == nil {
		return &ocm.ReceivedShare{
			State: altResult.State,
		}, nil
	}
	return &ocm.ReceivedShare{
		Id:            altResultShare.ID,
		RemoteShareId: altResultShare.RemoteShareID, // sic, see https://github.com/cs3org/reva/pull/3852#discussion_r1189681465
		Grantee: &provider.Grantee{
			Id: altResultShare.Grantee.ID,
		},
		Owner:   altResultShare.Owner,
		Creator: altResultShare.Creator,
		Ctime:   altResultShare.Ctime,
		Mtime:   altResultShare.Mtime,
		State:   altResult.State,
	}, nil
}

// UpdateReceivedShare updates the received share with share state.
func (sm *Manager) UpdateReceivedShare(ctx context.Context, user *userpb.User, share *ocm.ReceivedShare, fieldMask *field_mask.FieldMask) (*ocm.ReceivedShare, error) {
	type paramsObj struct {
		ReceivedShare *ocm.ReceivedShare    `json:"received_share"`
		FieldMask     *field_mask.FieldMask `json:"field_mask"`
	}

	bodyObj := &paramsObj{
		ReceivedShare: share,
		FieldMask:     fieldMask,
	}
	bodyStr, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, err
	}

	_, respBody, err := sm.do(ctx, Action{"UpdateReceivedShare", string(bodyStr)}, getUsername(user))
	if err != nil {
		return nil, err
	}

	var altResult ReceivedShareAltMap
	err = json.Unmarshal(respBody, &altResult)
	if err != nil {
		return nil, err
	}
	altResultShare := altResult.Share
	if altResultShare == nil {
		return &ocm.ReceivedShare{
			State: altResult.State,
		}, nil
	}
	return &ocm.ReceivedShare{
		Id:            altResultShare.ID,
		RemoteShareId: altResultShare.RemoteShareID, // sic, see https://github.com/cs3org/reva/pull/3852#discussion_r1189681465
		Grantee: &provider.Grantee{
			Id: altResultShare.Grantee.ID,
		},
		Owner:   altResultShare.Owner,
		Creator: altResultShare.Creator,
		Ctime:   altResultShare.Ctime,
		Mtime:   altResultShare.Mtime,
		State:   altResult.State,
	}, nil
}

func getUsername(user *userpb.User) string {
	if user != nil && len(user.Username) > 0 {
		return user.Username
	}
	if user != nil && len(user.Id.OpaqueId) > 0 {
		return user.Id.OpaqueId
	}

	return "empty-username"
}

func (sm *Manager) do(ctx context.Context, a Action, username string) (int, []byte, error) {
	url := sm.endPoint + "~" + username + "/api/ocm/" + a.verb

	log := appctx.GetLogger(ctx)
	log.Info().Msgf("am.do %s %s", url, a.argS)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(a.argS))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Reva-Secret", sm.sharedSecret)

	req.Header.Set("Content-Type", "application/json")
	resp, err := sm.client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	// curl -i -H 'application/json' -H 'X-Reva-Secret: shared-secret-1' -d '{"md":{"opaque_id":"fileid-/other/q/as"},"g":{"grantee":{"type":1,"Id":{"UserId":{"idp":"revanc2.docker","opaque_id":"marie"}}},"permissions":{"permissions":{"get_path":true,"initiate_file_download":true,"list_container":true,"list_file_versions":true,"stat":true}}},"provider_domain":"cern.ch","resource_type":"file","provider_id":2,"owner_opaque_id":"einstein","owner_display_name":"Albert Einstein","protocol":{"name":"webdav","options":{"sharedSecret":"secret","permissions":"webdav-property"}}}' https://nc1.docker/index.php/apps/sciencemesh/~/api/ocm/addSentShare

	log.Info().Msgf("am.do response %d %s", resp.StatusCode, body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, nil, fmt.Errorf("unexpected response code from EFSS API: %d", resp.StatusCode)
	}
	return resp.StatusCode, body, nil
}
