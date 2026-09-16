// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// ArbitraryMetadataUpdated is emitted when arbitrary metadata keys of a
// resource are set or unset. Keys lists the affected metadata keys.
type ArbitraryMetadataUpdated struct {
	SpaceOwner *user.UserId
	Ref        *provider.Reference
	Keys       []string
	Executant  *user.UserId
	Timestamp  *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (ArbitraryMetadataUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := ArbitraryMetadataUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}
