// Package varexpr provides some commonly ENV var parse functions.
//
// parse env value, allow expressions:
//
//	${VAR_NAME}            Only var name
//	${VAR_NAME | default}  With default value, if value is empty.
//	${VAR_NAME | ?error}   With error on value is empty.
//
// Examples:
//
//	only key     - "${SHELL}"
//	with default - "${NotExist | defValue}"
//	multi key    - "${GOPATH}/${APP_ENV | prod}/dir"
package varexpr

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

const (
	// SepChar separator char split var name and default value
	SepChar  = "|"
	VarLeft  = "${" // default var left format chars
	VarRight = "}"  // default var right format chars

	mustPrefix = '?' // must prefix char
)

// ParseOptFn option func
type ParseOptFn func(o *ParseOpts)

// ParseOpts parse options for ParseValue
type ParseOpts struct {
	// Getter Env value provider func.
	Getter func(string) string
	// ParseFn custom parse expr func. expr like "${SHELL}" "${NotExist|defValue}"
	ParseFn func(string) (string, error)
	// Regexp custom expression regex.
	Regexp *regexp.Regexp
	// var format chars for expression.
	// default left="${", right="}"
	VarLeft, VarRight string
}

func (opt *ParseOpts) useDefaultRegex() {
	opt.Regexp = envRegex
	opt.VarLeft = VarLeft
	opt.VarRight = VarRight
}

// must add "?" - To ensure that there is no greedy match
var envRegex = regexp.MustCompile(`\${.+?}`)
var std = New()

// Parse parse ENV var value from input string, support default value.
//
// Format:
//
//	${var_name}            Only var name
//	${var_name | default}  With default value
//	${var_name | ?error}   With error on value is empty.
//
// see Parser.Parse
func Parse(val string) (string, error) {
	return std.Parse(val)
}

// SafeParse parse ENV var value from input string, support default value.
//
// see Parser.Parse
func SafeParse(val string) string {
	s, _ := std.Parse(val)
	return s
}

// ParseWith parse ENV var value from input string, support default value.
func ParseWith(val string, optFns ...ParseOptFn) (string, error) {
	return New(optFns...).Parse(val)
}

// Parser parse ENV var value from input string, support default value.
type Parser struct {
	ParseOpts
}

// New create a new Parser
func New(optFns ...ParseOptFn) *Parser {
	opts := &ParseOpts{Getter: os.Getenv}
	opts.useDefaultRegex()

	for _, fn := range optFns {
		fn(opts)
	}
	return &Parser{ParseOpts: *opts}
}

// Parse parse ENV var value from input string, support default value.
//
// Format:
//
//	${var_name}            Only var name
//	${var_name | default}  With default value
//	${var_name | ?error}   With error on value is empty.
//	${VAR_NAME1}/path/${VAR_NAME2}  Allow multi var name.
func (p *Parser) Parse(val string) (newVal string, err error) {
	if p.Regexp == nil {
		p.useDefaultRegex()
	}

	times := strings.Count(val, p.VarLeft)
	if times == 0 {
		return val, nil
	}

	// enhance: see https://github.com/gookit/goutil/issues/135
	if times == 1 && strings.HasPrefix(val, p.VarLeft) && strings.HasSuffix(val, p.VarRight) {
		return p.parseOne(val)
	}

	// parse expression
	newVal = p.Regexp.ReplaceAllStringFunc(val, func(s string) string {
		if err != nil {
			return s
		}
		s, err = p.parseOne(s)
		return s
	})
	return
}

// parse one node expression.
func (p *Parser) parseOne(eVar string) (val string, err error) {
	if p.ParseFn != nil {
		return p.ParseFn(eVar)
	}

	// like "${NotExist | defValue}". first remove "${" and "}", then split it
	ss := strings.SplitN(eVar[2:len(eVar)-1], SepChar, 2)
	var name, def string

	// with default value.
	if len(ss) == 2 {
		name, def = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
	} else {
		name = strings.TrimSpace(ss[0])
	}

	// get ENV value by name
	val = p.Getter(name)
	if val == "" && def != "" {
		// check def is "?error"
		if def[0] == mustPrefix {
			msg := "value is required for var: " + name
			if len(def) > 1 {
				msg = def[1:]
			}
			err = errors.New(msg)
		} else {
			val = def
		}
	}
	return
}
