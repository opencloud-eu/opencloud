package enmime

import (
	"fmt"
)

const (
	// ErrorMalformedBase64 name.
	ErrorMalformedBase64 = "Malformed Base64"
	// ErrorMalformedHeader name.
	ErrorMalformedHeader = "Malformed Header"
	// ErrorMissingBoundary name.
	ErrorMissingBoundary = "Missing Boundary"
	// ErrorMissingContentType name.
	ErrorMissingContentType = "Missing Content-Type"
	// ErrorCharsetConversion name.
	ErrorCharsetConversion = "Character Set Conversion"
	// ErrorContentEncoding name.
	ErrorContentEncoding = "Content Encoding"
	// ErrorPlainTextFromHTML name.
	ErrorPlainTextFromHTML = "Plain Text from HTML"
	// ErrorCharsetDeclaration name.
	ErrorCharsetDeclaration = "Character Set Declaration Mismatch"
	// ErrorMissingRecipient name.
	ErrorMissingRecipient = "no recipients (to, cc, bcc) set"
	// ErrorMalformedChildPart name.
	ErrorMalformedChildPart = "Malformed child part"
)

// Error describes an error encountered while parsing.
type Error struct {
	Name   string // The name or type of error encountered, from Error consts.
	Detail string // Additional detail about the cause of the error, if available.
	Severe bool   // Indicates that a portion of the message was lost during parsing.
}

// Error formats the enmime.Error as a string.
func (e *Error) Error() string {
	sev := "W"
	if e.Severe {
		sev = "E"
	}
	return fmt.Sprintf("[%s] %s: %s", sev, e.Name, e.Detail)
}

// addError builds a severe Error and appends to the Part error slice.
func (p *Part) addError(name string, detail string) {
	p.addProblem(&Error{name, detail, true})
}

// addErrorf builds a severe Error and appends to the Part error slice.
func (p *Part) addErrorf(name string, detailFmt string, args ...interface{}) {
	p.addProblem(&Error{
		name,
		fmt.Sprintf(detailFmt, args...),
		true,
	})
}

// addWarning builds a non-severe Error and appends to the Part error slice.
func (p *Part) addWarning(name string, detail string) {
	p.addProblem(&Error{name, detail, false})
}

// addWarningf builds a non-severe Error and appends to the Part error slice.
func (p *Part) addWarningf(name string, detailFmt string, args ...interface{}) {
	p.addProblem(&Error{
		name,
		fmt.Sprintf(detailFmt, args...),
		false,
	})
}

// addProblem adds general *Error to the Part error slice.
func (p *Part) addProblem(err *Error) {
	maxErrors := 0
	if p.parser != nil {
		// Override global var.
		maxErrors = p.parser.maxStoredPartErrors
	}

	if (maxErrors == 0) || (len(p.Errors) < maxErrors) {
		p.Errors = append(p.Errors, err)
	}
}

// ErrorCollector is an interface for collecting errors and warnings during
// parsing.
type ErrorCollector interface {
	AddError(name string, detailFmt string, args ...any)
	AddWarning(name string, detailFmt string, args ...any)
}

type partErrorCollector struct {
	part *Part
}

func (p *partErrorCollector) AddError(name string, detailFmt string, args ...any) {
	p.part.addErrorf(name, detailFmt, args...)
}

func (p *partErrorCollector) AddWarning(name string, detailFmt string, args ...any) {
	p.part.addWarningf(name, detailFmt, args...)
}
