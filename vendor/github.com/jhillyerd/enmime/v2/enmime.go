// Package enmime implements a MIME encoding and decoding library.  It's built on top of Go's
// included mime/multipart support where possible, but is geared towards parsing MIME encoded
// emails.
//
// # Overview
//
// The enmime API has two conceptual layers.  The lower layer is a tree of Part structs,
// representing each component of a decoded MIME message.  The upper layer, called an Envelope
// provides an intuitive way to interact with a MIME message.
//
// # Part Tree
//
// Calling ReadParts causes enmime to parse the body of a MIME message into a tree of Part objects,
// each of which is aware of its content type, filename and headers.  The content of a Part is
// available as a slice of bytes via the Content field.
//
// If the part was encoded in quoted-printable or base64, it is decoded prior to being placed in
// Content.  If the Part contains text in a character set other than utf-8, enmime will attempt to
// convert it to utf-8.
//
// To locate a particular Part, pass a custom PartMatcher function into the BreadthMatchFirst() or
// DepthMatchFirst() methods to search the Part tree.  BreadthMatchAll() and DepthMatchAll() will
// collect all Parts matching your criteria.
//
// # Envelope
//
// ReadEnvelope returns an Envelope struct.  Behind the scenes a Part tree is constructed, and then
// sorted into the correct fields of the Envelope.
//
// The Envelope contains both the plain text and HTML portions of the email.  If there was no plain
// text Part available, the HTML Part will be down-converted using the html2text library[1].  The
// root of the Part tree, as well as slices of the inline and attachment Parts are also available.
//
// # Headers
//
// Every MIME Part has its own headers, accessible via the Part.Header field.  The raw headers for
// an Envelope are available in Root.Header.  Envelope also provides helper methods to fetch
// headers: GetHeader(key) will return the RFC 2047 decoded value of the specified header.
// AddressList(key) will convert the specified address header into a slice of net/mail.Address
// values.
//
// # Errors
//
// enmime attempts to be tolerant of poorly encoded MIME messages. In situations where parsing is
// not possible, the ReadEnvelope and ReadParts functions will return a hard error.  If enmime is
// able to continue parsing the message, it will add an entry to the Errors slice on the relevant
// Part.  After parsing is complete, all Part errors will be appended to the Envelope Errors slice.
// The Error* constants can be used to identify a specific class of error.
//
// Please note that enmime parses messages into memory, so it is not likely to perform well with
// multi-gigabyte attachments.
//
// enmime is open source software released under the MIT License.  The latest version can be found
// at https://github.com/jhillyerd/enmime/v2
//
// [1]: https://github.com/inbucket/html2text
package enmime // import "github.com/jhillyerd/enmime/v2"
