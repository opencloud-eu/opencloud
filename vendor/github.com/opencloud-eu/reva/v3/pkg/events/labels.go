// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// LabelAdded is emitted when a user adds a label to a resource
type LabelAdded struct {
	Ref       *provider.Reference
	Label     string
	Executant *user.UserId
	UserID    *user.UserId
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (LabelAdded) Unmarshal(v []byte) (interface{}, error) {
	e := LabelAdded{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LabelRemoved is emitted when a user removes a label from a resource
type LabelRemoved struct {
	Ref       *provider.Reference
	Label     string
	Executant *user.UserId
	UserID    *user.UserId
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (LabelRemoved) Unmarshal(v []byte) (interface{}, error) {
	e := LabelRemoved{}
	err := json.Unmarshal(v, &e)
	return e, err
}
