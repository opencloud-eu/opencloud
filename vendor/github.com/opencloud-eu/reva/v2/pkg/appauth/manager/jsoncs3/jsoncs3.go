package jsoncs3

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/argon2id"
	apppb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appauth"
	"github.com/opencloud-eu/reva/v2/pkg/appauth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/metadatacache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-diceware/diceware"
	"github.com/sethvargo/go-password/password"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/proto"
)

type PasswordGenerator interface {
	GeneratePassword() (string, error)
}

func init() {
	registry.Register("jsoncs3", New)
}

type manager struct {
	sync.RWMutex        // for lazy initialization
	mds                 metadata.Storage
	store               *metadatacache.Store[string, map[string]*apppb.AppPassword]
	generator           PasswordGenerator
	uTimeUpdateInterval time.Duration
	initialized         bool
}

type config struct {
	ProviderAddr      string         `mapstructure:"provider_addr"`
	ServiceUserID     string         `mapstructure:"service_user_id"`
	ServiceUserIdp    string         `mapstructure:"service_user_idp"`
	MachineAuthAPIKey string         `mapstructure:"machine_auth_apikey"`
	Generator         string         `mapstructure:"password_generator"`
	GeneratorConfig   map[string]any `mapstructure:"generator_config"`
	// Time interval in seconds to update the UTime of a token when calling GetAppPassword. Default is 5 min.
	// For testing set this -1 to disable automatic updates.
	UTimeUpdateInterval int `mapstructure:"utime_update_interval_seconds"`
	UpdateRetryCount    int `mapstructure:"update_retry_count"`
}

const tracerName = "jsoncs3"

func New(m map[string]any) (appauth.Manager, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error creating a new manager")
		return nil, err
	}

	if c.ProviderAddr == "" {
		return nil, fmt.Errorf("appauth jsoncs3 manager: provider_addr not set")
	}

	if c.ServiceUserID == "" {
		return nil, fmt.Errorf("appauth jsoncs3 manager: service_user_id not set")
	}

	if c.ServiceUserIdp == "" {
		return nil, fmt.Errorf("appauth jsoncs3 manager: service_user_idp not set")
	}

	if c.MachineAuthAPIKey == "" {
		return nil, fmt.Errorf("appauth jsoncs3 manager: machine_auth_apikey not set")
	}

	if c.Generator == "" {
		c.Generator = "diceware"
	}
	if c.UpdateRetryCount <= 0 {
		c.UpdateRetryCount = 5
	}

	var updateInterval time.Duration
	switch c.UTimeUpdateInterval {
	case -1:
		updateInterval = 0
	case 0:
		updateInterval = 5 * time.Minute
	default:
		updateInterval = time.Duration(c.UTimeUpdateInterval) * time.Second
	}

	var pwgen PasswordGenerator
	var err error
	switch c.Generator {
	case "diceware":
		pwgen, err = NewDicewareGenerator(c.GeneratorConfig)
	case "random":
		pwgen, err = NewRandGenerator(c.GeneratorConfig)
	default:
		return nil, fmt.Errorf("appauth jsoncs3 manager: unknown generator: %s", c.Generator)
	}

	if err != nil {
		return nil, fmt.Errorf("appauth jsoncs3 manager: failed initialize password generator: %w", err)
	}

	cs3, err := metadata.NewCS3Storage(c.ProviderAddr, c.ProviderAddr, c.ServiceUserID, c.ServiceUserIdp, c.MachineAuthAPIKey)
	if err != nil {
		return nil, err
	}

	return NewWithOptions(cs3, pwgen, updateInterval, c.UpdateRetryCount)
}

func NewWithOptions(mds metadata.Storage, generator PasswordGenerator, uTimeUpdateInterval time.Duration, updateRetries int) (*manager, error) {
	store := metadatacache.New(metadatacache.Options[string, map[string]*apppb.AppPassword]{
		Storage: mds,
		Path:    func(userID string) string { return userID + ".json" },
		Retries: updateRetries,
		Init:    func() map[string]*apppb.AppPassword { return map[string]*apppb.AppPassword{} },
	})
	return &manager{
		mds:                 mds,
		store:               store,
		generator:           generator,
		uTimeUpdateInterval: uTimeUpdateInterval,
	}, nil
}

// GenerateAppPassword creates a password with specified scope to be used by
// third-party applications.
func (m *manager) GenerateAppPassword(ctx context.Context, scope map[string]*authpb.Scope, label string, expiration *typespb.Timestamp) (*apppb.AppPassword, error) {
	logger := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GenerateAppPassword")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		logger.Error().Err(err).Msg("initializing appauth manager failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	token, err := m.generator.GeneratePassword()
	if err != nil {
		logger.Debug().Err(err).Msg("error generating new password")
		return nil, errors.Wrap(err, "error creating new token")
	}

	tokenHashed, err := argon2id.CreateHash(token, argon2id.DefaultParams)
	if err != nil {
		logger.Debug().Err(err).Msg("error generating password hash")
		return nil, errors.Wrap(err, "error creating new token")
	}

	var userID *userpb.UserId
	if user, ok := ctxpkg.ContextGetUser(ctx); ok {
		userID = user.GetId()
	} else {
		logger.Debug().Err(err).Msg("no user in context")
		return nil, errtypes.BadRequest("no user in context")
	}

	cTime := &typespb.Timestamp{Seconds: uint64(time.Now().Unix())}

	// For persisting we use the hashed password, since we don't
	// want to store it in cleartext.
	appPass := &apppb.AppPassword{
		Password:   tokenHashed,
		TokenScope: scope,
		Label:      label,
		Expiration: expiration,
		Ctime:      cTime,
		Utime:      cTime,
		User:       userID,
	}

	id := uuid.New().String()

	err = m.store.Update(ctx, userID.GetOpaqueId(), true, func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, bool, error) {
		a[id] = appPass
		return a, true, nil
	})
	if err != nil {
		logger.Debug().Err(err).Msg("failed to store new app password")
		return nil, err
	}

	// Return a fresh AppPassword with the cleartext token for the caller.
	// Constructing from scratch avoids copying the proto struct (which contains a mutex).
	return &apppb.AppPassword{
		Password:   token,
		TokenScope: scope,
		Label:      label,
		Expiration: expiration,
		Ctime:      cTime,
		Utime:      cTime,
		User:       userID,
	}, nil
}

// ListAppPasswords lists the application passwords created by a user.
func (m *manager) ListAppPasswords(ctx context.Context) ([]*apppb.AppPassword, error) {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "ListAppPasswords")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	var userID *userpb.UserId
	if user, ok := ctxpkg.ContextGetUser(ctx); ok {
		userID = user.GetId()
	} else {
		return nil, errtypes.BadRequest("no user in context")
	}

	unlock := m.store.Lock(userID.GetOpaqueId())
	defer unlock()

	passwords, ok, err := m.store.Get(ctx, userID.GetOpaqueId())
	if err != nil {
		log.Error().Err(err).Msg("store.Get failed")
		return nil, err
	}
	if !ok {
		return []*apppb.AppPassword{}, nil
	}

	result := make([]*apppb.AppPassword, 0, len(passwords))
	for id, p := range passwords {
		pw := proto.Clone(p).(*apppb.AppPassword)
		pw.Password = id
		result = append(result, pw)
	}
	return result, nil
}

// InvalidateAppPassword invalidates a generated password.
func (m *manager) InvalidateAppPassword(ctx context.Context, secretOrId string) error {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "InvalidateAppPassword")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var userID *userpb.UserId
	if user, ok := ctxpkg.ContextGetUser(ctx); ok {
		userID = user.GetId()
	} else {
		return errtypes.BadRequest("no user in context")
	}

	err := m.store.Update(ctx, userID.GetOpaqueId(), false, func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, bool, error) {
		// Allow deleting a token using the ID inside the password property. This is needed because of
		// some shortcomings of the CS3 APIs. On the API level tokens don't have IDs;
		// ListAppPasswords in this backend returns the ID as the password value.
		if _, ok := a[secretOrId]; ok {
			delete(a, secretOrId)
			return a, true, nil
		}

		// Check if the supplied parameter matches any of the stored password tokens.
		for key, pw := range a {
			ok, err := argon2id.ComparePasswordAndHash(secretOrId, pw.Password)
			switch {
			case err != nil:
				log.Debug().Err(err).Msg("Error comparing password and hash")
			case ok:
				delete(a, key)
				return a, true, nil
			}
		}
		return a, false, errtypes.NotFound("password not found")
	})
	if err != nil {
		log.Error().Err(err).Msg("store.Update failed")
		return errtypes.NotFound("password not found")
	}
	return nil
}

// GetAppPassword retrieves the password information by the combination of username and password.
func (m *manager) GetAppPassword(ctx context.Context, user *userpb.UserId, secret string) (*apppb.AppPassword, error) {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetAppPassword")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var (
		matchedPw *apppb.AppPassword
		matchedID string
	)

	err := m.store.Update(ctx, user.GetOpaqueId(), false, func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, bool, error) {
		matchedPw = nil
		for id, pw := range a {
			ok, err := argon2id.ComparePasswordAndHash(secret, pw.Password)
			switch {
			case err != nil:
				log.Debug().Err(err).Msg("Error comparing password and hash")
			case ok:
				// password found
				if pw.Expiration != nil && pw.Expiration.Seconds != 0 && uint64(time.Now().Unix()) > pw.Expiration.Seconds {
					log.Debug().Str("AppPasswordId", id).Msg("password expired")
					return nil, false, errtypes.NotFound("password not found")
				}

				matchedPw = pw
				matchedID = id
				// Updating the Utime will cause an Upload for every single GetAppPassword request. We are limiting this to one
				// update per 'uTimeUpdateInterval' (default 5 min) otherwise this backend will become unusable.
				if time.Since(utils.TSToTime(pw.Utime)) > m.uTimeUpdateInterval {
					a[id].Utime = utils.TSNow()
					return a, true, nil
				}
				return a, false, nil
			}
		}
		return nil, false, errtypes.NotFound("password not found")
	})
	if err != nil {
		return nil, errtypes.NotFound("password not found")
	}

	// Return a clone with the ID in the password field so the cached entry
	// is not corrupted.
	result := proto.Clone(matchedPw).(*apppb.AppPassword)
	result.Password = matchedID
	return result, nil
}

func (m *manager) initialize(ctx context.Context) error {
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "initialize")
	logger := appctx.GetLogger(ctx)
	defer span.End()
	if m.initialized {
		span.SetStatus(codes.Ok, "already initialized")
		return nil
	}

	m.Lock()
	defer m.Unlock()

	if m.initialized { // check if initialization happened while grabbing the lock
		span.SetStatus(codes.Ok, "initialized while grabbing lock")
		return nil
	}

	ctx = context.Background()
	ctx = appctx.WithLogger(ctx, logger)
	err := m.mds.Init(ctx, "jsoncs3-appauth-data")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	m.initialized = true
	return nil
}

type randomPassword struct {
	Strength int `mapstructure:"token_strength"`
}

func NewRandGenerator(config map[string]any) (*randomPassword, error) {
	r := &randomPassword{}
	if err := mapstructure.Decode(config, r); err != nil {
		err = errors.Wrap(err, "error configuring password generator")
		return nil, err
	}
	if r.Strength <= 0 {
		r.Strength = 11
	}
	return r, nil
}

func (r randomPassword) GeneratePassword() (string, error) {
	token, err := password.Generate(r.Strength, r.Strength/2, 0, false, false)
	if err != nil {
		return "", errors.Wrap(err, "error creating new token")
	}
	return token, nil
}

type dicewarePassword struct {
	NumWords int `mapstructure:"number_of_words"`
}

func NewDicewareGenerator(config map[string]any) (*dicewarePassword, error) {
	d := &dicewarePassword{}
	if err := mapstructure.Decode(config, d); err != nil {
		err = errors.Wrap(err, "error creating a new manager")
		return nil, err
	}
	if d.NumWords <= 0 {
		d.NumWords = 6
	}
	return d, nil
}

func (d dicewarePassword) GeneratePassword() (string, error) {
	token, err := diceware.Generate(d.NumWords)
	if err != nil {
		return "", errors.Wrap(err, "error creating new token")
	}
	return strings.Join(token, " "), nil
}
