// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package propfind

import (
	"encoding/xml"
	"strings"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/pkg/openextension"
)

// openExtensionProp renders a stored extension value with its xsi:type;
// false for a value the codec cannot read.
func openExtensionProp(name, key, raw string) (prop.PropertyXML, bool) {
	v, err := openextension.DecodeValue(raw)
	if err != nil {
		return prop.PropertyXML{}, false
	}
	d := openextension.ToDAV(v)
	p := prop.PropertyXML{
		XMLName:  xml.Name{Space: openextension.Namespace(name), Local: key},
		InnerXML: d.InnerXML,
	}
	if d.Type != "" {
		p.Attrs = append(p.Attrs,
			xml.Attr{Name: xml.Name{Local: "xmlns:xsi"}, Value: openextension.XSINamespace},
			xml.Attr{Name: xml.Name{Local: "xmlns:xs"}, Value: openextension.XSNamespace},
			xml.Attr{Name: xml.Name{Local: "xsi:type"}, Value: d.Type},
		)
	}
	if strings.Contains(string(d.InnerXML), "<oc:") {
		p.Attrs = append(p.Attrs, xml.Attr{Name: xml.Name{Local: "xmlns:oc"}, Value: openextension.OCNamespace})
	}
	return p, true
}
