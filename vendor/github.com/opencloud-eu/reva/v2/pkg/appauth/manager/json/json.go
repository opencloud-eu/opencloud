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

package json

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	apppb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appauth"
	"github.com/opencloud-eu/reva/v2/pkg/appauth/manager/registry"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/proto"
)

func init() {
	registry.Register("json", New)
}

type config struct {
	File             string `mapstructure:"file"`
	TokenStrength    int    `mapstructure:"token_strength"`
	PasswordHashCost int    `mapstructure:"password_hash_cost"`
}

type jsonManager struct {
	sync.Mutex
	config *config
	// map[userid][password]AppPassword
	passwords map[string]map[string]*apppb.AppPassword
}

// New returns a new mgr.
func New(m map[string]interface{}) (appauth.Manager, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, errors.Wrap(err, "error creating a new manager")
	}

	c.init()

	// load or create file
	manager, err := loadOrCreate(c.File)
	if err != nil {
		return nil, errors.Wrap(err, "error loading the file containing the application passwords")
	}

	manager.config = c

	return manager, nil
}

func (c *config) init() {
	if c.File == "" {
		c.File = "/var/tmp/reva/appauth.json"
	}
	if c.TokenStrength == 0 {
		c.TokenStrength = 16
	}
	if c.PasswordHashCost == 0 {
		c.PasswordHashCost = 11
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

func loadOrCreate(file string) (*jsonManager, error) {
	stat, err := os.Stat(file)
	if os.IsNotExist(err) || stat.Size() == 0 {
		if err = os.WriteFile(file, []byte("{}"), 0644); err != nil {
			return nil, errors.Wrapf(err, "error creating the file %s", file)
		}
	}

	fd, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening the file %s", file)
	}
	defer fd.Close()

	data, err := io.ReadAll(fd)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading the file %s", file)
	}

	m := &jsonManager{}
	if err = json.Unmarshal(data, &m.passwords); err != nil {
		return nil, errors.Wrapf(err, "error parsing the file %s", file)
	}

	if m.passwords == nil {
		m.passwords = make(map[string]map[string]*apppb.AppPassword)
	}

	return m, nil
}

func (mgr *jsonManager) GenerateAppPassword(ctx context.Context, scope map[string]*authpb.Scope, label string, expiration *typespb.Timestamp) (*apppb.AppPassword, error) {
	token, err := password.Generate(mgr.config.TokenStrength, mgr.config.TokenStrength/2, 0, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new token")
	}
	tokenHashed, err := bcrypt.GenerateFromPassword([]byte(token), mgr.config.PasswordHashCost)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new token")
	}
	userID := ctxpkg.ContextMustGetUser(ctx).GetId()
	ctime := now()

	password := string(tokenHashed)
	appPass := &apppb.AppPassword{
		Password:   password,
		TokenScope: scope,
		Label:      label,
		Expiration: expiration,
		Ctime:      ctime,
		Utime:      ctime,
		User:       userID,
	}
	mgr.Lock()
	defer mgr.Unlock()

	// check if user has some previous password
	if _, ok := mgr.passwords[userID.String()]; !ok {
		mgr.passwords[userID.String()] = make(map[string]*apppb.AppPassword)
	}

	mgr.passwords[userID.String()][password] = appPass

	err = mgr.save()
	if err != nil {
		return nil, errors.Wrap(err, "error saving new token")
	}

	clonedAppPass := proto.Clone(appPass).(*apppb.AppPassword)
	clonedAppPass.Password = token
	return clonedAppPass, nil
}

func (mgr *jsonManager) ListAppPasswords(ctx context.Context) ([]*apppb.AppPassword, error) {
	userID := ctxpkg.ContextMustGetUser(ctx).GetId()
	mgr.Lock()
	defer mgr.Unlock()
	appPasswords := make([]*apppb.AppPassword, 0, len(mgr.passwords[userID.String()]))
	for _, pw := range mgr.passwords[userID.String()] {
		appPasswords = append(appPasswords, pw)
	}
	return appPasswords, nil
}

func (mgr *jsonManager) InvalidateAppPassword(ctx context.Context, password string) error {
	userID := ctxpkg.ContextMustGetUser(ctx).GetId()
	mgr.Lock()
	defer mgr.Unlock()

	// see if user has a list of passwords
	appPasswords, ok := mgr.passwords[userID.String()]
	if !ok || len(appPasswords) == 0 {
		return errtypes.NotFound("password not found")
	}

	if _, ok := appPasswords[password]; !ok {
		return errtypes.NotFound("password not found")
	}
	delete(mgr.passwords[userID.String()], password)

	// if user has 0 passwords, delete user key from state map
	if len(mgr.passwords[userID.String()]) == 0 {
		delete(mgr.passwords, userID.String())
	}

	return mgr.save()
}

func (mgr *jsonManager) GetAppPassword(ctx context.Context, userID *userpb.UserId, password string) (*apppb.AppPassword, error) {
	mgr.Lock()
	defer mgr.Unlock()

	appPassword, ok := mgr.passwords[userID.String()]
	if !ok {
		return nil, errtypes.NotFound("password not found")
	}

	for hash, pw := range appPassword {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err == nil {
			// password found
			if pw.Expiration != nil && pw.Expiration.Seconds != 0 && uint64(time.Now().Unix()) > pw.Expiration.Seconds {
				// password expired
				return nil, errtypes.NotFound("password not found")
			}
			// password not expired
			// update last used time
			pw.Utime = now()
			if err := mgr.save(); err != nil {
				return nil, errors.Wrap(err, "error saving file")
			}

			return pw, nil
		}
	}

	return nil, errtypes.NotFound("password not found")
}

func now() *typespb.Timestamp {
	return &typespb.Timestamp{Seconds: uint64(time.Now().Unix())}
}

func (mgr *jsonManager) save() error {
	data, err := json.Marshal(mgr.passwords)
	if err != nil {
		return errors.Wrap(err, "error encoding json file")
	}

	if err = os.WriteFile(mgr.config.File, data, 0644); err != nil {
		return errors.Wrapf(err, "error writing to file %s", mgr.config.File)
	}

	return nil
}
