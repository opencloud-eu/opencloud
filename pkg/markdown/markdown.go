// Package markdown allows reading and editing Markdown files
package markdown

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Heading represents a markdown Heading
type Heading struct {
	Level   int    // the level of the heading. 1 means it's the H1
	Content string // the text of the heading
	Header  string // the heading itself
}

// MD represents a markdown file
type MD struct {
	Headings []Heading
}

// Bytes returns the markdown as []bytes, ignoring errors
func (md MD) Bytes() []byte {
	var b bytes.Buffer
	_, _ = md.WriteContent(&b)
	return b.Bytes()
}

// String returns the markdown as string, ignoring errors
func (md MD) String() string {
	var b strings.Builder
	_, _ = md.WriteContent(&b)
	return b.String()
}

// TocBytes returns the table of contents as []byte, ignoring errors
func (md MD) TocBytes() []byte {
	var b bytes.Buffer
	_, _ = md.WriteToc(&b)
	return b.Bytes()
}

// TocString returns the table of contents as string, ignoring errors
func (md MD) TocString() string {
	var b strings.Builder
	_, _ = md.WriteToc(&b)
	return b.String()
}

// WriteContent writes the MDs content to the given writer
func (md MD) WriteContent(w io.Writer) (int64, error) {
	written := int64(0)
	write := func(s string) error {
		n, err := w.Write([]byte(s))
		written += int64(n)
		return err
	}
	for _, h := range md.Headings {
		if err := write(strings.Repeat("#", h.Level) + " " + h.Header + "\n"); err != nil {
			return written, err
		}
		if len(h.Content) > 0 {
			if err := write(h.Content); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

// WriteToc writes the table of contents to the given writer
func (md MD) WriteToc(w io.Writer) (int64, error) {
	var written int64
	for _, h := range md.Headings {
		if h.Level == 1 {
			// main title not in toc
			continue
		}
		link := fmt.Sprintf("#%s", toAnchor(h.Header))
		s := fmt.Sprintf("%s* [%s](%s)\n", strings.Repeat("  ", h.Level-2), h.Header, link)
		n, err := w.Write([]byte(s))
		if err != nil {
			return written, err
		}
		written += int64(n)
	}
	return written, nil
}

// NewMD parses a new Markdown
func NewMD(b []byte) MD {
	var (
		md      MD
		heading Heading
		content strings.Builder
	)
	sendHeading := func() {
		if heading.Header != "" {
			heading.Content = content.String()
			md.Headings = append(md.Headings, heading)
			content = strings.Builder{}
		}
	}
	parts := strings.Split("\n"+string(b), "\n#")
	numparts := len(parts) - 1
	for i, p := range parts {
		if i == 0 {
			// omit part before first heading
			continue
		}

		all := strings.SplitN(p, "\n", 2)
		if len(all) != 2 {
			continue
		}

		head, con := all[0], all[1]
		// readd lost "#"
		heading = headingFromString("#" + head)
		_, _ = content.WriteString(con)
		// readd lost "\n" - omit for last part
		if i < numparts {
			_, _ = content.WriteString("\n")
		}
		// add heading
		sendHeading()
	}
	return md
}

func headingFromString(s string) Heading {
	i := strings.LastIndex(s, "#")
	levs, con := s[:i+1], s[i+1:]
	return Heading{
		Level:  len(levs),
		Header: strings.TrimPrefix(con, " "),
	}
}

func toAnchor(header string) string {
	// Remove everything except letters, numbers, and spaces
	reg := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
	anchor := reg.ReplaceAllString(header, "")
	// Replace spaces with hyphens and convert to lowercase
	anchor = strings.ReplaceAll(anchor, " ", "-")
	return strings.ToLower(anchor)
}
