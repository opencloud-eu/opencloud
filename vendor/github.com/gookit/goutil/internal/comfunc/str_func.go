package comfunc

import (
	"fmt"
	"strings"
)

// ParseEnvLineOption parse env line options
type ParseEnvLineOption struct {
	// NotInlineComments dont parse inline comments.
	//  - default: false. will parse inline comments
	NotInlineComments bool
	// SkipOnErrorLine skip error line, continue parse next line
	//  - False: return error, clear parsed map
	SkipOnErrorLine bool
}

// ParseEnvLines parse simple multiline k-v string to a string-map.
// Can use to parse simple INI or DOTENV file contents.
//
// NOTE:
//
//   - It's like INI/ENV format contents.
//   - Support comments line starts with: "#", ";", "//"
//   - Support inline comments split with: " #" eg: "name=tom # a comments"
//   - DON'T support submap parse.
func ParseEnvLines(text string, opt ParseEnvLineOption) (mp map[string]string, err error) {
	lines := strings.Split(text, "\n")
	ln := len(lines)
	if ln == 0 {
		return
	}

	strMap := make(map[string]string, ln)

	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		// skip comments line
		if line[0] == '#' || line[0] == ';' || strings.HasPrefix(line, "//") {
			continue
		}

		key, val := splitLineByChar(line, '=', !opt.NotInlineComments)
		// invalid line
		if key == "" {
			if opt.SkipOnErrorLine {
				continue
			}
			strMap = nil
			err = fmt.Errorf("invalid line contents: must match `KEY=VAL`(line: %s)", line)
			return
		}

		strMap[key] = val
	}

	return strMap, nil
}

// SplitLineToKv parse string line to k-v, not support comments.
//
// Example:
//
//	'DEBUG=true' => ['DEBUG', 'true']
//
// NOTE: line must contain '=', allow: 'ENV_KEY='
func SplitLineToKv(line, sep string) (string, string) {
	return SplitKvBySep(line, sep, false)
}

// SplitKvBySep parse string line to k-v, support parse comments.
//   - rmInlineComments: check and remove inline comments by ' #'
func SplitKvBySep(line, sep string, rmInlineComments bool) (key, val string) {
	sepPos := strings.Index(line, sep)
	if sepPos < 0 {
		return
	}

	return splitKvBySepPos(line, sepPos, len(sep), rmInlineComments)
}

func splitLineByChar(line string, sep byte, rmInlineComments bool) (key, val string) {
	sepPos := strings.IndexByte(line, sep)
	if sepPos < 0 {
		return
	}

	return splitKvBySepPos(line, sepPos, 1, rmInlineComments)
}

func splitKvBySepPos(line string, sepPos, sepLen int, rmInlineComments bool) (key, val string) {
	// key cannot be empty
	key = strings.TrimSpace(line[0:sepPos])
	if key == "" {
		return "", ""
	}
	val = strings.TrimSpace(line[sepPos+sepLen:])

	// check quotes if present
	if vln := len(val); vln >= 2 {
		// remove quotes
		if (val[0] == '"' && val[vln-1] == '"') || (val[0] == '\'' && val[vln-1] == '\'') {
			val = val[1 : vln-1]
			return
		}

		if !rmInlineComments {
			return
		}

		// value is empty, only inline comments
		if val[0] == '#' {
			val = ""
			return
		}

		// remove inline comments
		if pos := strings.Index(val, " #"); pos > 0 {
			val = strings.TrimRight(val[0:pos], " \t")
			vln = len(val)
			// remove quotes
			if (val[0] == '"' && val[vln-1] == '"') || (val[0] == '\'' && val[vln-1] == '\'') {
				val = val[1 : vln-1]
				return
			}
		}
	}

	return
}
