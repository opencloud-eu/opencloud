package jsoncs3

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/argon2id"
	apppb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appauth"
	"github.com/opencloud-eu/reva/v2/pkg/appauth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-diceware/diceware"
	"github.com/sethvargo/go-password/password"
	"go.opentelemetry.io/otel/codes"
)

type PasswordGenerator interface {
	GeneratePassword() (string, error)
}

func init() {
	registry.Register("jsoncs3", New)
}

type manager struct {
	sync.RWMutex // for lazy initialization
	mds          metadata.Storage
	generator    PasswordGenerator
	initialized  bool
}

type config struct {
	ProviderAddr      string         `mapstructure:"provider_addr"`
	ServiceUserID     string         `mapstructure:"service_user_id"`
	ServiceUserIdp    string         `mapstructure:"service_user_idp"`
	MachineAuthAPIKey string         `mapstructure:"machine_auth_apikey"`
	Generator         string         `mapstructure:"password_generator"`
	GeneratorConfig   map[string]any `mapstructure:"generator_config"`
}

type updaterFunc func(map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, error)

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

	return NewWithOptions(cs3, pwgen)
}

func NewWithOptions(mds metadata.Storage, generator PasswordGenerator) (*manager, error) {
	return &manager{
		mds:       mds,
		generator: generator,
	}, nil
}

// GenerateAppPassword creates a password with specified scope to be used by
// third-party applications.
func (m *manager) GenerateAppPassword(ctx context.Context, scope map[string]*authpb.Scope, label string, expiration *typespb.Timestamp) (*apppb.AppPassword, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GenerateAppPassword")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	token, err := m.generator.GeneratePassword()
	if err != nil {
		return nil, errors.Wrap(err, "error creating new token")
	}

	tokenHashed, err := argon2id.CreateHash(token, argon2id.DefaultParams)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new token")
	}

	var userID *userpb.UserId
	if user, ok := ctxpkg.ContextGetUser(ctx); ok {
		userID = user.GetId()
	} else {
		return nil, errtypes.BadRequest("no user in context")
	}

	cTime := &typespb.Timestamp{Seconds: uint64(time.Now().Unix())}

	// For persisting we use the hashed password, since we don't
	// want to store it in cleartext
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

	err = m.updateWithRetry(ctx, 5, true, userID, func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, error) {
		a[id] = appPass
		return a, nil
	})

	if err != nil {
		return nil, err
	}

	// Here we need to resplace the hash with the cleartext password again since
	// the requestor needs to know the cleartext value.
	appPass.Password = token

	return appPass, nil
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
	_, userAppPasswords, err := m.getUserAppPasswords(ctx, userID)
	if err != nil {
		if _, ok := err.(errtypes.NotFound); ok {
			return []*apppb.AppPassword{}, nil
		}
		log.Error().Err(err).Msg("getUserAppPasswords failed")
		return nil, err
	}

	userAppPasswordSlice := make([]*apppb.AppPassword, 0, len(userAppPasswords))

	for id, p := range userAppPasswords {
		p.Password = id
		userAppPasswordSlice = append(userAppPasswordSlice, p)
	}

	return userAppPasswordSlice, nil
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

	updater := func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, error) {
		// Allow deleting a token using the ID inside the password property. This is needed because of
		// some shortcomings of the CS3 APIs. On the API level tokens don't have IDs
		// ListAppPasswords in this backend returns the ID as the password value.
		if _, ok := a[secretOrId]; ok {
			delete(a, secretOrId)
			return a, nil
		}

		// Check if the supplied parameter matches any of the stored password tokens
		for key, pw := range a {
			ok, err := argon2id.ComparePasswordAndHash(secretOrId, pw.Password)
			switch {
			case err != nil:
				log.Debug().Err(err).Msg("Error comparing password and hash")
			case ok:
				delete(a, key)
				return a, nil
			}
		}
		return a, errtypes.NotFound("password not found")
	}

	err := m.updateWithRetry(ctx, 5, false, userID, updater)
	if err != nil {
		log.Error().Err(err).Msg("getUserAppPasswords failed")
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

	errUpdateSkipped := errors.New("update skipped")

	var (
		matchedPw *apppb.AppPassword
		matchedID string
	)
	updater := func(a map[string]*apppb.AppPassword) (map[string]*apppb.AppPassword, error) {
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
					return nil, errtypes.NotFound("password not found")
				}

				matchedPw = pw
				matchedID = id
				// password not expired
				// Updating the Utime will cause an Upload for every single GetAppPassword request. We are limiting this to one
				// update per 5 minutes otherwise this backend will become unusable.
				if time.Since(utils.TSToTime(pw.Utime)) > 5*time.Minute {
					a[id].Utime = utils.TSNow()
					return a, nil
				}
				return a, errUpdateSkipped
			}
		}
		return nil, errtypes.NotFound("password not found")
	}

	err := m.updateWithRetry(ctx, 5, false, user, updater)
	switch {
	case err == nil:
		fallthrough
	case errors.Is(err, errUpdateSkipped):
		// Don't return the hashed password, put the ID into the password field
		matchedPw.Password = matchedID
		return matchedPw, nil
	}

	return nil, errtypes.NotFound("password not found")
}

func (m *manager) initialize(ctx context.Context) error {
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "initialize")
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
	err := m.mds.Init(ctx, "jsoncs3-appauth-data")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	m.initialized = true
	return nil
}

func (m *manager) updateWithRetry(ctx context.Context, retries int, createIfNotFound bool, userid *userpb.UserId, updater updaterFunc) error {
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "initialize")
	defer span.End()

	retry := true
	var (
		etag             string
		userAppPasswords map[string]*apppb.AppPassword
		err              error
	)

	// retry for the specified number of times, then error out
	for i := 0; i < retries && retry; i++ {
		etag, userAppPasswords, err = m.getUserAppPasswords(ctx, userid)
		switch err.(type) {
		case nil:
			// empty
		case errtypes.NotFound:
			if createIfNotFound {
				userAppPasswords = map[string]*apppb.AppPassword{}
			} else {
				span.RecordError(err)
				span.SetStatus(codes.Error, "downloading app tokens failed")
				return err
			}
		default:
			span.RecordError(err)
			span.SetStatus(codes.Error, "downloading app tokens failed")
			return err
		}

		userAppPasswords, err = updater(userAppPasswords)
		if err != nil {
			return err
		}

		err = m.updateUserAppPassword(ctx, userid, userAppPasswords, etag)
		switch err.(type) {
		case nil:
			retry = false
		case errtypes.PreconditionFailed:
			retry = true
		default:
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	if retry {
		span.RecordError(err)
		span.SetStatus(codes.Error, "updating app tokens failed")
		return err
	}
	return nil
}

func (m *manager) updateUserAppPassword(ctx context.Context, userid *userpb.UserId, appPasswords map[string]*apppb.AppPassword, ifMatchEtag string) error {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "getUserAppPasswords")
	jsonPath := userAppTokenJSONPath(userid)

	pwBytes, err := json.Marshal(appPasswords)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	ur := metadata.UploadRequest{
		Path:        jsonPath,
		Content:     pwBytes,
		IfMatchEtag: ifMatchEtag,
	}

	// If there is no etag, make sure to only upload if the file wasn't craeted yet
	if ifMatchEtag == "" {
		ur.IfNoneMatch = []string{"*"}
	}
	_, err = m.mds.Upload(ctx, ur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Debug().Err(err).Msg("persisting provider cache failed")
		return err
	}
	return nil
}

func (m *manager) getUserAppPasswords(ctx context.Context, userid *userpb.UserId) (string, map[string]*apppb.AppPassword, error) {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "getUserAppPasswords")
	jsonPath := userAppTokenJSONPath(userid)
	dlreq := metadata.DownloadRequest{
		Path: jsonPath,
	}

	var userAppPasswords = map[string]*apppb.AppPassword{}
	dlres, err := m.mds.Download(ctx, dlreq)
	switch err.(type) {
	case nil:
		err = json.Unmarshal(dlres.Content, &userAppPasswords)
		if err != nil {
			log.Error().Err(err).Msg("unmarshaling app tokens failed")
			return "", nil, err
		}
	case errtypes.NotFound:
		return "", nil, errtypes.NotFound("password not found")
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, "downloading app tokens failed")
		return "", nil, err
	}
	return dlres.Etag, userAppPasswords, nil
}

func userAppTokenJSONPath(userID *userpb.UserId) string {
	return userID.GetOpaqueId() + ".json"
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
