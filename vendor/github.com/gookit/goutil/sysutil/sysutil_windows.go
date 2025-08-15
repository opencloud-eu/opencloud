//go:build windows
// +build windows

package sysutil

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// OsName system name. like runtime.GOOS. only allow: linux, windows, darwin
const OsName = Windows

// IsWin system. linux windows darwin
func IsWin() bool { return true }

// IsWindows system. linux windows darwin
func IsWindows() bool { return true }

// IsMac system
func IsMac() bool { return false }

// IsDarwin system
func IsDarwin() bool { return false }

// IsLinux system
func IsLinux() bool { return false }

// Kill a process by pid
func Kill(pid int, signal syscall.Signal) error {
	return errors.New("not support on Windows")
}

// ProcessExists check process exists by pid
func ProcessExists(pid int) bool {
	panic("TIP: please use sysutil/process.Exists()")
}

// OpenURL Open file or browser URL
//
// - refers https://github.com/pkg/browser
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
func OpenURL(url string) error {
	// return exec.Command("cmd", "/C", "start", URL).Run()
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(url), nil, nil, windows.SW_SHOWNORMAL)
}
