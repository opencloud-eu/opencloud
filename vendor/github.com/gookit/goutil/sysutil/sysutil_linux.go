package sysutil

import (
	"os/exec"
	"strings"
)

// OsName system name. like runtime.GOOS. allow: linux, windows, darwin
const OsName = Linux

// IsWin system. linux windows darwin
func IsWin() bool { return false }

// IsWindows system. linux windows darwin
func IsWindows() bool { return false }

// IsMac system
func IsMac() bool { return false }

// IsDarwin system
func IsDarwin() bool { return false }

// IsLinux system
func IsLinux() bool { return true }

// There are multiple possible providers to open a browser on linux
// One of them is xdg-open, another is x-www-browser, then there's www-browser, etc.
// Look for one that exists and run it
var openBins = []string{"xdg-open", "x-www-browser", "www-browser"}

// OpenURL Open file or browser URL
//
// Mac：
//
//	open 'https://github.com/inhere'
//
// Linux:
//
//	xdg-open URL
//	x-www-browser 'https://github.com/inhere'
//
// Windows:
//
//	cmd /c start https://github.com/inhere
func OpenURL(URL string) error {
	for _, bin := range openBins {
		if _, err := exec.LookPath(bin); err == nil {
			return exec.Command(bin, URL).Run()
		}
	}

	return &exec.Error{Name: strings.Join(openBins, ","), Err: exec.ErrNotFound}
}
