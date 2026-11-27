// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// ArbitraryMetadataChanged is emitted when arbitrary metadata keys are set or unset on a resource
type ArbitraryMetadataChanged struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	ChangedKeys       []string
	RemovedKeys       []string
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ArbitraryMetadataChanged) Unmarshal(v []byte) (interface{}, error) {
	e := ArbitraryMetadataChanged{}
	err := json.Unmarshal(v, &e)
	return e, err
}
