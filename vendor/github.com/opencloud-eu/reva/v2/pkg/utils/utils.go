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

package utils

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchEmail    = regexp.MustCompile(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)

	// ShareStorageProviderID is the provider id used by the sharestorageprovider
	ShareStorageProviderID = "a0ca6a90-a365-4782-871e-d44447bbc668"
	// ShareStorageSpaceID is the space id used by the sharestorageprovider share jail space
	ShareStorageSpaceID = "a0ca6a90-a365-4782-871e-d44447bbc668"

	// PublicStorageProviderID is the storage id used by the sharestorageprovider
	PublicStorageProviderID = "7993447f-687f-490d-875c-ac95e89a62a4"
	// PublicStorageSpaceID is the space id used by the sharestorageprovider
	PublicStorageSpaceID = "7993447f-687f-490d-875c-ac95e89a62a4"

	// OCMStorageProviderID is the storage id used by the ocmreceived storageprovider
	OCMStorageProviderID = "89f37a33-858b-45fa-8890-a1f2b27d90e1"
	// OCMStorageSpaceID is the space id used by the ocmreceived storageprovider
	OCMStorageSpaceID = "89f37a33-858b-45fa-8890-a1f2b27d90e1"

	// SpaceGrant is used to signal the storageprovider that the grant is on a space
	SpaceGrant struct{}
)

// Skip  evaluates whether a source endpoint contains any of the prefixes.
// i.e: /a/b/c/d/e contains prefix /a/b/c
func Skip(source string, prefixes []string) bool {
	for i := range prefixes {
		if strings.HasPrefix(source, prefixes[i]) {
			return true
		}
	}
	return false
}

// GetClientIP retrieves the client IP from incoming requests
func GetClientIP(r *http.Request) (string, error) {
	var clientIP string
	forwarded := r.Header.Get("X-FORWARDED-FOR")

	if forwarded != "" {
		clientIP = forwarded
	} else {
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
			ipObj := net.ParseIP(r.RemoteAddr)
			if ipObj == nil {
				return "", err
			}
			clientIP = ipObj.String()
		} else {
			clientIP = ip
		}
	}
	return clientIP, nil
}

// ToSnakeCase converts a CamelCase string to a snake_case string.
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ResolvePath converts relative local paths to absolute paths
func ResolvePath(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	homeDir := usr.HomeDir

	if path == "~" {
		path = homeDir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}

	return filepath.Abs(path)
}

// RandString is a helper to create tokens.
func RandString(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	var l = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = l[rand.Intn(len(l))]
	}
	return string(b)
}

// TSToUnixNano converts a protobuf Timestamp to uint64
// with nanoseconds resolution.
func TSToUnixNano(ts *types.Timestamp) uint64 {
	if ts == nil {
		return 0
	}
	return uint64(time.Unix(int64(ts.Seconds), int64(ts.Nanos)).UnixNano())
}

// TSToTime converts a protobuf Timestamp to Go's time.Time.
func TSToTime(ts *types.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return time.Unix(int64(ts.Seconds), int64(ts.Nanos))
}

// TimeToTS converts Go's time.Time to a protobuf Timestamp.
func TimeToTS(t time.Time) *types.Timestamp {
	return &types.Timestamp{
		Seconds: uint64(t.Unix()), // implicitly returns UTC
		Nanos:   uint32(t.Nanosecond()),
	}
}

// LaterTS returns the timestamp which occurs later.
func LaterTS(t1 *types.Timestamp, t2 *types.Timestamp) *types.Timestamp {
	if TSToUnixNano(t1) > TSToUnixNano(t2) {
		return t1
	}
	return t2
}

// TSNow returns the current UTC timestamp
func TSNow() *types.Timestamp {
	t := time.Now().UTC()
	return &types.Timestamp{
		Seconds: uint64(t.Unix()),
		Nanos:   uint32(t.Nanosecond()),
	}
}

// MTimeToTS converts a string in the form "<unix>.<nanoseconds>" into a CS3 Timestamp
func MTimeToTS(v string) (ts types.Timestamp, err error) {
	p := strings.SplitN(v, ".", 2)
	var sec, nsec uint64
	if sec, err = strconv.ParseUint(p[0], 10, 64); err == nil {
		if len(p) > 1 {
			nsec, err = strconv.ParseUint(p[1], 10, 32)
		}
	}
	return types.Timestamp{Seconds: sec, Nanos: uint32(nsec)}, err
}

// MTimeToTime converts a string in the form "<unix>.<nanoseconds>" into a go time.Time
func MTimeToTime(v string) (t time.Time, err error) {
	p := strings.SplitN(v, ".", 2)
	var sec, nsec int64
	if sec, err = strconv.ParseInt(p[0], 10, 64); err == nil {
		if len(p) > 1 {
			nsec, err = strconv.ParseInt(p[1], 10, 64)
		}
	}
	return time.Unix(sec, nsec), err
}

// TimeToOCMtime converts a Go time.Time to a string in the form "<unix>.<nanoseconds>"
func TimeToOCMtime(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10) + "." + strconv.FormatInt(int64(t.Nanosecond()), 10)
}

// ExtractGranteeID returns the ID, user or group, set in the GranteeId object
func ExtractGranteeID(grantee *provider.Grantee) (*userpb.UserId, *grouppb.GroupId) {
	switch t := grantee.Id.(type) {
	case *provider.Grantee_UserId:
		return t.UserId, nil
	case *provider.Grantee_GroupId:
		return nil, t.GroupId
	default:
		return nil, nil
	}
}

// UserEqual returns whether two users have the same field values.
func UserEqual(u, v *userpb.UserId) bool {
	return u != nil && v != nil && u.Idp == v.Idp && u.OpaqueId == v.OpaqueId
}

// UserIDEqual returns whether two users have the same opaqueid values. The idp is ignored
func UserIDEqual(u, v *userpb.UserId) bool {
	return u != nil && v != nil && u.OpaqueId == v.OpaqueId
}

// GroupEqual returns whether two groups have the same field values.
func GroupEqual(u, v *grouppb.GroupId) bool {
	return u != nil && v != nil && u.Idp == v.Idp && u.OpaqueId == v.OpaqueId
}

// ResourceIDEqual returns whether two resources have the same field values.
func ResourceIDEqual(u, v *provider.ResourceId) bool {
	return u != nil && v != nil && u.StorageId == v.StorageId && u.OpaqueId == v.OpaqueId && u.SpaceId == v.SpaceId
}

// ResourceEqual returns whether two resources have the same field values.
func ResourceEqual(u, v *provider.Reference) bool {
	return u != nil && v != nil && u.Path == v.Path && ((u.ResourceId == nil && v.ResourceId == nil) || (ResourceIDEqual(u.ResourceId, v.ResourceId)))
}

// GranteeEqual returns whether two grantees have the same field values.
func GranteeEqual(u, v *provider.Grantee) bool {
	if u == nil || v == nil {
		return false
	}
	uu, ug := ExtractGranteeID(u)
	vu, vg := ExtractGranteeID(v)
	return u.Type == v.Type && (UserEqual(uu, vu) || GroupEqual(ug, vg))
}

// IsEmailValid checks whether the provided email has a valid format.
func IsEmailValid(e string) bool {
	if len(e) < 3 || len(e) > 254 {
		return false
	}
	return matchEmail.MatchString(e)
}

// IsValidWebAddress checks whether the provided address is a valid URL.
func IsValidWebAddress(address string) bool {
	_, err := url.ParseRequestURI(address)
	return err == nil
}

// IsValidPhoneNumber checks whether the provided phone number has a valid format.
func IsValidPhoneNumber(number string) bool {
	re := regexp.MustCompile(`^(?:(?:\(?(?:00|\+)([1-4]\d\d|[1-9]\d?)\)?)?[\-\.\ \\\/]?)?((?:\(?\d{1,}\)?[\-\.\ \\\/]?){0,})(?:[\-\.\ \\\/]?(?:#|ext\.?|extension|x)[\-\.\ \\\/]?(\d+))?$`)
	return re.MatchString(number)
}

// IsValidName cheks if the given name doesn't contain any non-alpha, space or dash characters.
func IsValidName(name string) bool {
	re := regexp.MustCompile(`^[A-Za-z\s\-]*$`)
	return re.MatchString(name)
}

// MarshalProtoV1ToJSON marshals a proto V1 message to a JSON byte array
// TODO: update this once we start using V2 in CS3APIs
func MarshalProtoV1ToJSON(m proto.Message) ([]byte, error) {
	mV2 := proto.MessageV2(m)
	return protojson.Marshal(mV2)
}

// UnmarshalJSONToProtoV1 decodes a JSON byte array to a specified proto message type
// TODO: update this once we start using V2 in CS3APIs
func UnmarshalJSONToProtoV1(b []byte, m proto.Message) error {
	mV2 := proto.MessageV2(m)
	if err := protojson.Unmarshal(b, mV2); err != nil {
		return err
	}
	return nil
}

// IsRelativeReference returns true if the given reference qualifies as relative
// when the resource id is set and the path starts with a .
//
// TODO(corby): Currently if the path begins with a dot, the ResourceId is set but has empty storageId and OpaqueId
// then the reference is still being viewed as relative. We need to check if we want that because in some
// places we might not want to set both StorageId and OpaqueId so we can't do a hard check if they are set.
func IsRelativeReference(ref *provider.Reference) bool {
	return ref.ResourceId != nil && strings.HasPrefix(ref.Path, ".")
}

// IsAbsoluteReference returns true if the given reference qualifies as absolute
// when either only the resource id is set or only the path is set and starts with /
//
// TODO(corby): Currently if the path is empty, the ResourceId is set but has empty storageId and OpaqueId
// then the reference is still being viewed as absolute. We need to check if we want that because in some
// places we might not want to set both StorageId and OpaqueId so we can't do a hard check if they are set.
func IsAbsoluteReference(ref *provider.Reference) bool {
	return (ref.ResourceId != nil && ref.Path == "") || (ref.ResourceId == nil) && strings.HasPrefix(ref.Path, "/")
}

// IsAbsolutePathReference returns true if the given reference qualifies as a global path
// when only the path is set and starts with /
func IsAbsolutePathReference(ref *provider.Reference) bool {
	return ref.ResourceId == nil && strings.HasPrefix(ref.Path, "/")
}

// MakeRelativePath prefixes the path with a . to use it in a relative reference
func MakeRelativePath(p string) string {
	p = path.Join("/", p)

	if p == "/" {
		return "."
	}
	return "." + p
}

// UserTypeMap translates account type string to CS3 UserType
func UserTypeMap(accountType string) userpb.UserType {
	var t userpb.UserType
	switch accountType {
	case "primary":
		t = userpb.UserType_USER_TYPE_PRIMARY
	case "secondary":
		t = userpb.UserType_USER_TYPE_SECONDARY
	case "service":
		t = userpb.UserType_USER_TYPE_SERVICE
	case "application":
		t = userpb.UserType_USER_TYPE_APPLICATION
	case "guest":
		t = userpb.UserType_USER_TYPE_GUEST
	case "federated":
		t = userpb.UserType_USER_TYPE_FEDERATED
	case "lightweight":
		t = userpb.UserType_USER_TYPE_LIGHTWEIGHT
	// FIXME new user type
	case "spaceowner":
		t = 8
	}
	return t
}

// UserTypeToString translates CS3 UserType to user-readable string
func UserTypeToString(accountType userpb.UserType) string {
	var t string
	switch accountType {
	case userpb.UserType_USER_TYPE_PRIMARY:
		t = "primary"
	case userpb.UserType_USER_TYPE_SECONDARY:
		t = "secondary"
	case userpb.UserType_USER_TYPE_SERVICE:
		t = "service"
	case userpb.UserType_USER_TYPE_APPLICATION:
		t = "application"
	case userpb.UserType_USER_TYPE_GUEST:
		t = "guest"
	case userpb.UserType_USER_TYPE_FEDERATED:
		t = "federated"
	case userpb.UserType_USER_TYPE_LIGHTWEIGHT:
		t = "lightweight"
	// FIXME new user type
	case 8:
		t = "spaceowner"
	}
	return t
}

// GetViewMode converts a human-readable string to a view mode for opening a resource in an app.
func GetViewMode(viewMode string) gateway.OpenInAppRequest_ViewMode {
	switch viewMode {
	case "view":
		return gateway.OpenInAppRequest_VIEW_MODE_VIEW_ONLY
	case "read":
		return gateway.OpenInAppRequest_VIEW_MODE_READ_ONLY
	case "write":
		return gateway.OpenInAppRequest_VIEW_MODE_READ_WRITE
	default:
		return gateway.OpenInAppRequest_VIEW_MODE_INVALID
	}
}

// GetAppViewMode converts a human-readable string to an appprovider view mode for opening a resource in an app.
func GetAppViewMode(viewMode string) appprovider.ViewMode {
	switch viewMode {
	case "view":
		return appprovider.ViewMode_VIEW_MODE_VIEW_ONLY
	case "read":
		return appprovider.ViewMode_VIEW_MODE_READ_ONLY
	case "write":
		return appprovider.ViewMode_VIEW_MODE_READ_WRITE
	case "preview":
		return appprovider.ViewMode_VIEW_MODE_PREVIEW
	default:
		return appprovider.ViewMode_VIEW_MODE_INVALID
	}
}

// AppendPlainToOpaque adds a new key value pair as a plain string on the given opaque and returns it
func AppendPlainToOpaque(o *types.Opaque, key, value string) *types.Opaque {
	o = ensureOpaque(o)

	o.Map[key] = &types.OpaqueEntry{
		Decoder: "plain",
		Value:   []byte(value),
	}
	return o
}

// AppendJSONToOpaque adds a new key value pair as a json on the given opaque and returns it. Ignores errors
func AppendJSONToOpaque(o *types.Opaque, key string, value interface{}) *types.Opaque {
	o = ensureOpaque(o)

	b, _ := json.Marshal(value)
	o.Map[key] = &types.OpaqueEntry{
		Decoder: "json",
		Value:   b,
	}
	return o
}

// ReadPlainFromOpaque reads a plain string from the given opaque map
func ReadPlainFromOpaque(o *types.Opaque, key string) string {
	if o.GetMap() == nil {
		return ""
	}
	if e, ok := o.Map[key]; ok && e.Decoder == "plain" {
		return string(e.Value)
	}
	return ""
}

// ReadJSONFromOpaque reads and unmarshals a value from the opaque in the given interface{} (Make sure it's a pointer!)
func ReadJSONFromOpaque(o *types.Opaque, key string, valptr interface{}) error {
	if o.GetMap() == nil {
		return errors.New("not found")
	}

	e, ok := o.Map[key]
	if !ok || e.Decoder != "json" {
		return errors.New("not found")
	}

	return json.Unmarshal(e.Value, valptr)
}

// ExistsInOpaque returns true if the key exists in the opaque (ignoring the value)
func ExistsInOpaque(o *types.Opaque, key string) bool {
	if o.GetMap() == nil {
		return false
	}

	_, ok := o.Map[key]
	return ok
}

// SpaceGrantOpaque returns an Opaque with the "spacegrant" key set, which
// signals to storage and event middleware that a grant targets a space root.
func SpaceGrantOpaque() *types.Opaque {
	return &types.Opaque{
		Map: map[string]*types.OpaqueEntry{
			"spacegrant": {},
		},
	}
}

// MergeOpaques will merge the opaques. If a key exists in both opaques
// the values from the first opaque will be taken
func MergeOpaques(o *types.Opaque, p *types.Opaque) *types.Opaque {
	p = ensureOpaque(p)
	for k, v := range o.GetMap() {
		p.Map[k] = v
	}
	return p
}

// ensures the opaque is initialized
func ensureOpaque(o *types.Opaque) *types.Opaque {
	if o == nil {
		o = &types.Opaque{}
	}
	if o.Map == nil {
		o.Map = map[string]*types.OpaqueEntry{}
	}
	return o
}

// RemoveItem removes the given item, its children and all empty parent folders
func RemoveItem(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	for {
		path = filepath.Dir(path)
		if err := os.Remove(path); err != nil {
			// remove will fail when the dir is not empty.
			// We can exit in that case
			return nil
		}

	}

}
