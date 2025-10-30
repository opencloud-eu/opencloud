// Copyright 2021 CERN
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

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/go-ldap/ldap/v3"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	ldapReconnect "github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"github.com/pkg/errors"
)

// LDAPConn holds the basic parameter for setting up an
// LDAP connection.
type LDAPConn struct {
	URI          string `mapstructure:"uri"`
	Insecure     bool   `mapstructure:"insecure"`
	CACert       string `mapstructure:"cacert"`
	BindDN       string `mapstructure:"bind_username"`
	BindPassword string `mapstructure:"bind_password"`
}

// GetLDAPClientWithReconnect initializes a long-lived LDAP connection that
// automatically reconnects on connection errors. It allows to set TLS options
// e.g. to add trusted Certificates or disable Certificate verification
func GetLDAPClientWithReconnect(c *LDAPConn) (ldap.Client, error) {
	var tlsConf *tls.Config
	if c.Insecure {
		logger.New().Warn().Msg("SSL Certificate verification is disabled. This is strongly discouraged for production environments.")
		tlsConf = &tls.Config{
			//nolint:gosec // We need the ability to run with "insecure" (dev/testing)
			InsecureSkipVerify: true,
		}
	}
	if !c.Insecure && c.CACert != "" {
		if pemBytes, err := os.ReadFile(c.CACert); err == nil {
			rpool, _ := x509.SystemCertPool()
			rpool.AppendCertsFromPEM(pemBytes)
			tlsConf = &tls.Config{
				RootCAs: rpool,
			}
		} else {
			return nil, errors.Wrapf(err, "Error reading LDAP CA Cert '%s.'", c.CACert)
		}
	}

	conn := ldapReconnect.NewLDAPWithReconnect(
		ldapReconnect.Config{
			URI:          c.URI,
			BindDN:       c.BindDN,
			BindPassword: c.BindPassword,
			TLSConfig:    tlsConf,
		},
	)
	return conn, nil
}

// GetLDAPClientForAuth initializes an LDAP connection. The connection is not authenticated
// when returned. The main purpose for GetLDAPClientForAuth is to get and LDAP connection that
// can be used to issue a single bind request to authenticate a user.
func GetLDAPClientForAuth(ctx context.Context, c *LDAPConn) (ldap.Client, error) {
	var tlsConf *tls.Config
	if c.Insecure {
		appctx.GetLogger(ctx).Warn().Msg("SSL Certificate verification is disabled. Is is strongly discouraged for production environments.")
		tlsConf = &tls.Config{
			//nolint:gosec // We need the ability to run with "insecure" (dev/testing)
			InsecureSkipVerify: true,
		}
	}
	if !c.Insecure && c.CACert != "" {
		if pemBytes, err := os.ReadFile(c.CACert); err == nil {
			rpool, _ := x509.SystemCertPool()
			rpool.AppendCertsFromPEM(pemBytes)
			tlsConf = &tls.Config{
				RootCAs: rpool,
			}
		} else {
			return nil, errors.Wrapf(err, "Error reading LDAP CA Cert '%s.'", c.CACert)
		}
	}
	l, err := ldap.DialURL(c.URI, ldap.DialWithTLSConfig(tlsConf))
	if err != nil {
		return nil, err
	}

	return l, nil
}
