// Copyright 2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.
package ocdav

import (
	"errors"
	"testing"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/test-go/testify/require"
)

func TestWrapResourceID(t *testing.T) {
	expected := "storageid" + "$" + "spaceid" + "!" + "opaqueid"
	wrapped := storagespace.FormatResourceID(&sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "opaqueid"})

	if wrapped != expected {
		t.Errorf("wrapped id doesn't have the expected format: got %s expected %s", wrapped, expected)
	}
}

func TestNameNotEmpty(t *testing.T) {
	expErr := errors.New("must not be empty")
	tests := map[string]error{
		"":      expErr,
		" ":     expErr,
		"\n":    expErr,
		"name":  nil,
		"empty": nil,
	}

	for name, expected := range tests {
		rule := notEmpty()
		require.Equal(t, expected, rule(name), name)
	}
}

func TestNameDoesNotContain(t *testing.T) {
	tests := []struct {
		excludedChars []string
		tests         map[string]error
	}{
		{
			[]string{"a"},
			map[string]error{
				"foo": nil,
				"bar": errors.New("must not contain a"),
			},
		},
		{
			[]string{"a", "b"},
			map[string]error{
				"foo": nil,
				"bar": errors.New("must not contain a"),
				"car": errors.New("must not contain a"),
				"bor": errors.New("must not contain b"),
			},
		},
	}

	for _, tt := range tests {
		rule := doesNotContain(tt.excludedChars)
		for name, expected := range tt.tests {
			require.Equal(t, expected, rule(name), name)
		}
	}
}

func TestNameMaxLength(t *testing.T) {
	name := "123456789"
	tests := []struct {
		MaxLength int
		Error     error
	}{
		{12, nil},
		{8, errors.New("must be shorter than 8")},
		{4, errors.New("must be shorter than 4")},
	}
	for _, tt := range tests {
		rule := isShorterThan(tt.MaxLength)
		require.Equal(t, tt.Error, rule(name), tt.MaxLength)
	}
}
