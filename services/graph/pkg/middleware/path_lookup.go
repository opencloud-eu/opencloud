package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// The middleware is attached to the /drives/{driveID} sub-routers, so by the
// time it runs chi has already consumed the {version}/drives/{driveID} prefix
// and exposes the remainder via chi.RouteContext().RoutePath. parseColonPath
// therefore only needs to handle the part below the drive:
//
//	root-anchored: /root:/<path>[:/<suffix>][:]
//	item-anchored: /items/{itemID}:/<path>[:/<suffix>][:]

type contextKey string

// OriginalPathContextKey holds the pre-rewrite request path for downstream
// tracing/logging consumers.
const OriginalPathContextKey contextKey = "graph.original_path"

// Sentinels distinguishing the resolution outcomes that map to specific HTTP
// statuses. Anything else surfaces as 500.
//
//	errPathNotFound   - path doesn't exist or the user lacks permission to
//	                    see it. Both collapse to 404 (no existence disclosure).
//	errInvalidRequest - client sent a malformed input (unparseable drive/item
//	                    id, drive/item mismatch in item-anchored form). 400.
//	errUnauthenticated - the gateway said the caller isn't authenticated for
//	                    the lookup (token expired, cross-storage auth, etc.). 401.
var (
	errPathNotFound    = errors.New("path not found")
	errInvalidRequest  = errors.New("invalid request")
	errUnauthenticated = errors.New("unauthenticated")
)

// ResolveGraphPath returns middleware that detects MS Graph colon-syntax path
// lookup URLs and rewrites chi's internal route path to the canonical
// /items/{resolvedItemID}{suffix} form so the request lands on the existing
// /drives/{driveID}/items/{itemID}... routes.
//
// It must be attached to the /drives/{driveID} sub-routers (it reads driveID
// from chi.URLParam and matches against chi.RouteContext().RoutePath). chi runs
// a sub-router's middleware chain before that sub-router performs its own route
// matching, so rewriting RoutePath here re-routes the request to a different
// leaf. (Rewriting r.URL.Path would NOT work at this level: once chi has
// descended into a sub-router, routeHTTP matches against rctx.RoutePath and
// ignores r.URL.Path.)
//
// Two URL shapes are recognized:
//
//	/drives/{driveID}/root:/<path>[:/<suffix>][:]
//	/drives/{driveID}/items/{itemID}:/<path>[:/<suffix>][:]
//
// Path resolution runs as the request user via CS3 Stat. NOT_FOUND and
// PERMISSION_DENIED collapse to 404 (no existence disclosure); operational
// failures (gateway selection, RPC transport, unexpected status) surface
// as 5xx so outages aren't masked.
//
// Requests whose RoutePath contains no colon fast-path through untouched.
func ResolveGraphPath(gws pool.Selectable[gateway.GatewayAPIClient], logger log.Logger) func(http.Handler) http.Handler {
	l := logger.With().Str("middleware", "graphPathLookup").Logger()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rctx := chi.RouteContext(r.Context())

			// Fast-path: skip anything that can't be colon-syntax. RoutePath
			// is the part below /drives/{driveID}, e.g. "/items/{id}/children"
			// for a normal request - no colon, so this returns immediately.
			if rctx == nil || !strings.Contains(rctx.RoutePath, ":") {
				next.ServeHTTP(w, r)
				return
			}

			driveID := chi.URLParam(r, "driveID")
			original := r.URL.Path
			rewritten, err := rewriteColonPath(r.Context(), gws, l, driveID, rctx.RoutePath)
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
				// No colon-syntax match - pass through untouched.
				next.ServeHTTP(w, r)
				return
			}

			l.Debug().
				Str("original", original).
				Str("rewritten", rewritten).
				Msg("colon-path resolution: rewrote")

			// Preserve the original (unmodified) request path for downstream
			// tracing/logging. r.URL.Path itself stays untouched; only chi's
			// internal RoutePath is rewritten.
			r = r.WithContext(context.WithValue(r.Context(), OriginalPathContextKey, original))
			rctx.RoutePath = rewritten
			next.ServeHTTP(w, r)
		})
	}
}

// colonMatch is the normalized result of parseColonPath for either the root-
// or item-anchored colon-syntax shape. The two shapes carry different parts;
// this struct erases that difference for downstream resolution.
//
// itemAnchorID, relPath and suffix hold the raw (still percent-encoded)
// substrings from RoutePath; rewriteColonPath decodes them as needed.
type colonMatch struct {
	isItemAnchored bool   // item-anchored form: anchor is itemAnchorID, validate against driveID
	itemAnchorID   string // itemID from the path for the item-anchored form (empty for root-anchored)
	relPath        string // relative path with leading slash
	suffix         string // suffix with leading slash (e.g. "/children"); may be empty
}

// rewriteColonPath returns:
//   - ""        + nil                - no colon-syntax pattern matched (passthrough)
//   - rewritten + nil                - matched and resolved to a canonical RoutePath
//   - ""        + errPathNotFound    - path doesn't exist or user lacks permission (404)
//   - ""        + errInvalidRequest  - malformed input (400)
//   - ""        + errUnauthenticated - gateway said caller isn't authenticated (401)
//   - ""        + other error        - operational / internal failure (5xx)
//
// driveIDParam is the {driveID} route param (raw chi.URLParam value); routePath
// is chi.RouteContext().RoutePath (the part below /drives/{driveID}).
func rewriteColonPath(
	ctx context.Context,
	gws pool.Selectable[gateway.GatewayAPIClient],
	logger zerolog.Logger,
	driveIDParam string,
	routePath string,
) (string, error) {
	match, ok := parseColonPath(routePath)
	if !ok {
		return "", nil
	}

	// RoutePath follows chi's RawPath, i.e. the percent-encoded wire form
	// (e.g. "/Documents/My%20File"). A single PathUnescape reproduces exactly
	// what net/http put in r.URL.Path; it is NOT a double-decode (a crafted
	// "%252F" decodes once to "%2F", matching the decoded r.URL.Path, not "/").
	driveID, err := url.PathUnescape(driveIDParam)
	if err != nil {
		logger.Debug().Err(err).Str("driveID", driveIDParam).Msg("undecodable drive id in colon path")
		return "", errInvalidRequest
	}

	anchorIDStr := driveID
	if match.isItemAnchored {
		anchorIDStr, err = url.PathUnescape(match.itemAnchorID)
		if err != nil {
			logger.Debug().Err(err).Str("itemID", match.itemAnchorID).Msg("undecodable item id in colon path")
			return "", errInvalidRequest
		}
	}

	anchor, err := storagespace.ParseID(anchorIDStr)
	if err != nil {
		// Unparseable input is malformed by the client, not "not found".
		logger.Debug().Err(err).Str("anchor", anchorIDStr).Msg("invalid anchor id in colon path")
		return "", errInvalidRequest
	}

	// Item-anchored form: the itemID comes from the path, driveID from the
	// route param. Validate the itemID belongs to the given driveID (storage +
	// space prefix) - otherwise the request is malformed and we short-circuit
	// instead of doing an unnecessary CS3 Stat.
	if match.isItemAnchored {
		drive, err := storagespace.ParseID(driveID)
		if err != nil {
			logger.Debug().Err(err).Str("driveID", driveID).Msg("invalid drive id in colon path")
			return "", errInvalidRequest
		}
		if drive.GetStorageId() != anchor.GetStorageId() || drive.GetSpaceId() != anchor.GetSpaceId() {
			logger.Debug().
				Str("driveID", driveID).
				Str("itemID", anchorIDStr).
				Msg("drive id does not match item id storage/space")
			return "", errInvalidRequest
		}
	}

	relPath, err := url.PathUnescape(match.relPath)
	if err != nil {
		logger.Debug().Err(err).Str("relPath", match.relPath).Msg("undecodable path in colon path")
		return "", errInvalidRequest
	}

	itemID, err := resolvePath(ctx, gws, &anchor, relPath)
	if err != nil {
		return "", err
	}
	return buildCanonicalRoutePath(itemID, match.suffix), nil
}

// parseColonPath splits a colon-syntax RoutePath (the part below
// /drives/{driveID}) into its parts, or reports ok=false when the path is not
// colon-syntax and should pass through untouched.
//
// Below the drive the grammar is one of:
//
//	/root:/<path>[:/<suffix>][:]
//	/items/<itemID>:/<path>[:/<suffix>][:]
//
// The structural delimiter is ":/" - a colon immediately followed by the
// leading slash of the path or suffix - not a bare ":". Splitting on ":/"
// keeps a ':' *inside* a file or directory name as data instead of mistaking
// it for a delimiter (segments are '/'-separated and never start with ':').
// For example:
//
//	/root:/Documents                 -> path "/Documents"
//	/root:/Documents:                -> path "/Documents"
//	/root:/Documents:/children       -> path "/Documents", suffix "/children"
//	/root:/a:b.txt                   -> path "/a:b.txt" (colon kept in the name)
//	/items/{id}:/notes.txt:/children -> itemID "{id}", path "/notes.txt", suffix "/children"
//
// A raw ':' at a segment boundary is ambiguous - "/root:/foo:/bar" reads as
// path "/foo" with suffix "/bar", and a trailing ':' is the no-suffix
// terminator. To address a name whose ':' sits at a boundary (e.g. a name
// ending in ':'), percent-encode it as "%3A": the split works on the literal
// ":/", so "%3A" is never a delimiter and decodes back to ':' with the rest of
// the path (e.g. "/root:/weird%3A/file.txt" -> "/weird:/file.txt"). MS Graph
// sidesteps this by forbidding ':' in names entirely; OpenCloud allows it (via
// WebDAV), so "%3A" is how such names are reached here.
//
// The returned fields are the raw (still percent-encoded) substrings; the
// caller decodes them.
func parseColonPath(routePath string) (colonMatch, bool) {
	var m colonMatch

	// The anchor is separated from the path by the first ":/"; no ":/" at all
	// means this isn't colon syntax and should pass through.
	anchor, rest, found := strings.Cut(routePath, ":/")
	if !found {
		return m, false
	}
	// Cut consumed the leading '/' of the path along with the ":/" delimiter;
	// the path is absolute, so put it back.
	rest = "/" + rest

	switch {
	case anchor == "/root":
		// Root-anchored: path resolution later anchors on the {driveID} route
		// param, so there's no item id to read from the URL.
	case strings.HasPrefix(anchor, "/items/"):
		// Item-anchored: the anchor is /items/{itemID} with a single-segment
		// id. A '/' inside the id means this is an ordinary /items/{id}/...
		// request that just happens to contain a ":/" further along.
		itemID := strings.TrimPrefix(anchor, "/items/")
		if itemID == "" || strings.Contains(itemID, "/") {
			return m, false
		}
		m.isItemAnchored = true
		m.itemAnchorID = itemID
	default:
		return m, false
	}

	// rest is "/<path>[:/<suffix>][:]". Split path from an optional suffix on
	// the next ":/" (the suffix regains the leading '/' Cut consumed), then
	// drop a single trailing ':' - the "...:" no-suffix terminator. A suffix
	// therefore requires an explicit ":/", so "/root:/foo/children" is the path
	// "/foo/children", not "/foo" with suffix "/children".
	m.relPath = rest
	if path, suffix, found := strings.Cut(rest, ":/"); found {
		m.relPath, m.suffix = path, "/"+suffix
	}
	m.relPath = strings.TrimSuffix(m.relPath, ":")

	// Shape requirements: the path must be absolute and non-empty ("/x"), and a
	// present suffix must be absolute, non-empty and colon-free (suffixes are
	// fixed API routes like "/children", never file paths).
	if len(m.relPath) < 2 || m.relPath[0] != '/' {
		return m, false
	}
	if m.suffix != "" && (len(m.suffix) < 2 || strings.Contains(m.suffix, ":")) {
		return m, false
	}
	return m, true
}

// resolvePath translates a relative filesystem path (anchored at the given
// CS3 resource id) into the resolved item's id, running with the request
// user's permissions via CS3 Stat.
func resolvePath(
	ctx context.Context,
	gws pool.Selectable[gateway.GatewayAPIClient],
	anchor *storageprovider.ResourceId,
	relPath string,
) (string, error) {
	gw, err := gws.Next()
	if err != nil {
		return "", fmt.Errorf("gateway selector: %w", err)
	}

	// relPath is already decoded (PathUnescape'd once by the caller), matching
	// the form a normal handler would receive from r.URL.Path.
	statRes, err := gw.Stat(ctx, &storageprovider.StatRequest{
		Ref: &storageprovider.Reference{
			ResourceId: anchor,
			Path:       utils.MakeRelativePath(relPath),
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

// buildCanonicalRoutePath produces the RoutePath chi should match against after
// the rewrite, relative to /drives/{driveID}: /items/{itemID}{suffix}.
//
// itemID is inserted literally (the FormatResourceID output, e.g. with `$` and
// `!` sub-delims). chi binds it verbatim into the {itemID} param; downstream
// handlers (parseIDParam, GetDriveAndItemIDParam) call url.PathUnescape on the
// param, which is a no-op for these literal sub-delims, so the id round-trips.
func buildCanonicalRoutePath(itemID, suffix string) string {
	return "/items/" + itemID + suffix
}
