package termenv

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gookit/goutil/internal/checkfn"
	"github.com/gookit/goutil/internal/comfunc"
	"golang.org/x/term"
)

var (
	cmdList  = []string{"cmd", "cmd.exe"}
	pwshList = []string{"powershell", "powershell.exe", "pwsh", "pwsh.exe"}
)

// IsTerminal 检查是否为终端设备中
func IsTerminal() bool {
	fd := int(os.Stdout.Fd())
	return term.IsTerminal(fd)
}

// CurrentShell get current used shell env file.
//
// eg "/bin/zsh" "/bin/bash".
// if onlyName=true, will return "zsh", "bash"
func CurrentShell(onlyName bool, fallbackShell ...string) string {
	return comfunc.CurrentShell(onlyName, fallbackShell...)
}

// HasShellEnv has shell env check.
//
// Usage:
//
//	HasShellEnv("sh")
//	HasShellEnv("bash")
func HasShellEnv(shell string) bool {
	// can also use: "echo $0"
	out, err := shellExec("echo OK", shell)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "OK"
}

// IsShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func IsShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

func shellExec(expr, shell string) (string, error) {
	// "-c" for bash,sh,zsh shell
	mark := "-c"
	if shell == "" {
		shell = CurrentShell(true, "sh")
	}

	// special for Windows shell
	if runtime.GOOS == "windows" {
		// use cmd.exe, mark is "/c"
		if checkfn.StringsContains(cmdList, shell) {
			mark = "/c"
		} else if checkfn.StringsContains(pwshList, shell) {
			// "-Command" for powershell
			mark = "-Command"
		}
	}

	cmd := exec.Command(shell, mark, expr)
	bs, err := cmd.CombinedOutput()
	return string(bs), err
}
