package middleware

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	rjwt "github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
	"github.com/rs/zerolog"
	microstore "go-micro.dev/v4/store"
	"google.golang.org/grpc/metadata"
)

type key int

const (
	wopiContextKey key = iota
)

// WopiContext wraps all the information we need for WOPI
type WopiContext struct {
	AccessToken       string
	ViewOnlyToken     string
	FileReference     *providerv1beta1.Reference
	TemplateReference *providerv1beta1.Reference
	ViewMode          appproviderv1beta1.ViewMode
}

// WopiContextAuthMiddleware will prepare an HTTP handler to be used as
// middleware. The handler will create a WopiContext by parsing the
// access_token (which must be provided as part of the URL query).
// The access_token is required.
//
// This middleware will add the following to the request's context:
// * The access token as metadata for outgoing requests (for the
// authentication against the CS3 API, the "x-access-token" header).
// * The created WopiContext for the request
// * A contextual zerologger containing information about the request
// and the WopiContext
func WopiContextAuthMiddleware(cfg *config.Config, st microstore.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// include additional info in the context's logger
		// we might need to check https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/common-headers
		// although some headers might not be sent depending on the client.
		logger := zerolog.Ctx(ctx)
		wopiLogger := logger.With().
			Str("WopiSessionId", r.Header.Get("X-WOPI-SessionId")).
			Str("WopiOverride", r.Header.Get("X-WOPI-Override")).
			Str("WopiProof", r.Header.Get("X-WOPI-Proof")).
			Str("WopiProofOld", r.Header.Get("X-WOPI-ProofOld")).
			Str("WopiStamp", r.Header.Get("X-WOPI-TimeStamp")).
			Logger()

		accessToken := r.URL.Query().Get("access_token")
		if accessToken == "" {
			wopiLogger.Error().Msg("missing access token")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if cfg.Wopi.ShortTokens {
			records, err := st.Read(accessToken)
			if err != nil {
				wopiLogger.Error().Err(err).Msg("cannot retrieve access token from store")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if len(records) != 1 {
				wopiLogger.Error().Int("records", len(records)).Msg("no record found for the token")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			accessToken = string(records[0].Value)
		}

		claims := &Claims{}
		_, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(cfg.Wopi.Secret), nil
		})

		if err != nil {
			wopiLogger.Error().Err(err).Msg("failed to parse jwt token")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		wopiContextAccessToken, err := DecryptAES([]byte(cfg.Wopi.Secret), claims.WopiContext.AccessToken)
		if err != nil {
			wopiLogger.Error().Err(err).Msg("failed to decrypt reva access token")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		tokenManager, err := rjwt.New(map[string]interface{}{
			"secret":  cfg.TokenManager.JWTSecret,
			"expires": int64(24 * 60 * 60),
		})
		if err != nil {
			wopiLogger.Error().Err(err).Msg("failed to get a reva token manager")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		user, scopes, err := tokenManager.DismantleToken(ctx, wopiContextAccessToken)
		if err != nil {
			wopiLogger.Error().Err(err).Msg("failed to dismantle reva token manager")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims.WopiContext.AccessToken = wopiContextAccessToken

		ctx = context.WithValue(ctx, wopiContextKey, claims.WopiContext)
		// authentication for the CS3 api
		ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, claims.WopiContext.AccessToken)
		ctx = ctxpkg.ContextSetUser(ctx, user)
		ctx = ctxpkg.ContextSetScopes(ctx, scopes)

		// include additional info in the context's logger
		wopiLogger = wopiLogger.With().
			Str("FileReference", claims.WopiContext.FileReference.String()).
			Str("ViewMode", claims.WopiContext.ViewMode.String()).
			Str("Requester", user.GetId().String()).
			Logger()
		ctx = wopiLogger.WithContext(ctx)

		// Validate that the file ID in the URL matches the WOPI token's file
		// reference. This check only applies to /wopi/files/ and /wopi/templates/
		// paths. Other WOPI-authenticated endpoints (e.g. /wopi/avatars/) don't
		// carry a file ID in the URL — they only need a valid WOPI token.
		if strings.Contains(r.URL.Path, "/files/") || strings.Contains(r.URL.Path, "/templates/") {
			hashedRef := helpers.HashResourceId(claims.WopiContext.FileReference.GetResourceId())
			fileID := parseWopiFileID(cfg, r.URL.Path)
			if claims.WopiContext.TemplateReference != nil {
				hashedTemplateRef := helpers.HashResourceId(claims.WopiContext.TemplateReference.GetResourceId())
				// the fileID could be one of the references within the access token if both are set
				// because we can use the access token to get the contents of the template file
				if fileID != hashedTemplateRef && fileID != hashedRef {
					wopiLogger.Error().Msg("file reference in the URL doesn't match the one inside the access token")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			} else {
				if fileID != hashedRef {
					wopiLogger.Error().Msg("file reference in the URL doesn't match the one inside the access token")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Extract a WopiContext from the context if possible. An error will be
// returned if there is no WopiContext
func WopiContextFromCtx(ctx context.Context) (WopiContext, error) {
	if wopiContext, ok := ctx.Value(wopiContextKey).(WopiContext); ok {
		return wopiContext, nil
	}
	return WopiContext{}, errors.New("no wopi context found")
}

// Set a WopiContext in the context. A new context will be returned with the
// add WopiContext inside. Note that the old one won't have the WopiContext set.
//
// This method is used for testing. The WopiContextAuthMiddleware should be
// used instead in order to provide a valid WopiContext
func WopiContextToCtx(ctx context.Context, wopiContext WopiContext) context.Context {
	return context.WithValue(ctx, wopiContextKey, wopiContext)
}

// The access token inside the wopiContext is expected to be decrypted.
// In order to generate the access token for WOPI, the reva token inside the
// wopiContext will be encrypted
func GenerateWopiToken(wopiContext WopiContext, cfg *config.Config, st microstore.Store) (string, int64, error) {
	if cfg.Wopi.ShortTokens && st == nil {
		return "", 0, errors.New("Cannot generate a short token without microstore")
	}

	cryptedReqAccessToken, err := EncryptAES([]byte(cfg.Wopi.Secret), wopiContext.AccessToken)
	if err != nil {
		return "", 0, err
	}

	cs3Claims := &jwt.RegisteredClaims{}
	cs3JWTparser := jwt.Parser{}
	_, _, err = cs3JWTparser.ParseUnverified(wopiContext.AccessToken, cs3Claims)
	if err != nil {
		return "", 0, err
	}

	wopiContext.AccessToken = cryptedReqAccessToken
	claims := &Claims{
		WopiContext: wopiContext,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: cs3Claims.ExpiresAt,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(cfg.Wopi.Secret))

	if cfg.Wopi.ShortTokens {
		c := md5.New()
		c.Write([]byte(accessToken))
		shortAccessToken := hex.EncodeToString(c.Sum(nil)) + strconv.FormatInt(time.Now().UnixNano(), 16)

		errWrite := st.Write(&microstore.Record{
			Key:    shortAccessToken,
			Value:  []byte(accessToken),
			Expiry: time.Until(claims.ExpiresAt.Time),
		})

		return shortAccessToken, claims.ExpiresAt.UnixMilli(), errWrite
	}

	return accessToken, claims.ExpiresAt.UnixMilli(), err
}

// parseWopiFileID extracts the file id from a wopi path
//
// If the file id is a jwt, it will be decoded and the file id will be extracted from the jwt claims.
// If the file id is not a jwt, it will be returned as is.
func parseWopiFileID(cfg *config.Config, path string) string {
	s := strings.Split(path, "/")
	if len(s) < 4 || (s[1] != "wopi" && s[2] != "files") {
		return path
	}
	// check if the fileid is a jwt
	if strings.Contains(s[3], ".") {
		token, err := jwt.Parse(s[3], func(_ *jwt.Token) (interface{}, error) {
			return []byte(cfg.Wopi.ProxySecret), nil
		})
		if err != nil {
			return s[3]
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return s[3]
		}

		f, ok := claims["f"].(string)
		if !ok {
			return s[3]
		}
		return f
	}
	// fileid is not a jwt
	return s[3]
}
