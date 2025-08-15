//go:build darwin
package sysutil

import "os/exec"

// OsName system name. like runtime.GOOS. allow: linux, windows, darwin
const OsName = Darwin

// IsWin system. linux windows darwin
func IsWin() bool { return false }

// IsWindows system. linux windows darwin
func IsWindows() bool { return false }

// IsMac system
func IsMac() bool { return true }

// IsDarwin system
func IsDarwin() bool { return true }

// IsLinux system
func IsLinux() bool { return false }

// OpenURL Open browser URL
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
	return exec.Command("open", URL).Run()
}
