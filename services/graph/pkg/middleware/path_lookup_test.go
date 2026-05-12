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

// recordingHandler captures the request URL path it receives so tests can assert
// the middleware rewrote (or passed through) correctly.
func recordingHandler() (http.Handler, *string) {
	var seen string
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}), &seen
}

func TestResolveGraphPath(t *testing.T) {
	tests := []struct {
		name             string
		urlPath          string
		statCode         cs3rpc.Code
		statErr          error
		expectStatCalled bool
		expectStatus     int
		expectURLPath    string // empty = handler not invoked
	}{
		{
			name:             "non-colon URL passes through unchanged",
			urlPath:          "/graph/v1.0/me/drives",
			expectStatCalled: false,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/me/drives",
		},
		{
			name:             "URL with colon but not matching pattern passes through",
			urlPath:          "/graph/v1.0/some/other/route:/foo",
			expectStatCalled: false,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/some/other/route:/foo",
		},
		{
			name:             "v1.0 root-anchored with /children rewrites",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d/children",
		},
		{
			name:             "v1beta1 root-anchored with /createLink rewrites",
			urlPath:          "/graph/v1beta1/drives/" + testDriveID + "/root:/Documents:/createLink",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1beta1/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d/createLink",
		},
		{
			name:             "trailing colon (no suffix) rewrites to bare item URL",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d",
		},
		{
			name:             "no trailing colon, no suffix rewrites to bare item URL",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d",
		},
		{
			name:             "deep path rewrites correctly",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents/Reports:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d/children",
		},
		{
			name:             "item-anchored colon syntax rewrites",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/items/" + testItemID + ":/notes.txt:/children",
			statCode:         cs3rpc.Code_CODE_OK,
			expectStatCalled: true,
			expectStatus:     http.StatusOK,
			expectURLPath:    "/graph/v1.0/drives/" + testDriveID + "/items/storage-users-1$f503f6fe-2656-4b0f-8289-fb3184962dfd!f0e20017-9cba-498a-87e5-3467b976604d/children",
		},
		{
			name:             "Stat NOT_FOUND returns 404 without invoking handler",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Missing:",
			statCode:         cs3rpc.Code_CODE_NOT_FOUND,
			expectStatCalled: true,
			expectStatus:     http.StatusNotFound,
			expectURLPath:    "", // handler must NOT be called
		},
		{
			// CRITICAL security test: PERMISSION_DENIED must not leak existence.
			// We collapse it to 404, identical to NOT_FOUND, so an unauthorized
			// caller can't distinguish "doesn't exist" from "exists but you can't see it".
			name:             "Stat PERMISSION_DENIED returns 404 (not 403) — don't disclose existence",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Restricted:",
			statCode:         cs3rpc.Code_CODE_PERMISSION_DENIED,
			expectStatCalled: true,
			expectStatus:     http.StatusNotFound,
			expectURLPath:    "",
		},
		{
			// Operational/unexpected CS3 statuses must NOT be collapsed to 404 —
			// that would mask outages. Surface as 500 like other graph handlers do.
			name:             "Stat unexpected status returns 500 (not 404 — don't mask outages)",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Anything:",
			statCode:         cs3rpc.Code_CODE_INTERNAL,
			expectStatCalled: true,
			expectStatus:     http.StatusInternalServerError,
			expectURLPath:    "",
		},
		{
			// UNAUTHENTICATED is its own distinct outcome — must surface as 401,
			// not 500, so clients can detect "your token is bad" vs "server error".
			name:             "Stat UNAUTHENTICATED returns 401",
			urlPath:          "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:",
			statCode:         cs3rpc.Code_CODE_UNAUTHENTICATED,
			expectStatCalled: true,
			expectStatus:     http.StatusUnauthorized,
			expectURLPath:    "",
		},
		{
			// Item-anchored form with a driveID that doesn't match the itemID's
			// storage/space — the request is malformed; short-circuit to 400
			// instead of doing a Stat that would only fail downstream.
			name:             "drive id and item id storage/space mismatch returns 400",
			urlPath:          "/graph/v1.0/drives/storage-users-2$other-space-id/items/" + testItemID + ":/notes.txt:/children",
			expectStatCalled: false,
			expectStatus:     http.StatusBadRequest,
			expectURLPath:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := &cs3mocks.GatewayAPIClient{}
			if tt.expectStatCalled {
				gw.On("Stat", mock.Anything, mock.Anything).
					Return(statResponse(tt.statCode, tt.statCode == cs3rpc.Code_CODE_OK), tt.statErr)
			}

			selector := newTestSelector(t, gw)
			handler, seen := recordingHandler()
			mw := middleware.ResolveGraphPath(selector, log.NopLogger())

			req := httptest.NewRequest(http.MethodGet, "http://localhost"+tt.urlPath, nil)
			rr := httptest.NewRecorder()

			mw(handler).ServeHTTP(rr, req)

			assert.Equal(t, tt.expectStatus, rr.Code, "status code")
			assert.Equal(t, tt.expectURLPath, *seen, "URL seen by next handler")

			if tt.expectStatCalled {
				gw.AssertCalled(t, "Stat", mock.Anything, mock.Anything)
			} else {
				gw.AssertNotCalled(t, "Stat", mock.Anything, mock.Anything)
			}
		})
	}
}

// TestResolveGraphPath_ChiParamRoundTripWithSubDelims pins down what chi
// actually returns from URLParam for rewritten URLs whose IDs contain `$`
// and `!`, and verifies the PathUnescape round-trip downstream handlers
// rely on still recovers the original ID.
//
// Specifically: r.URL.RawPath = r.URL.EscapedPath() leaves `$` literal but
// encodes `!` as `%21` (net/url's encodePath behavior). chi.URLParam returns
// the matched RawPath segment as-is, so the bound param contains `%21`.
// Existing handlers (parseIDParam, GetDriveAndItemIDParam) call
// url.PathUnescape on the param before parsing the ID, which recovers `!`.
//
// This test guards against any future change to the encoding strategy that
// would silently break that round-trip, for example, switching to
// url.PathEscape(itemID) and stuffing the result into r.URL.Path (which
// expects the decoded form) and thereby double-encoding `!` to `%2521`.
func TestResolveGraphPath_ChiParamRoundTripWithSubDelims(t *testing.T) {
	gw := &cs3mocks.GatewayAPIClient{}
	gw.On("Stat", mock.Anything, mock.Anything).
		Return(statResponse(cs3rpc.Code_CODE_OK, true), nil)

	selector := newTestSelector(t, gw)

	var gotDriveID, gotItemID string
	r := chi.NewRouter()
	r.Use(middleware.ResolveGraphPath(selector, log.NopLogger()))
	r.Route("/graph/v1.0/drives/{driveID}", func(r chi.Router) {
		r.Get("/items/{itemID}/children", func(w http.ResponseWriter, r *http.Request) {
			gotDriveID = chi.URLParam(r, "driveID")
			gotItemID = chi.URLParam(r, "itemID")
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"http://localhost/graph/v1.0/drives/"+testDriveID+"/root:/Documents:/children",
		nil,
	)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "chi must route the rewritten URL to the handler")

	// driveID has only `$` (a sub-delim left literal by EscapedPath), so chi
	// returns it without any percent-encoding.
	assert.Equal(t, testDriveID, gotDriveID,
		"chi.URLParam should return driveID unchanged: only `$` sub-delim, left literal by EscapedPath")

	// itemID has both `$` and `!`. EscapedPath encodes `!` as `%21`, so chi
	// returns the param with `%21` literal; PathUnescape recovers the
	// original ID. This is the round-trip downstream handlers rely on.
	unescaped, err := url.PathUnescape(gotItemID)
	assert.NoError(t, err, "URLParam value must be a valid percent-encoded string")
	assert.Equal(t, testItemID, unescaped,
		"PathUnescape(chi.URLParam(itemID)) must recover the original ID")
}

// TestResolveGraphPath_OriginalPathContext verifies the rewrite preserves the
// original URL in request context for downstream tracing/logging.
func TestResolveGraphPath_OriginalPathContext(t *testing.T) {
	gw := &cs3mocks.GatewayAPIClient{}
	gw.On("Stat", mock.Anything, mock.Anything).
		Return(statResponse(cs3rpc.Code_CODE_OK, true), nil)

	selector := newTestSelector(t, gw)
	original := "/graph/v1.0/drives/" + testDriveID + "/root:/Documents:/children"

	var capturedOriginal interface{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOriginal = r.Context().Value(middleware.OriginalPathContextKey)
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.ResolveGraphPath(selector, log.NopLogger())
	req := httptest.NewRequest(http.MethodGet, "http://localhost"+original, nil)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, original, capturedOriginal, "original URL must be available via OriginalPathContextKey")
}
