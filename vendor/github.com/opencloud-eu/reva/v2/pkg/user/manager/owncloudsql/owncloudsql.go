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

package owncloudsql

import (
	"context"
	"database/sql"
	"fmt"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/owncloudsql/accounts"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"github.com/pkg/errors"

	// Provides mysql drivers
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	registry.Register("owncloudsql", NewMysql)
}

type manager struct {
	c  *config
	db *accounts.Accounts
}

type config struct {
	DbUsername         string `mapstructure:"dbusername"`
	DbPassword         string `mapstructure:"dbpassword"`
	DbHost             string `mapstructure:"dbhost"`
	DbPort             int    `mapstructure:"dbport"`
	DbName             string `mapstructure:"dbname"`
	Idp                string `mapstructure:"idp"`
	Nobody             int64  `mapstructure:"nobody"`
	JoinUsername       bool   `mapstructure:"join_username"`
	JoinOwnCloudUUID   bool   `mapstructure:"join_ownclouduuid"`
	EnableMedialSearch bool   `mapstructure:"enable_medial_search"`
}

// NewMysql returns a new user manager connection to an owncloud mysql database
func NewMysql(m map[string]interface{}) (user.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		err = errors.Wrap(err, "error creating a new manager")
		return nil, err
	}

	mgr.db, err = accounts.NewMysql(
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mgr.c.DbUsername, mgr.c.DbPassword, mgr.c.DbHost, mgr.c.DbPort, mgr.c.DbName),
		mgr.c.JoinUsername,
		mgr.c.JoinOwnCloudUUID,
		mgr.c.EnableMedialSearch,
	)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}

	if c.Nobody == 0 {
		c.Nobody = 99
	}

	m.c = c
	return nil
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, &c); err != nil {
		return nil, err
	}
	return c, nil
}

func (m *manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	// search via the user_id
	a, err := m.db.GetAccountByClaim(ctx, "userid", uid.OpaqueId)
	if err == sql.ErrNoRows {
		return nil, errtypes.NotFound(uid.OpaqueId)
	}
	return m.convertToCS3User(ctx, a, skipFetchingGroups)
}

func (m *manager) GetUserByClaim(ctx context.Context, claim, value string, skipFetchingGroups bool) (*userpb.User, error) {
	a, err := m.db.GetAccountByClaim(ctx, claim, value)
	if err == sql.ErrNoRows {
		return nil, errtypes.NotFound(claim + "=" + value)
	} else if err != nil {
		return nil, err
	}
	return m.convertToCS3User(ctx, a, skipFetchingGroups)
}

func (m *manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	if tenantID != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in opencloudsql user manager")
	}

	accounts, err := m.db.FindAccounts(ctx, query)
	if err == sql.ErrNoRows {
		return nil, errtypes.NotFound("no users found for " + query)
	} else if err != nil {
		return nil, err
	}

	users := make([]*userpb.User, 0, len(accounts))
	for i := range accounts {
		u, err := m.convertToCS3User(ctx, &accounts[i], skipFetchingGroups)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Interface("account", accounts[i]).Msg("could not convert account, skipping")
			continue
		}
		users = append(users, u)
	}

	return users, nil
}

func (m *manager) GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error) {
	groups, err := m.db.GetAccountGroups(ctx, uid.OpaqueId)
	if err == sql.ErrNoRows {
		return nil, errtypes.NotFound("no groups found for uid " + uid.OpaqueId)
	} else if err != nil {
		return nil, err
	}
	return groups, nil
}

func (m *manager) convertToCS3User(ctx context.Context, a *accounts.Account, skipFetchingGroups bool) (*userpb.User, error) {
	u := &userpb.User{
		Id: &userpb.UserId{
			Idp:      m.c.Idp,
			OpaqueId: a.OwnCloudUUID.String,
			Type:     userpb.UserType_USER_TYPE_PRIMARY,
		},
		Username:    a.Username.String,
		Mail:        a.Email.String,
		DisplayName: a.DisplayName.String,
		//Groups:      groups,
		GidNumber: m.c.Nobody,
		UidNumber: m.c.Nobody,
	}
	// https://github.com/cs3org/reva/pull/4135
	// fall back to userid
	if u.Id.OpaqueId == "" {
		u.Id.OpaqueId = a.UserID
	}
	if u.Username == "" {
		u.Username = u.Id.OpaqueId
	}
	if u.DisplayName == "" {
		u.DisplayName = u.Id.OpaqueId
	}

	if !skipFetchingGroups {
		var err error
		if u.Groups, err = m.GetUserGroups(ctx, u.Id); err != nil {
			return nil, err
		}
	}
	return u, nil
}
