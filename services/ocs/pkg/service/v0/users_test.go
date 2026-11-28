package svc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cs3user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/ocs/pkg/service/v0/data"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

type signingKeyResponse struct {
	OCS struct {
		Meta struct {
			StatusCode int `json:"statuscode"`
		} `json:"meta"`
		Data struct {
			User       string `json:"user"`
			SigningKey string `json:"signing-key"`
		} `json:"data"`
	} `json:"ocs"`
}

func getSigningKey(t *testing.T, s store.Store, u *cs3user.User) signingKeyResponse {
	t.Helper()
	o := Ocs{logger: log.NewLogger(), store: s}

	req := httptest.NewRequest(http.MethodGet, "/ocs/v2.php/cloud/user/signing-key?format=json", nil)
	req = req.WithContext(revactx.ContextSetUser(req.Context(), u))
	w := httptest.NewRecorder()
	o.GetSigningKey(w, req)

	var body signingKeyResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return body
}

func TestGetSigningKey(t *testing.T) {
	s := store.NewMemoryStore()
	u := &cs3user.User{Id: &cs3user.UserId{OpaqueId: "alice-id", Type: cs3user.UserType_USER_TYPE_PRIMARY}}

	body := getSigningKey(t, s, u)
	assert.Equal(t, "alice-id", body.OCS.Data.User)
	require.NotEmpty(t, body.OCS.Data.SigningKey)

	// the key is persisted and returned again on the next request
	rec, err := s.Read("alice-id")
	require.NoError(t, err)
	require.Len(t, rec, 1)
	assert.Equal(t, body.OCS.Data.SigningKey, string(rec[0].Value))
	assert.Equal(t, body.OCS.Data.SigningKey, getSigningKey(t, s, u).OCS.Data.SigningKey)
}

// A public link token carries the link owner as user, marked with the
// public-share-role opaque entry by the publicshares auth manager. The owner's
// signing key must never be handed out in that context, not even if it exists.
func TestGetSigningKeyRefusesPublicLinkAndGuests(t *testing.T) {
	linkVisitor := &cs3user.User{
		Id: &cs3user.UserId{OpaqueId: "alice-id", Type: cs3user.UserType_USER_TYPE_PRIMARY},
		Opaque: &cs3types.Opaque{Map: map[string]*cs3types.OpaqueEntry{
			"public-share-role": {Decoder: "plain", Value: []byte("viewer")},
		}},
	}
	guest := &cs3user.User{Id: &cs3user.UserId{OpaqueId: "guest-id", Type: cs3user.UserType_USER_TYPE_GUEST}}

	for name, u := range map[string]*cs3user.User{"public link": linkVisitor, "guest": guest} {
		t.Run(name, func(t *testing.T) {
			s := store.NewMemoryStore()
			require.NoError(t, s.Write(&store.Record{Key: u.GetId().GetOpaqueId(), Value: []byte("existing-key")}))

			body := getSigningKey(t, s, u)
			assert.Equal(t, data.MetaUnauthorized.StatusCode, body.OCS.Meta.StatusCode)
			assert.Empty(t, body.OCS.Data.SigningKey)
			assert.Empty(t, body.OCS.Data.User)
		})
	}

	t.Run("does not create a key", func(t *testing.T) {
		s := store.NewMemoryStore()
		getSigningKey(t, s, linkVisitor)
		_, err := s.Read("alice-id")
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
}
