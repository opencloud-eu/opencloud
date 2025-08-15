package comfunc

import (
	"fmt"
	"strings"
)

// Cmdline build
func Cmdline(args []string, binName ...string) string {
	b := new(strings.Builder)

	if len(binName) > 0 {
		b.WriteString(binName[0])
		b.WriteByte(' ')
	}

	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}

		if strings.ContainsRune(a, '"') {
			b.WriteString(fmt.Sprintf(`'%s'`, a))
		} else if a == "" || strings.ContainsRune(a, '\'') || strings.ContainsRune(a, ' ') {
			b.WriteString(fmt.Sprintf(`"%s"`, a))
		} else {
			b.WriteString(a)
		}
	}
	return b.String()
}

// ShellQuote quote a string on contains ', ", SPACE. refer strconv.Quote()
func ShellQuote(a string) string {
	if a == "" {
		return `""`
	}

	// use quote char
	var quote byte

	// has double quote
	if pos := strings.IndexByte(a, '"'); pos > -1 {
		if !checkNeedQuote(a, pos, '"') {
			return a
		}

		quote = '\''
	} else if pos := strings.IndexByte(a, '\''); pos > -1 {
		// single quote
		if !checkNeedQuote(a, pos, '\'') {
			return a
		}
		quote = '"'
	} else if strings.IndexByte(a, ' ') > -1 {
		quote = '"'
	}

	// no quote char OR not need quote
	if quote == 0 {
		return a
	}
	return fmt.Sprintf("%c%s%c", quote, a, quote)
}

func checkNeedQuote(a string, pos int, char byte) bool {
	// end with char. eg: "
	lastIsQ := a[len(a)-1] == char

	// start with char. eg: "
	if pos == 0 {
		if lastIsQ {
			return false
		}

		if pos1 := strings.IndexByte(a[pos+1:], char); pos1 > -1 {
			// eg: `"one two" three four`
			lastS := a[pos1+pos+1:]
			if !strings.ContainsRune(lastS, ' ') {
				return false
			}
		}
	} else {
		startS := a[:pos]

		// eg: `--one="two three"`
		if lastIsQ && strings.IndexByte(startS, ' ') == -1 {
			return false
		}
	}

	return true
}
