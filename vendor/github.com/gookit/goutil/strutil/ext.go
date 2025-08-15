package strutil

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/mathutil"
)

// SimilarComparator definition
//
// links:
//
//	https://github.com/mkideal/cli/blob/master/fuzzy.go
type SimilarComparator struct {
	src, dst string
}

// NewComparator create
func NewComparator(src, dst string) *SimilarComparator {
	return &SimilarComparator{src, dst}
}

// Similarity calc for two string.
//
// Usage:
//
//	rate, ok := Similarity("hello", "he")
func Similarity(s, t string, rate float32) (float32, bool) {
	return NewComparator(s, t).Similar(rate)
}

// Similar by minDifferRate
//
// Usage:
//
//	c := NewComparator("hello", "he")
//	rate, ok :c.Similar(0.3)
func (c *SimilarComparator) Similar(minDifferRate float32) (float32, bool) {
	dist := c.editDistance([]byte(c.src), []byte(c.dst))
	differRate := dist / float32(mathutil.Max(len(c.src), len(c.dst))+4)

	return differRate, differRate >= minDifferRate
}

func (c *SimilarComparator) editDistance(s, t []byte) float32 {
	var (
		m = len(s)
		n = len(t)
		d = make([][]float32, m+1)
	)
	for i := 0; i < m+1; i++ {
		d[i] = make([]float32, n+1)
		d[i][0] = float32(i)
	}
	for j := 0; j < n+1; j++ {
		d[0][j] = float32(j)
	}

	for j := 1; j < n+1; j++ {
		for i := 1; i < m+1; i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				d[i][j] = mathutil.Min(d[i-1][j]+1, mathutil.Min(d[i][j-1]+1, d[i-1][j-1]+1))
			}
		}
	}

	return d[m][n]
}

// Builder struct
type Builder struct {
	strings.Builder
}

// Write bytes and no error report
func (b *Builder) Write(p []byte) {
	_, _ = b.Builder.Write(p)
}

// WriteRune and no error report
func (b *Builder) WriteRune(r rune) {
	_, _ = b.Builder.WriteRune(r)
}

// WriteByteNE write byte and no error report
func (b *Builder) WriteByteNE(c byte) {
	_ = b.WriteByte(c)
}

// WriteString to builder
func (b *Builder) WriteString(s string) {
	_, _ = b.Builder.WriteString(s)
}

// Writef write string by fmt.Sprintf formatted
func (b *Builder) Writef(tpl string, vs ...any) {
	_, _ = b.Builder.WriteString(fmt.Sprintf(tpl, vs...))
}

// Writeln write string with newline.
func (b *Builder) Writeln(s string) {
	_, _ = b.Builder.WriteString(s)
	_ = b.WriteByte('\n')
}

// WriteAny write any type value.
func (b *Builder) WriteAny(v any) {
	_, _ = b.Builder.WriteString(QuietString(v))
}

// WriteAnys write any type values.
func (b *Builder) WriteAnys(vs ...any) {
	for _, v := range vs {
		_, _ = b.Builder.WriteString(QuietString(v))
	}
}

// WriteMulti write multi byte at once.
func (b *Builder) WriteMulti(bs ...byte) {
	for _, b2 := range bs {
		_ = b.WriteByte(b2)
	}
}

// WriteStrings write multi string at once.
func (b *Builder) WriteStrings(ss ...string) {
	for _, s := range ss {
		_, _ = b.Builder.WriteString(s)
	}
}

// ResetGet return current string and reset builder
func (b *Builder) ResetGet() string {
	s := b.String()
	b.Reset()
	return s
}
