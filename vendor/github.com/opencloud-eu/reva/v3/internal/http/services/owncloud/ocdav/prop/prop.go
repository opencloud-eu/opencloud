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

package prop

import (
	"bytes"
	"encoding/xml"
	"unicode/utf8"
)

// PropertyXML represents a single DAV resource property as defined in RFC 4918.
// http://www.webdav.org/specs/rfc4918.html#data.model.for.resource.properties
type PropertyXML struct {
	// XMLName is the fully qualified name that identifies this property.
	XMLName xml.Name

	// Lang is an optional xml:lang attribute.
	Lang string `xml:"xml:lang,attr,omitempty"`

	// InnerXML contains the XML representation of the property value.
	// See http://www.webdav.org/specs/rfc4918.html#property_values
	//
	// Property values of complex type or mixed-content must have fully
	// expanded XML namespaces or be self-contained with according
	// XML namespace declarations. They must not rely on any XML
	// namespace declarations within the scope of the XML document,
	// even including the DAV: namespace.
	InnerXML []byte `xml:",innerxml"`
}

func xmlEscaped(val string) []byte {
	buf := new(bytes.Buffer)
	xml.Escape(buf, []byte(val))
	return buf.Bytes()
}

// EscapedNS returns a new PropertyXML instance while xml-escaping the value
func EscapedNS(namespace string, local string, val string) PropertyXML {
	return PropertyXML{
		XMLName:  xml.Name{Space: namespace, Local: local},
		Lang:     "",
		InnerXML: xmlEscaped(val),
	}
}

var (
	escAmp  = []byte("&amp;")
	escLT   = []byte("&lt;")
	escGT   = []byte("&gt;")
	escFFFD = []byte(string(utf8.RuneError)) // Unicode replacement character
)

// Decide whether the given rune is in the XML Character Range, per
// the Char production of https://www.xml.com/axml/testaxml.htm,
// Section 2.2 Characters.
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= utf8.RuneError ||
		r >= 0x10000 && r <= 0x10FFFF
}

// Escaped returns a new PropertyXML instance while replacing only
// * `&` with `&amp;`
// * `<` with `&lt;`
// * `>` with `&gt;`
// as defined in https://www.w3.org/TR/REC-xml/#syntax:
//
// > The ampersand character (&) and the left angle bracket (<) must not appear
// > in their literal form, except when used as markup delimiters, or within a
// > comment, a processing instruction, or a CDATA section. If they are needed
// > elsewhere, they must be escaped using either numeric character references
// > or the strings " &amp; " and " &lt; " respectively. The right angle
// > bracket (>) may be represented using the string " &gt; ", and must, for
// > compatibility, be escaped using either " &gt; " or a character reference
// > when it appears in the string " ]]> " in content, when that string is not
// > marking the end of a CDATA section.
//
// The code ignores errors as the legacy Escaped() does
// TODO properly use the space
func Escaped(key, val string) PropertyXML {
	s := []byte(val)
	w := bytes.NewBuffer(make([]byte, 0, len(s)))
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '>':
			esc = escGT
		default:
			if !isInCharacterRange(r) || (r == utf8.RuneError && width == 1) {
				esc = escFFFD
				break
			}
			continue
		}
		if _, err := w.Write(s[last : i-width]); err != nil {
			break
		}
		if _, err := w.Write(esc); err != nil {
			break
		}
		last = i
	}
	_, _ = w.Write(s[last:])
	return PropertyXML{
		XMLName:  xml.Name{Space: "", Local: key},
		Lang:     "",
		InnerXML: w.Bytes(),
	}
}

// NotFound returns a new PropertyXML instance with an empty value
func NotFound(key string) PropertyXML {
	return PropertyXML{
		XMLName: xml.Name{Space: "", Local: key},
		Lang:    "",
	}
}

// NotFoundNS returns a new PropertyXML instance with the given namespace and an empty value
func NotFoundNS(namespace, key string) PropertyXML {
	return PropertyXML{
		XMLName: xml.Name{Space: namespace, Local: key},
		Lang:    "",
	}
}

// Raw returns a new PropertyXML instance for the given key/value pair
// TODO properly use the space
func Raw(key, val string) PropertyXML {
	return PropertyXML{
		XMLName:  xml.Name{Space: "", Local: key},
		Lang:     "",
		InnerXML: []byte(val),
	}
}

// Next returns the next token, if any, in the XML stream of d.
// RFC 4918 requires to ignore comments, processing instructions
// and directives.
// http://www.webdav.org/specs/rfc4918.html#property_values
// http://www.webdav.org/specs/rfc4918.html#xml-extensibility
func Next(d *xml.Decoder) (xml.Token, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return t, err
		}
		switch t.(type) {
		case xml.Comment, xml.Directive, xml.ProcInst:
			continue
		default:
			return t, nil
		}
	}
}

// ActiveLock holds active lock xml data
//
//	http://www.webdav.org/specs/rfc4918.html#ELEMENT_activelock
//
// <!ELEMENT activelock (lockscope, locktype, depth, owner?, timeout?,
//
//	locktoken?, lockroot)>
type ActiveLock struct {
	XMLName   xml.Name  `xml:"activelock"`
	Exclusive *struct{} `xml:"lockscope>exclusive,omitempty"`
	Shared    *struct{} `xml:"lockscope>shared,omitempty"`
	Write     *struct{} `xml:"locktype>write,omitempty"`
	Depth     string    `xml:"depth"`
	Owner     Owner     `xml:"owner,omitempty"`
	Timeout   string    `xml:"timeout,omitempty"`
	Locktoken string    `xml:"locktoken>href"`
	Lockroot  string    `xml:"lockroot>href,omitempty"`
}

// Owner captures the inner UML of a lock owner element http://www.webdav.org/specs/rfc4918.html#ELEMENT_owner
type Owner struct {
	InnerXML string `xml:",innerxml"`
}

// Escape repaces ", &, ', < and > with their xml representation
func Escape(s string) string {
	b := bytes.NewBuffer(nil)
	_ = xml.EscapeText(b, []byte(s))
	return b.String()
}
