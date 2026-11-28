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

package response

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
)

type key int

const (
	apiVersionKey key = 1
)

var (
	defaultStatusCodeMapper = OcsV2StatusCodes
)

// Response is the top level response structure
type Response struct {
	OCS *Payload `json:"ocs"`
}

// Payload combines response metadata and data
type Payload struct {
	XMLName struct{}    `json:"-" xml:"ocs"`
	Meta    Meta        `json:"meta" xml:"meta"`
	Data    interface{} `json:"data,omitempty" xml:"data,omitempty"`
}

var (
	elementStartElement = xml.StartElement{Name: xml.Name{Local: "element"}}
	metaStartElement    = xml.StartElement{Name: xml.Name{Local: "meta"}}
	ocsName             = xml.Name{Local: "ocs"}
	dataName            = xml.Name{Local: "data"}
)

// MarshalXML handles ocs specific wrapping of array members in 'element' tags for the data
func (p Payload) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	// first the easy part
	// use ocs as the surrounding tag
	start.Name = ocsName
	if err = e.EncodeToken(start); err != nil {
		return
	}

	// encode the meta tag
	if err = e.EncodeElement(p.Meta, metaStartElement); err != nil {
		return
	}

	// we need to use reflection to determine if p.Data is an array or a slice
	rt := reflect.TypeOf(p.Data)
	if rt != nil && (rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice) {
		// this is how to wrap the data elements in their own <element> tag
		v := reflect.ValueOf(p.Data)
		if err = e.EncodeToken(xml.StartElement{Name: dataName}); err != nil {
			return
		}
		for i := 0; i < v.Len(); i++ {
			if err = e.EncodeElement(v.Index(i).Interface(), elementStartElement); err != nil {
				return
			}
		}
		if err = e.EncodeToken(xml.EndElement{Name: dataName}); err != nil {
			return
		}
	} else if err = e.EncodeElement(p.Data, xml.StartElement{Name: dataName}); err != nil {
		return
	}

	// write the closing <ocs> tag
	if err = e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return
	}
	return
}

// Meta holds response metadata
type Meta struct {
	Status       string `json:"status" xml:"status"`
	StatusCode   int    `json:"statuscode" xml:"statuscode"`
	Message      string `json:"message" xml:"message"`
	TotalItems   string `json:"totalitems,omitempty" xml:"totalitems,omitempty"`
	ItemsPerPage string `json:"itemsperpage,omitempty" xml:"itemsperpage,omitempty"`
}

// MetaOK is the default ok response
var MetaOK = Meta{Status: "ok", StatusCode: 100, Message: "OK"}

// MetaFailure is a failure response with code 101
var MetaFailure = Meta{Status: "", StatusCode: 101, Message: "Failure"}

// MetaInvalidInput is an error response with code 102
var MetaInvalidInput = Meta{Status: "", StatusCode: 102, Message: "Invalid Input"}

// MetaForbidden is an error response with code 104
var MetaForbidden = Meta{Status: "", StatusCode: 104, Message: "Forbidden"}

// MetaBadRequest is used for unknown errors
var MetaBadRequest = Meta{Status: "error", StatusCode: 400, Message: "Bad Request"}

// MetaPathNotFound is returned when trying to share not existing resources
var MetaPathNotFound = Meta{Status: "failure", StatusCode: 404, Message: MessagePathNotFound}

// MetaLocked is returned when trying to share not existing resources
var MetaLocked = Meta{Status: "failure", StatusCode: 423, Message: "The file is locked"}

// MetaServerError is returned on server errors
var MetaServerError = Meta{Status: "error", StatusCode: 996, Message: "Server Error"}

// MetaUnauthorized is returned on unauthorized requests
var MetaUnauthorized = Meta{Status: "error", StatusCode: 997, Message: "Unauthorised"}

// MetaNotFound is returned when trying to access not existing resources
var MetaNotFound = Meta{Status: "error", StatusCode: 998, Message: "Not Found"}

// MetaUnknownError is used for unknown errors
var MetaUnknownError = Meta{Status: "error", StatusCode: 999, Message: "Unknown Error"}

// MessageUserNotFound is  used when a user can not be found
var MessageUserNotFound = "The requested user could not be found"

// MessageGroupNotFound is used when a group can not be found
var MessageGroupNotFound = "The requested group could not be found"

// MessagePathNotFound is used when a file or folder can not be found
var MessagePathNotFound = "Wrong path, file/folder doesn't exist"

// MessageShareExists is used when a user tries to create a new share for the same user
var MessageShareExists = "A share for the recipient already exists"

// MessageLockedForSharing is used when a user tries to create a new share until the file is in use by at least one user
var MessageLockedForSharing = "The file is locked until the file is in use by at least one user"

// WriteOCSSuccess handles writing successful ocs response data
func WriteOCSSuccess(w http.ResponseWriter, r *http.Request, d interface{}) {
	WriteOCSData(w, r, MetaOK, d, nil)
}

// WriteOCSError handles writing error ocs responses
func WriteOCSError(w http.ResponseWriter, r *http.Request, c int, m string, err error) {
	WriteOCSData(w, r, Meta{Status: "error", StatusCode: c, Message: m}, nil, err)
}

// WriteOCSData handles writing ocs data in json and xml
func WriteOCSData(w http.ResponseWriter, r *http.Request, m Meta, d interface{}, err error) {
	WriteOCSResponse(w, r, Response{
		OCS: &Payload{
			Meta: m,
			Data: d,
		},
	}, err)
}

// WriteOCSResponse handles writing ocs responses in json and xml
func WriteOCSResponse(w http.ResponseWriter, r *http.Request, res Response, err error) {
	if err != nil {
		appctx.GetLogger(r.Context()).
			Debug().
			Err(err).
			Str("ocs_msg", res.OCS.Meta.Message).
			Msg("writing ocs error response")
	}

	version := APIVersion(r.Context())
	m := statusCodeMapper(version)
	statusCode := m(res.OCS.Meta)
	if version == "2" && statusCode == http.StatusOK {
		res.OCS.Meta.StatusCode = statusCode
	}

	var encoder func(Response) ([]byte, error)
	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder = encodeJSON
	} else {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		encoder = encodeXML
	}
	w.WriteHeader(statusCode)
	encoded, err := encoder(res)
	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error encoding ocs response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(encoded)
	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error writing ocs response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func encodeXML(res Response) ([]byte, error) {
	marshalled, err := xml.Marshal(res.OCS)
	if err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	b.WriteString(xml.Header)
	b.Write(marshalled)
	return b.Bytes(), nil
}

func encodeJSON(res Response) ([]byte, error) {
	return json.Marshal(res)
}

// OcsV1StatusCodes returns the http status codes for the OCS API v1.
func OcsV1StatusCodes(meta Meta) int {
	return http.StatusOK
}

// OcsV2StatusCodes maps the OCS codes to http status codes for the ocs API v2.
func OcsV2StatusCodes(meta Meta) int {
	sc := meta.StatusCode
	switch sc {
	case MetaNotFound.StatusCode:
		return http.StatusNotFound
	case MetaUnknownError.StatusCode:
		fallthrough
	case MetaServerError.StatusCode:
		return http.StatusInternalServerError
	case MetaUnauthorized.StatusCode:
		return http.StatusUnauthorized
	case MetaOK.StatusCode:
		meta.StatusCode = http.StatusOK
		return http.StatusOK
	case MetaForbidden.StatusCode:
		return http.StatusForbidden
	}
	// any 2xx, 4xx and 5xx will be used as is
	if sc >= 200 && sc < 600 {
		return sc
	}

	// any error codes > 100 are treated as client errors
	if sc > 100 && sc < 200 {
		return http.StatusBadRequest
	}

	// TODO change this status code?
	return http.StatusOK
}

// WithAPIVersion puts the api version in the context.
func VersionCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := chi.URLParam(r, "version")
		if version == "" {
			WriteOCSError(w, r, MetaBadRequest.StatusCode, "unknown ocs api version", nil)
			return
		}
		w.Header().Set("Ocs-Api-Version", version)

		// store version in context so handlers can access it
		ctx := context.WithValue(r.Context(), apiVersionKey, version)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// APIVersion retrieves the api version from the context.
func APIVersion(ctx context.Context) string {
	value := ctx.Value(apiVersionKey)
	if value != nil {
		return value.(string)
	}
	return ""
}

func statusCodeMapper(version string) func(Meta) int {
	var mapper func(Meta) int
	switch version {
	case "1":
		mapper = OcsV1StatusCodes
	case "2":
		mapper = OcsV2StatusCodes
	default:
		mapper = defaultStatusCodeMapper
	}
	return mapper
}
