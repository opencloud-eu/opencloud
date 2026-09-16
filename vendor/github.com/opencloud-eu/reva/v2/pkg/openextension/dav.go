// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package openextension

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RFC 4316 datatypes
const (
	XSINamespace = "http://www.w3.org/2001/XMLSchema-instance"
	XSNamespace  = "http://www.w3.org/2001/XMLSchema"
	OCNamespace  = "http://owncloud.org/ns"
)

// xsi:type values; a string carries none
const (
	DAVTypeInteger  = "xs:integer"
	DAVTypeDecimal  = "xs:decimal"
	DAVTypeBoolean  = "xs:boolean"
	DAVTypeDateTime = "xs:dateTime"
	DAVTypeGeo      = "oc:geoCoordinates"
	DAVTypeList     = "oc:list"
)

// DAVProperty is the rendering of one property; the caller declares the xsi,
// xs and oc prefixes on the element.
type DAVProperty struct {
	Type     string
	InnerXML []byte
}

func ToDAV(v Value) DAVProperty {
	if v.Array {
		var b bytes.Buffer
		for _, item := range elementsOf(v) {
			p := ToDAV(item)
			b.WriteString("<oc:item")
			if p.Type != "" {
				b.WriteString(` xsi:type="` + p.Type + `"`)
			}
			b.WriteByte('>')
			b.Write(p.InnerXML)
			b.WriteString("</oc:item>")
		}
		return DAVProperty{Type: DAVTypeList, InnerXML: b.Bytes()}
	}
	switch v.Kind {
	case KindNumber:
		typ := DAVTypeDecimal
		if v.IsInteger() {
			typ = DAVTypeInteger
		}
		return DAVProperty{Type: typ, InnerXML: v.Raw}
	case KindBool:
		return DAVProperty{Type: DAVTypeBoolean, InnerXML: v.Raw}
	case KindDate:
		return DAVProperty{Type: DAVTypeDateTime, InnerXML: escapeText(v.Strings()[0])}
	case KindGeo:
		p, _ := v.Geo()
		var b bytes.Buffer
		fmt.Fprintf(&b, "<oc:latitude>%s</oc:latitude><oc:longitude>%s</oc:longitude>", formatFloat(p.Latitude), formatFloat(p.Longitude))
		if p.Altitude != nil {
			fmt.Fprintf(&b, "<oc:altitude>%s</oc:altitude>", formatFloat(*p.Altitude))
		}
		return DAVProperty{Type: DAVTypeGeo, InnerXML: b.Bytes()}
	default:
		s := v.Strings()
		if len(s) == 0 {
			return DAVProperty{}
		}
		return DAVProperty{InnerXML: escapeText(s[0])}
	}
}

func elementsOf(v Value) []Value {
	var elems []json.RawMessage
	if err := json.Unmarshal(v.Raw, &elems); err != nil {
		return nil
	}
	out := make([]Value, 0, len(elems))
	for _, e := range elems {
		out = append(out, Value{Kind: v.Kind, Raw: e})
	}
	return out
}

func formatFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func escapeText(s string) []byte {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.Bytes()
}

// FromDAV parses a PROPPATCH property value; an unknown or missing xsi:type
// is a string.
func FromDAV(key string, innerXML []byte, xsiType string) (Value, error) {
	if err := ValidateKey(key); err != nil {
		return Value{}, err
	}
	if len(innerXML) > MaxValueSize {
		return Value{}, invalid("property %s is %d bytes, at most %d are allowed", key, len(innerXML), MaxValueSize)
	}
	switch davKind(xsiType) {
	case "list":
		items, err := childElements(innerXML)
		if err != nil {
			return Value{}, invalid("property %s: %v", key, err)
		}
		raws := make([]string, 0, len(items))
		for i, item := range items {
			if item.Name != "item" {
				return Value{}, invalid("property %s: a list holds oc:item elements only", key)
			}
			ev, err := FromDAV(fmt.Sprintf("%s_%d", key, i), item.InnerXML, item.Type)
			if err != nil {
				return Value{}, err
			}
			if ev.Array || ev.Kind == KindGeo {
				return Value{}, invalid("property %s: arrays hold scalars only", key)
			}
			raws = append(raws, string(ev.Raw))
		}
		return valueOf(key, json.RawMessage("["+strings.Join(raws, ",")+"]"), "")
	case "geo":
		children, err := childElements(innerXML)
		if err != nil {
			return Value{}, invalid("property %s: %v", key, err)
		}
		members := make([]string, 0, len(children))
		for _, c := range children {
			text, err := textOf(c.InnerXML)
			if err != nil {
				return Value{}, invalid("property %s: %v", key, err)
			}
			if _, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err != nil {
				return Value{}, invalid("property %s: %s is not a number", key, c.Name)
			}
			members = append(members, fmt.Sprintf("%q:%s", c.Name, strings.TrimSpace(text)))
		}
		return valueOf(key, json.RawMessage("{"+strings.Join(members, ",")+"}"), ODataGeoCoordinates)
	}

	text, err := textOf(innerXML)
	if err != nil {
		return Value{}, invalid("property %s: %v", key, err)
	}
	text = strings.TrimSpace(text)
	switch davKind(xsiType) {
	case "number":
		if _, err := strconv.ParseFloat(text, 64); err != nil || !json.Valid([]byte(text)) {
			return Value{}, invalid("property %s: %q is not a number", key, text)
		}
		return valueOf(key, json.RawMessage(text), "")
	case "bool":
		switch text {
		case "true", "1":
			return Value{Kind: KindBool, Raw: json.RawMessage("true")}, nil
		case "false", "0":
			return Value{Kind: KindBool, Raw: json.RawMessage("false")}, nil
		}
		return Value{}, invalid("property %s: %q is not a boolean", key, text)
	case "date":
		if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
			return Value{}, invalid("property %s: %q is not an RFC 3339 date-time", key, text)
		}
		return Value{Kind: KindDate, Raw: jsonString(text)}, nil
	}
	return Value{Kind: KindString, Raw: jsonString(text)}, nil
}

// jsonString encodes without HTML escaping, the stored value reads as written.
func jsonString(s string) json.RawMessage {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return json.RawMessage(bytes.TrimSpace(b.Bytes()))
}

// davKind classifies an xsi:type by its local part: encoding/xml does not
// resolve prefixes in attribute values.
func davKind(xsiType string) string {
	local := xsiType
	if i := strings.LastIndex(local, ":"); i >= 0 {
		local = local[i+1:]
	}
	switch strings.ToLower(strings.TrimSpace(local)) {
	case "integer", "int", "long", "short", "byte", "decimal", "double", "float":
		return "number"
	case "boolean":
		return "bool"
	case "datetime":
		return "date"
	case "geocoordinates":
		return "geo"
	case "list":
		return "list"
	}
	return "string"
}

type childElement struct {
	Name     string
	Type     string
	InnerXML []byte
}

// childElements lists the direct children of a property value. The fragment
// may use prefixes declared on the request root, so the decoder is lenient and
// elements are matched by local name.
func childElements(innerXML []byte) ([]childElement, error) {
	d := xml.NewDecoder(bytes.NewReader(append(append([]byte("<r>"), innerXML...), []byte("</r>")...)))
	d.Strict = false
	var out []childElement
	depth := 0
	for {
		tok, err := d.Token()
		if err != nil {
			return out, nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth != 2 {
				continue
			}
			var raw struct {
				InnerXML []byte `xml:",innerxml"`
			}
			if err := d.DecodeElement(&raw, &t); err != nil {
				return nil, err
			}
			depth--
			c := childElement{Name: t.Name.Local, InnerXML: raw.InnerXML}
			for _, a := range t.Attr {
				if a.Name.Local == "type" {
					c.Type = a.Value
				}
			}
			out = append(out, c)
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 1 && len(bytes.TrimSpace(t)) > 0 {
				return nil, fmt.Errorf("mixed content is not allowed")
			}
		}
	}
}

func textOf(innerXML []byte) (string, error) {
	var v struct {
		Text string `xml:",chardata"`
	}
	d := xml.NewDecoder(bytes.NewReader(append(append([]byte("<r>"), innerXML...), []byte("</r>")...)))
	d.Strict = false
	if err := d.Decode(&v); err != nil {
		return "", err
	}
	return v.Text, nil
}
