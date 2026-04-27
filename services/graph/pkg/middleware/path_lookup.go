package middleware

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// {HTTP.Root}/{version}/drives/{driveID}/root:/<path>[:/<suffix>][:]
// where {version} is "v1.0" or "v1beta1" — preserves the requested version
// so it lands on a registered route.
//   group 1: prefix up to and including the driveID
//   group 2: driveID (URL-encoded)
//   group 3: path (with leading slash)
//   group 4: suffix (with leading slash) — empty for direct item lookup
var rootColonRe = regexp.MustCompile(`^(.*?/v1(?:\.0|beta1)/drives/([^/]+))/root:(/.+?)(?::(/[^:]+))?:?$`)

// {HTTP.Root}/{version}/drives/{driveID}/items/{itemID}:/<path>[:/<suffix>][:]
//   group 1: prefix up to and including the driveID
//   group 2: driveID (URL-encoded)
//   group 3: anchor itemID (URL-encoded)
//   group 4: path (with leading slash)
//   group 5: suffix (with leading slash)
var itemColonRe = regexp.MustCompile(`^(.*?/v1(?:\.0|beta1)/drives/([^/]+))/items/([^/]+):(/.+?)(?::(/[^:]+))?:?$`)

type contextKey string

// OriginalPathContextKey holds the pre-rewrite request path for downstream
// tracing/logging consumers.
const OriginalPathContextKey contextKey = "graph.original_path"

// ResolveGraphPath returns middleware that detects MS Graph colon-syntax
// path lookup URLs and rewrites them to the canonical
// /v1beta1/drives/{driveID}/items/{resolvedItemID}/{suffix} form before
// chi performs route matching.
//
// Two URL shapes are recognized:
//
//	/v1beta1/drives/{driveID}/root:/<path>[:/<suffix>][:]
//	/v1beta1/drives/{driveID}/items/{itemID}:/<path>[:/<suffix>][:]
//
// Path resolution runs as the request user via CS3 Stat. Both NOT_FOUND
// and PERMISSION_DENIED collapse to a 404 response so existence isn't
// disclosed to unauthorized callers.
//
// URLs without colon syntax fast-path through the middleware with a
// single substring check.
func ResolveGraphPath(gws pool.Selectable[gateway.GatewayAPIClient], logger log.Logger) func(http.Handler) http.Handler {
	l := logger.With().Str("middleware", "graphPathLookup").Logger()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fast-path: skip URLs that can't be colon-syntax
			if !strings.Contains(r.URL.Path, ":") {
				next.ServeHTTP(w, r)
				return
			}

			original := r.URL.Path
			rewritten, matched := rewriteColonPath(r.Context(), gws, l, original)
			if !matched {
				next.ServeHTTP(w, r)
				return
			}
			if rewritten == "" {
				// Resolution failed: not found, permission denied, or invalid input.
				// Collapse to 404 to avoid disclosing existence.
				l.Debug().Str("original", original).Msg("colon-path resolution failed; returning 404")
				errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
				return
			}

			l.Debug().
				Str("original", original).
				Str("rewritten", rewritten).
				Msg("rewrote MS Graph colon-syntax path")

			// Stash original URL for downstream logging/tracing.
			ctx := context.WithValue(r.Context(), OriginalPathContextKey, original)
			r = r.WithContext(ctx)
			r.URL.Path = rewritten
			r.URL.RawPath = ""
			next.ServeHTTP(w, r)
		})
	}
}

// rewriteColonPath returns:
//   - rewritten URL + true  when the URL matched a colon-syntax pattern and resolution succeeded
//   - "" + true             when the URL matched a colon-syntax pattern but resolution failed (caller should 404)
//   - "" + false            when the URL did not match any colon-syntax pattern
func rewriteColonPath(ctx context.Context, gws pool.Selectable[gateway.GatewayAPIClient], logger zerolog.Logger, urlPath string) (string, bool) {
	if m := rootColonRe.FindStringSubmatch(urlPath); m != nil {
		prefix, driveIDStr, relPath, suffix := m[1], m[2], m[3], m[4]
		driveID, err := decodeAndParseID(driveIDStr)
		if err != nil {
			logger.Debug().Err(err).Str("driveID", driveIDStr).Msg("invalid driveID in colon path")
			return "", true
		}
		itemID, ok := resolvePath(ctx, gws, logger, &driveID, relPath)
		if !ok {
			return "", true
		}
		return buildCanonicalPath(prefix, itemID, suffix), true
	}

	if m := itemColonRe.FindStringSubmatch(urlPath); m != nil {
		prefix, _, anchorIDStr, relPath, suffix := m[1], m[2], m[3], m[4], m[5]
		anchorID, err := decodeAndParseID(anchorIDStr)
		if err != nil {
			logger.Debug().Err(err).Str("itemID", anchorIDStr).Msg("invalid item anchor in colon path")
			return "", true
		}
		itemID, ok := resolvePath(ctx, gws, logger, &anchorID, relPath)
		if !ok {
			return "", true
		}
		return buildCanonicalPath(prefix, itemID, suffix), true
	}

	return "", false
}

func resolvePath(ctx context.Context, gws pool.Selectable[gateway.GatewayAPIClient], logger zerolog.Logger, anchor *storageprovider.ResourceId, rawPath string) (string, bool) {
	gw, err := gws.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select gateway client")
		return "", false
	}

	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		logger.Debug().Err(err).Str("path", rawPath).Msg("failed to URL-decode path segment")
		return "", false
	}

	statRes, err := gw.Stat(ctx, &storageprovider.StatRequest{
		Ref: &storageprovider.Reference{
			ResourceId: anchor,
			Path:       utils.MakeRelativePath(decoded),
		},
	})
	if err != nil {
		logger.Error().Err(err).Msg("Stat call failed during colon-path resolution")
		return "", false
	}

	switch statRes.GetStatus().GetCode() {
	case cs3rpc.Code_CODE_OK:
		// fall through
	case cs3rpc.Code_CODE_NOT_FOUND, cs3rpc.Code_CODE_PERMISSION_DENIED:
		// Both collapse to "not found" — never tell unauthorized callers
		// that the resource exists.
		return "", false
	default:
		logger.Debug().
			Str("code", statRes.GetStatus().GetCode().String()).
			Str("message", statRes.GetStatus().GetMessage()).
			Msg("unexpected Stat status during colon-path resolution")
		return "", false
	}

	id := statRes.GetInfo().GetId()
	if id == nil {
		return "", false
	}
	return storagespace.FormatResourceID(id), true
}

func decodeAndParseID(s string) (storageprovider.ResourceId, error) {
	decoded, err := url.PathUnescape(s)
	if err != nil {
		return storageprovider.ResourceId{}, err
	}
	return storagespace.ParseID(decoded)
}

func buildCanonicalPath(prefix, itemID, suffix string) string {
	// r.URL.Path is the decoded form per Go's net/http convention; chi tree
	// matching reads it directly. Insert the raw itemID without escaping so
	// chi's {itemID} param captures the same string the handler will see
	// after parseIDParam unescapes (no-op for already-decoded chars).
	return prefix + "/items/" + itemID + suffix
}
