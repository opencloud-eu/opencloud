package enmime

import (
	"strings"

	inttp "github.com/jhillyerd/enmime/v2/internal/textproto"
)

// detectMultipartMessage returns true if the message has a recognized multipart Content-Type header
func detectMultipartMessage(root *Part, multipartWOBoundaryAsSinglepart bool) bool {
	// Parse top-level multipart
	ctype := root.Header.Get(hnContentType)
	mtype, params, _, err := root.parseMediaType(ctype)
	if err != nil {
		return false
	}

	if boundary := params[hpBoundary]; multipartWOBoundaryAsSinglepart && boundary == "" {
		return false
	}

	// According to rfc2046#section-5.1.7 all other multipart should
	// be treated as multipart/mixed
	return strings.HasPrefix(mtype, ctMultipartPrefix)
}

// detectAttachmentHeader returns true, if the given header defines an attachment. First it checks
// if the Content-Disposition header defines either an attachment part or an inline part with at
// least one parameter. If this test is false, the Content-Type header is checked for attachment,
// but not inline. Email clients use inline for their text bodies.
//
// Valid Attachment-Headers:
//
//   - Content-Disposition: attachment; filename="frog.jpg"
//   - Content-Disposition: inline; filename="frog.jpg"
//   - Content-Type: attachment; filename="frog.jpg"
func detectAttachmentHeader(root *Part, header inttp.MIMEHeader) bool {
	mtype, params, _, _ := root.parseMediaType(header.Get(hnContentDisposition))
	if strings.ToLower(mtype) == cdAttachment ||
		(strings.ToLower(mtype) == cdInline && len(params) > 0) {
		return true
	}

	mtype, _, _, _ = root.parseMediaType(header.Get(hnContentType))
	return strings.ToLower(mtype) == cdAttachment
}

// detectTextHeader returns true, if the the MIME headers define a valid 'text/plain' or 'text/html'
// part.  If the emptyContentTypeIsPlain argument is set to true, a missing Content-Type header will
// result in a positive plain part detection.
func detectTextHeader(root *Part, header inttp.MIMEHeader, emptyContentTypeIsText bool) bool {
	ctype := header.Get(hnContentType)
	if ctype == "" && emptyContentTypeIsText {
		return true
	}

	if mtype, _, _, err := root.parseMediaType(ctype); err == nil {
		switch mtype {
		case ctTextPlain, ctTextHTML:
			return true
		}
	}

	return false
}

// detectBinaryBody returns true if the mail header defines a binary body.
func detectBinaryBody(root *Part) bool {
	header := inttp.MIMEHeader(root.Header) // Use internal header methods.
	if detectTextHeader(root, header, true) {
		// It is text/plain, but an attachment.
		// Content-Type: text/plain; name="test.csv"
		// Content-Disposition: attachment; filename="test.csv"
		// Check for attachment only, or inline body is marked
		// as attachment, too.
		mtype, _, _, _ := root.parseMediaType(header.Get(hnContentDisposition))
		return strings.ToLower(mtype) == cdAttachment
	}

	isBin := detectAttachmentHeader(root, header)
	if !isBin {
		// This must be an attachment, if the Content-Type is not
		// 'text/plain' or 'text/html'.
		// Example:
		// Content-Type: application/pdf; name="doc.pdf"
		mtype, _, _, _ := root.parseMediaType(header.Get(hnContentType))
		mtype = strings.ToLower(mtype)
		if mtype != ctTextPlain && mtype != ctTextHTML {
			return true
		}
	}

	return isBin
}
