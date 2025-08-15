package comfunc

import (
	"fmt"
	"strings"
)

var commentsPrefixes = []string{"#", ";", "//"}

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
//   - Support inline comments split with: " #" eg: name=tom # a comments
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

		// invalid line
		if strings.IndexByte(line, '=') < 1 {
			if opt.SkipOnErrorLine {
				continue
			}

			strMap = nil
			err = fmt.Errorf("invalid line contents: must match `KEY=VAL`(line: %s)", line)
			return
		}

		key, value := SplitLineToKv(line, "=")

		// check and remove inline comments
		if !opt.NotInlineComments {
			if pos := strings.Index(value, " #"); pos > 0 {
				value = strings.TrimRight(value[0:pos], " \t")
			}
		}

		strMap[key] = value
	}

	return strMap, nil
}

// SplitLineToKv parse string line to k-v. eg:
//
//	'DEBUG=true' => ['DEBUG', 'true']
//
// NOTE: line must contain '=', allow: 'ENV_KEY='
func SplitLineToKv(line, sep string) (string, string) {
	nodes := strings.SplitN(line, sep, 2)
	envKey := strings.TrimSpace(nodes[0])

	// key cannot be empty
	if envKey == "" {
		return "", ""
	}

	if len(nodes) < 2 {
		if strings.Contains(line, sep) {
			return envKey, ""
		}
		return "", ""
	}
	return envKey, strings.TrimSpace(nodes[1])
}
