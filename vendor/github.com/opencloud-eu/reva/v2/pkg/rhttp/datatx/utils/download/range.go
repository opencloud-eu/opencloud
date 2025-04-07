// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package download

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"strings"
)

// taken from https://golang.org/src/net/http/fs.go

// ErrInvalidRange is returned by serveContent's parseRange if the Range is
// malformed or invalid.
var ErrInvalidRange = errors.New("invalid range")

// ErrNoOverlap is returned by serveContent's parseRange if first-byte-pos of
// all of the byte-range-spec values is greater than the content size.
var ErrNoOverlap = fmt.Errorf("%w: failed to overlap", ErrInvalidRange)

// HTTPRange specifies the byte range to be sent to the client.
type HTTPRange struct {
	Start, Length int64
}

// ContentRange formats a Range header string as per RFC 7233.
func (r HTTPRange) ContentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.Start, r.Start+r.Length-1, size)
}

// MimeHeader creates range relevant MimeHeaders
func (r HTTPRange) MimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.ContentRange(size)},
		"Content-Type":  {contentType},
	}
}

// ParseRange parses a Range header string as per RFC 7233.
// errNoOverlap is returned if none of the ranges overlap.
func ParseRange(s string, size int64) ([]HTTPRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, ErrInvalidRange
	}
	ranges := []HTTPRange{}
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, ErrInvalidRange
		}
		start, end := textproto.TrimString(ra[:i]), textproto.TrimString(ra[i+1:])
		var r HTTPRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, ErrInvalidRange
			}
			if i > size {
				i = size
			}
			r.Start = size - i
			r.Length = size - r.Start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, ErrInvalidRange
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.Start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.Length = size - r.Start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.Start > i {
					return nil, ErrInvalidRange
				}
				if i >= size {
					i = size - 1
				}
				r.Length = i - r.Start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, ErrNoOverlap
	}
	return ranges, nil
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// RangesMIMESize returns the number of bytes it takes to encode the
// provided ranges as a multipart response.
func RangesMIMESize(ranges []HTTPRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		// CreatePart might return an error if the io.Copy for the boundaries fails
		// here parts are not filled, so we assume for now thet this will always succeed
		_, _ = mw.CreatePart(ra.MimeHeader(contentType, contentSize))
		encSize += ra.Length
	}
	_ = mw.Close()
	encSize += int64(w)
	return
}

// SumRangesSize adds up the length of all ranges
func SumRangesSize(ranges []HTTPRange) (size int64) {
	for _, ra := range ranges {
		size += ra.Length
	}
	return
}
