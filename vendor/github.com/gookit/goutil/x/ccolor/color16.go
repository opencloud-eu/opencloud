package ccolor

import (
	"fmt"
	"strconv"
)

// Color Color16, 16 color value type
// 3(2^3=8) OR 4(2^4=16) bite color.
type Color uint8

/*************************************************************
 * Basic 16 color definition
 *************************************************************/

// Boundary value for foreground/background color 16
//
//   - base: fg 30~37, bg 40~47
//   - light: fg 90~97, bg 100~107
const (
	fgBase uint8 = 30
	fgMax  uint8 = 37
	bgBase uint8 = 40
	bgMax  uint8 = 47

	hiFgBase uint8 = 90
	hiFgMax  uint8 = 97
	hiBgBase uint8 = 100
	hiBgMax  uint8 = 107

	// optMax max option value. range: 0 - 9
	optMax = 10
)

// Foreground colors. basic foreground colors 30 - 37
const (
	FgBlack Color = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta // 品红
	FgCyan    // 青色
	FgWhite
	// FgDefault revert default FG
	FgDefault Color = 39
)

// Extra foreground color 90 - 97(非标准)
const (
	FgDarkGray Color = iota + 90 // 亮黑（灰）
	FgLightRed
	FgLightGreen
	FgLightYellow
	FgLightBlue
	FgLightMagenta
	FgLightCyan
	FgLightWhite
	// FgGray is alias of FgDarkGray
	FgGray Color = 90 // 亮黑（灰）
)

// Background colors. basic background colors 40 - 47
const (
	BgBlack Color = iota + 40
	BgRed
	BgGreen
	BgYellow // BgBrown like yellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
	// BgDefault revert default BG
	BgDefault Color = 49
)

// Extra background color 100 - 107 (non-standard)
const (
	BgDarkGray Color = iota + 100
	BgLightRed
	BgLightGreen
	BgLightYellow
	BgLightBlue
	BgLightMagenta
	BgLightCyan
	BgLightWhite
	// BgGray is alias of BgDarkGray
	BgGray Color = 100
)

// Option settings. range: 0 - 9
const (
	OpReset         Color = iota // 0 重置所有设置
	OpBold                       // 1 加粗
	OpFuzzy                      // 2 模糊(不是所有的终端仿真器都支持)
	OpItalic                     // 3 斜体(不是所有的终端仿真器都支持)
	OpUnderscore                 // 4 下划线
	OpBlink                      // 5 闪烁
	OpFastBlink                  // 6 快速闪烁(未广泛支持)
	OpReverse                    // 7 颠倒的 交换背景色与前景色
	OpConcealed                  // 8 隐匿的
	OpStrikethrough              // 9 删除的，删除线(未广泛支持)
)

// There are basic and light foreground color aliases
const (
	Red     = FgRed
	Cyan    = FgCyan
	Gray    = FgDarkGray // is light Black
	Blue    = FgBlue
	Black   = FgBlack
	Green   = FgGreen
	White   = FgWhite
	Yellow  = FgYellow
	Magenta = FgMagenta

	// special

	Bold   = OpBold
	Normal = FgDefault

	// extra light

	LightRed     = FgLightRed
	LightCyan    = FgLightCyan
	LightBlue    = FgLightBlue
	LightGreen   = FgLightGreen
	LightWhite   = FgLightWhite
	LightYellow  = FgLightYellow
	LightMagenta = FgLightMagenta

	HiRed     = FgLightRed
	HiCyan    = FgLightCyan
	HiBlue    = FgLightBlue
	HiGreen   = FgLightGreen
	HiWhite   = FgLightWhite
	HiYellow  = FgLightYellow
	HiMagenta = FgLightMagenta

	BgHiRed     = BgLightRed
	BgHiCyan    = BgLightCyan
	BgHiBlue    = BgLightBlue
	BgHiGreen   = BgLightGreen
	BgHiWhite   = BgLightWhite
	BgHiYellow  = BgLightYellow
	BgHiMagenta = BgLightMagenta
)

/*************************************************************
 * Color render methods
 *************************************************************/

// Text render a text message
func (c Color) Text(message string) string { return RenderString(c.String(), message) }

// Render messages by color setting
//
// Usage:
//
//	green := ccolor.FgGreen.Render
//	fmt.Println(green("message"))
func (c Color) Render(a ...any) string { return RenderCode(c.String(), a...) }

// Renderln messages by color setting.
// like fmt.Println, will add spaces for each argument
//
// Usage:
//
//	green := ccolor.FgGreen.Renderln
//	fmt.Println(green("message"))
func (c Color) Renderln(a ...any) string { return RenderWithSpaces(c.String(), a...) }

// Sprint render messages by color setting. is alias of the Render()
func (c Color) Sprint(a ...any) string { return RenderCode(c.String(), a...) }

// Sprintf format and render message.
//
// Usage:
//
//		green := ccolor.Green.Sprintf
//	 	colored := green("message")
func (c Color) Sprintf(format string, args ...any) string {
	return RenderString(c.String(), fmt.Sprintf(format, args...))
}

// Print messages.
//
// Usage:
//
//	ccolor.Green.Print("message")
//
// OR:
//
//	green := ccolor.FgGreen.Print
//	green("message")
func (c Color) Print(args ...any) {
	doPrint(c.Code(), fmt.Sprint(args...))
}

// Printf format and print messages.
//
// Usage:
//
//	ccolor.Cyan.Printf("string %s", "arg0")
func (c Color) Printf(format string, a ...any) {
	doPrint(c.Code(), fmt.Sprintf(format, a...))
}

// Println messages with new line
func (c Color) Println(a ...any) { doPrintln(c.String(), a) }

// Light current color. eg: 36(FgCyan) -> 96(FgLightCyan).
//
// Usage:
//
//	lightCyan := Cyan.Light()
//	lightCyan.Print("message")
func (c Color) Light() Color {
	val := uint8(c)
	if val >= 30 && val <= 47 {
		return Color(val + 60)
	}

	// don't change
	return c
}

// Darken current color. eg. 96(FgLightCyan) -> 36(FgCyan)
//
// Usage:
//
//	cyan := LightCyan.Darken()
//	cyan.Print("message")
func (c Color) Darken() Color {
	val := uint8(c)
	if val >= 90 && val <= 107 {
		return Color(val - 60)
	}

	// don't change
	return c
}

// ToFg always convert fg
func (c Color) ToFg() Color {
	val := uint8(c)
	// option code, don't change
	if val < 10 {
		return c
	}
	return Color(Bg2Fg(val))
}

// ToBg always convert bg
func (c Color) ToBg() Color {
	val := uint8(c)
	// option code, don't change
	if val < 10 {
		return c
	}
	return Color(Fg2Bg(val))
}

// Code convert to code string. eg "35"
func (c Color) Code() string {
	return strconv.FormatInt(int64(c), 10)
}

// String convert to code string. eg "35"
func (c Color) String() string {
	return strconv.FormatInt(int64(c), 10)
}

// IsBg check is background color
func (c Color) IsBg() bool {
	val := uint8(c)
	return val >= bgBase && val <= bgMax || val >= hiBgBase && val <= hiBgMax
}

// IsFg check is foreground color
func (c Color) IsFg() bool {
	val := uint8(c)
	return val >= fgBase && val <= fgMax || val >= hiFgBase && val <= hiFgMax
}

// IsOption check is option code: 0-9
func (c Color) IsOption() bool { return uint8(c) < optMax }

// IsValid color value
func (c Color) IsValid() bool { return uint8(c) < hiBgMax }

/*************************************************************
 * basic color maps
 *************************************************************/

// FgColors foreground colors map
var FgColors = map[string]Color{
	"black":   FgBlack,
	"red":     FgRed,
	"green":   FgGreen,
	"yellow":  FgYellow,
	"blue":    FgBlue,
	"magenta": FgMagenta,
	"cyan":    FgCyan,
	"white":   FgWhite,
	"default": FgDefault,
}

// BgColors background colors map
var BgColors = map[string]Color{
	"black":   BgBlack,
	"red":     BgRed,
	"green":   BgGreen,
	"yellow":  BgYellow,
	"blue":    BgBlue,
	"magenta": BgMagenta,
	"cyan":    BgCyan,
	"white":   BgWhite,
	"default": BgDefault,
}

// ExFgColors extra foreground colors map
var ExFgColors = map[string]Color{
	"darkGray":     FgDarkGray,
	"lightRed":     FgLightRed,
	"lightGreen":   FgLightGreen,
	"lightYellow":  FgLightYellow,
	"lightBlue":    FgLightBlue,
	"lightMagenta": FgLightMagenta,
	"lightCyan":    FgLightCyan,
	"lightWhite":   FgLightWhite,
}

// ExBgColors extra background colors map
var ExBgColors = map[string]Color{
	"darkGray":     BgDarkGray,
	"lightRed":     BgLightRed,
	"lightGreen":   BgLightGreen,
	"lightYellow":  BgLightYellow,
	"lightBlue":    BgLightBlue,
	"lightMagenta": BgLightMagenta,
	"lightCyan":    BgLightCyan,
	"lightWhite":   BgLightWhite,
}

// AllOptions color options map
var AllOptions = map[string]Color{
	"reset":      OpReset,
	"bold":       OpBold,
	"fuzzy":      OpFuzzy,
	"italic":     OpItalic,
	"underscore": OpUnderscore,
	"blink":      OpBlink,
	"reverse":    OpReverse,
	"concealed":  OpConcealed,
}

// Bg2Fg bg color value to fg value
func Bg2Fg(val uint8) uint8 {
	if val >= bgBase && val <= 47 { // is bg
		val = val - 10
	} else if val >= hiBgBase && val <= 107 { // is hi bg
		val = val - 10
	}
	return val
}

// Fg2Bg fg color value to bg value
func Fg2Bg(val uint8) uint8 {
	if val >= fgBase && val <= 37 { // is fg
		val = val + 10
	} else if val >= hiFgBase && val <= 97 { // is hi fg
		val = val + 10
	}
	return val
}
