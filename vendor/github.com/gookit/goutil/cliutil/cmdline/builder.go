package cmdline

import (
	"strings"

	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/strutil"
)

// LineBuilder build command line string.
// codes refer from strings.Builder
type LineBuilder struct {
	strings.Builder
}

// NewBuilder create
func NewBuilder(binFile string, args ...string) *LineBuilder {
	b := &LineBuilder{}

	if binFile != "" {
		b.AddArg(binFile)
	}

	b.AddArray(args)
	return b
}

// ResetGet value, will reset after get.
func (b *LineBuilder) ResetGet() string {
	defer b.Reset()
	return b.String()
}

// AddArg to builder
func (b *LineBuilder) AddArg(arg string) {
	_, _ = b.WriteString(arg)
}

// AddArgs to builder
func (b *LineBuilder) AddArgs(args ...string) {
	b.AddArray(args)
}

// AddArray to builder
func (b *LineBuilder) AddArray(args []string) {
	for _, arg := range args {
		_, _ = b.WriteString(arg)
	}
}

// AddAny args to builder
func (b *LineBuilder) AddAny(args ...any) {
	for _, arg := range args {
		_, _ = b.WriteString(strutil.SafeString(arg))
	}
}

// WriteString arg string to the builder, will auto quote special string.
// refer strconv.Quote()
func (b *LineBuilder) WriteString(a string) (int, error) {
	// add sep on not-first write.
	if b.Len() != 0 {
		_ = b.WriteByte(' ')
	}

	return b.Builder.WriteString(comfunc.ShellQuote(a))
}
