// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

package appctx

import (
	"context"
	"reflect"
	"unsafe"
)

// PutKeyValuesToCtx puts all the key-value pairs from the provided map to a background context.
func PutKeyValuesToCtx(m map[interface{}]interface{}) context.Context {
	ctx := context.Background()
	for key, value := range m {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

// GetKeyValuesFromCtx retrieves all the key-value pairs from the provided context.
func GetKeyValuesFromCtx(ctx context.Context) map[interface{}]interface{} {
	m := make(map[interface{}]interface{})
	getKeyValue(ctx, m)
	return m
}

func getKeyValue(ctx interface{}, m map[interface{}]interface{}) {
	// This is a dirty hack to run with go 1.21.x
	reflectCtxValues := reflect.ValueOf(ctx)
	if reflectCtxValues.Kind() != reflect.Struct {
		ctxVals := reflectCtxValues.Elem()
		ctxType := reflect.TypeOf(ctx).Elem()

		if ctxType.Kind() == reflect.Struct {
			for i := 0; i < ctxVals.NumField(); i++ {
				currField, currIf := extractField(ctxVals, ctxType, i)
				switch currField {
				case "Context":
					getKeyValue(currIf, m)
				case "key":
					nextField, nextIf := extractField(ctxVals, ctxType, i+1)
					if nextField == "val" {
						m[currIf] = nextIf
						i++
					}
				}
			}
		}
	}
}

func extractField(vals reflect.Value, fieldType reflect.Type, pos int) (string, interface{}) {
	currVal := vals.Field(pos)
	currVal = reflect.NewAt(currVal.Type(), unsafe.Pointer(currVal.UnsafeAddr())).Elem()
	currField := fieldType.Field(pos)
	return currField.Name, currVal.Interface()
}
