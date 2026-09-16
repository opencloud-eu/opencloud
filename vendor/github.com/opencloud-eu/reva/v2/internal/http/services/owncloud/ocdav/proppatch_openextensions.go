// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package ocdav

import (
	"encoding/xml"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/pkg/openextension"
)

// openExtensionValue validates a PROPPATCH property in an extension namespace
// and returns the stored form of its value, empty for a removal.
func openExtensionValue(name string, p prop.PropertyXML, remove bool) (string, error) {
	if err := openextension.ValidateName(name); err != nil {
		return "", err
	}
	if remove {
		return "", openextension.ValidateKey(p.XMLName.Local)
	}
	v, err := openextension.FromDAV(p.XMLName.Local, p.InnerXML, xsiType(p.Attrs))
	if err != nil {
		return "", err
	}
	return openextension.EncodeValue(v), nil
}

func xsiType(attrs []xml.Attr) string {
	for _, a := range attrs {
		if a.Name.Local == "type" && (a.Name.Space == openextension.XSINamespace || a.Name.Space == "xsi") {
			return a.Value
		}
	}
	return ""
}
