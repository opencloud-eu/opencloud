// Package textutil provide some extensions text handle util functions.
package textutil

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
)

// ReplaceVars by regex replace given tpl vars.
//
// If a format is empty, will use {const defaultVarFormat}
func ReplaceVars(text string, vars map[string]any, format string) string {
	return NewVarReplacer(format).Replace(text, vars)
}

// RenderSMap by regex replacement given tpl vars.
//
// If a format is empty, will use {const defaultVarFormat}
func RenderSMap(text string, vars map[string]string, format string) string {
	return NewVarReplacer(format).RenderSimple(text, vars)
}

// IsMatchAll keywords in the give text string.
//
// TIP: can use ^ for exclude match.
func IsMatchAll(s string, keywords []string) bool {
	return strutil.SimpleMatch(s, keywords)
}

// ParseInlineINI parse config string to string-map. it's like INI format contents.
//
// Examples:
//
//	eg: "name=val0;shorts=i;required=true;desc=a message"
//	=>
//	{name: val0, shorts: i, required: true, desc: a message}
func ParseInlineINI(tagVal string, keys ...string) (mp maputil.SMap, err error) {
	ss := strutil.Split(tagVal, ";")
	ln := len(ss)
	if ln == 0 {
		return
	}

	mp = make(maputil.SMap, ln)
	for _, s := range ss {
		if !strings.ContainsRune(s, '=') {
			err = fmt.Errorf("parse inline config error: must match `KEY=VAL`")
			return
		}

		key, val := strutil.TrimCut(s, "=")
		if len(keys) > 0 && !arrutil.StringsHas(keys, key) {
			err = fmt.Errorf("parse inline config error: invalid key name %q", key)
			return
		}

		mp[key] = val
	}
	return
}

// ParseSimpleINI parse simple multiline config string to a string-map.
// Can use to parse simple INI or dotenv file contents.
//
// NOTE:
//
//   - it's like INI format contents.
//   - support comments line with: "#", ";", "//"
//   - support inline comments with: " #" eg: name=tom # a comments
//   - DON'T support submap parse.
func ParseSimpleINI(text string) (mp maputil.SMap, err error) {
	return comfunc.ParseEnvLines(text, comfunc.ParseEnvLineOption{})
}
