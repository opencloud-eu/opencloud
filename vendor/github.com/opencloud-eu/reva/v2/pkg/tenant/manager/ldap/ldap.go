// Copyright 2018-2021 CERN
// Copyright 2026 OpenCloud GmbH
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

package ldap

import (
	"context"
	"fmt"

	tenantpb "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	"github.com/go-ldap/ldap/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/tenant"
	"github.com/opencloud-eu/reva/v2/pkg/tenant/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	ldapIdentity "github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("ldap", New)
}

type config struct {
	utils.LDAPConn `mapstructure:",squash"`
	LDAPIdentity   ldapIdentity.Identity `mapstructure:",squash"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{
		LDAPIdentity: ldapIdentity.New(),
	}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

type manager struct {
	conf *config
	ldap ldap.Client
}

// New returns a new user manager.
func New(m map[string]interface{}) (tenant.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}

	mgr.ldap, err = utils.GetLDAPClientWithReconnect(&mgr.conf.LDAPConn)

	return mgr, err
}

func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}
	if err = c.LDAPIdentity.Setup(); err != nil {
		return fmt.Errorf("error setting up Identity config: %w", err)
	}
	m.conf = c

	return nil
}

func (m *manager) GetTenant(ctx context.Context, id string) (*tenantpb.Tenant, error) {
	log := appctx.GetLogger(ctx)

	tenantEntry, err := m.conf.LDAPIdentity.GetLDAPTenantByID(ctx, m.ldap, id)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("entry", tenantEntry).Msg("entries")

	t, err := m.ldapEntryToTenant(tenantEntry)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (m *manager) GetTenantByClaim(ctx context.Context, claim, value string) (*tenantpb.Tenant, error) {
	tenantEntry, err := m.conf.LDAPIdentity.GetLDAPTenantByAttribute(ctx, m.ldap, claim, value)
	if err != nil {
		return nil, err
	}
	return m.ldapEntryToTenant(tenantEntry)
}

func (m *manager) ldapEntryToTenant(entry *ldap.Entry) (*tenantpb.Tenant, error) {
	t := &tenantpb.Tenant{
		Id:         entry.GetEqualFoldAttributeValue(m.conf.LDAPIdentity.Tenant.Schema.ID),
		ExternalId: entry.GetEqualFoldAttributeValue(m.conf.LDAPIdentity.Tenant.Schema.ExternalID),
		Name:       entry.GetEqualFoldAttributeValue(m.conf.LDAPIdentity.Tenant.Schema.Name),
	}
	return t, nil
}
