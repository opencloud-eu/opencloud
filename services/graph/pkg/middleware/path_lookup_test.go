package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
)

const (
	testDriveID = "storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd"
	testItemID  = "storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d"
)

func newTestSelector(t *testing.T, gw *cs3mocks.GatewayAPIClient) pool.Selectable[gateway.GatewayAPIClient] {
	t.Helper()
	// Unique key per test so pool's selector cache doesn't hand back a stale gw.
	// t.Cleanup removes the entry from pool's global selectors map after the
	// subtest, so global state doesn't grow across the suite.
	svcName := "TestGatewaySelector"
	key := "test.gateway." + t.Name()
	pool.RemoveSelector(svcName + key)
	t.Cleanup(func() { pool.RemoveSelector(svcName + key) })
	return pool.GetSelector[gateway.GatewayAPIClient](
		svcName,
		key,
		func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient { return gw },
	)
}

// statResponse builds a CS3 StatResponse with the given status code, optionally
// returning a resource info populated with testItemID for OK responses.
func statResponse(code cs3rpc.Code, withInfo bool) *storageprovider.StatResponse {
	res := &storageprovider.StatResponse{Status: &cs3rpc.Status{Code: code}}
	if withInfo {
		res.Info = &storageprovider.ResourceInfo{
			Id: &storageprovider.ResourceId{
				StorageId: "storage-users-1",
				SpaceId:   "f503f6fe-2656-4b0f-8289-fb3184962dfd",
				OpaqueId:  "f0e20017-9cba-498a-87e5-3467b976604d",
			},
		}
	}
	return res
}

// leafCapture records what the matched leaf handler saw, so tests can assert
// the middleware rewrote chi's route path correctly and the request reached the
// intended /items/{itemID}... handler with the resolved id bound as a param.
type leafCapture struct {
	hit      string // which leaf was reached ("" = none)
	urlPath  string // r.URL.Path as seen by the handler (must stay the original)
	driveID  string // chi.URLParam(driveID)
	itemID   string // resolved item id, decoded via PathUnescape
	original any    // OriginalPathContextKey value
}

// newGraphTestRouter wires ResolveGraphPath into a chi router that mirrors the
// production /drives/{driveID} nesting for both API versions, including the
// RawPath workaround Graph.ServeHTTP applies. Driving requests through real chi
// routing is deliberate: it exercises chi's actual behavior (sub-router
// middleware ordering, RoutePath encoding, param round-trip) indirectly, so a
// chi upgrade that changes any of it fails these tests instead of silently
// breaking colon-path lookups in production.
func newGraphTestRouter(t *testing.T, gw *cs3mocks.GatewayAPIClient) (http.Handler, *leafCapture) {
	t.Helper()
	cap := &leafCapture{}
	selector := newTestSelector(t, gw)

	leaf := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cap.hit = name
			cap.urlPath = r.URL.Path
			cap.driveID = chi.URLParam(r, "driveID")
			// v1.0 binds the item param as {driveItemID}, v1beta1 as {itemID}.
			raw := chi.URLParam(r, "itemID")
			if raw == "" {
				raw = chi.URLParam(r, "driveItemID")
			}
			// Downstream handlers PathUnescape the param before parsing the id;
			// mirror that here so we assert on the recovered id.
			cap.itemID, _ = url.PathUnescape(raw)
			cap.original = r.Context().Value(middleware.OriginalPathContextKey)
			w.WriteHeader(http.StatusOK)
		}
	}

	m := chi.NewMux()
	// Mirror Graph.ServeHTTP: RawPath drives chi's tree walk, which is what
	// makes RoutePath carry the percent-encoded wire form.
	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.RawPath = r.URL.EscapedPath()
			next.ServeHTTP(w, r)
		})
	})
	// Leaf routes mirror production exactly (service.go), per version, so the
	// tests assert reachability that production actually reproduces: a rewrite
	// to a suffix that doesn't exist for that version must 404 here too.
	m.Route("/graph", func(r chi.Router) {
		r.Route("/v1beta1/drives/{driveID}", func(r chi.Router) {
			r.Use(middleware.ResolveGraphPath(selector, log.NopLogger()))
			r.Route("/items/{itemID}", func(r chi.Router) {
				r.Get("/", leaf("item"))
				r.Post("/createLink", leaf("createLink"))
				r.Route("/permissions", func(r chi.Router) {
					r.Get("/", leaf("permissions"))
					r.Post("/{permissionID}/setPassword", leaf("setPassword"))
				})
			})
		})
		r.Route("/v1.0/drives/{driveID}", func(r chi.Router) {
			r.Use(middleware.ResolveGraphPath(selector, log.NopLogger()))
			r.Route("/items/{driveItemID}", func(r chi.Router) {
				r.Get("/", leaf("item"))
				r.Get("/children", leaf("children"))
			})
		})
	})
	return m, cap
}

func TestResolveGraphPath(t *testing.T) {
	tests := []struct {
		name             string
		method           string // defaults to GET when empty
		urlPath          string
		statCode         cs3rpc.Code
		expectStatCalled bool
		expectStatus     int
		expectHit        string // leaf the request must land on ("" = none reached)
		expectItemID     string // resolved item id the leaf must see (after PathUnescape)
	}{
		{
			name:             "non-colon URL routes normally without a Stat",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/items/" + testItemID + "/children",
			expectStatCalled: false,
			expectStatus:     http.StatusOK,
			expectHit:        "children",
			expectItemID:     testItemID,
		},
		{
			name:             "colon URL not matching the lookup pattern passes through (404, no Stat)",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/items/" + testItemID + "/foo:bar",
			expectStatCalled: false,
			expectStatus:     http.StatusNotFound,
			expectHit:        "",
		},
		{
			// Anchoring: the colon-syntax shape appearing under a junk prefix
			// (i.e. NOT at the configured HTTP root) must not trigger a rewrite
			// or a CS3 Stat. Because the middleware is mounted under the
			// /graph root, chi never routes such a path into it (404 first) -
			// no /foo/.../v1.0/drives/...:/... can over-match.
			name:             "colon-syntax under a junk prefix does not match (anchored on HTTP root)",
			urlPath:          "/foo/graph/v1.0/drives/" + testDriveID + "/root:/Documents:/children",
			expectStatCalled: false,
			expectStatus:     http.StatusNotFound,
			expectHit:        "",
		},
		{
			name:             "v1.0 root-anchored with /children rewrites and routes",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "children",
			expectItemID:     testItemID,
		},
		{
			// Mirrors acceptance scenario 5: v1beta1 root-anchored with a
			// /permissions suffix (a real v1beta1-only route, GET).
			name:             "v1beta1 root-anchored with /permissions rewrites and routes",
			urlPath:          "/graph/v1beta1/drives/" + testDriveID + "/root:/folder1:/permissions",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "permissions",
			expectItemID:     testItemID,
		},
		{
			// Non-GET colon request: createLink is POST on v1beta1.
			name:             "v1beta1 root-anchored with /createLink (POST) rewrites and routes",
			method:           http.MethodPost,
			urlPath:          "/graph/v1beta1/drives/" + testDriveID + "/root:/folder1:/createLink",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "createLink",
			expectItemID:     testItemID,
		},
		{
			// Multi-segment suffix: /permissions/{permissionID}/setPassword on
			// v1beta1 (POST). Pins that parseColonPath carries a suffix with
			// embedded slashes through the rewrite into a real nested route.
			name:             "v1beta1 root-anchored with multi-segment suffix rewrites and routes",
			method:           http.MethodPost,
			urlPath:          "/graph/v1beta1/drives/" + testDriveID + "/root:/folder1:/permissions/perm-1/setPassword",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "setPassword",
			expectItemID:     testItemID,
		},
		{
			// Item-anchored colon syntax on v1beta1 (POST createLink).
			name:             "v1beta1 item-anchored with /createLink (POST) rewrites and routes",
			method:           http.MethodPost,
			urlPath:          "/graph/v1beta1/drives/" + testDriveID + "/items/" + testItemID + ":/notes.txt:/createLink",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "createLink",
			expectItemID:     testItemID,
		},
		{
			name:             "trailing colon (no suffix) rewrites to bare item URL",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "item",
			expectItemID:     testItemID,
		},
		{
			name:             "no trailing colon, no suffix rewrites to bare item URL",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "item",
			expectItemID:     testItemID,
		},
		{
			name:             "deep path rewrites correctly",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents/Reports:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "children",
			expectItemID:     testItemID,
		},
		{
			// A trailing segment that looks like a suffix ("children") is only a
			// suffix when a SECOND colon introduces it. Without it, the whole
			// thing is the path, so this must resolve to the bare item (GET "/"),
			// NOT to /items/{id}/children. If the parser wrongly split it, the
			// request would land on the "children" leaf and this fails.
			name:             "trailing suffix-looking segment without a second colon stays in the path",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "item",
			expectItemID:     testItemID,
		},
		{
			// A ':' inside a file name is data, not a delimiter (the delimiter
			// is ":/"). This must resolve the whole path including the colon and
			// route to the bare item.
			name:             "colon inside a file name is kept in the path",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/folder1/re:port.txt",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "item",
			expectItemID:     testItemID,
		},
		{
			// Colon inside the file name AND a real ":/"-delimited suffix: the
			// name keeps its colon, the suffix still routes.
			name:             "colon in file name with a real suffix",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/folder1/re:port.txt:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "children",
			expectItemID:     testItemID,
		},
		{
			name:             "item-anchored colon syntax rewrites",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/items/" + testItemID + ":/notes.txt:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectHit:        "children",
			expectItemID:     testItemID,
		},
		{
			name:             "Stat NOT_FOUND returns 404 without reaching a leaf",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Missing:",
			statCode:         cs3rpc.Code_CODE_NOT_FOUND,
			expectStatCalled: true,
			expectStatus:     http.StatusNotFound,
			expectHit:        "",
		},
		{
			// CRITICAL security test: PERMISSION_DENIED must not leak existence.
			// We collapse it to 404, identical to NOT_FOUND, so an unauthorized
			// caller can't distinguish "doesn't exist" from "exists but hidden".
			name:             "Stat PERMISSION_DENIED returns 404 (not 403) - don't disclose existence",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Restricted:",
			statCode:         cs3rpc.Code_CODE_PERMISSION_DENIED,
			expectStatCalled: true,
			expectStatus:     http.StatusNotFound,
			expectHit:        "",
		},
		{
			// Operational/unexpected CS3 statuses must NOT collapse to 404 -
			// that would mask outages. Surface as 500 like other graph handlers.
			name:             "Stat unexpected status returns 500 (not 404 - don't mask outages)",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Anything:",
			statCode:         cs3rpc.Code_CODE_INTERNAL,
			expectStatCalled: true,
			expectStatus:     http.StatusInternalServerError,
			expectHit:        "",
		},
		{
			// UNAUTHENTICATED is its own distinct outcome - must surface as 401,
			// not 500, so clients can detect "your token is bad" vs "server error".
			name:             "Stat UNAUTHENTICATED returns 401",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:",
			statCode:         cs3rpc.Code_CODE_UNAUTHENTICATED,
			expectStatCalled: true,
			expectStatus:     http.StatusUnauthorized,
			expectHit:        "",
		},
		{
			// Item-anchored form with a driveID that doesn't match the itemID's
			// storage/space - the request is malformed; short-circuit to 400
			// instead of doing a Stat that would only fail downstream.
			name:             "drive id and item id storage/space mismatch returns 400",
			urlPath:          "/graph/v1.0/drives/storage-users-2$other-space-id/items/" + testItemID + ":/notes.txt:/children",
			expectStatCalled: false,
			expectStatus:     http.StatusBadRequest,
			expectHit:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := &cs3mocks.GatewayAPIClient{}
			if tt.expectStatCalled {
				gw.On("Stat", mock.Anything, mock.Anything).
					Return(statResponse(tt.statCode, tt.statCode == cs3rpc.Code_CODE_OK), nil)
			}

			method := tt.method
			if method == "" {
				method = http.MethodGet
			}
			router, cap := newGraphTestRouter(t, gw)
			req := httptest.NewRequest(method, "http://localhost"+tt.urlPath, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectStatus, rr.Code, "status code")
			assert.Equal(t, tt.expectHit, cap.hit, "leaf handler reached")

			if tt.expectHit != "" {
				assert.Equal(t, testDriveID, cap.driveID, "driveID param")
				assert.Equal(t, tt.expectItemID, cap.itemID, "resolved item id seen by leaf")
				// r.URL.Path must stay the original; only chi's RoutePath is rewritten.
				assert.Equal(t, tt.urlPath, cap.urlPath, "r.URL.Path must remain the original request path")
			}

			if tt.expectStatCalled {
				gw.AssertCalled(t, "Stat", mock.Anything, mock.Anything)
			} else {
				gw.AssertNotCalled(t, "Stat", mock.Anything, mock.Anything)
			}
		})
	}
}

// TestResolveGraphPath_DecodesEncodedPath pins the decoding contract: chi's
// RoutePath carries the percent-encoded wire form (because Graph.ServeHTTP sets
// RawPath), and the middleware must hand CS3 Stat the decoded path - exactly
// what a normal handler reading r.URL.Path would get. Driving this through real
// chi routing means a chi change to RoutePath encoding would break this test.
func TestResolveGraphPath_DecodesEncodedPath(t *testing.T) {
	tests := []struct {
		name         string
		urlPath      string
		expectedStat string
	}{
		{
			name:         "space encoded as %20 is decoded for Stat",
			urlPath:      "/graph/v1.0/drives/" + testDriveID + "/root:/Documents/My%20File:/children",
			expectedStat: "./Documents/My File",
		},
		{
			// A crafted double-encoding must be decoded exactly once (to the
			// literal "%2F"), never twice into a path separator.
			name:         "double-encoded %252F decodes once, not twice",
			urlPath:      "/graph/v1.0/drives/" + testDriveID + "/root:/a%252Fb:/children",
			expectedStat: "./a%2Fb",
		},
		{
			// Without a second colon the whole remainder is the path: Stat must
			// receive "/Documents/children", not "/Documents" (which would mean
			// "/children" was wrongly treated as a suffix).
			name:         "suffix-looking segment without a second colon is part of the Stat path",
			urlPath:      "/graph/v1.0/drives/" + testDriveID + "/root:/Documents/children",
			expectedStat: "./Documents/children",
		},
		{
			// A ':' inside a file name (delimiter is ":/") must reach Stat as
			// part of the path, not be treated as a path/suffix separator.
			name:         "colon inside a file name reaches Stat as part of the path",
			urlPath:      "/graph/v1.0/drives/" + testDriveID + "/root:/folder1/re:port.txt",
			expectedStat: "./folder1/re:port.txt",
		},
		{
			// A ':' at a segment boundary is ambiguous raw, so it must be
			// percent-encoded. "%3A" is never seen as the ":/" delimiter and
			// decodes back to a literal ':' - here a directory named "weird:".
			name:         "percent-encoded colon (%3A) at a boundary reaches Stat as a literal colon",
			urlPath:      "/graph/v1.0/drives/" + testDriveID + "/root:/weird%3A/file.txt",
			expectedStat: "./weird:/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var statPath string
			gw := &cs3mocks.GatewayAPIClient{}
			gw.On("Stat", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					req := args.Get(1).(*storageprovider.StatRequest)
					statPath = req.GetRef().GetPath()
				}).
				Return(statResponse(cs3rpc.Code_CODE_OK, true), nil)

			router, _ := newGraphTestRouter(t, gw)
			req := httptest.NewRequest(http.MethodGet, "http://localhost"+tt.urlPath, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, tt.expectedStat, statPath, "path passed to CS3 Stat must be decoded exactly once")
		})
	}
}

// TestResolveGraphPath_ItemIDRoundTrip guards the sub-delimiter round-trip: the
// resolved id contains `$` and `!`, and after the RoutePath rewrite chi must
// bind it to the {itemID} param such that the downstream PathUnescape recovers
// the original id. This exercises chi's param binding indirectly; a regression
// in how chi stores matched segments would surface here.
func TestResolveGraphPath_ItemIDRoundTrip(t *testing.T) {
	gw := &cs3mocks.GatewayAPIClient{}
	gw.On("Stat", mock.Anything, mock.Anything).
		Return(statResponse(cs3rpc.Code_CODE_OK, true), nil)

	router, cap := newGraphTestRouter(t, gw)
	req := httptest.NewRequest(
		http.MethodGet,
		"http://localhost/graph/v1.0/drives/"+testDriveID+"/root:/Documents:/children",
		nil,
	)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "chi must route the rewritten path to the leaf")
	assert.Equal(t, "children", cap.hit)
	assert.Equal(t, testItemID, cap.itemID,
		"PathUnescape(chi.URLParam(itemID)) must recover the original id (with `$` and `!`)")
}

// TestResolveGraphPath_OriginalPathContext verifies the original URL is
// preserved in request context for downstream tracing/logging, and that
// r.URL.Path itself is left untouched by the rewrite.
func TestResolveGraphPath_OriginalPathContext(t *testing.T) {
	gw := &cs3mocks.GatewayAPIClient{}
	gw.On("Stat", mock.Anything, mock.Anything).
		Return(statResponse(cs3rpc.Code_CODE_OK, true), nil)

	original := "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:/children"
	router, cap := newGraphTestRouter(t, gw)
	req := httptest.NewRequest(http.MethodGet, "http://localhost"+original, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, original, cap.original, "original URL must be available via OriginalPathContextKey")
	assert.Equal(t, original, cap.urlPath, "r.URL.Path must remain the original request path")
}
