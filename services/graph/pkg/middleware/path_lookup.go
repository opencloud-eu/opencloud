package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
//   group 2: driveID
//   group 3: path (with leading slash)
//   group 4: suffix (with leading slash) — empty for direct item lookup
var rootColonRe = regexp.MustCompile(`^(.*?/v1(?:\.0|beta1)/drives/([^/]+))/root:(/.+?)(?::(/[^:]+))?:?$`)

// {HTTP.Root}/{version}/drives/{driveID}/items/{itemID}:/<path>[:/<suffix>][:]
//   group 1: prefix up to and including the driveID
//   group 2: driveID
//   group 3: anchor itemID
//   group 4: path (with leading slash)
//   group 5: suffix (with leading slash)
var itemColonRe = regexp.MustCompile(`^(.*?/v1(?:\.0|beta1)/drives/([^/]+))/items/([^/]+):(/.+?)(?::(/[^:]+))?:?$`)

type contextKey string

// OriginalPathContextKey holds the pre-rewrite request path for downstream
// tracing/logging consumers.
const OriginalPathContextKey contextKey = "graph.original_path"

// Sentinels distinguishing the resolution outcomes that map to specific HTTP
// statuses. Anything else surfaces as 500.
//
//   errPathNotFound   — path doesn't exist or the user lacks permission to
//                       see it. Both collapse to 404 (no existence disclosure).
//   errInvalidRequest — client sent a malformed input (unparseable drive/item
//                       id, drive/item mismatch in item-anchored form). 400.
//   errUnauthenticated — the gateway said the caller isn't authenticated for
//                       the lookup (token expired, cross-storage auth, etc.). 401.
var (
	errPathNotFound    = errors.New("path not found")
	errInvalidRequest  = errors.New("invalid request")
	errUnauthenticated = errors.New("unauthenticated")
)

// ResolveGraphPath returns middleware that detects MS Graph colon-syntax
// path lookup URLs and rewrites them to the canonical
// /{version}/drives/{driveID}/items/{resolvedItemID}{suffix} form before
// chi performs route matching. The requested API version is preserved
// (for example, "v1.0" or "v1beta1").
//
// Two URL shapes are recognized:
//
//	/{version}/drives/{driveID}/root:/<path>[:/<suffix>][:]
//	/{version}/drives/{driveID}/items/{itemID}:/<path>[:/<suffix>][:]
//
// Path resolution runs as the request user via CS3 Stat. NOT_FOUND and
// PERMISSION_DENIED collapse to 404 (no existence disclosure); operational
// failures (gateway selection, RPC transport, unexpected status) surface
// as 5xx so outages aren't masked.
//
// URLs without colon syntax fast-path through with a single substring check.
func ResolveGraphPath(gws pool.Selectable[gateway.GatewayAPIClient], logger log.Logger) func(http.Handler) http.Handler {
	l := logger.With().Str("middleware", "graphPathLookup").Logger()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fast-path: skip URLs that can't be colon-syntax.
			if !strings.Contains(r.URL.Path, ":") {
				next.ServeHTTP(w, r)
				return
			}

			original := r.URL.Path
			rewritten, err := rewriteColonPath(r.Context(), gws, l, original)
			switch {
			case errors.Is(err, errPathNotFound):
				l.Debug().Str("original", original).Msg("colon-path resolution: not found")
				errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
				return
			case errors.Is(err, errInvalidRequest):
				l.Debug().Str("original", original).Msg("colon-path resolution: invalid request")
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request")
				return
			case errors.Is(err, errUnauthenticated):
				l.Debug().Str("original", original).Msg("colon-path resolution: unauthenticated")
				errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, "unauthenticated")
				return
			case err != nil:
				l.Error().Err(err).Str("original", original).Msg("colon-path resolution: internal error")
				errorcode.GeneralException.Render(
					w, r, http.StatusInternalServerError, "internal error resolving path",
				)
				return
			case rewritten == "":
				// No colon-syntax match — pass through untouched.
				next.ServeHTTP(w, r)
				return
			}

			l.Debug().
				Str("original", original).
				Str("rewritten", rewritten).
				Msg("rewrote MS Graph colon-syntax path")

			ctx := context.WithValue(r.Context(), OriginalPathContextKey, original)
			r = r.WithContext(ctx)
			r.URL.Path = rewritten
			// Match the chi-escaping workaround in Graph.ServeHTTP — RawPath
			// must be a properly-escaped form of Path so chi's parameter
			// binding works for rewritten requests too. Drive/item IDs
			// contain `$` and `!`, both of which need percent-encoding for
			// chi to round-trip them through its tree match.
			// (See services/graph/pkg/service/v0/graph.go ServeHTTP and
			// https://github.com/go-chi/chi/issues/641#issuecomment-883156692.)
			//
			// EscapedPath() re-encodes Path because the existing RawPath
			// (set by Graph.ServeHTTP for the original URL) no longer
			// unescapes to our rewritten Path.
			r.URL.RawPath = r.URL.EscapedPath()
			next.ServeHTTP(w, r)
		})
	}
}

// colonMatch is the normalized result of matching either the root- or
// item-anchored colon-syntax regex. The two regexes capture different
// groups; this struct erases that difference for downstream resolution.
type colonMatch struct {
	prefix      string // canonical prefix up to and including /drives/{driveID}
	driveIDStr  string // captured driveID for item-anchored form (empty for root-anchored, where anchorIDStr already IS the driveID)
	anchorIDStr string // resource id to anchor path resolution (driveID for root, itemID for item-anchored)
	relPath     string // relative path with leading slash
	suffix      string // suffix with leading slash (e.g. "/children"); may be empty
}

// rewriteColonPath returns:
//   - ""        + nil                — no colon-syntax pattern matched (passthrough)
//   - rewritten + nil                — matched and resolved to a canonical URL
//   - ""        + errPathNotFound    — path doesn't exist or user lacks permission (404)
//   - ""        + errInvalidRequest  — malformed input (400)
//   - ""        + errUnauthenticated — gateway said caller isn't authenticated (401)
//   - ""        + other error        — operational / internal failure (5xx)
func rewriteColonPath(
	ctx context.Context,
	gws pool.Selectable[gateway.GatewayAPIClient],
	logger zerolog.Logger,
	urlPath string,
) (string, error) {
	var match colonMatch
	switch {
	case matchInto(rootColonRe, urlPath, &match, rootMatchExtract):
	case matchInto(itemColonRe, urlPath, &match, itemMatchExtract):
	default:
		return "", nil
	}

	// r.URL.Path is already URL-decoded by net/http; do NOT call url.PathUnescape
	// here, that would double-decode and let crafted inputs like "%252F" become
	// "/", changing path semantics.
	anchor, err := storagespace.ParseID(match.anchorIDStr)
	if err != nil {
		// Unparseable input is malformed by the client, not "not found".
		logger.Debug().Err(err).Str("anchor", match.anchorIDStr).Msg("invalid anchor id in colon path")
		return "", errInvalidRequest
	}

	// Item-anchored form ("/drives/{driveID}/items/{itemID}:/...") captures
	// driveID separately. Validate the itemID belongs to the given driveID
	// (storage + space prefix) — otherwise the request is malformed and we
	// short-circuit instead of doing an unnecessary CS3 Stat.
	if match.driveIDStr != "" {
		drive, err := storagespace.ParseID(match.driveIDStr)
		if err != nil {
			logger.Debug().Err(err).Str("driveID", match.driveIDStr).Msg("invalid drive id in colon path")
			return "", errInvalidRequest
		}
		if drive.GetStorageId() != anchor.GetStorageId() || drive.GetSpaceId() != anchor.GetSpaceId() {
			logger.Debug().
				Str("driveID", match.driveIDStr).
				Str("itemID", match.anchorIDStr).
				Msg("drive id does not match item id storage/space")
			return "", errInvalidRequest
		}
	}

	itemID, err := resolvePath(ctx, gws, logger, &anchor, match.relPath)
	if err != nil {
		return "", err
	}
	return buildCanonicalPath(match.prefix, itemID, match.suffix), nil
}

// matchInto runs the regex; on a hit, populates *out via extract and returns true.
// A small indirection that lets the switch in rewriteColonPath stay flat.
func matchInto(re *regexp.Regexp, s string, out *colonMatch, extract func([]string) colonMatch) bool {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	*out = extract(m)
	return true
}

func rootMatchExtract(m []string) colonMatch {
	// Root-anchored: anchorIDStr IS the driveID, so leave driveIDStr empty
	// to skip the drive/item match check (it would compare driveID to itself).
	return colonMatch{prefix: m[1], anchorIDStr: m[2], relPath: m[3], suffix: m[4]}
}

func itemMatchExtract(m []string) colonMatch {
	// Item-anchored: capture driveID (m[2]) so the resolver can validate it
	// matches the itemID's storage/space.
	return colonMatch{prefix: m[1], driveIDStr: m[2], anchorIDStr: m[3], relPath: m[4], suffix: m[5]}
}

// resolvePath translates a relative filesystem path (anchored at the given
// CS3 resource id) into the resolved item's id, running with the request
// user's permissions via CS3 Stat.
func resolvePath(
	ctx context.Context,
	gws pool.Selectable[gateway.GatewayAPIClient],
	logger zerolog.Logger,
	anchor *storageprovider.ResourceId,
	rawPath string,
) (string, error) {
	gw, err := gws.Next()
	if err != nil {
		return "", fmt.Errorf("gateway selector: %w", err)
	}

	// rawPath comes from r.URL.Path which is already decoded by net/http —
	// no extra unescape (would double-decode crafted "%25xx" inputs).
	statRes, err := gw.Stat(ctx, &storageprovider.StatRequest{
		Ref: &storageprovider.Reference{
			ResourceId: anchor,
			Path:       utils.MakeRelativePath(rawPath),
		},
	})
	if err != nil {
		return "", fmt.Errorf("CS3 Stat: %w", err)
	}

	switch statRes.GetStatus().GetCode() {
	case cs3rpc.Code_CODE_OK:
		// fall through
	case cs3rpc.Code_CODE_NOT_FOUND, cs3rpc.Code_CODE_PERMISSION_DENIED:
		return "", errPathNotFound
	case cs3rpc.Code_CODE_UNAUTHENTICATED:
		return "", errUnauthenticated
	default:
		return "", fmt.Errorf(
			"CS3 Stat returned %s: %s",
			statRes.GetStatus().GetCode(),
			statRes.GetStatus().GetMessage(),
		)
	}

	id := statRes.GetInfo().GetId()
	if id == nil {
		return "", fmt.Errorf("CS3 Stat returned OK but missing Info.Id")
	}
	return storagespace.FormatResourceID(id), nil
}

func buildCanonicalPath(prefix, itemID, suffix string) string {
	// r.URL.Path is the decoded form per Go's net/http convention; chi tree
	// matching reads it directly. Insert the raw itemID without escaping so
	// chi's {itemID} param captures the same string the handler will see
	// after parseIDParam unescapes (no-op for already-decoded chars).
	return prefix + "/items/" + itemID + suffix
}
