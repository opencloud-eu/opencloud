package ldap

import (
	"fmt"
	"log"
	"os"
	"strings"

	ber "github.com/go-asn1-ber/asn1-ber"
)

// LDAP Application Codes
const (
	ApplicationBindRequest           = 0
	ApplicationBindResponse          = 1
	ApplicationUnbindRequest         = 2
	ApplicationSearchRequest         = 3
	ApplicationSearchResultEntry     = 4
	ApplicationSearchResultDone      = 5
	ApplicationModifyRequest         = 6
	ApplicationModifyResponse        = 7
	ApplicationAddRequest            = 8
	ApplicationAddResponse           = 9
	ApplicationDelRequest            = 10
	ApplicationDelResponse           = 11
	ApplicationModifyDNRequest       = 12
	ApplicationModifyDNResponse      = 13
	ApplicationCompareRequest        = 14
	ApplicationCompareResponse       = 15
	ApplicationAbandonRequest        = 16
	ApplicationSearchResultReference = 19
	ApplicationExtendedRequest       = 23
	ApplicationExtendedResponse      = 24
	ApplicationIntermediateResponse  = 25
)

// ApplicationMap contains human readable descriptions of LDAP Application Codes
var ApplicationMap = map[uint8]string{
	ApplicationBindRequest:           "Bind Request",
	ApplicationBindResponse:          "Bind Response",
	ApplicationUnbindRequest:         "Unbind Request",
	ApplicationSearchRequest:         "Search Request",
	ApplicationSearchResultEntry:     "Search Result Entry",
	ApplicationSearchResultDone:      "Search Result Done",
	ApplicationModifyRequest:         "Modify Request",
	ApplicationModifyResponse:        "Modify Response",
	ApplicationAddRequest:            "Add Request",
	ApplicationAddResponse:           "Add Response",
	ApplicationDelRequest:            "Del Request",
	ApplicationDelResponse:           "Del Response",
	ApplicationModifyDNRequest:       "Modify DN Request",
	ApplicationModifyDNResponse:      "Modify DN Response",
	ApplicationCompareRequest:        "Compare Request",
	ApplicationCompareResponse:       "Compare Response",
	ApplicationAbandonRequest:        "Abandon Request",
	ApplicationSearchResultReference: "Search Result Reference",
	ApplicationExtendedRequest:       "Extended Request",
	ApplicationExtendedResponse:      "Extended Response",
	ApplicationIntermediateResponse:  "Intermediate Response",
}

// Ldap Behera Password Policy Draft 10 (https://tools.ietf.org/html/draft-behera-ldap-password-policy-10)
const (
	BeheraPasswordExpired             = 0
	BeheraAccountLocked               = 1
	BeheraChangeAfterReset            = 2
	BeheraPasswordModNotAllowed       = 3
	BeheraMustSupplyOldPassword       = 4
	BeheraInsufficientPasswordQuality = 5
	BeheraPasswordTooShort            = 6
	BeheraPasswordTooYoung            = 7
	BeheraPasswordInHistory           = 8
)

// BeheraPasswordPolicyErrorMap contains human readable descriptions of Behera Password Policy error codes
var BeheraPasswordPolicyErrorMap = map[int8]string{
	BeheraPasswordExpired:             "Password expired",
	BeheraAccountLocked:               "Account locked",
	BeheraChangeAfterReset:            "Password must be changed",
	BeheraPasswordModNotAllowed:       "Policy prevents password modification",
	BeheraMustSupplyOldPassword:       "Policy requires old password in order to change password",
	BeheraInsufficientPasswordQuality: "Password fails quality checks",
	BeheraPasswordTooShort:            "Password is too short for policy",
	BeheraPasswordTooYoung:            "Password has been changed too recently",
	BeheraPasswordInHistory:           "New password is in list of old passwords",
}

var logger = log.New(os.Stderr, "", log.LstdFlags)

// Logger allows clients to override the default logger
func Logger(l *log.Logger) {
	logger = l
}

// Adds descriptions to an LDAP Response packet for debugging
func addLDAPDescriptions(packet *ber.Packet) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewError(ErrorDebugging, fmt.Errorf("ldap: cannot process packet to add descriptions: %s", r))
		}
	}()
	packet.Description = "LDAP Response"
	messageID, err := packetChild(packet, 0)
	if err != nil {
		return err
	}
	messageID.Description = "Message ID"

	protocolOp, err := packetChild(packet, 1)
	if err != nil {
		return err
	}
	application := uint8(protocolOp.Tag)
	protocolOp.Description = ApplicationMap[application]

	switch application {
	case ApplicationBindRequest:
		err = addRequestDescriptions(packet)
	case ApplicationBindResponse:
		err = addDefaultLDAPResponseDescriptions(packet)
	case ApplicationUnbindRequest:
		err = addRequestDescriptions(packet)
	case ApplicationSearchRequest:
		err = addRequestDescriptions(packet)
	case ApplicationSearchResultEntry:
		objectName, err := packetChild(protocolOp, 0)
		if err != nil {
			return err
		}
		objectName.Description = "Object Name"
		attrs, err := packetChild(protocolOp, 1)
		if err != nil {
			return err
		}
		attrs.Description = "Attributes"
		for _, child := range attrs.Children {
			if child == nil {
				continue
			}
			child.Description = "Attribute"
			nameChild, err := packetChild(child, 0)
			if err != nil {
				return err
			}
			nameChild.Description = "Attribute Name"
			valuesChild, err := packetChild(child, 1)
			if err != nil {
				return err
			}
			valuesChild.Description = "Attribute Values"
			for _, grandchild := range valuesChild.Children {
				if grandchild == nil {
					continue
				}
				grandchild.Description = "Attribute Value"
			}
		}
		controls, ok, err := packetChildIfPresent(packet, 2)
		if err != nil {
			return err
		}
		if ok {
			if err = addControlDescriptions(controls); err != nil {
				return err
			}
		}
	case ApplicationSearchResultDone:
		err = addDefaultLDAPResponseDescriptions(packet)
	case ApplicationModifyRequest:
		err = addRequestDescriptions(packet)
	case ApplicationModifyResponse:
	case ApplicationAddRequest:
		err = addRequestDescriptions(packet)
	case ApplicationAddResponse:
	case ApplicationDelRequest:
		err = addRequestDescriptions(packet)
	case ApplicationDelResponse:
	case ApplicationModifyDNRequest:
		err = addRequestDescriptions(packet)
	case ApplicationModifyDNResponse:
	case ApplicationCompareRequest:
		err = addRequestDescriptions(packet)
	case ApplicationCompareResponse:
	case ApplicationAbandonRequest:
		err = addRequestDescriptions(packet)
	case ApplicationSearchResultReference:
	case ApplicationExtendedRequest:
		err = addRequestDescriptions(packet)
	case ApplicationExtendedResponse:
	}

	return err
}

func addControlDescriptions(packet *ber.Packet) error {
	packet.Description = "Controls"
	for _, child := range packet.Children {
		if child == nil {
			return fmt.Errorf("nil control packet found")
		}
		var value *ber.Packet
		controlType := ""
		child.Description = "Control"

		controlChildren, err := packetChildCount(child, 1, 3, "control packet")
		if err != nil {
			return err
		}

		typeChild, err := packetChild(child, 0)
		if err != nil {
			return err
		}
		controlType, err = packetString(typeChild)
		if err != nil {
			return err
		}
		typeChild.Description = "Control Type (" + ControlTypeMap[controlType] + ")"

		switch len(controlChildren) {
		case 2:
			second, err := packetChild(child, 1)
			if err != nil {
				return err
			}
			// Children[1] is criticality or value; identify it by its ASN.1 tag.
			if second.ClassType == ber.ClassUniversal && second.Tag == ber.TagBoolean {
				second.Description = "Criticality"
			} else {
				second.Description = "Control Value"
				value = second
			}

		case 3:
			// criticality and value present
			criticality, err := packetChild(child, 1)
			if err != nil {
				return err
			}
			criticality.Description = "Criticality"
			valueChild, err := packetChild(child, 2)
			if err != nil {
				return err
			}
			valueChild.Description = "Control Value"
			value = valueChild
		}

		if value == nil {
			continue
		}
		switch controlType {
		case ControlTypePaging:
			value.Description += " (Paging)"
			if value.Value != nil {
				data, err := packetData(value)
				if err != nil {
					return fmt.Errorf("failed to decode data bytes: %w", err)
				}
				valueChildren, err := ber.DecodePacketErr(data)
				if err != nil {
					return fmt.Errorf("failed to decode data bytes: %w", err)
				}
				value.Data.Truncate(0)
				value.Value = nil
				cookieChild, err := packetChild(valueChildren, 1)
				if err != nil {
					return err
				}
				cookie, err := packetData(cookieChild)
				if err != nil {
					return err
				}
				cookieChild.Value = cookie
				value.AppendChild(valueChildren)
			}
			real, err := packetChild(value, 0)
			if err != nil {
				return err
			}
			real.Description = "Real Search Control Value"
			sizeChild, err := packetChild(real, 0)
			if err != nil {
				return err
			}
			cookieChild, err := packetChild(real, 1)
			if err != nil {
				return err
			}
			sizeChild.Description = "Paging Size"
			cookieChild.Description = "Cookie"

		case ControlTypeBeheraPasswordPolicy:
			value.Description += " (Password Policy - Behera Draft)"
			if value.Value != nil {
				data, err := packetData(value)
				if err != nil {
					return fmt.Errorf("failed to decode data bytes: %w", err)
				}
				valueChildren, err := ber.DecodePacketErr(data)
				if err != nil {
					return fmt.Errorf("failed to decode data bytes: %w", err)
				}
				value.Data.Truncate(0)
				value.Value = nil
				value.AppendChild(valueChildren)
			}
			sequence, err := packetChild(value, 0)
			if err != nil {
				return err
			}
			for _, child := range sequence.Children {
				if child == nil {
					continue
				}
				switch child.Tag {
				case 0:
					// Warning
					warningPacket, err := packetChild(child, 0)
					if err != nil {
						return err
					}
					data, err := packetData(warningPacket)
					if err != nil {
						return fmt.Errorf("failed to decode data bytes: %w", err)
					}
					val, err := ber.ParseInt64(data)
					if err != nil {
						return fmt.Errorf("failed to decode data bytes: %w", err)
					}
					switch warningPacket.Tag {
					case 0:
						// timeBeforeExpiration
						value.Description += " (TimeBeforeExpiration)"
						warningPacket.Value = val
					case 1:
						// graceAuthNsRemaining
						value.Description += " (GraceAuthNsRemaining)"
						warningPacket.Value = val
					}
				case 1:
					// Error
					bs, err := packetData(child)
					if err != nil {
						return err
					}
					if len(bs) != 1 || bs[0] > 8 {
						return malformedf("invalid PasswordPolicyResponse enum value")
					}
					val := int8(bs[0])
					child.Description = "Error"
					child.Value = val
				}
			}
		}
	}
	return nil
}

func addRequestDescriptions(packet *ber.Packet) error {
	packet.Description = "LDAP Request"
	messageID, err := packetChild(packet, 0)
	if err != nil {
		return err
	}
	messageID.Description = "Message ID"
	protocolOp, err := packetChild(packet, 1)
	if err != nil {
		return err
	}
	protocolOp.Description = ApplicationMap[uint8(protocolOp.Tag)]
	controls, ok, err := packetChildIfPresent(packet, 2)
	if err != nil {
		return err
	}
	if ok {
		return addControlDescriptions(controls)
	}
	return nil
}

func addDefaultLDAPResponseDescriptions(packet *ber.Packet) error {
	resultCode := uint16(LDAPResultSuccess)
	matchedDN := ""
	description := "Success"
	if err := GetLDAPError(packet); err != nil {
		resultCode = err.(*Error).ResultCode
		matchedDN = err.(*Error).MatchedDN
		description = "Error Message"
	}

	protocolOp, err := packetChild(packet, 1)
	if err != nil {
		return err
	}
	resultCodeChild, err := packetChild(protocolOp, 0)
	if err != nil {
		return err
	}
	resultCodeChild.Description = "Result Code (" + LDAPResultCodeMap[resultCode] + ")"
	matchedDNChild, err := packetChild(protocolOp, 1)
	if err != nil {
		return err
	}
	matchedDNChild.Description = "Matched DN (" + matchedDN + ")"
	errorMessageChild, err := packetChild(protocolOp, 2)
	if err != nil {
		return err
	}
	errorMessageChild.Description = description
	referral, ok, err := packetChildIfPresent(protocolOp, 3)
	if err != nil {
		return err
	}
	if ok {
		referral.Description = "Referral"
	}
	controls, ok, err := packetChildIfPresent(packet, 2)
	if err != nil {
		return err
	}
	if ok {
		return addControlDescriptions(controls)
	}
	return nil
}

// DebugBinaryFile reads and prints packets from the given filename
func DebugBinaryFile(fileName string) error {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return NewError(ErrorDebugging, err)
	}
	ber.PrintBytes(os.Stdout, file, "")
	packet, err := ber.DecodePacketErr(file)
	if err != nil {
		return fmt.Errorf("failed to decode packet: %w", err)
	}
	if err := addLDAPDescriptions(packet); err != nil {
		return err
	}
	ber.PrintPacket(packet)

	return nil
}

func mustEscape(c byte) bool {
	return c > 0x7f || c == '(' || c == ')' || c == '\\' || c == '*' || c == 0
}

// EscapeFilter escapes from the provided LDAP filter string the special
// characters in the set `()*\` and those out of the range 0 < c < 0x80,
// as defined in RFC4515.
func EscapeFilter(filter string) string {
	const hexValues = "0123456789abcdef"
	escape := 0
	for i := 0; i < len(filter); i++ {
		if mustEscape(filter[i]) {
			escape++
		}
	}
	if escape == 0 {
		return filter
	}
	buf := make([]byte, len(filter)+escape*2)
	for i, j := 0, 0; i < len(filter); i++ {
		c := filter[i]
		if mustEscape(c) {
			buf[j+0] = '\\'
			buf[j+1] = hexValues[c>>4]
			buf[j+2] = hexValues[c&0xf]
			j += 3
		} else {
			buf[j] = c
			j++
		}
	}
	return string(buf)
}

// EscapeDN escapes distinguished names as described in RFC4514. Characters in the
// set `"+,;<>\` are escaped by prepending a backslash, which is also done for trailing
// spaces or a leading `#`. Null bytes are replaced with `\00`.
func EscapeDN(dn string) string {
	if dn == "" {
		return ""
	}

	builder := strings.Builder{}

	for i, r := range dn {
		// Escape leading and trailing spaces
		if (i == 0 || i == len(dn)-1) && r == ' ' {
			builder.WriteRune('\\')
			builder.WriteRune(r)
			continue
		}

		// Escape leading '#'
		if i == 0 && r == '#' {
			builder.WriteRune('\\')
			builder.WriteRune(r)
			continue
		}

		// Escape characters as defined in RFC4514
		switch r {
		case '"', '+', ',', ';', '<', '>', '\\':
			builder.WriteRune('\\')
			builder.WriteRune(r)
		case '\x00': // Null byte may not be escaped by a leading backslash
			builder.WriteString("\\00")
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
