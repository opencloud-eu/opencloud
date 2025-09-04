package stringutil

import (
	"bytes"
	"net/mail"
	"strings"
)

// JoinAddress formats a slice of Address structs such that they can be used in a To or Cc header.
func JoinAddress(addrs []mail.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	buf := &bytes.Buffer{}
	for i, a := range addrs {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString(a.String())
	}
	return buf.String()
}

// EnsureCommaDelimitedAddresses is used by AddressList to ensure that address lists are properly
// delimited.
func EnsureCommaDelimitedAddresses(s string) string {
	// This normalizes the whitespace, but may interfere with CFWS (comments with folding whitespace)
	// RFC-5322 3.4.0:
	//      because some legacy implementations interpret the comment,
	//      comments generally SHOULD NOT be used in address fields
	//      to avoid confusing such implementations.
	s = strings.Join(strings.Fields(s), " ")

	inQuotes := false
	inDomain := false
	escapeSequence := false
	inAngles := false
	sb := strings.Builder{}
	for i, r := range s {
		if escapeSequence {
			escapeSequence = false
			sb.WriteRune(r)
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			sb.WriteRune(r)
			continue
		}

		if r == '<' {
			inAngles = true
			sb.WriteRune(r)
			continue
		}

		if r == '>' {
			inAngles = false
			sb.WriteRune(r)
			continue
		}

		if inQuotes {
			if r == '\\' {
				escapeSequence = true
				sb.WriteRune(r)
				continue
			}
		} else {
			if r == '@' {
				inDomain = true
				sb.WriteRune(r)
				continue
			}
			if inDomain {
				if r == ';' {
					inDomain = false
					if i == len(s)-1 {
						// omit trailing semicolon
						continue
					}

					sb.WriteRune(',')
					continue
				}
				if r == ',' {
					inDomain = false
					sb.WriteRune(r)
					continue
				}
				if r == ' ' && !inAngles {
					inDomain = false
					sb.WriteRune(',')
					sb.WriteRune(r)
					continue
				}
			}

			if inAngles && r == ' ' {
				continue
			}
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
