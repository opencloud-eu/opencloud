// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

// Package openextension models open extensions (libregraph openTypeExtension).
// Every property is one arbitrary metadata key,
// http://opencloud.eu/ns/extensions/<extensionName>/<property>, whose value
// carries its kind as a one-letter prefix, see EncodeValue. Values are
// scalars, homogeneous arrays of scalars or a geoCoordinates object. In the
// API the kind follows from the JSON type except for date-times and geo
// points, which carry a `<property>@odata.type` annotation.
package openextension

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// NamespacePrefix is the WebDAV namespace of an extension and the prefix
	// of its metadata keys: <NamespacePrefix><extensionName>/<property>.
	NamespacePrefix = "http://opencloud.eu/ns/extensions/"

	ODataTypeSuffix     = "@odata.type"
	ExtensionNameMember = "extensionName"

	MaxProperties = 64
	MaxValueSize  = 4 * 1024 // bytes of one stored value
)

// OData type annotations.
const (
	ODataString         = "#String"
	ODataInt64          = "#Int64"
	ODataDouble         = "#Double"
	ODataBoolean        = "#Boolean"
	ODataDateTimeOffset = "#DateTimeOffset"
	ODataGeoCoordinates = "#microsoft.graph.geoCoordinates"
	ODataOpenType       = "#microsoft.graph.openTypeExtension"
)

// Kind is the value kind of a property.
type Kind string

const (
	KindString Kind = "string"
	KindNumber Kind = "number"
	KindBool   Kind = "bool"
	KindDate   Kind = "date"
	KindGeo    Kind = "geo"
)

// ErrInvalid marks a payload the codec rejects.
var ErrInvalid = errors.New("invalid extension")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

var (
	nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*(\.[A-Za-z0-9][A-Za-z0-9_-]*)+$`) // reverse DNS
	keyRe  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)                             // OData simple identifier
)

func ValidateName(name string) error {
	if len(name) > 128 || !nameRe.MatchString(name) {
		return invalid("extension name %q must be a reverse DNS name such as com.example.app", name)
	}
	return nil
}

func ValidateKey(key string) error {
	if !keyRe.MatchString(key) {
		return invalid("property %q must be a simple identifier of letters, digits and underscores", key)
	}
	return nil
}

func Namespace(name string) string { return NamespacePrefix + name }

// NameFromNamespace returns the extension name of a WebDAV namespace.
func NameFromNamespace(ns string) (string, bool) {
	name, ok := strings.CutPrefix(ns, NamespacePrefix)
	if !ok || name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

// Key is the metadata key of one property.
func Key(name, property string) string { return Namespace(name) + "/" + property }

// SplitKey returns extension name and property of a metadata key.
func SplitKey(key string) (name, property string, ok bool) {
	rest, ok := strings.CutPrefix(key, NamespacePrefix)
	if !ok {
		return "", "", false
	}
	i := strings.LastIndexByte(rest, '/')
	if i <= 0 || i == len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

type GeoPoint struct {
	Latitude  float64
	Longitude float64
	Altitude  *float64
}

// Value is one property, the JSON as the client wrote it.
type Value struct {
	Kind  Kind
	Array bool
	Raw   json.RawMessage
}

type OpenExtension struct {
	Name   string
	Values map[string]Value
}

// Keys returns the property names sorted.
func (e OpenExtension) Keys() []string {
	keys := make([]string, 0, len(e.Values))
	for k := range e.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Strings returns the elements of a string or date property, nil for other kinds.
func (v Value) Strings() []string {
	if v.Kind != KindString && v.Kind != KindDate {
		return nil
	}
	var out []string
	if v.Array {
		_ = json.Unmarshal(v.Raw, &out)
		return out
	}
	var s string
	if json.Unmarshal(v.Raw, &s) != nil {
		return nil
	}
	return []string{s}
}

func (v Value) Numbers() []float64 {
	if v.Kind != KindNumber {
		return nil
	}
	var out []float64
	if v.Array {
		_ = json.Unmarshal(v.Raw, &out)
		return out
	}
	var f float64
	if json.Unmarshal(v.Raw, &f) != nil {
		return nil
	}
	return []float64{f}
}

func (v Value) Bools() []bool {
	if v.Kind != KindBool {
		return nil
	}
	var out []bool
	if v.Array {
		_ = json.Unmarshal(v.Raw, &out)
		return out
	}
	var b bool
	if json.Unmarshal(v.Raw, &b) != nil {
		return nil
	}
	return []bool{b}
}

func (v Value) Times() []time.Time {
	if v.Kind != KindDate {
		return nil
	}
	var out []time.Time
	for _, s := range v.Strings() {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (v Value) Geo() (GeoPoint, bool) {
	if v.Kind != KindGeo || v.Array {
		return GeoPoint{}, false
	}
	return parseGeo(v.Raw)
}

// IsInteger reports whether a scalar number was written without fraction or exponent.
func (v Value) IsInteger() bool {
	if v.Kind != KindNumber || v.Array {
		return false
	}
	return isIntegerLiteral(string(v.Raw))
}

func isIntegerLiteral(s string) bool {
	return !strings.ContainsAny(s, ".eE")
}

func (v Value) ODataType() string {
	var elem string
	switch v.Kind {
	case KindString:
		elem = ODataString
	case KindNumber:
		elem = ODataDouble
		if !v.Array && v.IsInteger() {
			elem = ODataInt64
		}
	case KindBool:
		elem = ODataBoolean
	case KindDate:
		elem = ODataDateTimeOffset
	case KindGeo:
		return ODataGeoCoordinates
	}
	if v.Array {
		return "#Collection(" + strings.TrimPrefix(elem, "#") + ")"
	}
	return elem
}

// Inferable reports whether the kind follows from the JSON value alone.
func (v Value) Inferable() bool {
	return v.Kind != KindDate && v.Kind != KindGeo
}

// The stored form is <code>:<payload>. Lower case codes are scalars, upper
// case codes arrays of that kind holding the JSON array. A value without a
// known code is unreadable; readers skip it.
var kindCodes = map[Kind]byte{KindString: 's', KindNumber: 'n', KindBool: 'b', KindDate: 'd', KindGeo: 'g'}

var (
	codeKinds       = map[byte]Kind{'s': KindString, 'n': KindNumber, 'b': KindBool, 'd': KindDate}
	codeAnnotations = map[byte]string{'S': "#Collection(String)", 'N': "#Collection(Double)", 'B': "#Collection(Boolean)", 'D': "#Collection(DateTimeOffset)"}
)

func EncodeValue(v Value) string {
	code := kindCodes[v.Kind]
	if v.Array {
		return string(code-'a'+'A') + ":" + string(v.Raw)
	}
	switch v.Kind {
	case KindString, KindDate:
		s := v.Strings()
		if len(s) == 0 {
			return string(code) + ":"
		}
		return string(code) + ":" + s[0]
	case KindGeo:
		p, ok := v.Geo()
		if !ok {
			return string(code) + ":"
		}
		payload := formatFloat(p.Latitude) + "," + formatFloat(p.Longitude)
		if p.Altitude != nil {
			payload += "," + formatFloat(*p.Altitude)
		}
		return string(code) + ":" + payload
	default:
		return string(code) + ":" + string(v.Raw)
	}
}

func DecodeValue(raw string) (Value, error) {
	if len(raw) < 2 || raw[1] != ':' {
		return Value{}, invalid("stored value %q has no type code", raw)
	}
	code, payload := raw[0], raw[2:]
	if annotation, ok := codeAnnotations[code]; ok {
		v, err := valueOf("value", json.RawMessage(payload), annotation)
		if err != nil {
			return Value{}, err
		}
		if !v.Array {
			return Value{}, invalid("stored value %q is not an array", raw)
		}
		return v, nil
	}
	switch kind := codeKinds[code]; kind {
	case KindString:
		return Value{Kind: KindString, Raw: jsonString(payload)}, nil
	case KindDate:
		if _, err := time.Parse(time.RFC3339Nano, payload); err != nil {
			return Value{}, invalid("stored value %q is not an RFC 3339 date-time", raw)
		}
		return Value{Kind: KindDate, Raw: jsonString(payload)}, nil
	case KindNumber, KindBool:
		v, err := valueOf("value", json.RawMessage(payload), "")
		if err != nil || v.Kind != kind || v.Array {
			return Value{}, invalid("stored value %q is not a %s", raw, kind)
		}
		return v, nil
	}
	if code == kindCodes[KindGeo] {
		parts := strings.Split(payload, ",")
		if len(parts) < 2 || len(parts) > 3 {
			return Value{}, invalid("stored value %q is not a geo point", raw)
		}
		members := []string{`"latitude":` + parts[0], `"longitude":` + parts[1]}
		if len(parts) == 3 {
			members = append(members, `"altitude":`+parts[2])
		}
		obj := json.RawMessage("{" + strings.Join(members, ",") + "}")
		if _, ok := parseGeo(obj); !ok {
			return Value{}, invalid("stored value %q is not a geo point", raw)
		}
		return Value{Kind: KindGeo, Raw: obj}, nil
	}
	return Value{}, invalid("stored value %q has an unknown type code", raw)
}

// Patch is a client write; members sent as null are removed.
type Patch struct {
	Set    map[string]Value
	Remove []string
}

// Parse reads a libregraph openTypeExtension body. Annotations are validated
// against their values and consumed.
func Parse(body []byte) (Patch, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		return Patch{}, invalid("body must be a JSON object: %v", err)
	}
	delete(members, ExtensionNameMember)
	delete(members, "id")
	delete(members, ODataTypeSuffix)

	annotations := map[string]string{}
	for key, raw := range members {
		if prop, ok := strings.CutSuffix(key, ODataTypeSuffix); ok {
			var typ string
			if err := json.Unmarshal(raw, &typ); err != nil {
				return Patch{}, invalid("%s must be a string", key)
			}
			annotations[prop] = typ
			delete(members, key)
		}
	}

	patch := Patch{Set: map[string]Value{}}
	for key, raw := range members {
		if err := ValidateKey(key); err != nil {
			return Patch{}, err
		}
		if strings.TrimSpace(string(raw)) == "null" {
			patch.Remove = append(patch.Remove, key)
			continue
		}
		v, err := valueOf(key, raw, annotations[key])
		if err != nil {
			return Patch{}, err
		}
		patch.Set[key] = v
	}
	for prop := range annotations {
		if _, ok := members[prop]; !ok {
			return Patch{}, invalid("%s%s annotates a property that is not in the body", prop, ODataTypeSuffix)
		}
	}
	sort.Strings(patch.Remove)
	return patch, nil
}

// Metadata returns the keys a patch writes and removes, sorted.
func (p Patch) Metadata(name string) (set map[string]string, unset []string) {
	set = make(map[string]string, len(p.Set))
	for key, v := range p.Set {
		set[Key(name, key)] = EncodeValue(v)
	}
	unset = make([]string, 0, len(p.Remove))
	for _, key := range p.Remove {
		unset = append(unset, Key(name, key))
	}
	sort.Strings(unset)
	return set, unset
}

func (e *OpenExtension) Apply(p Patch) {
	if e.Values == nil {
		e.Values = map[string]Value{}
	}
	for key, v := range p.Set {
		e.Values[key] = v
	}
	for _, key := range p.Remove {
		delete(e.Values, key)
	}
}

// MarshalJSON renders the libregraph shape, annotating only what JSON cannot express.
func (e OpenExtension) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	n, _ := json.Marshal(e.Name)
	b.WriteString(`"` + ExtensionNameMember + `":`)
	b.Write(n)
	for _, key := range e.Keys() {
		v := e.Values[key]
		k, _ := json.Marshal(key)
		b.WriteByte(',')
		b.Write(k)
		b.WriteByte(':')
		b.Write(v.Raw)
		if !v.Inferable() {
			a, _ := json.Marshal(key + ODataTypeSuffix)
			t, _ := json.Marshal(v.ODataType())
			b.WriteByte(',')
			b.Write(a)
			b.WriteByte(':')
			b.Write(t)
		}
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

func valueOf(key string, raw json.RawMessage, annotation string) (Value, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return Value{}, invalid("property %s has no value", key)
	}
	if len(raw) > MaxValueSize {
		return Value{}, invalid("property %s is %d bytes, at most %d are allowed", key, len(raw), MaxValueSize)
	}
	elemType, isCollection := annotation, false
	if inner, ok := strings.CutPrefix(annotation, "#Collection("); ok {
		if !strings.HasSuffix(inner, ")") {
			return Value{}, invalid("property %s: unknown type annotation %q", key, annotation)
		}
		elemType, isCollection = "#"+strings.TrimSuffix(inner, ")"), true
	}

	switch raw[0] {
	case '[':
		if annotation != "" && !isCollection {
			return Value{}, invalid("property %s is an array but annotated as %s", key, annotation)
		}
		var elems []json.RawMessage
		if err := json.Unmarshal(raw, &elems); err != nil {
			return Value{}, invalid("property %s: %v", key, err)
		}
		kind := Kind("")
		for i, elem := range elems {
			ev, err := valueOf(fmt.Sprintf("%s[%d]", key, i), elem, elemType)
			if err != nil {
				return Value{}, err
			}
			if ev.Array || ev.Kind == KindGeo {
				return Value{}, invalid("property %s: arrays hold scalars only", key)
			}
			if kind != "" && ev.Kind != kind {
				return Value{}, invalid("property %s: array elements must all be %s", key, kind)
			}
			kind = ev.Kind
		}
		if kind == "" {
			kind = KindString // an empty array carries no type, treat it as strings
			if elemType != "" {
				if k, err := scalarKind(key, elemType); err == nil {
					kind = k
				}
			}
		}
		return Value{Kind: kind, Array: true, Raw: raw}, nil
	case '{':
		if elemType != ODataGeoCoordinates {
			return Value{}, invalid("property %s: objects are only allowed as geoCoordinates annotated with %s", key, ODataGeoCoordinates)
		}
		if isCollection {
			return Value{}, invalid("property %s: a collection of geoCoordinates is not supported", key)
		}
		if _, ok := parseGeo(raw); !ok {
			return Value{}, invalid("property %s: geoCoordinates need numeric latitude and longitude and nothing but an optional altitude", key)
		}
		return Value{Kind: KindGeo, Raw: raw}, nil
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return Value{}, invalid("property %s: %v", key, err)
		}
		switch elemType {
		case "", ODataString:
			return Value{Kind: KindString, Raw: raw}, nil
		case ODataDateTimeOffset:
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return Value{}, invalid("property %s: %q is not an RFC 3339 date-time", key, s)
			}
			return Value{Kind: KindDate, Raw: raw}, nil
		}
		return Value{}, invalid("property %s is a string but annotated as %s", key, elemType)
	case 't', 'f':
		if elemType != "" && elemType != ODataBoolean {
			return Value{}, invalid("property %s is a boolean but annotated as %s", key, elemType)
		}
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return Value{}, invalid("property %s: %v", key, err)
		}
		return Value{Kind: KindBool, Raw: raw}, nil
	case 'n':
		return Value{}, invalid("property %s: null is only allowed at the top level to remove a property", key)
	default:
		if _, err := strconv.ParseFloat(string(raw), 64); err != nil {
			return Value{}, invalid("property %s: %q is not a number", key, string(raw))
		}
		switch elemType {
		case "", ODataDouble:
		case ODataInt64:
			if !isIntegerLiteral(string(raw)) {
				return Value{}, invalid("property %s: %s is not an integer", key, string(raw))
			}
		default:
			return Value{}, invalid("property %s is a number but annotated as %s", key, elemType)
		}
		return Value{Kind: KindNumber, Raw: raw}, nil
	}
}

func scalarKind(key, annotation string) (Kind, error) {
	switch annotation {
	case ODataString:
		return KindString, nil
	case ODataInt64, ODataDouble:
		return KindNumber, nil
	case ODataBoolean:
		return KindBool, nil
	case ODataDateTimeOffset:
		return KindDate, nil
	case ODataGeoCoordinates:
		return KindGeo, nil
	}
	return "", invalid("property %s: unknown type annotation %q", key, annotation)
}

func parseGeo(raw json.RawMessage) (GeoPoint, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return GeoPoint{}, false
	}
	num := func(name string) (float64, bool) {
		r, ok := obj[name]
		if !ok {
			return 0, false
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(string(r)), 64)
		return f, err == nil
	}
	lat, okLat := num("latitude")
	lon, okLon := num("longitude")
	if !okLat || !okLon || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return GeoPoint{}, false
	}
	p := GeoPoint{Latitude: lat, Longitude: lon}
	for name := range obj {
		switch name {
		case "latitude", "longitude":
		case "altitude":
			alt, ok := num("altitude")
			if !ok {
				return GeoPoint{}, false
			}
			p.Altitude = &alt
		default:
			return GeoPoint{}, false
		}
	}
	return p, true
}

// FromMetadata decodes the extensions in an arbitrary metadata map, sorted by
// name. Unreadable values are skipped, the map is user-writable.
func FromMetadata(metadata map[string]string) []OpenExtension {
	byName := map[string]*OpenExtension{}
	for key, raw := range metadata {
		name, property, ok := SplitKey(key)
		if !ok {
			continue
		}
		v, err := DecodeValue(raw)
		if err != nil {
			continue
		}
		ext, known := byName[name]
		if !known {
			ext = &OpenExtension{Name: name, Values: map[string]Value{}}
			byName[name] = ext
		}
		ext.Values[property] = v
	}
	out := make([]OpenExtension, 0, len(byName))
	for _, ext := range byName {
		out = append(out, *ext)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup returns one extension; ok reports whether the item stores any of its keys.
func Lookup(metadata map[string]string, name string) (OpenExtension, bool) {
	ext := OpenExtension{Name: name, Values: map[string]Value{}}
	found := false
	prefix := Namespace(name) + "/"
	for key, raw := range metadata {
		property, ok := strings.CutPrefix(key, prefix)
		if !ok || property == "" || strings.Contains(property, "/") {
			continue
		}
		found = true
		if v, err := DecodeValue(raw); err == nil {
			ext.Values[property] = v
		}
	}
	return ext, found
}

// MetadataKeys returns the keys of an extension in an arbitrary metadata map, sorted.
func MetadataKeys(metadata map[string]string, name string) []string {
	prefix := Namespace(name) + "/"
	var keys []string
	for key := range metadata {
		if property, ok := strings.CutPrefix(key, prefix); ok && property != "" && !strings.Contains(property, "/") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
