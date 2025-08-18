//go:build windows

package termenv

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// Get the Windows Version and Build Number
var majorVersion, _, buildNumber = windows.RtlGetNtVersionNumbers()

// refer
//
//	https://github.com/Delta456/box-cli-maker/blob/7b5a1ad8a016ce181e7d8b05e24b54ff60b4b38a/detect_windows.go#L30-L57
//	https://github.com/gookit/color/issues/25#issuecomment-738727917
//
// detects the color level supported on Windows: CMD, PowerShell
func detectSpecialTermColor(_ string) (tl ColorLevel, needVTP bool) {
	if os.Getenv("ConEmuANSI") == "ON" {
		debugf("True Color support by ConEmuANSI=ON")
		// ConEmuANSI is "ON" for generic ANSI support
		// but True Color option is enabled by default
		// I am just assuming that people wouldn't have disabled it
		// Even if it is not enabled then ConEmu will auto round off accordingly
		return TermColorTrue, false
	}

	// Before Windows 10 Build Number 10586, console never supported ANSI Colors
	if buildNumber < 10586 || majorVersion < 10 {
		// Detect if using ANSICON on older systems
		if os.Getenv("ANSICON") != "" {
			conVersion := os.Getenv("ANSICON_VER")
			// 8-bit Colors were only supported after v1.81 release
			if conVersion >= "181" {
				return TermColor256, false
			}
			return TermColor16, false
		}

		return TermColorNone, false
	}

	// True Color is not available before build 14931 so fallback to 8-bit color.
	if buildNumber < 14931 {
		return TermColor256, true
	}

	// Windows 10 build 14931 is the first release that supports 16m/TrueColor
	debugf("support True Color on windows version is >= build 14931")
	return TermColorTrue, true
}

// TryEnableVTP try force enables colors on Windows terminal
func TryEnableVTP(enable bool) bool {
	if !enable {
		return false
	}

	// enable colors on Windows terminal
	if tryEnableOnCONOUT() {
		debugf("True-Color by enable VirtualTerminalProcessing on windows")
		return true
	}

	// initKernel32Proc()
	suc := tryEnableOnStdout()
	debugf("True-Color by enable VirtualTerminalProcessing on windows")
	return suc
}

func tryEnableOnCONOUT() bool {
	outHandle, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil {
		setLastErr(err)
		return false
	}

	err = EnableVTProcessing(outHandle, true)
	if err != nil {
		setLastErr(err)
		return false
	}

	return true
}

func tryEnableOnStdout() bool {
	// try direct open syscall.Stdout
	err := EnableVTProcessing(syscall.Stdout, true)
	if err != nil {
		setLastErr(err)
		return false
	}

	return true
}

// related docs
// https://docs.microsoft.com/zh-cn/windows/console/console-virtual-terminal-sequences
// https://docs.microsoft.com/zh-cn/windows/console/console-virtual-terminal-sequences#samples
var (
	// isMSys bool
	kernel32 *syscall.LazyDLL

	procGetConsoleMode *syscall.LazyProc
	procSetConsoleMode *syscall.LazyProc
)

func initKernel32Proc() {
	if kernel32 != nil {
		return
	}

	// load related Windows dll
	// https://docs.microsoft.com/en-us/windows/console/setconsolemode
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
}

/*************************************************************
 * render full color code on Windows(8,16,24bit color)
 *************************************************************/

// EnableVTProcessing Enable virtual terminal processing on Windows
//
// ref from github.com/konsorten/go-windows-terminal-sequences
// doc https://docs.microsoft.com/zh-cn/windows/console/console-virtual-terminal-sequences#samples
//
// Usage:
//
//	err := EnableVTProcessing(syscall.Stdout, true)
//	// support print color text
//	err = EnableVTProcessing(syscall.Stdout, false)
func EnableVTProcessing(stream syscall.Handle, enable bool) error {
	var mode uint32
	// Check if it is currently in the terminal
	// err := syscall.GetConsoleMode(syscall.Stdout, &mode)
	err := syscall.GetConsoleMode(stream, &mode)
	if err != nil {
		debugf("enable Windows VirtualTerminalProcessing error: %v", err)
		return err
	}

	// docs https://docs.microsoft.com/zh-cn/windows/console/getconsolemode#parameters
	if enable {
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	} else {
		mode &^= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	}

	err = windows.SetConsoleMode(windows.Handle(stream), mode)
	if err != nil {
		return err
	}

	// ret, _, err := procSetConsoleMode.Call(uintptr(stream), uintptr(mode))
	// if ret == 0 {
	// 	return err
	// }
	return nil
}

// on Windows, must convert 'syscall.Stdin' to int
func syscallStdinFd() int {
	return int(syscall.Stdin)
}
