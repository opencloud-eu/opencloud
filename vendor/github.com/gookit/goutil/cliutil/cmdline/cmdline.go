// Package cmdline provide quick build and parse cmd line string.
package cmdline

import "github.com/gookit/goutil/internal/comfunc"

// LineBuild build command line string by given args.
func LineBuild(binFile string, args []string) string {
	return NewBuilder(binFile, args...).String()
}

// ParseLine input command line text. alias of the StringToOSArgs()
func ParseLine(line string) []string { return NewParser(line).Parse() }

// Quote string in shell command env
func Quote(s string) string { return comfunc.ShellQuote(s) }
