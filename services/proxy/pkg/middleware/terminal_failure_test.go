package middleware

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hostileShareID would break out of an XML/JSON document if it were not escaped.
const hostileShareID = `<script>alert("x")</script>&'`

func TestRenderTerminalFailure_Generic(t *testing.T) {
	t.Run("non-dav request", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/me/drive/sharedWithMe", nil)

		require.NoError(t, renderTerminalFailure(rr, req, TerminalFailed()))

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Empty(t, rr.Body.String())
	})

	t.Run("dav request", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("PROPFIND", "/dav/spaces/abc", nil)

		require.NoError(t, renderTerminalFailure(rr, req, TerminalFailed()))

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "application/xml; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(rr.Body.String(), xml.Header))
		assert.Contains(t, rr.Body.String(), "Sabre\\DAV\\Exception\\NotAuthenticated")
	})
}

func TestRenderTerminalFailure_GuestSessionExpiredJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/me/drive/sharedWithMe", nil)

	require.NoError(t, renderTerminalFailure(rr, req, AuthenticationResult{
		State:        AuthenticationFailed,
		Terminal:     true,
		ErrorDetails: GuestSessionExpiredDetails{PermissionID: hostileShareID},
	}))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body struct {
		ErrorType    string `json:"error_type"`
		PermissionID string `json:"permissionId"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "session_expired", body.ErrorType)
	assert.Equal(t, hostileShareID, body.PermissionID)
	// json.Encoder escapes <, > and & so the body is safe even if it were rendered as HTML
	assert.NotContains(t, rr.Body.String(), "<script>")
}

func TestRenderTerminalFailure_GuestSessionExpiredDAV(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PROPFIND", "/dav/spaces/abc", nil)

	require.NoError(t, renderTerminalFailure(rr, req, AuthenticationResult{
		State:        AuthenticationFailed,
		Terminal:     true,
		ErrorDetails: GuestSessionExpiredDetails{PermissionID: hostileShareID, IsDAV: true},
	}))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/xml; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(rr.Body.String(), xml.Header))

	// encoding/xml cannot unmarshal prefixed element names such as d:error, so
	// inspect the document directly.
	body := rr.Body.String()
	assert.Contains(t, body, "<s:Exception>Sabre\\DAV\\Exception\\NotAuthenticated</s:Exception>")
	assert.Contains(t, body, "<opencloud:error_type>session_expired</opencloud:error_type>")
	assert.Contains(t, body, "<opencloud:share_id>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;&amp;&#39;</opencloud:share_id>")
	assert.NotContains(t, body, "<script>")
}
