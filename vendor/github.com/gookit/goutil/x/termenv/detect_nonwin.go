//go:build !windows

package termenv

import (
	"strings"
	"syscall"

	"github.com/gookit/goutil/internal/checkfn"
)

// detect special term color support on macOS, linux, unix
func detectSpecialTermColor(termVal string) (ColorLevel, bool) {
	if termVal == "" {
		// detect WSL as it has True Color support
		// on Windows WSL:
		// - runtime.GOOS == "Linux"
		// - support true-color
		if checkfn.IsWSL() {
			debugf("True Color support on WSL environment")
			return TermColorTrue, false
		}
		return TermColorNone, false
	}

	debugf("terminfo check - fallback detect color by check TERM value")

	// on TERM=screen:
	// - support 256, not support true-color. test on macOS
	if termVal == noTrueColorTerm {
		return TermColor256, false
	}

	if strings.Contains(termVal, "256color") {
		return TermColor256, false
	}

	if strings.Contains(termVal, "xterm") {
		return TermColor256, false
	}
	return TermColor16, false
}

func syscallStdinFd() int {
	return syscall.Stdin
}