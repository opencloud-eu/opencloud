package svc

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	cs3user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/services/ocs/pkg/service/v0/data"
	"github.com/opencloud-eu/opencloud/services/ocs/pkg/service/v0/response"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"go-micro.dev/v4/store"
)

// GetSigningKey returns the signing key for the current user. It will create it on the fly if it does not exist
// The signing key is part of the user settings and is used by the proxy to authenticate requests
// Currently, the username is used as the OC-Credential
func (o Ocs) GetSigningKey(w http.ResponseWriter, r *http.Request) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		//o.logger.Error().Msg("missing user in context")
		o.mustRender(w, r, response.ErrRender(data.MetaBadRequest.StatusCode, "missing user in context"))
		return
	}

	// A public link token carries the link owner as user. Never hand out the owner's
	// signing key to guests or link visitors, it would let them sign urls for
	// everything the owner can access.
	if u.GetId().GetType() == cs3user.UserType_USER_TYPE_GUEST || utils.ExistsInOpaque(u.GetOpaque(), "public-share-role") {
		o.mustRender(w, r, response.ErrRender(data.MetaUnauthorized.StatusCode, "unauthorized"))
		return
	}

	// use the user's UUID
	userID := u.Id.OpaqueId

	res, err := o.store.Read(userID)
	if err == nil && len(res) > 0 {
		o.mustRender(w, r, response.DataRender(&data.SigningKey{
			User:       userID,
			SigningKey: string(res[0].Value),
		}))
		return
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// not found is ok, so we can continue and generate the key on the fly
		} else {
			o.logger.Error().Err(err).Msg("error reading from server")
			o.mustRender(w, r, response.ErrRender(data.MetaServerError.StatusCode, "error reading from store"))
			return
		}
	}

	// try creating it
	key := make([]byte, 64)
	_, err = rand.Read(key[:])
	if err != nil {
		o.mustRender(w, r, response.ErrRender(data.MetaServerError.StatusCode, "could not generate signing key"))
		return
	}
	signingKey := hex.EncodeToString(key)

	err = o.store.Write(&store.Record{Key: userID, Value: []byte(signingKey)})
	if err != nil {
		o.mustRender(w, r, response.ErrRender(data.MetaServerError.StatusCode, "could not persist signing key"))
		return
	}

	o.mustRender(w, r, response.DataRender(&data.SigningKey{
		User:       userID,
		SigningKey: signingKey,
	}))
}
