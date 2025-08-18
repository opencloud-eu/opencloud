package termenv

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ColorLevel is the color level supported by a terminal.
type ColorLevel uint8

const (
	TermColorNone ColorLevel = iota // not support color
	TermColor16                     // 16(4bit) ANSI color supported
	TermColor256                    // 256(8bit) color supported
	TermColorTrue                   // support TRUE(RGB) color
)

// String returns the string name of the color level.
func (l ColorLevel) String() string {
	switch l {
	case TermColor16:
		return "ansiColor"
	case TermColor256:
		return "256color"
	case TermColorTrue:
		return "trueColor"
	default:
		return "none"
	}
}

// NoColor returns true if the NO_COLOR environment variable is set.
func NoColor() bool { return noColor }

// TermColorLevel returns the color support level for the current terminal.
func TermColorLevel() ColorLevel { return colorLevel }

// IsSupportColor returns true if the terminal supports color.
func IsSupportColor() bool { return colorLevel > TermColorNone }

// IsSupport256Color returns true if the terminal supports 256 colors.
func IsSupport256Color() bool { return colorLevel >= TermColor256 }

// IsSupportTrueColor returns true if the terminal supports true color.
func IsSupportTrueColor() bool { return colorLevel == TermColorTrue }

//
// ---------------- Force set color support ----------------
//

var backLevel ColorLevel

// SetColorLevel value force.
func SetColorLevel(level ColorLevel) {
	// backup old value
	backLevel = colorLevel

	// force set color level
	colorLevel = level
	supportColor = level > TermColorNone
	noColor = supportColor == false
}

// DisableColor in the current terminal
func DisableColor() {
	// backup old value
	backLevel = colorLevel

	// force disable color
	noColor = true
	supportColor = false
	colorLevel = TermColorNone
}

// ForceEnableColor flags value. TIP: use for unit testing.
//
// Usage:
//
//	ccolor.ForceEnableColor()
//	defer ccolor.RevertColorSupport()
func ForceEnableColor() {
	// backup old value
	backLevel = colorLevel

	// force enables color
	noColor = false
	supportColor = true
	colorLevel = TermColor256
	// return colorLevel
}

// RevertColorSupport flags to init value.
func RevertColorSupport() {
	// revert color flags var
	colorLevel = backLevel
	supportColor = backLevel > TermColorNone
	noColor = os.Getenv("NO_COLOR") == ""
}

/*************************************************************
 * helper methods for detect color supports
 *************************************************************/

// DetectColorLevel for current env
//
// NOTICE: The method will detect terminal info each time.
//
//	if only want to get current color level, please direct call IsSupportColor() or TermColorLevel()
func DetectColorLevel() ColorLevel {
	level, _ := detectTermColorLevel()
	return level
}

// on TERM=screen: not support true-color
const noTrueColorTerm = "screen"

// detect terminal color support level
//
// refer https://github.com/Delta456/box-cli-maker
func detectTermColorLevel() (level ColorLevel, needVTP bool) {
	isWin := runtime.GOOS == "windows"
	termVal := os.Getenv("TERM")

	if termVal != noTrueColorTerm {
		// On JetBrains Terminal
		// - TERM value not set, but support true-color
		// env:
		// 	TERMINAL_EMULATOR=JetBrains-JediTerm
		val := os.Getenv("TERMINAL_EMULATOR")
		if val == "JetBrains-JediTerm" {
			debugf("True Color support on JetBrains-JediTerm, is win: %v", isWin)
			return TermColorTrue, false
		}
	}

	level = detectColorLevelFromEnv(termVal, isWin)

	// fallback: simple detect by TERM value string.
	if level == TermColorNone {
		debugf("level=none - fallback check special term color support")
		level, needVTP = detectSpecialTermColor(termVal)
		debugf("color level by detectSpecialTermColor: %s", level.String())
	} else {
		debugf("color level by detectColorLevelFromEnv: %s", level.String())
	}
	return
}

// detectColorFromEnv returns the color level COLORTERM, FORCE_COLOR,
// TERM_PROGRAM, or determined from the TERM environment variable.
//
// refer the github.com/xo/terminfo.ColorLevelFromEnv()
// https://en.wikipedia.org/wiki/Terminfo
func detectColorLevelFromEnv(termVal string, isWin bool) ColorLevel {
	if termVal == noTrueColorTerm { // on TERM=screen: not support true-color
		return TermColor256
	}

	// check for overriding environment variables
	colorTerm, termProg, forceColor := os.Getenv("COLORTERM"), os.Getenv("TERM_PROGRAM"), os.Getenv("FORCE_COLOR")
	switch {
	case strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit"):
		return TermColorTrue
	case colorTerm != "" || forceColor != "":
		return TermColor16
	case termProg == "Apple_Terminal":
		return TermColor256
	case termProg == "Terminus" || termProg == "Hyper":
		return TermColorTrue
	case termProg == "iTerm.app":
		// check iTerm version
		termVer := os.Getenv("TERM_PROGRAM_VERSION")
		if termVer != "" {
			i, err := strconv.Atoi(strings.Split(termVer, ".")[0])
			if err != nil {
				setLastErr(errors.New("invalid TERM_PROGRAM_VERSION=" + termVer))
				return TermColor256 // return TermColorNone
			}
			if i == 3 {
				return TermColorTrue
			}
		}
		return TermColor256
	}

	// otherwise determine from TERM's max_colors capability
	// if !isWin && termVal != "" {
	// 	debugf("TERM=%s - TODO check color level by load terminfo file", termVal)
	// 	return TermColor16
	// }

	// no TERM env value. default return none level
	return TermColorNone
}
