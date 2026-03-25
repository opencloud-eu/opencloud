/*
 * Copyright 2018-2024 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/nats-io/nkeys"
)

// NoLimit is used to indicate a limit field is unlimited in value.
const (
	NoLimit    = -1
	AnyAccount = "*"
)

type AccountLimits struct {
	Imports         int64 `json:"imports,omitempty"`         // Max number of imports
	Exports         int64 `json:"exports,omitempty"`         // Max number of exports
	WildcardExports bool  `json:"wildcards,omitempty"`       // Are wildcards allowed in exports
	DisallowBearer  bool  `json:"disallow_bearer,omitempty"` // User JWT can't be bearer token
	Conn            int64 `json:"conn,omitempty"`            // Max number of active connections
	LeafNodeConn    int64 `json:"leaf,omitempty"`            // Max number of active leaf node connections
}

// IsUnlimited returns true if all limits are unlimited
func (a *AccountLimits) IsUnlimited() bool {
	return *a == AccountLimits{NoLimit, NoLimit, true, false, NoLimit, NoLimit}
}

type NatsLimits struct {
	Subs    int64 `json:"subs,omitempty"`    // Max number of subscriptions
	Data    int64 `json:"data,omitempty"`    // Max number of bytes
	Payload int64 `json:"payload,omitempty"` // Max message payload
}

// IsUnlimited returns true if all limits are unlimited
func (n *NatsLimits) IsUnlimited() bool {
	return *n == NatsLimits{NoLimit, NoLimit, NoLimit}
}

type JetStreamLimits struct {
	MemoryStorage        int64 `json:"mem_storage,omitempty"`           // Max number of bytes stored in memory across all streams. (0 means disabled)
	DiskStorage          int64 `json:"disk_storage,omitempty"`          // Max number of bytes stored on disk across all streams. (0 means disabled)
	Streams              int64 `json:"streams,omitempty"`               // Max number of streams
	Consumer             int64 `json:"consumer,omitempty"`              // Max number of consumers
	MaxAckPending        int64 `json:"max_ack_pending,omitempty"`       // Max ack pending of a Stream
	MemoryMaxStreamBytes int64 `json:"mem_max_stream_bytes,omitempty"`  // Max bytes a memory backed stream can have. (0 means disabled/unlimited)
	DiskMaxStreamBytes   int64 `json:"disk_max_stream_bytes,omitempty"` // Max bytes a disk backed stream can have. (0 means disabled/unlimited)
	MaxBytesRequired     bool  `json:"max_bytes_required,omitempty"`    // Max bytes required by all Streams
}

// IsUnlimited returns true if all limits are unlimited
func (j *JetStreamLimits) IsUnlimited() bool {
	lim := *j
	// workaround in case NoLimit was used instead of 0
	if lim.MemoryMaxStreamBytes < 0 {
		lim.MemoryMaxStreamBytes = 0
	}
	if lim.DiskMaxStreamBytes < 0 {
		lim.DiskMaxStreamBytes = 0
	}
	if lim.MaxAckPending < 0 {
		lim.MaxAckPending = 0
	}
	return lim == JetStreamLimits{NoLimit, NoLimit, NoLimit, NoLimit, 0, 0, 0, false}
}

type JetStreamTieredLimits map[string]JetStreamLimits

// OperatorLimits are used to limit access by an account
type OperatorLimits struct {
	NatsLimits
	AccountLimits
	JetStreamLimits
	JetStreamTieredLimits `json:"tiered_limits,omitempty"`
}

// IsJSEnabled returns if this account claim has JS enabled either through a tier or the non tiered limits.
func (o *OperatorLimits) IsJSEnabled() bool {
	if len(o.JetStreamTieredLimits) > 0 {
		for _, l := range o.JetStreamTieredLimits {
			if l.MemoryStorage != 0 || l.DiskStorage != 0 {
				return true
			}
		}
		return false
	}
	l := o.JetStreamLimits
	return l.MemoryStorage != 0 || l.DiskStorage != 0
}

// IsEmpty returns true if all limits are 0/false/empty.
func (o *OperatorLimits) IsEmpty() bool {
	return o.NatsLimits == NatsLimits{} &&
		o.AccountLimits == AccountLimits{} &&
		o.JetStreamLimits == JetStreamLimits{} &&
		len(o.JetStreamTieredLimits) == 0
}

// IsUnlimited returns true if all limits are unlimited
func (o *OperatorLimits) IsUnlimited() bool {
	return o.AccountLimits.IsUnlimited() && o.NatsLimits.IsUnlimited() &&
		o.JetStreamLimits.IsUnlimited() && len(o.JetStreamTieredLimits) == 0
}

// Validate checks that the operator limits contain valid values
func (o *OperatorLimits) Validate(vr *ValidationResults) {
	// negative values mean unlimited, so all numbers are valid
	if len(o.JetStreamTieredLimits) > 0 {
		if (o.JetStreamLimits != JetStreamLimits{}) {
			vr.AddError("JetStream Limits and tiered JetStream Limits are mutually exclusive")
		}
		if _, ok := o.JetStreamTieredLimits[""]; ok {
			vr.AddError(`Tiered JetStream Limits can not contain a blank "" tier name`)
		}
	}
}

// WeightedMapping for publishes
type WeightedMapping struct {
	Subject Subject `json:"subject"`
	Weight  uint8   `json:"weight,omitempty"`
	Cluster string  `json:"cluster,omitempty"`
}

func (m *WeightedMapping) GetWeight() uint8 {
	if m.Weight == 0 {
		return 100
	}
	return m.Weight
}

type Mapping map[Subject][]WeightedMapping

func (m *Mapping) Validate(vr *ValidationResults) {
	for ubFrom, wm := range (map[Subject][]WeightedMapping)(*m) {
		ubFrom.Validate(vr)
		perCluster := make(map[string]uint32)
		total := uint32(0)
		for _, e := range wm {
			e.Subject.Validate(vr)
			if e.GetWeight() > 100 {
				vr.AddError("Mapping %q has a weight %d that exceeds 100", ubFrom, e.GetWeight())
			}
			if e.Cluster != "" {
				t := perCluster[e.Cluster]
				t += uint32(e.GetWeight())
				perCluster[e.Cluster] = t
				if t > 100 {
					vr.AddError("Mapping %q in cluster %q exceeds 100%% among all of it's weighted to mappings", ubFrom, e.Cluster)
				}
			} else {
				total += uint32(e.GetWeight())
			}
		}
		if total > 100 {
			vr.AddError("Mapping %q exceeds 100%% among all of it's weighted to mappings", ubFrom)
		}
	}
}

func (a *Account) AddMapping(sub Subject, to ...WeightedMapping) {
	a.Mappings[sub] = to
}

// ExternalAuthorization enables external authorization for account users.
// AuthUsers are those users specified to bypass the authorization callout and should be used for the authorization service itself.
// AllowedAccounts specifies which accounts, if any, that the authorization service can bind an authorized user to.
// The authorization response, a user JWT, will still need to be signed by the correct account.
// If optional XKey is specified, that is the public xkey (x25519) and the server will encrypt the request such that only the
// holder of the private key can decrypt. The auth service can also optionally encrypt the response back to the server using it's
// public xkey which will be in the authorization request.
type ExternalAuthorization struct {
	AuthUsers       StringList `json:"auth_users,omitempty"`
	AllowedAccounts StringList `json:"allowed_accounts,omitempty"`
	XKey            string     `json:"xkey,omitempty"`
}

func (ac *ExternalAuthorization) IsEnabled() bool {
	return len(ac.AuthUsers) > 0
}

// HasExternalAuthorization helper function to determine if external authorization is enabled.
func (a *Account) HasExternalAuthorization() bool {
	return a.Authorization.IsEnabled()
}

// EnableExternalAuthorization helper function to setup external authorization.
func (a *Account) EnableExternalAuthorization(users ...string) {
	a.Authorization.AuthUsers.Add(users...)
}

func (ac *ExternalAuthorization) Validate(vr *ValidationResults) {
	if len(ac.AllowedAccounts) > 0 && len(ac.AuthUsers) == 0 {
		vr.AddError("External authorization cannot have accounts without users specified")
	}
	// Make sure users are all valid user nkeys.
	// Make sure allowed accounts are all valid account nkeys.
	for _, u := range ac.AuthUsers {
		if !nkeys.IsValidPublicUserKey(u) {
			vr.AddError("AuthUser %q is not a valid user public key", u)
		}
	}
	for _, a := range ac.AllowedAccounts {
		if a == AnyAccount && len(ac.AllowedAccounts) > 1 {
			vr.AddError("AllowedAccounts can only be a list of accounts or %q", AnyAccount)
			continue
		} else if a == AnyAccount {
			continue
		} else if !nkeys.IsValidPublicAccountKey(a) {
			vr.AddError("Account %q is not a valid account public key", a)
		}
	}
	if ac.XKey != "" && !nkeys.IsValidPublicCurveKey(ac.XKey) {
		vr.AddError("XKey %q is not a valid public xkey", ac.XKey)
	}
}

const (
	ClusterTrafficSystem = "system"
	ClusterTrafficOwner  = "owner"
)

type ClusterTraffic string

func (ct ClusterTraffic) Valid() error {
	if ct == "" || ct == ClusterTrafficSystem || ct == ClusterTrafficOwner {
		return nil
	}
	return fmt.Errorf("unknown cluster traffic option: %q", ct)
}

// Account holds account specific claims data
type Account struct {
	Imports            Imports               `json:"imports,omitempty"`
	Exports            Exports               `json:"exports,omitempty"`
	Limits             OperatorLimits        `json:"limits,omitempty"`
	SigningKeys        SigningKeys           `json:"signing_keys,omitempty"`
	Revocations        RevocationList        `json:"revocations,omitempty"`
	DefaultPermissions Permissions           `json:"default_permissions,omitempty"`
	Mappings           Mapping               `json:"mappings,omitempty"`
	Authorization      ExternalAuthorization `json:"authorization,omitempty"`
	Trace              *MsgTrace             `json:"trace,omitempty"`
	ClusterTraffic     ClusterTraffic        `json:"cluster_traffic,omitempty"`
	Info
	GenericFields
}

// MsgTrace holds distributed message tracing configuration
type MsgTrace struct {
	// Destination is the subject the server will send message traces to
	// if the inbound message contains the "traceparent" header and has
	// its sampled field indicating that the trace should be triggered.
	Destination Subject `json:"dest,omitempty"`
	// Sampling is used to set the probability sampling, that is, the
	// server will get a random number between 1 and 100 and trigger
	// the trace if the number is lower than this Sampling value.
	// The valid range is [1..100]. If the value is not set Validate()
	// will set the value to 100.
	Sampling int `json:"sampling,omitempty"`
}

// Validate checks if the account is valid, based on the wrapper
func (a *Account) Validate(acct *AccountClaims, vr *ValidationResults) {
	a.Imports.Validate(acct.Subject, vr)
	a.Exports.Validate(vr)
	a.Limits.Validate(vr)
	a.DefaultPermissions.Validate(vr)
	a.Mappings.Validate(vr)
	a.Authorization.Validate(vr)
	if a.Trace != nil {
		tvr := CreateValidationResults()
		a.Trace.Destination.Validate(tvr)
		if !tvr.IsEmpty() {
			vr.AddError("the account Trace.Destination %s", tvr.Issues[0].Description)
		}
		if a.Trace.Destination.HasWildCards() {
			vr.AddError("the account Trace.Destination subject %q is not a valid publish subject", a.Trace.Destination)
		}
		if a.Trace.Sampling < 0 || a.Trace.Sampling > 100 {
			vr.AddError("the account Trace.Sampling value '%d' is not valid, should be in the range [1..100]", a.Trace.Sampling)
		} else if a.Trace.Sampling == 0 {
			a.Trace.Sampling = 100
		}
	}

	if !a.Limits.IsEmpty() && a.Limits.Imports >= 0 && int64(len(a.Imports)) > a.Limits.Imports {
		vr.AddError("the account contains more imports than allowed by the operator")
	}

	// Check Imports and Exports for limit violations.
	if a.Limits.Imports != NoLimit {
		if int64(len(a.Imports)) > a.Limits.Imports {
			vr.AddError("the account contains more imports than allowed by the operator")
		}
	}
	if a.Limits.Exports != NoLimit {
		if int64(len(a.Exports)) > a.Limits.Exports {
			vr.AddError("the account contains more exports than allowed by the operator")
		}
		// Check for wildcard restrictions
		if !a.Limits.WildcardExports {
			for _, ex := range a.Exports {
				if ex.Subject.HasWildCards() {
					vr.AddError("the account contains wildcard exports that are not allowed by the operator")
				}
			}
		}
	}
	a.SigningKeys.Validate(vr)
	a.Info.Validate(vr)

	if err := a.ClusterTraffic.Valid(); err != nil {
		vr.AddError("%s", err.Error())
	}
}

// AccountClaims defines the body of an account JWT
type AccountClaims struct {
	ClaimsData
	Account `json:"nats,omitempty"`
}

// NewAccountClaims creates a new account JWT
func NewAccountClaims(subject string) *AccountClaims {
	if subject == "" {
		return nil
	}
	c := &AccountClaims{}
	c.SigningKeys = make(SigningKeys)
	// Set to unlimited to start. We do it this way so we get compiler
	// errors if we add to the OperatorLimits.
	c.Limits = OperatorLimits{
		NatsLimits{NoLimit, NoLimit, NoLimit},
		AccountLimits{NoLimit, NoLimit, true, false, NoLimit, NoLimit},
		JetStreamLimits{0, 0, 0, 0, 0, 0, 0, false},
		JetStreamTieredLimits{},
	}
	c.Subject = subject
	c.Mappings = Mapping{}
	return c
}

// Encode converts account claims into a JWT string
func (a *AccountClaims) Encode(pair nkeys.KeyPair) (string, error) {
	return a.EncodeWithSigner(pair, nil)
}

func (a *AccountClaims) EncodeWithSigner(pair nkeys.KeyPair, fn SignFn) (string, error) {
	if !nkeys.IsValidPublicAccountKey(a.Subject) {
		return "", errors.New("expected subject to be account public key")
	}
	sort.Sort(a.Exports)
	sort.Sort(a.Imports)
	a.Type = AccountClaim
	return a.ClaimsData.encode(pair, a, fn)
}

// DecodeAccountClaims decodes account claims from a JWT string
func DecodeAccountClaims(token string) (*AccountClaims, error) {
	claims, err := Decode(token)
	if err != nil {
		return nil, err
	}
	ac, ok := claims.(*AccountClaims)
	if !ok {
		return nil, errors.New("not account claim")
	}
	return ac, nil
}

func (a *AccountClaims) String() string {
	return a.ClaimsData.String(a)
}

// Payload pulls the accounts specific payload out of the claims
func (a *AccountClaims) Payload() interface{} {
	return &a.Account
}

// Validate checks the accounts contents
func (a *AccountClaims) Validate(vr *ValidationResults) {
	a.ClaimsData.Validate(vr)
	a.Account.Validate(a, vr)

	if nkeys.IsValidPublicAccountKey(a.ClaimsData.Issuer) {
		if !a.Limits.IsEmpty() {
			vr.AddWarning("self-signed account JWTs shouldn't contain operator limits")
		}
	}
}

func (a *AccountClaims) ClaimType() ClaimType {
	return a.Type
}

func (a *AccountClaims) updateVersion() {
	a.GenericFields.Version = libVersion
}

// ExpectedPrefixes defines the types that can encode an account jwt, account and operator
func (a *AccountClaims) ExpectedPrefixes() []nkeys.PrefixByte {
	return []nkeys.PrefixByte{nkeys.PrefixByteAccount, nkeys.PrefixByteOperator}
}

// Claims returns the accounts claims data
func (a *AccountClaims) Claims() *ClaimsData {
	return &a.ClaimsData
}
func (a *AccountClaims) GetTags() TagList {
	return a.Account.Tags
}

// DidSign checks the claims against the account's public key and its signing keys
func (a *AccountClaims) DidSign(c Claims) bool {
	if c != nil {
		issuer := c.Claims().Issuer
		if issuer == a.Subject {
			return true
		}
		uc, ok := c.(*UserClaims)
		if ok && uc.IssuerAccount == a.Subject {
			return a.SigningKeys.Contains(issuer)
		}
		at, ok := c.(*ActivationClaims)
		if ok && at.IssuerAccount == a.Subject {
			return a.SigningKeys.Contains(issuer)
		}
	}
	return false
}

// Revoke enters a revocation by public key using time.Now().
func (a *AccountClaims) Revoke(pubKey string) {
	a.RevokeAt(pubKey, time.Now())
}

// RevokeAt enters a revocation by public key and timestamp into this account
// This will revoke all jwt issued for pubKey, prior to timestamp
// If there is already a revocation for this public key that is newer, it is kept.
// The value is expected to be a public key or "*" (means all public keys)
func (a *AccountClaims) RevokeAt(pubKey string, timestamp time.Time) {
	if a.Revocations == nil {
		a.Revocations = RevocationList{}
	}
	a.Revocations.Revoke(pubKey, timestamp)
}

// ClearRevocation removes any revocation for the public key
func (a *AccountClaims) ClearRevocation(pubKey string) {
	a.Revocations.ClearRevocation(pubKey)
}

// isRevoked checks if the public key is in the revoked list with a timestamp later than the one passed in.
// Generally this method is called with the subject and issue time of the jwt to be tested.
// DO NOT pass time.Now(), it will not produce a stable/expected response.
func (a *AccountClaims) isRevoked(pubKey string, claimIssuedAt time.Time) bool {
	return a.Revocations.IsRevoked(pubKey, claimIssuedAt)
}

// IsClaimRevoked checks if the account revoked the claim passed in.
// Invalid claims (nil, no Subject or IssuedAt) will return true.
func (a *AccountClaims) IsClaimRevoked(claim *UserClaims) bool {
	if claim == nil || claim.IssuedAt == 0 || claim.Subject == "" {
		return true
	}
	return a.isRevoked(claim.Subject, time.Unix(claim.IssuedAt, 0))
}
