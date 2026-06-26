/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ldap

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	uuid "github.com/google/uuid"
	"github.com/libregraph/oidc-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/config"
	"github.com/libregraph/lico/identifier/backends"
	"github.com/libregraph/lico/identifier/meta/scopes"
)

const ldapIdentifierBackendName = "identifier-ldap"

var ldapSupportedScopes = []string{
	oidc.ScopeProfile,
	oidc.ScopeEmail,
	konnect.ScopeUniqueUserID,
	konnect.ScopeRawSubject,
}

// LDAPIdentifierBackend is a backend for the Identifier which connects LDAP.
type LDAPIdentifierBackend struct {
	addr         string
	isTLS        bool
	bindDN       string
	bindPassword string

	baseDN       string
	scope        int
	searchFilter string
	getFilter    string

	entryIDMapping   []string
	attributeMapping ldapAttributeMapping
	supportedScopes  []string

	logger    logrus.FieldLogger
	dialer    *net.Dialer
	tlsConfig *tls.Config

	timeout int
	limiter *rate.Limiter
}

type ldapAttributeMapping map[string]string

var ldapDefaultAttributeMapping = ldapAttributeMapping{
	AttributeLogin:                        AttributeLogin,
	AttributeEmail:                        AttributeEmail,
	AttributeName:                         AttributeName,
	AttributeFamilyName:                   AttributeFamilyName,
	AttributeGivenName:                    AttributeGivenName,
	AttributeUUID:                         AttributeUUID,
	fmt.Sprintf("%s_type", AttributeUUID): AttributeValueTypeText,
}

func (m ldapAttributeMapping) attributes() []string {
	attributes := make([]string, len(m)+1)
	attributes[0] = AttributeDN
	idx := 1
	for _, attribute := range m {
		attributes[idx] = attribute
		idx++
	}

	return attributes
}

type ldapUser struct {
	entryID string
	id      int64
	data    ldapAttributeMapping
}

func newLdapUser(entryID string, mapping ldapAttributeMapping, entry *ldap.Entry) (*ldapUser, error) {
	// Go through all returned attributes, add them to the local data set if
	// we know them in the mapping.
	var id int64
	data := make(ldapAttributeMapping)
	for _, attribute := range entry.Attributes {
		if len(attribute.Values) == 0 {
			continue
		}
		for n, mapped := range mapping {
			// LDAP attribute descriptors / short names are case insensitive. See
			// https://tools.ietf.org/html/rfc4512#page-4.
			if strings.ToLower(attribute.Name) == strings.ToLower(mapped) {
				// Check if we need conversion.
				switch mapping[fmt.Sprintf("%s_type", n)] {
				case AttributeValueTypeBinary:
					// Binary gets encoded witih Base64.
					data[n] = base64.StdEncoding.EncodeToString(attribute.ByteValues[0])
				case AttributeValueTypeUUID:
					// Try to decode as UUID https://tools.ietf.org/html/rfc4122 and
					// serialize to string.
					if value, err := uuid.FromBytes(attribute.ByteValues[0]); err == nil {
						data[n] = value.String()
					}
				default:
					data[n] = attribute.Values[0]
				}

				if n == AttributeNumericUID {
					numericID, err := strconv.ParseInt(data[n], 10, 64)
					if err != nil {
						return nil, fmt.Errorf("invalid numeric ID %v in record", err)
					}
					id = numericID
				}
			}
		}
	}

	return &ldapUser{
		entryID: entryID,
		id:      id,
		data:    data,
	}, nil
}

func (u *ldapUser) getAttributeValue(n string) string {
	if n == "" {
		return ""
	}

	return u.data[n]
}

func (u *ldapUser) Subject() string {
	return u.entryID
}

func (u *ldapUser) Email() string {
	return u.getAttributeValue(AttributeEmail)
}

func (u *ldapUser) EmailVerified() bool {
	return false
}

func (u *ldapUser) Name() string {
	return u.getAttributeValue(AttributeName)
}

func (u *ldapUser) FamilyName() string {
	return u.getAttributeValue(AttributeFamilyName)
}

func (u *ldapUser) GivenName() string {
	return u.getAttributeValue(AttributeGivenName)
}

func (u *ldapUser) Username() string {
	return u.getAttributeValue(AttributeLogin)
}

func (u *ldapUser) ID() int64 {
	return u.id
}

func (u *ldapUser) UniqueID() string {
	return u.getAttributeValue(AttributeUUID)
}

func (u *ldapUser) BackendClaims() map[string]interface{} {
	claims := make(map[string]interface{})
	claims[konnect.IdentifiedUserIDClaim] = u.entryID

	return claims
}

func (u *ldapUser) BackendScopes() []string {
	return nil
}

func (u *ldapUser) RequiredScopes() []string {
	return nil
}

// NewLDAPIdentifierBackend creates a new LDAPIdentifierBackend with the provided
// parameters.
func NewLDAPIdentifierBackend(
	c *config.Config,
	tlsConfig *tls.Config,
	uriString,
	bindDN,
	bindPassword,
	baseDN,
	scopeString,
	filter string,
	subAttributes []string,
	mappedAttributes map[string]string,
) (*LDAPIdentifierBackend, error) {
	var err error
	var scope int
	var uri *url.URL
	for {
		if uriString == "" {
			err = fmt.Errorf("server must not be empty")
			break
		}
		uri, err = url.Parse(uriString)
		if err != nil {
			break
		}

		if bindDN == "" && bindPassword != "" {
			err = fmt.Errorf("bind DN must not be empty when bind password is given")
			break
		}
		if baseDN == "" {
			err = fmt.Errorf("base DN must not be empty")
			break
		}
		switch scopeString {
		case "sub":
			scope = ldap.ScopeWholeSubtree
		case "one":
			scope = ldap.ScopeSingleLevel
		case "base":
			scope = ldap.ScopeBaseObject
		case "":
			scope = ldap.ScopeWholeSubtree
		default:
			err = fmt.Errorf("unknown scope value: %v, must be one of sub, one or base", scopeString)
		}
		if err != nil {
			break
		}

		break
	}
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend %v", err)
	}

	attributeMapping := ldapAttributeMapping{}
	for k, v := range ldapDefaultAttributeMapping {
		if mapped, ok := mappedAttributes[k]; ok && mapped != "" {
			v = mapped
		}
		attributeMapping[k] = v
		c.Logger.WithField("attribute", fmt.Sprintf("%v:%v", k, v)).Debugln("ldap identifier backend set attribute")
	}

	// Build supported scopes based on default scopes and scope mapping.
	supportedScopes := make([]string, len(ldapSupportedScopes))
	copy(supportedScopes, ldapSupportedScopes)
	if numericUIDAttribute := mappedAttributes[AttributeNumericUID]; numericUIDAttribute != "" {
		supportedScopes = append(supportedScopes, konnect.ScopeNumericID)
		attributeMapping[AttributeNumericUID] = numericUIDAttribute
		c.Logger.WithField("attribute", fmt.Sprintf("%v:%v", AttributeNumericUID, numericUIDAttribute)).Debugln("ldap identifier backend use attribute")
	}

	if filter == "" {
		filter = "(objectClass=inetOrgPerson)"
	}
	c.Logger.WithField("filter", filter).Debugln("ldap identifier backend set filter")

	loginAttribute := attributeMapping[AttributeLogin]

	addr := uri.Host
	isTLS := false

	switch uri.Scheme {
	case "":
		uri.Scheme = "ldap"
		fallthrough
	case "ldap":
		if uri.Port() == "" {
			addr += ":389"
		}
	case "ldaps":
		if uri.Port() == "" {
			addr += ":636"
		}
		// To be able to verify the servers TLS certificate we need to set the
		// server's hostname. (Normally tls.DialWithDialer() would take care of
		// that, but we're not using that in LDAPIdentifierBackend.connect())
		if !tlsConfig.InsecureSkipVerify && tlsConfig.ServerName == "" {
			tlsConfig.ServerName = uri.Hostname()
		}
		isTLS = true
	default:
		err = fmt.Errorf("invalid URI scheme: %v", uri.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend %v", err)
	}

	var entryIDMapping []string
	if len(subAttributes) > 0 {
		entryIDMapping = subAttributes
		c.Logger.WithField("mapping", entryIDMapping).Debugln("ldap identifier sub is mapped")
	}

	b := &LDAPIdentifierBackend{
		addr:         addr,
		isTLS:        isTLS,
		bindDN:       bindDN,
		bindPassword: bindPassword,
		baseDN:       baseDN,
		scope:        scope,
		searchFilter: fmt.Sprintf("(&(%s)(%s=%%s))", filter, loginAttribute),
		getFilter:    filter,

		entryIDMapping:   entryIDMapping,
		attributeMapping: attributeMapping,
		supportedScopes:  supportedScopes,

		logger: c.Logger,
		dialer: &net.Dialer{
			Timeout:   ldap.DefaultTimeout,
			DualStack: true,
		},
		tlsConfig: tlsConfig,

		timeout: 60,                        //XXX(longsleep): make timeout configuration.
		limiter: rate.NewLimiter(100, 200), //XXX(longsleep): make rate limits configuration.
	}

	b.logger.WithField("ldap", fmt.Sprintf("%s://%s ", uri.Scheme, addr)).Infoln("ldap server identifier backend set up")

	return b, nil
}

// RunWithContext implements the Backend interface.
func (b *LDAPIdentifierBackend) RunWithContext(ctx context.Context) error {
	return nil
}

// Logon implements the Backend interface, enabling Logon with user name and
// password as provided. Requests are bound to the provided context.
func (b *LDAPIdentifierBackend) Logon(ctx context.Context, audience, username, password string) (bool, *string, *string, backends.UserFromBackend, error) {
	loginAttributeName := b.attributeMapping[AttributeLogin]
	if loginAttributeName == "" {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon impossible as no login attribute is set")
	}

	l, err := b.connect(ctx)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon connect error: %v", err)
	}
	defer l.Close()

	// Search for the given username.
	entry, err := b.searchUsername(l, username, b.attributeMapping.attributes())
	switch {
	case ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject):
		return false, nil, nil, nil, nil
	}
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon search error: %v", err)
	}
	if !strings.EqualFold(entry.GetEqualFoldAttributeValue(loginAttributeName), username) {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon search returned wrong user")
	}

	// Bind as the user to verify the password.
	err = l.Bind(entry.DN, password)
	switch {
	case ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials):
		return false, nil, nil, nil, nil
	}

	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon error: %v", err)
	}

	entryID := b.entryIDFromEntry(b.attributeMapping, entry)
	if entryID == "" {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon entry without entry ID: %v", entry.DN)
	}

	user, err := newLdapUser(entryID, b.attributeMapping, entry)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("ldap identifier backend logon entry data error: %v", err)
	}

	// Use the users subject as user id.
	userID := user.Subject()

	b.logger.WithFields(logrus.Fields{
		"username": user.Username(),
		"id":       userID,
	}).Debugln("ldap identifier backend logon")

	return true, &userID, nil, user, nil
}

// ResolveUserByUsername implements the Beckend interface, providing lookup for
// user by providing the username. Requests are bound to the provided context.
func (b *LDAPIdentifierBackend) ResolveUserByUsername(ctx context.Context, username string) (backends.UserFromBackend, error) {
	loginAttributeName := b.attributeMapping[AttributeLogin]
	if loginAttributeName == "" {
		return nil, fmt.Errorf("ldap identifier backend resolve impossible as no login attribute is set")
	}

	l, err := b.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend resolve connect error: %v", err)
	}
	defer l.Close()

	// Search for the given username.
	entry, err := b.searchUsername(l, username, b.attributeMapping.attributes())
	switch {
	case ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject):
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend resolve search error: %v", err)
	}
	if !strings.EqualFold(entry.GetEqualFoldAttributeValue(loginAttributeName), username) {
		return nil, fmt.Errorf("ldap identifier backend resolve search returned wrong user")
	}

	newEntryID := b.entryIDFromEntry(b.attributeMapping, entry)

	user, err := newLdapUser(newEntryID, b.attributeMapping, entry)
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend resolve entry data error: %v", err)
	}

	return user, nil
}

// GetUser implements the Backend interface, providing user meta data retrieval
// for the user specified by the userID. Requests are bound to the provided
// context.
func (b *LDAPIdentifierBackend) GetUser(ctx context.Context, entryID string, sessionRef *string, requestedScopes map[string]bool) (backends.UserFromBackend, error) {
	l, err := b.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend get user connect error: %v", err)
	}
	defer l.Close()

	entry, err := b.getUser(l, entryID, b.attributeMapping.attributes())
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend get user error: %v", err)
	}

	newEntryID := b.entryIDFromEntry(b.attributeMapping, entry)
	if !strings.EqualFold(newEntryID, entryID) {
		return nil, fmt.Errorf("ldap identifier backend get user returned wrong user")
	}

	user, err := newLdapUser(newEntryID, b.attributeMapping, entry)
	if err != nil {
		return nil, fmt.Errorf("ldap identifier backend get user entry data error: %v", err)
	}

	return user, err
}

// RefreshSession implements the Backend interface.
func (b *LDAPIdentifierBackend) RefreshSession(ctx context.Context, userID string, sessionRef *string, claims map[string]interface{}) error {
	return nil
}

// DestroySession implements the Backend interface providing destroy to KC session.
func (b *LDAPIdentifierBackend) DestroySession(ctx context.Context, sessionRef *string) error {
	return nil
}

// UserClaims implements the Backend interface, providing user specific claims
// for the user specified by the userID.
func (b *LDAPIdentifierBackend) UserClaims(userID string, authorizedScopes map[string]bool) map[string]interface{} {
	return nil
}

// ScopesSupported implements the Backend interface, providing supported scopes
// when running this backend.
func (b *LDAPIdentifierBackend) ScopesSupported() []string {
	return b.supportedScopes
}

// ScopesMeta implements the Backend interface, providing meta data for
// supported scopes.
func (b *LDAPIdentifierBackend) ScopesMeta() *scopes.Scopes {
	return nil
}

// Name implements the Backend interface.
func (b *LDAPIdentifierBackend) Name() string {
	return ldapIdentifierBackendName
}

func (b *LDAPIdentifierBackend) connect(parentCtx context.Context) (*ldap.Conn, error) {
	// A timeout for waiting for a limiter slot. The timeout also includes the
	// time to connect to the LDAP server which as a consequence means that both
	// getting a free slot and establishing the connection are one timeout.
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(b.timeout)*time.Second)
	defer cancel()

	err := b.limiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	c, err := b.dialer.DialContext(ctx, "tcp", b.addr)
	if err != nil {
		return nil, ldap.NewError(ldap.ErrorNetwork, err)
	}

	var l *ldap.Conn
	if b.isTLS {
		sc := tls.Client(c, b.tlsConfig)
		err = sc.Handshake()
		if err != nil {
			c.Close()
			return nil, ldap.NewError(ldap.ErrorNetwork, err)
		}
		l = ldap.NewConn(sc, true)

	} else {
		l = ldap.NewConn(c, false)
	}

	l.Start()

	// Bind with general user (which is preferably read only).
	if b.bindDN != "" {
		err = l.Bind(b.bindDN, b.bindPassword)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (b *LDAPIdentifierBackend) searchUsername(l *ldap.Conn, username string, attributes []string) (*ldap.Entry, error) {
	base, filter := b.baseAndSearchFilterFromUsername(username)
	// Search for the given username.
	searchRequest := ldap.NewSearchRequest(
		base,
		b.scope, ldap.NeverDerefAliases, 1, b.timeout, false,
		filter,
		attributes,
		nil,
	)
	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	switch len(sr.Entries) {
	case 0:
		// Nothing found.
		return nil, ldap.NewError(ldap.LDAPResultNoSuchObject, err)
	case 1:
		// Exactly one found, success.
		return sr.Entries[0], nil
	default:
		// Invalid when multiple matched.
		return nil, fmt.Errorf("user too many entries returned")
	}
}

func (b *LDAPIdentifierBackend) getUser(l *ldap.Conn, entryID string, attributes []string) (*ldap.Entry, error) {
	base, filter := b.baseAndGetFilterFromEntryID(entryID)
	if base == "" || filter == "" || entryID == "" {
		return nil, fmt.Errorf("ldap identifier backend get user invalid user ID: %v", entryID)
	}

	scope := b.scope
	if base == entryID {
		// Ensure that scope is limited, when directly requesting an entry.
		scope = ldap.ScopeBaseObject
	}

	// search for the given DN.
	searchRequest := ldap.NewSearchRequest(
		base,
		scope, ldap.NeverDerefAliases, 1, b.timeout, false,
		filter,
		attributes,
		nil,
	)
	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	if len(sr.Entries) != 1 {
		return nil, fmt.Errorf("user does not exist or too many entries returned")
	}

	return sr.Entries[0], nil
}

func (b *LDAPIdentifierBackend) entryIDFromEntry(mapping ldapAttributeMapping, entry *ldap.Entry) string {
	if b.entryIDMapping != nil {
		// Encode as URL query.
		values := url.Values{}
		for _, k := range b.entryIDMapping {
			v := entry.GetEqualFoldAttributeValues(k)
			if len(v) > 0 {
				values[k] = v
			}
		}
		// URL encode values to string.
		return values.Encode()
	}

	// Use DN by default is no mapping is set.
	return entry.DN
}

func (b *LDAPIdentifierBackend) baseAndGetFilterFromEntryID(entryID string) (string, string) {
	if b.entryIDMapping != nil {
		// Parse entryID as URL encoded query values, and build & filter to search for them all.
		if values, err := url.ParseQuery(entryID); err == nil {
			filter := ""
			for k, values := range values {
				for _, value := range values {
					filter = fmt.Sprintf("%s(%s=%s)", filter, k, ldap.EscapeFilter(value))
				}
			}
			if filter != "" {
				return b.baseDN, fmt.Sprintf("(&%s%s)", b.getFilter, filter)
			}
		}
		// Failed to parse entry ID.
		return "", ""
	}

	// Map DN to entryID.
	_, err := ldap.ParseDN(entryID)
	if err != nil {
		return "", ""
	}
	return entryID, b.getFilter
}

func (b *LDAPIdentifierBackend) baseAndSearchFilterFromUsername(username string) (string, string) {
	// Build search filter with username.
	return b.baseDN, fmt.Sprintf(b.searchFilter, ldap.EscapeFilter(username))
}
