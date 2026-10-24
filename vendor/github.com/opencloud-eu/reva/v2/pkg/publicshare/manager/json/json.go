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
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/json/persistence"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/json/persistence/cs3"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/json/persistence/file"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/json/persistence/memory"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// defaultStatConcurrency bounds concurrent Stat RPCs in ListPublicShares, mirroring the decomposedfs share manager.
const defaultStatConcurrency = 5

func init() {
	registry.Register("json", NewFile)
	registry.Register("jsoncs3", NewCS3)
	registry.Register("jsonmemory", NewMemory)
}

// NewFile returns a new filesystem public shares manager.
func NewFile(c map[string]interface{}) (publicshare.Manager, error) {
	conf := &fileConfig{}
	if err := mapstructure.Decode(c, conf); err != nil {
		return nil, err
	}

	conf.init()
	if conf.File == "" {
		conf.File = "/var/tmp/reva/publicshares"
	}

	p := file.New(conf.File)
	return New(conf.GatewayAddr, conf.SharePasswordHashCost, conf.JanitorRunInterval, conf.EnableExpiredSharesCleanup, p)
}

// NewMemory returns a new in-memory public shares manager.
func NewMemory(c map[string]interface{}) (publicshare.Manager, error) {
	conf := &commonConfig{}
	if err := mapstructure.Decode(c, conf); err != nil {
		return nil, err
	}

	conf.init()
	p := memory.New()

	return New(conf.GatewayAddr, conf.SharePasswordHashCost, conf.JanitorRunInterval, conf.EnableExpiredSharesCleanup, p)
}

// NewCS3 returns a new cs3 public shares manager.
func NewCS3(c map[string]interface{}) (publicshare.Manager, error) {
	conf := &cs3Config{}
	if err := mapstructure.Decode(c, conf); err != nil {
		return nil, err
	}

	conf.init()

	s, err := metadata.NewCS3Storage(conf.ProviderAddr, conf.ProviderAddr, conf.ServiceUserID, conf.ServiceUserIdp, conf.MachineAuthAPIKey)
	if err != nil {
		return nil, err
	}
	p := cs3.New(s)

	return New(conf.GatewayAddr, conf.SharePasswordHashCost, conf.JanitorRunInterval, conf.EnableExpiredSharesCleanup, p)
}

// New returns a new public share manager instance
func New(gwAddr string, pwHashCost, janitorRunInterval int, enableCleanup bool, p persistence.Persistence) (publicshare.Manager, error) {
	m := &manager{
		gatewayAddr:                gwAddr,
		maxConcurrency:             defaultStatConcurrency,
		mutex:                      &sync.Mutex{},
		passwordHashCost:           pwHashCost,
		janitorRunInterval:         janitorRunInterval,
		enableExpiredSharesCleanup: enableCleanup,
		persistence:                p,
	}

	go m.startJanitorRun()
	return m, nil
}

type commonConfig struct {
	GatewayAddr                string `mapstructure:"gateway_addr"`
	SharePasswordHashCost      int    `mapstructure:"password_hash_cost"`
	JanitorRunInterval         int    `mapstructure:"janitor_run_interval"`
	EnableExpiredSharesCleanup bool   `mapstructure:"enable_expired_shares_cleanup"`
}

type fileConfig struct {
	commonConfig `mapstructure:",squash"`

	File string `mapstructure:"file"`
}

type cs3Config struct {
	commonConfig `mapstructure:",squash"`

	ProviderAddr      string `mapstructure:"provider_addr"`
	ServiceUserID     string `mapstructure:"service_user_id"`
	ServiceUserIdp    string `mapstructure:"service_user_idp"`
	MachineAuthAPIKey string `mapstructure:"machine_auth_apikey"`
}

func (c *commonConfig) init() {
	if c.SharePasswordHashCost == 0 {
		c.SharePasswordHashCost = 11
	}
	if c.JanitorRunInterval == 0 {
		c.JanitorRunInterval = 60
	}
}

type manager struct {
	// maxConcurrency bounds concurrent Stat RPCs in ListPublicShares.
	maxConcurrency int

	gatewayAddr string
	mutex       *sync.Mutex
	persistence persistence.Persistence

	passwordHashCost           int
	janitorRunInterval         int
	enableExpiredSharesCleanup bool
}

func (m *manager) init() error {
	return m.persistence.Init(context.Background())
}

func (m *manager) startJanitorRun() {
	if !m.enableExpiredSharesCleanup {
		return
	}

	ticker := time.NewTicker(time.Duration(m.janitorRunInterval) * time.Second)
	work := make(chan os.Signal, 1)
	signal.Notify(work, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-work:
			return
		case <-ticker.C:
			m.cleanupExpiredShares()
		}
	}
}

// Dump exports public shares to channels (e.g. during migration)
func (m *manager) Dump(ctx context.Context, shareChan chan<- *publicshare.WithPassword) error {
	log := appctx.GetLogger(ctx)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return err
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return err
	}

	for _, v := range db {
		var local publicshare.WithPassword
		if err := utils.UnmarshalJSONToProtoV1([]byte(v.(map[string]interface{})["share"].(string)), &local.PublicShare); err != nil {
			log.Error().Err(err).Msg("error unmarshalling share")
		}
		local.Password = v.(map[string]interface{})["password"].(string)
		shareChan <- &local
	}

	return nil
}

// Load imports public shares and received shares from channels (e.g. during migration)
func (m *manager) Load(ctx context.Context, shareChan <-chan *publicshare.WithPassword) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return err
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return err
	}

	for ps := range shareChan {
		encShare, err := utils.MarshalProtoV1ToJSON(&ps.PublicShare)
		if err != nil {
			return err
		}

		db[ps.PublicShare.Id.GetOpaqueId()] = map[string]interface{}{
			"share":    string(encShare),
			"password": ps.Password,
		}
	}
	return m.persistence.Write(ctx, db)
}

// CreatePublicShare adds a new entry to manager.shares
func (m *manager) CreatePublicShare(ctx context.Context, u *user.User, rInfo *provider.ResourceInfo, g *link.Grant) (*link.PublicShare, error) {
	id := &link.PublicShareId{
		OpaqueId: utils.RandString(15),
	}

	tkn := utils.RandString(15)
	now := time.Now().UnixNano()

	displayName, ok := rInfo.ArbitraryMetadata.Metadata["name"]
	if !ok {
		displayName = tkn
	}

	quicklink, _ := strconv.ParseBool(rInfo.ArbitraryMetadata.Metadata["quicklink"])

	var passwordProtected bool
	password := g.Password
	if len(password) > 0 {
		h, err := bcrypt.GenerateFromPassword([]byte(password), m.passwordHashCost)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash share password")
		}
		password = string(h)
		passwordProtected = true
	}

	createdAt := &typespb.Timestamp{
		Seconds: uint64(now / int64(time.Second)),
		Nanos:   uint32(now % int64(time.Second)),
	}

	s := &link.PublicShare{
		Id:                id,
		Owner:             rInfo.GetOwner(),
		Creator:           u.Id,
		ResourceId:        rInfo.Id,
		Token:             tkn,
		Permissions:       g.Permissions,
		Ctime:             createdAt,
		Mtime:             createdAt,
		PasswordProtected: passwordProtected,
		Expiration:        g.Expiration,
		DisplayName:       displayName,
		Quicklink:         quicklink,
	}

	ps := &publicShare{
		Password: password,
	}
	proto.Merge(&ps.PublicShare, s)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return nil, err
	}

	encShare, err := utils.MarshalProtoV1ToJSON(&ps.PublicShare)
	if err != nil {
		return nil, err
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, err
	}

	if _, ok := db[s.Id.GetOpaqueId()]; !ok {
		db[s.Id.GetOpaqueId()] = map[string]interface{}{
			"share":    string(encShare),
			"password": ps.Password,
		}
	} else {
		return nil, errors.New("key already exists")
	}

	err = m.persistence.Write(ctx, db)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// UpdatePublicShare updates the public share
func (m *manager) UpdatePublicShare(ctx context.Context, u *user.User, req *link.UpdatePublicShareRequest) (*link.PublicShare, error) {
	log := appctx.GetLogger(ctx)
	share, err := m.GetPublicShare(ctx, u, req.Ref, false)
	if err != nil {
		return nil, errors.New("ref does not exist")
	}

	now := time.Now().UnixNano()
	var newPasswordEncoded string
	passwordChanged := false

	switch req.GetUpdate().GetType() {
	case link.UpdatePublicShareRequest_Update_TYPE_DISPLAYNAME:
		log.Debug().Str("json", "update display name").Msgf("from: `%v` to `%v`", share.DisplayName, req.Update.GetDisplayName())
		share.DisplayName = req.Update.GetDisplayName()
	case link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS:
		old, _ := json.Marshal(share.Permissions)
		new, _ := json.Marshal(req.Update.GetGrant().Permissions)

		if req.GetUpdate().GetGrant().GetPassword() != "" {
			passwordChanged = true
			h, err := bcrypt.GenerateFromPassword([]byte(req.Update.GetGrant().Password), m.passwordHashCost)
			if err != nil {
				return nil, errors.Wrap(err, "could not hash share password")
			}
			newPasswordEncoded = string(h)
			share.PasswordProtected = true
		}

		log.Debug().Str("json", "update grants").Msgf("from: `%v`\nto\n`%v`", old, new)
		share.Permissions = req.Update.GetGrant().GetPermissions()
	case link.UpdatePublicShareRequest_Update_TYPE_EXPIRATION:
		old, _ := json.Marshal(share.Expiration)
		new, _ := json.Marshal(req.Update.GetGrant().Expiration)
		log.Debug().Str("json", "update expiration").Msgf("from: `%v`\nto\n`%v`", old, new)
		share.Expiration = req.Update.GetGrant().Expiration
	case link.UpdatePublicShareRequest_Update_TYPE_PASSWORD:
		passwordChanged = true
		if req.Update.GetGrant().Password == "" {
			share.PasswordProtected = false
			newPasswordEncoded = ""
		} else {
			h, err := bcrypt.GenerateFromPassword([]byte(req.Update.GetGrant().Password), m.passwordHashCost)
			if err != nil {
				return nil, errors.Wrap(err, "could not hash share password")
			}
			newPasswordEncoded = string(h)
			share.PasswordProtected = true
		}
	default:
		return nil, fmt.Errorf("invalid update type: %v", req.GetUpdate().GetType())
	}

	share.Mtime = &typespb.Timestamp{
		Seconds: uint64(now / int64(time.Second)),
		Nanos:   uint32(now % int64(time.Second)),
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return nil, err
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, err
	}

	encShare, err := utils.MarshalProtoV1ToJSON(share)
	if err != nil {
		return nil, err
	}

	data, ok := db[share.Id.OpaqueId].(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}

	if ok && passwordChanged {
		data["password"] = newPasswordEncoded
	}
	data["share"] = string(encShare)

	db[share.Id.OpaqueId] = data

	err = m.persistence.Write(ctx, db)
	if err != nil {
		return nil, err
	}

	return share, nil
}

// GetPublicShare gets a public share either by ID or Token.
func (m *manager) GetPublicShare(ctx context.Context, u *user.User, ref *link.PublicShareReference, sign bool) (*link.PublicShare, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return nil, err
	}

	if ref.GetToken() != "" {
		ps, pw, err := m.getByToken(ctx, ref.GetToken())
		if err != nil {
			return nil, errtypes.NotFound("no shares found by token")
		}
		if ps.PasswordProtected && sign {
			err := publicshare.AddSignature(ps, pw)
			if err != nil {
				return nil, err
			}
		}
		return ps, nil
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range db {
		d := v.(map[string]interface{})["share"]
		passDB := v.(map[string]interface{})["password"].(string)

		var ps link.PublicShare
		if err := utils.UnmarshalJSONToProtoV1([]byte(d.(string)), &ps); err != nil {
			return nil, err
		}

		if ref.GetId().GetOpaqueId() == ps.Id.OpaqueId {
			if publicshare.IsExpired(&ps) {
				if err := m.revokeExpiredPublicShare(ctx, &ps); err != nil {
					return nil, err
				}
				return nil, errtypes.NotFound("no shares found by id:" + ref.GetId().String())
			}
			if ps.PasswordProtected && sign {
				err := publicshare.AddSignature(&ps, passDB)
				if err != nil {
					return nil, err
				}
			}
			return &ps, nil
		}

	}
	return nil, errtypes.NotFound("no shares found by id:" + ref.GetId().String())
}

// collectShares splits the persisted shares into own and foreign ones, plus the distinct resources the foreign ones point at.
func (m *manager) collectShares(ctx context.Context, u *user.User, filters []*link.ListPublicSharesRequest_Filter) ([]*publicShare, []*publicShare, map[string]*provider.ResourceId, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return nil, nil, nil, err
	}

	log := appctx.GetLogger(ctx)

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	ownShares := make([]*publicShare, 0)
	foreignShares := make([]*publicShare, 0)
	foreignResourceIDs := make(map[string]*provider.ResourceId)

	for _, v := range db {
		var local publicShare
		if err := utils.UnmarshalJSONToProtoV1([]byte(v.(map[string]interface{})["share"].(string)), &local.PublicShare); err != nil {
			return nil, nil, nil, err
		}

		if publicshare.IsExpired(&local.PublicShare) {
			if err := m.revokeExpiredPublicShare(ctx, &local.PublicShare); err != nil {
				log.Error().Err(err).
					Str("share_token", local.Token).
					Msg("failed to revoke expired public share")
			}
			continue
		}

		if !publicshare.MatchesFilters(&local.PublicShare, filters) {
			continue
		}

		if local.ResourceId == nil {
			log.Warn().
				Str("share_id", local.PublicShare.GetId().GetOpaqueId()).
				Str("share_token", local.Token).
				Msg("ListPublicShares: skipping share with nil resource_id")
			continue
		}

		if publicshare.IsCreatedByUser(&local.PublicShare, u) {
			ownShares = append(ownShares, &local)
			continue
		}

		foreignShares = append(foreignShares, &local)
		foreignResourceIDs[storagespace.FormatResourceID(local.ResourceId)] = local.ResourceId
	}

	return ownShares, foreignShares, foreignResourceIDs, nil
}

// ListPublicShares retrieves all the shares on the manager that are valid.
func (m *manager) ListPublicShares(ctx context.Context, u *user.User, filters []*link.ListPublicSharesRequest_Filter, sign bool) ([]*link.PublicShare, error) {
	// collectShares holds the lock; nothing below it touches manager state
	ownShares, foreignShares, foreignResourceIDs, err := m.collectShares(ctx, u, filters)
	if err != nil {
		return nil, err
	}

	// stat each distinct foreign resource once, concurrently, within a time budget
	var permitted map[string]bool
	if len(foreignResourceIDs) > 0 {
		client, err := pool.GetGatewayServiceClient(m.gatewayAddr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list shares")
		}
		permitted = m.statForeignResources(ctx, u, client, foreignResourceIDs)
	}

	shares := make([]*link.PublicShare, 0, len(ownShares)+len(foreignShares))
	for _, local := range ownShares {
		if local.PasswordProtected && sign {
			if err := publicshare.AddSignature(&local.PublicShare, local.Password); err != nil {
				return nil, err
			}
		}
		shares = append(shares, &local.PublicShare)
	}
	for _, local := range foreignShares {
		// fail closed: a resource left undecided is absent here and stays excluded
		if !permitted[storagespace.FormatResourceID(local.ResourceId)] {
			continue
		}
		if local.PasswordProtected && sign {
			if err := publicshare.AddSignature(&local.PublicShare, local.Password); err != nil {
				return nil, err
			}
		}
		shares = append(shares, &local.PublicShare)
	}
	return shares, nil
}

// statForeignResources stats each resource once within a time budget and reports who may list its grants; anything absent was never decided.
func (m *manager) statForeignResources(ctx context.Context, u *user.User, client gateway.GatewayAPIClient, resourceIDs map[string]*provider.ResourceId) map[string]bool {
	log := appctx.GetLogger(ctx)

	statCtx, cancel := m.statBudgetContext(ctx)
	defer cancel()

	results := newStatResults()

	numWorkers := m.maxConcurrency
	if numWorkers > len(resourceIDs) {
		numWorkers = len(resourceIDs)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	type job struct {
		rid *provider.ResourceId
	}
	jobs := make(chan job)

	g, gctx := errgroup.WithContext(statCtx)

	// stop feeding jobs once the budget runs out so workers drain instead of blocking
	g.Go(func() error {
		defer close(jobs)
		for _, rid := range resourceIDs {
			select {
			case jobs <- job{rid}:
			case <-gctx.Done():
				return nil
			}
		}
		return nil
	})

	// workers, bounded by numWorkers <= m.maxConcurrency concurrent Stat RPCs
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for j := range jobs {
				if gctx.Err() != nil {
					// budget exhausted: leave the rest undecided, the caller treats them as denied
					continue
				}
				m.userCanListGrants(statCtx, client, results, j.rid)
			}
			return nil
		})
	}
	_ = g.Wait()

	result := results.snapshot()
	if skipped := len(resourceIDs) - len(result); skipped > 0 {
		log.Warn().
			Str("user_id", u.GetId().GetOpaqueId()).
			Int("resources_checked", len(result)).
			Int("resources_skipped", skipped).
			Msg("ListPublicShares: stat time budget exhausted before every resource could be checked, returned list may be incomplete")
	}
	return result
}

// statBudgetContext bounds the stat fan-out to the caller's own deadline minus a margin, or not at all if it has none.
func (m *manager) statBudgetContext(ctx context.Context) (context.Context, context.CancelFunc) {
	const margin = 200 * time.Millisecond

	deadline, ok := ctx.Deadline()
	if !ok {
		return ctx, func() {}
	}

	budget := time.Until(deadline) - margin
	if budget < 0 {
		budget = 0
	}
	return context.WithTimeout(ctx, budget)
}

// statResults collects ListGrants answers keyed by storagespace.FormatResourceID, safe for concurrent use.
type statResults struct {
	mu   sync.Mutex
	data map[string]bool
}

func newStatResults() *statResults {
	return &statResults{data: make(map[string]bool)}
}

func (r *statResults) set(key string, allowed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = allowed
}

// snapshot copies the results collected so far; call it only once no writers remain.
func (r *statResults) snapshot() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]bool, len(r.data))
	for k, v := range r.data {
		out[k] = v
	}
	return out
}

// userCanListGrants stats the resource, records whether the user may list its grants, and reports the answer.
func (m *manager) userCanListGrants(ctx context.Context, client gateway.GatewayAPIClient, results *statResults, rid *provider.ResourceId) bool {
	log := appctx.GetLogger(ctx)
	key := storagespace.FormatResourceID(rid)

	sRes, err := client.Stat(ctx, &provider.StatRequest{
		Ref:       &provider.Reference{ResourceId: rid},
		FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"permissions"}},
	})
	switch {
	case err != nil:
		log.Error().Err(err).Interface("resource_id", rid).Msg("ListShares: an error occurred during stat on the resource")
		results.set(key, false)
		return false
	case sRes.Status.Code == rpc.Code_CODE_NOT_FOUND:
		log.Debug().Str("message", sRes.Status.Message).Interface("status", sRes.Status).Interface("resource_id", rid).Msg("ListShares: Resource not found")
		results.set(key, false)
		return false
	case sRes.Status.Code != rpc.Code_CODE_OK:
		log.Error().Str("message", sRes.Status.Message).Interface("status", sRes.Status).Interface("resource_id", rid).Msg("ListShares: could not stat resource")
		results.set(key, false)
		return false
	}

	allowed := sRes.GetInfo().GetPermissionSet().GetListGrants()
	results.set(key, allowed)
	return allowed
}

func (m *manager) cleanupExpiredShares() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return
	}

	db, _ := m.persistence.Read(context.Background())

	for _, v := range db {
		d := v.(map[string]interface{})["share"]

		var ps link.PublicShare
		_ = utils.UnmarshalJSONToProtoV1([]byte(d.(string)), &ps)

		if publicshare.IsExpired(&ps) {
			_ = m.revokeExpiredPublicShare(context.Background(), &ps)
		}
	}
}

// revokeExpiredPublicShare doesn't have a lock inside, ensure a lock before call
func (m *manager) revokeExpiredPublicShare(ctx context.Context, s *link.PublicShare) error {
	if !m.enableExpiredSharesCleanup {
		return nil
	}

	err := m.revokePublicShare(ctx, &link.PublicShareReference{
		Spec: &link.PublicShareReference_Id{
			Id: &link.PublicShareId{
				OpaqueId: s.Id.OpaqueId,
			},
		},
	})
	if err != nil {
		log.Err(err).Msg(fmt.Sprintf("publicShareJSONManager: error deleting public share with opaqueId: %s", s.Id.OpaqueId))
		return err
	}

	return nil
}

// RevokePublicShare undocumented.
func (m *manager) RevokePublicShare(ctx context.Context, _ *user.User, ref *link.PublicShareReference) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return err
	}

	return m.revokePublicShare(ctx, ref)
}

// revokePublicShare doesn't have a lock inside, ensure a lock before call
func (m *manager) revokePublicShare(ctx context.Context, ref *link.PublicShareReference) error {
	db, err := m.persistence.Read(ctx)
	if err != nil {
		return err
	}

	switch {
	case ref.GetId() != nil && ref.GetId().OpaqueId != "":
		if _, ok := db[ref.GetId().OpaqueId]; ok {
			delete(db, ref.GetId().OpaqueId)
		} else {
			return errors.New("reference does not exist")
		}
	case ref.GetToken() != "":
		share, _, err := m.getByToken(ctx, ref.GetToken())
		if err != nil {
			return err
		}
		delete(db, share.Id.OpaqueId)
	default:
		return errors.New("reference does not exist")
	}

	return m.persistence.Write(ctx, db)
}

// getByToken doesn't have a lock inside, ensure a lock before call
func (m *manager) getByToken(ctx context.Context, token string) (*link.PublicShare, string, error) {
	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, "", err
	}

	for _, v := range db {
		var local link.PublicShare
		if err := utils.UnmarshalJSONToProtoV1([]byte(v.(map[string]interface{})["share"].(string)), &local); err != nil {
			return nil, "", err
		}

		if local.Token == token {
			passDB := v.(map[string]interface{})["password"].(string)
			return &local, passDB, nil
		}
	}

	return nil, "", fmt.Errorf("share with token: `%v` not found", token)
}

// GetPublicShareByToken gets a public share by its opaque token.
func (m *manager) GetPublicShareByToken(ctx context.Context, token string, auth *link.PublicShareAuthentication, sign bool) (*link.PublicShare, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err := m.init(); err != nil {
		return nil, err
	}

	db, err := m.persistence.Read(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range db {
		passDB := v.(map[string]interface{})["password"].(string)
		var local link.PublicShare
		if err := utils.UnmarshalJSONToProtoV1([]byte(v.(map[string]interface{})["share"].(string)), &local); err != nil {
			return nil, err
		}

		if local.Token == token {
			if publicshare.IsExpired(&local) {
				if err := m.revokeExpiredPublicShare(ctx, &local); err != nil {
					return nil, err
				}
				break
			}

			if local.PasswordProtected {
				if publicshare.Authenticate(&local, passDB, auth) {
					if sign {
						err := publicshare.AddSignature(&local, passDB)
						if err != nil {
							return nil, err
						}
					}
					return &local, nil
				}

				return nil, errtypes.InvalidCredentials("json: invalid password")
			}
			return &local, nil
		}
	}

	return nil, errtypes.NotFound(fmt.Sprintf("share with token: `%v` not found", token))
}

type publicShare struct {
	link.PublicShare
	Password string `json:"password"`
}
