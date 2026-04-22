package response

import (
	"encoding/xml"
	"net/http"
	"reflect"

	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/services/ocs/pkg/service/v0/data"
)

// Response is the top level response structure
type Response struct {
	OCS *Payload `json:"ocs" xml:"ocs"`
}

var (
	elementStartElement = xml.StartElement{Name: xml.Name{Local: "element"}}
	metaStartElement    = xml.StartElement{Name: xml.Name{Local: "meta"}}
	ocsName             = xml.Name{Local: "ocs"}
	dataName            = xml.Name{Local: "data"}
)

// Payload combines response metadata and data
type Payload struct {
	Meta data.Meta `json:"meta" xml:"meta"`
	Data any       `json:"data,omitempty" xml:"data,omitempty"`
}

// MarshalXML handles ocs specific wrapping of array members in 'element' tags for the data
func (rsp Response) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	// first the easy part
	// use ocs as the surrounding tag
	start.Name = ocsName
	if err = e.EncodeToken(start); err != nil {
		return
	}

	// encode the meta tag
	if err = e.EncodeElement(rsp.OCS.Meta, metaStartElement); err != nil {
		return
	}

	// we need to use reflection to determine if p.Data is an array or a slice
	rt := reflect.TypeOf(rsp.OCS.Data)
	if rt != nil && (rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice) {
		// this is how to wrap the data elements in their own <element> tag
		v := reflect.ValueOf(rsp.OCS.Data)
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
	} else if err = e.EncodeElement(rsp.OCS.Data, xml.StartElement{Name: dataName}); err != nil {
		return
	}

	// write the closing <ocs> tag
	if err = e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return
	}
	return
}

// Render sets the status code of the http response, taking the ocs version into account
func (rsp *Response) Render(w http.ResponseWriter, r *http.Request) error {
	version := APIVersion(r.Context())
	m := statusCodeMapper(version)
	statusCode := m(rsp.OCS.Meta)
	render.Status(r, statusCode)
	if version == ocsVersion2 && statusCode == http.StatusOK {
		rsp.OCS.Meta.StatusCode = statusCode
	}
	return nil
}

// DataRender creates an OK Payload for the given data
func DataRender(d any) render.Renderer {
	return &Response{
		&Payload{
			Meta: data.MetaOK,
			Data: d,
		},
	}
}

// ErrRender creates an Error Payload with the given OCS error code and message
// The httpcode will be determined using the API version stored in the context
func ErrRender(c int, m string) render.Renderer {
	return &Response{
		&Payload{
			Meta: data.Meta{Status: "error", StatusCode: c, Message: m},
		},
	}
}

func statusCodeMapper(version string) func(data.Meta) int {
	var mapper func(data.Meta) int
	switch version {
	case ocsVersion1:
		mapper = OcsV1StatusCodes
	case ocsVersion2:
		mapper = OcsV2StatusCodes
	default:
		mapper = defaultStatusCodeMapper
	}
	return mapper
}
