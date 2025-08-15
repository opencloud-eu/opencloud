package ccolor

import (
	"fmt"
	"io"
	"strings"
)

// String color style string. TODO
// eg:
// 		s := String("red,bold")
//		s.Println("some text message")
type String string

// Style for color render.
type Style struct {
	Fg   Color
	Bg   Color
	Opts []Color
}

// NewStyle fg, bg and options
func NewStyle(fg Color, bg Color, opts ...Color) *Style {
	return &Style{
		Fg:   fg,
		Bg:   bg,
		Opts: opts,
	}
}

// Print like fmt.Print, but with color
func (s *Style) Print(v ...any) {
	doPrint(s.String(), fmt.Sprint(v...))
}

// Println like fmt.Println, but with color
func (s *Style) Println(v ...any) {
	doPrintln(s.String(), v)
}

// Printf render and print text
func (s *Style) Printf(format string, v ...any) {
	doPrint(s.String(), fmt.Sprintf(format, v...))
}

// Sprint like fmt.Sprint, but with color
func (s *Style) Sprint(v ...any) string {
	return RenderString(s.String(), fmt.Sprint(v...))
}

// Sprintln like fmt.Sprintln, but with color
func (s *Style) Sprintln(v ...any) string {
	return RenderWithSpaces(s.String(), v...)
}

// Sprintf format and render message.
func (s *Style) Sprintf(format string, v ...any) string {
	return RenderString(s.String(), fmt.Sprintf(format, v...))
}

// Fprint like fmt.Fprint, but with color
func (s *Style) Fprint(w io.Writer, v ...any) {
	doPrintTo(w, s.String(), fmt.Sprint(v...))
}

// String convert style setting to color code string.
func (s *Style) String() string {
	var codes []string
	if s.Fg.IsFg() {
		codes = append(codes, s.Fg.String())
	}
	if s.Bg.IsBg() {
		codes = append(codes, s.Bg.String())
	}

	if len(s.Opts) > 0 {
		codes = append(codes, ColorsToCode(s.Opts...))
	}

	if len(codes) == 0 {
		return ""
	}
	return strings.Join(codes, ";")
}

var (
	// Info color style
	Info = &Style{Fg: FgGreen}
	// Warn color style
	Warn = &Style{Fg: FgYellow}
	// Error color style
	Error = NewStyle(FgLightWhite, BgRed)
	// Debug color style
	Debug = &Style{Fg: FgCyan}
	// Success color style
	Success = &Style{Fg: FgGreen, Opts: []Color{OpBold}}
)
