package ldap

import (
	"errors"
	"fmt"
	"strings"
)

var ErrEmptyPostalAddress = errors.New("ldap: postal address cannot be empty")

// PostalAddress represents an RFC 4517 Postal Address
// A postal address is a sequence of strings of one or more arbitrary UCS
// characters, which form the lines of the address.
type PostalAddress struct {
	lines []string
}

// NewPostalAddress creates a new PostalAddress by copying non-empty lines from the provided slice of strings.
func NewPostalAddress(lines []string) (*PostalAddress, error) {
	copiedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		copiedLines = append(copiedLines, line)
	}

	if len(copiedLines) == 0 {
		return nil, ErrEmptyPostalAddress
	}

	return &PostalAddress{lines: copiedLines}, nil
}

// Lines returns a copy of the address lines as a slice of strings.
func (p *PostalAddress) Lines() []string {
	copiedLines := make([]string, len(p.lines))
	copy(copiedLines, p.lines)
	return copiedLines
}

// String returns the postal address as a single string, with lines joined by newline characters.
func (p *PostalAddress) String() string {
	return strings.Join(p.lines, "\n")
}

// Escape encodes special characters in the PostalAddress lines as per RFC 4517 and appends a `$` at the end of each line.
func (p *PostalAddress) Escape() string {
	builder := &strings.Builder{}

	for _, line := range p.lines {
		for _, char := range line {
			switch char {
			case '\\':
				builder.WriteString("\\5C")
			case '$':
				builder.WriteString("\\24")
			default:
				builder.WriteRune(char)
			}
		}

		builder.WriteRune('$')
	}

	return builder.String()
}

// ParsePostalAddress parses an RFC 4517 escaped postal address string into a PostalAddress object or returns an error.
func ParsePostalAddress(escaped string) (*PostalAddress, error) {
	lines := strings.Split(escaped, "$")
	parsedLines := make([]string, 0, len(lines))
	const totalEscapeLen = 3

	for _, line := range lines {
		if line == "" {
			// Skip empty lines
			continue
		}

		builder := &strings.Builder{}
		for i := 0; i < len(line); i++ {
			char := line[i]
			if char == '\\' && i+totalEscapeLen <= len(line) {
				escapeSeq := line[i+1 : i+totalEscapeLen]
				switch escapeSeq {
				case "5C", "5c":
					builder.WriteRune('\\')
					i += 2
				case "24":
					builder.WriteRune('$')
					i += 2
				default:
					return nil, fmt.Errorf("invalid escape sequence: \\%s at position %d", escapeSeq, i)
				}
			} else if char == '\\' {
				return nil, fmt.Errorf("incomplete escape sequence at position %d", i)
			} else {
				builder.WriteByte(char)
			}
		}
		parsedLines = append(parsedLines, builder.String())
	}

	if len(parsedLines) == 0 {
		return nil, ErrEmptyPostalAddress
	}

	return &PostalAddress{lines: parsedLines}, nil
}

// Equal compares the current PostalAddress with another PostalAddress and returns true if they are identical.
func (p *PostalAddress) Equal(other *PostalAddress) bool {
	if p == other {
		return true
	}
	if p == nil || other == nil {
		return false
	}

	if len(p.lines) != len(other.lines) {
		return false
	}
	for i := range p.lines {
		if p.lines[i] != other.lines[i] {
			return false
		}
	}
	return true
}
