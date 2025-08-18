package ccolor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gookit/goutil/x/termenv"
)

// output colored text like uses custom tag.
const (
	// MatchExpr regex to match color tags
	//
	// Notice: golang 不支持反向引用. 即不支持使用 \1 引用第一个匹配 ([a-z=;]+)
	// MatchExpr = `<([a-z=;]+)>(.*?)<\/\1>`
	// 所以调整一下 统一使用 `</>` 来结束标签，例如 "<info>some text</>"
	//
	// - NOTE: 不支持自定义属性。如有需要请使用 gookit/color 包
	//
	// (?s:...) s - 让 "." 匹配换行
	MatchExpr = `<([0-9a-zA-Z_]+)>(?s:(.*?))<\/>`

	// StripExpr regex used for removing color tags
	// StripExpr = `<[\/]?[a-zA-Z=;]+>`
	// 随着上面的做一些调整
	StripExpr = `<[\/]?[0-9a-zA-Z_=,;]*>`
)

var (
	matchRegex = regexp.MustCompile(MatchExpr)
	stripRegex = regexp.MustCompile(StripExpr)
)

/*************************************************************
 * internal defined color tags
 *************************************************************/

// There are internal defined fg color tags
//
// Usage:
//
//	<tag>content text</>
//
// @notice 加 0 在前面是为了防止之前的影响到现在的设置
var colorTags = map[string]string{
	// basic tags
	"red":      "0;31",
	"red1":     "1;31", // with bold
	"redB":     "1;31",
	"red_b":    "1;31",
	"blue":     "0;34",
	"blue1":    "1;34", // with bold
	"blueB":    "1;34",
	"blue_b":   "1;34",
	"cyan":     "0;36",
	"cyan1":    "1;36", // with bold
	"cyanB":    "1;36",
	"cyan_b":   "1;36",
	"green":    "0;32",
	"green1":   "1;32", // with bold
	"greenB":   "1;32",
	"green_b":  "1;32",
	"black":    "0;30",
	"white":    "1;37",
	"default":  "0;39", // no color
	"normal":   "0;39", // no color
	"brown":    "0;33", // #A52A2A
	"yellow":   "0;33",
	"ylw0":     "0;33",
	"yellowB":  "1;33", // with bold
	"ylw1":     "1;33",
	"ylwB":     "1;33",
	"magenta":  "0;35",
	"mga":      "0;35", // short name
	"magentaB": "1;35", // with bold
	"magenta1": "1;35",
	"mgb":      "1;35",
	"mga1":     "1;35",
	"mgaB":     "1;35",

	// light/hi tags

	"gray":          "0;90",
	"darkGray":      "0;90",
	"dark_gray":     "0;90",
	"lightYellow":   "0;93",
	"light_yellow":  "0;93",
	"hiYellow":      "0;93",
	"hi_yellow":     "0;93",
	"hiYellowB":     "1;93", // with bold
	"hi_yellow_b":   "1;93",
	"lightMagenta":  "0;95",
	"light_magenta": "0;95",
	"hiMagenta":     "0;95",
	"hi_magenta":    "0;95",
	"lightMagenta1": "1;95", // with bold
	"hiMagentaB":    "1;95", // with bold
	"hi_magenta_b":  "1;95",
	"lightRed":      "0;91",
	"light_red":     "0;91",
	"hiRed":         "0;91",
	"hi_red":        "0;91",
	"lightRedB":     "1;91", // with bold
	"light_red_b":   "1;91",
	"hi_red_b":      "1;91",
	"lightGreen":    "0;92",
	"light_green":   "0;92",
	"hiGreen":       "0;92",
	"hi_green":      "0;92",
	"lightGreenB":   "1;92",
	"light_green_b": "1;92",
	"hi_green_b":    "1;92",
	"lightBlue":     "0;94",
	"light_blue":    "0;94",
	"hiBlue":        "0;94",
	"hi_blue":       "0;94",
	"lightBlueB":    "1;94",
	"light_blue_b":  "1;94",
	"hi_blue_b":     "1;94",
	"lightCyan":     "0;96",
	"light_cyan":    "0;96",
	"hiCyan":        "0;96",
	"hi_cyan":       "0;96",
	"lightCyanB":    "1;96",
	"light_cyan_b":  "1;96",
	"hi_cyan_b":     "1;96",
	"lightWhite":    "0;97;40",
	"light_white":   "0;97;40",

	// option
	"bold":       "1",
	"b":          "1",
	"italic":     "3",
	"i":          "3", // italic
	"underscore": "4",
	"us":         "4", // short name for 'underscore'
	"blink":      "5",
	"fb":         "6", // fast blink
	"reverse":    "7",
	"st":         "9", // strikethrough

	// alert tags, like bootstrap's alert
	"suc":     "1;32", // same "green" and "bold"
	"success": "1;32",
	"info":    "0;32", // same "green",
	"comment": "0;33", // same "brown"
	"note":    "36;1",
	"notice":  "36;4",
	"warn":    "0;1;33",
	"warning": "0;30;43",
	"primary": "0;34",
	"danger":  "1;31", // same "red" but add bold
	"err":     "97;41",
	"error":   "97;41", // fg light white; bg red
}

/*************************************************************
 * parse color tags
 *************************************************************/

// Render parse color tags, return rendered string.
//
// Usage:
//
//	text := Render("<info>hello</> <cyan>world</>!")
//	fmt.Println(text)
func Render(a ...any) string {
	if len(a) == 0 {
		return ""
	}
	return ReplaceTag(fmt.Sprint(a...))
}

// ReplaceTag parse string, replace color tag and return rendered string
func ReplaceTag(str string) string { return ParseTagByEnv(str) }

// ParseTagByEnv parse given string. will check package setting.
func ParseTagByEnv(str string) string {
	// disable OR not support color
	if termenv.NoColor() || !termenv.IsSupportColor() {
		return ClearTag(str)
	}
	return ParseTag(str)
}

// ParseTag parse given string, replace color tag and return rendered string
//
// Use built in tags:
//
//	<TAG_NAME>CONTENT</>
//	// e.g: `<info>message</>`
//
// TIP: code is from gookit/color package
//
//   - Not support custom attributes
//   - Not support c256 or rgb color
func ParseTag(str string) string {
	// not contains color tag
	if !strings.Contains(str, "</>") {
		return str
	}

	// find color tags by regex. str eg: "<fg=white;bg=blue;op=bold>content</>"
	matched := matchRegex.FindAllStringSubmatch(str, -1)

	// item: 0 full text 1 tag name 2 tag content
	for _, item := range matched {
		full, tag, body := item[0], item[1], item[2]

		// use defined color tag name: "<info>content</>" -> tag: "info"
		if code := colorTags[tag]; len(code) > 0 {
			str = strings.Replace(str, full, RenderString(code, body), 1)
		}
	}

	return str
}

// ClearTag clear-all tag for a string
func ClearTag(s string) string {
	if !strings.Contains(s, "</>") {
		return s
	}
	return stripRegex.ReplaceAllString(s, "")
}

/*************************************************************
 * helper methods
 *************************************************************/

// GetTagCode get color code by tag name
func GetTagCode(name string) string { return colorTags[name] }

// ApplyTag for messages
func ApplyTag(tag string, a ...any) string {
	return RenderCode(GetTagCode(tag), a...)
}

// WrapTag wrap a tag for a string "<tag>content</>"
func WrapTag(s string, tag string) string {
	if s == "" || tag == "" {
		return s
	}
	return fmt.Sprintf("<%s>%s</>", tag, s)
}

// ColorTags get all internal color tags
func ColorTags() map[string]string { return colorTags }

// IsDefinedTag is defined tag name
func IsDefinedTag(name string) bool {
	_, ok := colorTags[name]
	return ok
}
