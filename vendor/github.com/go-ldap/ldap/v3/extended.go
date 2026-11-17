package ldap

import (
	"errors"
	"fmt"

	ber "github.com/go-asn1-ber/asn1-ber"
)

// ExtendedRequest represents an extended request to send to the server
// See: https://www.rfc-editor.org/rfc/rfc4511#section-4.12
type ExtendedRequest struct {
	// ExtendedRequest ::= [APPLICATION 23] SEQUENCE {
	// 	requestName      [0] LDAPOID,
	// 	requestValue     [1] OCTET STRING OPTIONAL }

	Name     string
	Value    *ber.Packet
	Controls []Control
}

// NewExtendedRequest returns a new ExtendedRequest. The value can be
// nil depending on the type of request
func NewExtendedRequest(name string, value *ber.Packet) *ExtendedRequest {
	return &ExtendedRequest{
		Name:  name,
		Value: value,
	}
}

func (er ExtendedRequest) appendTo(envelope *ber.Packet) error {
	pkt := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ApplicationExtendedRequest, nil, "Extended Request")
	pkt.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, ber.TagEOC, er.Name, "Extended Request Name"))
	if er.Value != nil {
		pkt.AppendChild(encodeExtendedRequestValue(er.Value))
	}
	envelope.AppendChild(pkt)
	if len(er.Controls) > 0 {
		envelope.AppendChild(encodeControls(er.Controls))
	}
	return nil
}

// encodeExtendedRequestValue wraps the caller payload as RFC 4511 requestValue
// [1] OCTET STRING. Callers that already built that field (context class, tag 1)
// are left unchanged so Password Modify-style packets are not double-wrapped.
func encodeExtendedRequestValue(value *ber.Packet) *ber.Packet {
	if value.ClassType == ber.ClassContext && value.Tag == 1 {
		return value
	}
	return ber.NewString(ber.ClassContext, ber.TypePrimitive, 1, string(value.Bytes()), "Extended Request Value")
}

// ExtendedResponse represents the response from the directory server
// after sending an extended request
// See: https://www.rfc-editor.org/rfc/rfc4511#section-4.12
type ExtendedResponse struct {
	// ExtendedResponse ::= [APPLICATION 24] SEQUENCE {
	//   COMPONENTS OF LDAPResult,
	//   responseName     [10] LDAPOID OPTIONAL,
	//   responseValue    [11] OCTET STRING OPTIONAL }

	Name     string
	Value    *ber.Packet
	Controls []Control
}

// Extended performs an extended request. The resulting
// ExtendedResponse may return a value in the form of a *ber.Packet
func (l *Conn) Extended(er *ExtendedRequest) (*ExtendedResponse, error) {
	if er == nil {
		return nil, NewError(ErrorNetwork, errors.New("ExtendedRequest cannot be nil"))
	}

	msgCtx, err := l.doRequest(er)
	if err != nil {
		return nil, err
	}
	defer l.finishMessage(msgCtx)

	packet, err := l.readPacket(msgCtx)
	if err != nil {
		return nil, err
	}

	return decodeExtendedResponse(packet)
}

// decodeExtendedResponse decodes an extended response envelope, including any
// response controls. It is separated from the network loop so the full
// response decoder can be fuzzed directly.
func decodeExtendedResponse(packet *ber.Packet) (*ExtendedResponse, error) {
	if err := GetLDAPError(packet); err != nil {
		return nil, err
	}

	extResp, err := packetChild(packet, 1)
	if err != nil {
		return nil, err
	}
	if _, err := packetChildCount(extResp, 3, -1, "extended response"); err != nil {
		return nil, err
	}

	response := &ExtendedResponse{
		Controls: make([]Control, 0),
	}

	for _, child := range extResp.Children {
		// responseName [10] and responseValue [11] are context-class and
		// optional. The preceding resultCode is a universal ENUMERATED whose
		// tag number (10) is the same as responseName, so a child must be
		// matched on its class as well, otherwise the resultCode is read as
		// the responseName whenever the server omits the latter.
		if child.ClassType != ber.ClassContext {
			continue
		}
		switch child.Tag {
		case ber.TagEnumerated:
			name, err := packetData(child)
			if err != nil {
				return nil, err
			}
			response.Name = string(name)
		case ber.TagEmbeddedPDV:
			response.Value = child
		}
	}

	controlsChild, ok, err := packetChildIfPresent(packet, 2)
	if err != nil {
		return nil, err
	}
	if ok {
		for _, child := range controlsChild.Children {
			decodedChild, decodeErr := DecodeControl(child)
			if decodeErr != nil {
				return nil, fmt.Errorf("failed to decode child control: %w", decodeErr)
			}
			response.Controls = append(response.Controls, decodedChild)
		}
	}

	return response, nil
}
