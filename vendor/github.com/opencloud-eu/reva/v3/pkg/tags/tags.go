// Copyright 2018-2022 CERN
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

package tags

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// character used to separate tags in lists
	_tagsep = ","
	// maximum number of tags
	_maxtags = 100
)

// Tags is a helper struct for merging, deleting and deduplicating the tags while preserving the order
type Tags struct {
	sep     string
	maxtags int

	t       []string
	exists  map[string]bool
	numtags int
}

// New creates a Tag struct from a slice of tags, e.g. ["tag1", "tag2"] or a list of tags, e.g. "tag1,tag2"
func New(ts ...string) *Tags {
	t := &Tags{sep: _tagsep, maxtags: _maxtags, exists: make(map[string]bool), t: make([]string, 0)}
	t.addTags(t.normalize(ts))
	return t
}

// Add appends a list of new tags and returns true if at least one was appended
func (t *Tags) Add(ts ...string) bool {
	return len(t.addTags(t.normalize(ts))) > 0
}

// AddValidated works like Add, but first passes the normalized tags to the
// given validator: if it returns an error, no tag is added and the error is
// returned, e.g. to reject a request that contains a tag that is too long.
func (t *Tags) AddValidated(validator func([]string) error, ts ...string) (bool, error) {
	tags := t.normalize(ts)
	if err := validator(tags); err != nil {
		return false, err
	}
	return len(t.addTags(tags)) > 0, nil
}

// MaxLengthValidator returns a validator for AddValidated that rejects tags
// that are longer than maxTagLength characters, counted as Unicode code points
// rather than bytes, and tags that are not valid UTF-8. The error lists all the
// offending tags at once. A maxTagLength of zero or less disables the length
// check, but invalid UTF-8 is still rejected.
func MaxLengthValidator(maxTagLength int) func([]string) error {
	return func(tags []string) error {
		var invalid, tooLong []string
		for _, tag := range tags {
			switch {
			case !utf8.ValidString(tag):
				invalid = append(invalid, fmt.Sprintf("%q", tag))
			case maxTagLength > 0 && utf8.RuneCountInString(tag) > maxTagLength:
				tooLong = append(tooLong, fmt.Sprintf("%q", tag))
			}
		}
		switch {
		case len(invalid) > 0:
			return fmt.Errorf("tags are not valid UTF-8: %s", strings.Join(invalid, ", "))
		case len(tooLong) > 0:
			return fmt.Errorf("tags are longer than %d characters: %s", maxTagLength, strings.Join(tooLong, ", "))
		}
		return nil
	}
}

// Remove removes a list of tags and returns true if at least one was removed
func (t *Tags) Remove(s ...string) bool {
	var removed bool

	for _, tag := range t.normalize(s) {
		if !t.exists[tag] {
			continue
		}

		for i, tt := range t.t {
			if tt == tag {
				t.t = append(t.t[:i], t.t[i+1:]...)
				break
			}
		}

		delete(t.exists, tag)
		t.numtags--
		removed = true
	}
	return removed
}

// AsList returns the tags converted to a list
func (t *Tags) AsList() string {
	return strings.Join(t.t, t.sep)
}

// AsSlice returns the tags as slice of strings
func (t *Tags) AsSlice() []string {
	return t.t
}

// addTags adds the given tags, which must already be normalized (see normalize),
// in front of the existing ones, and returns the tags that were actually added:
// tags that already exist are skipped, and so is everything beyond maxtags.
func (t *Tags) addTags(tags []string) []string {
	added := make([]string, 0, len(tags))
	for _, tag := range tags {
		if t.exists[tag] {
			// tag is already existing
			continue
		}

		if t.numtags >= t.maxtags {
			// max number of tags reached. We return silently without warning anyone
			break
		}

		added = append(added, tag)
		t.exists[tag] = true
		t.numtags++
	}

	t.t = append(added, t.t...)
	return added
}

// normalize turns the given arguments into a flat list of tags: every argument
// may itself be a list of tags joined by the separator, e.g. "tag1,tag2", and
// whitespace and invisible zero-width characters around each tag are removed
// (see isTrimmed), so that " tag1" and "tag1" are the same tag. Tags that are
// empty after that are dropped.
func (t *Tags) normalize(s []string) []string {
	tags := make([]string, 0, len(s))
	for _, list := range s {
		for _, tag := range strings.Split(list, t.sep) {
			if tag = strings.TrimFunc(tag, isTrimmed); tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

// isTrimmed reports whether r is removed from the start and the end of a tag:
// all Unicode whitespace, plus the invisible characters zero-width space, word
// joiner and zero-width no-break space (BOM). These aren't whitespace for
// Unicode, but a tag that starts or ends with them looks exactly like the tag
// without them. Zero-width joiner and non-joiner are kept, emoji sequences and
// some scripts need them.
func isTrimmed(r rune) bool {
	switch r {
	case '\u200b', '\u2060', '\ufeff':
		return true
	}
	return unicode.IsSpace(r)
}
