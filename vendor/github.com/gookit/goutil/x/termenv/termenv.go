// Package termenv provides detect color support of the current terminal.
// And with some utils for terminal env.
package termenv

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var (
	lastErr error
	// debug mode
	debugMode bool
	// support color of current terminal
	supportColor bool

	// value of os color render and display
	//
	// NOTICE:
	// if ENV: NO_COLOR is not empty, will disable color render.
	noColor = os.Getenv("NO_COLOR") != ""

	// the color support level for current terminal
	// needVTP - need enable VTP, only for Windows OS
	colorLevel, needVTP = detectTermColorLevel()
)

// SetDebugMode sets debug mode.
func SetDebugMode(enable bool) { debugMode = enable }

// LastErr returns the last error.
func LastErr() error {
	defer func() {
		lastErr = nil // reset on get
	}()
	return lastErr
}

func debugf(tpl string, v ...any) {
	if debugMode {
		fmt.Printf("TERMENV: "+tpl+"\n", v...)
	}
}

func setLastErr(err error) {
	if err != nil {
		debugf("TERMENV: last error: %v", err)
		lastErr = err
	}
}

// exec: `stty -a 2>&1`
// const (
// mac: speed 9600 baud; 97 rows; 362 columns;
// macSttyMsgPattern = `(\d+)\s+rows;\s*(\d+)\s+columns;`
// linux: speed 38400 baud; rows 97; columns 362; line = 0;
// linuxSttyMsgPattern = `rows\s+(\d+);\s*columns\s+(\d+);`
// )
var terminalWidth, terminalHeight int

// GetTermSize for current console terminal.
func GetTermSize(refresh ...bool) (w int, h int) {
	if terminalWidth > 0 && len(refresh) > 0 && !refresh[0] {
		return terminalWidth, terminalHeight
	}

	var err error
	w, h, err = term.GetSize(syscallStdinFd())
	if err != nil {
		return
	}

	// cache result
	terminalWidth, terminalHeight = w, h
	return
}

// ReadPassword from console terminal
func ReadPassword(question ...string) string {
	if len(question) > 0 {
		print(question[0])
	} else {
		print("Enter Password: ")
	}

	bs, err := term.ReadPassword(syscallStdinFd())
	if err != nil {
		return ""
	}

	println() // new line
	return string(bs)
}
