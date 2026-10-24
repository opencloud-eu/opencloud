package webdav

import (
	"encoding/xml"
	"io"
	"net/http"
)

type code int

const (
	// SabredavBadRequest maps to HTTP 400
	SabredavBadRequest code = iota
	// SabredavMethodNotAllowed maps to HTTP 405
	SabredavMethodNotAllowed
	// SabredavNotAuthenticated maps to HTTP 401
	SabredavNotAuthenticated
	// SabredavPreconditionFailed maps to HTTP 412
	SabredavPreconditionFailed
	// SabredavPermissionDenied maps to HTTP 403
	SabredavPermissionDenied
	// SabredavNotFound maps to HTTP 404
	SabredavNotFound
	// SabredavConflict maps to HTTP 409
	SabredavConflict
)

var (
	codesEnum = []string{
		"Sabre\\DAV\\Exception\\BadRequest",
		"Sabre\\DAV\\Exception\\MethodNotAllowed",
		"Sabre\\DAV\\Exception\\NotAuthenticated",
		"Sabre\\DAV\\Exception\\PreconditionFailed",
		"Sabre\\DAV\\Exception\\PermissionDenied",
		"Sabre\\DAV\\Exception\\NotFound",
		"Sabre\\DAV\\Exception\\Conflict",
	}
)

type Exception struct {
	Code    code
	Message string
	Header  string
}

// Marshal just calls the xml marshaller for a given Exception.
func Marshal(e Exception) ([]byte, error) {
	xmlstring, err := xml.Marshal(&errorXML{
		Xmlnsd:    "DAV",
		Xmlnss:    "http://sabredav.org/ns",
		Exception: codesEnum[e.Code],
		Message:   e.Message,
		Header:    e.Header,
	})
	if err != nil {
		return []byte(""), err
	}
	return []byte(xml.Header + string(xmlstring)), err
}

// Encode writes the given Exception as a SabreDAV error document, including the
// XML header, to w.
func Encode(w io.Writer, e Exception) error {
	return EncodeXML(w, &errorXML{
		Xmlnsd:    "DAV",
		Xmlnss:    "http://sabredav.org/ns",
		Exception: codesEnum[e.Code],
		Message:   e.Message,
		Header:    e.Header,
	})
}

// EncodeXML writes v as an XML document, including the XML header, to w.
func EncodeXML(w io.Writer, v any) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(v)
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_error
type errorXML struct {
	XMLName   xml.Name `xml:"d:error"`
	Xmlnsd    string   `xml:"xmlns:d,attr"`
	Xmlnss    string   `xml:"xmlns:s,attr"`
	Exception string   `xml:"s:Exception"`
	Message   string   `xml:"s:Message"`
	InnerXML  []byte   `xml:",innerxml"`
	Header    string   `xml:"s:Header,omitempty"`
}

func HandleWebdavError(w http.ResponseWriter, b []byte, err error) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(b)
}
