package comfunc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Workdir get
func Workdir() string {
	dir, _ := os.Getwd()
	return dir
}

// ExpandHome will parse first `~` as user home dir path.
func ExpandHome(pathStr string) string {
	if len(pathStr) == 0 {
		return pathStr
	}

	if pathStr[0] != '~' {
		return pathStr
	}

	if len(pathStr) > 1 && pathStr[1] != '/' && pathStr[1] != '\\' {
		return pathStr
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return pathStr
	}
	return homeDir + pathStr[1:]
}

// ExecCmd an command and return output.
//
// Usage:
//
//	ExecCmd("ls", []string{"-al"})
func ExecCmd(binName string, args []string, workDir ...string) (string, error) {
	// create a new Cmd instance
	cmd := exec.Command(binName, args...)
	if len(workDir) > 0 {
		cmd.Dir = workDir[0]
	}

	bs, err := cmd.Output()
	return string(bs), err
}

var (
	cmdList  = []string{"cmd", "cmd.exe"}
	pwshList = []string{"powershell", "powershell.exe", "pwsh", "pwsh.exe"}
)

// ShellExec exec command by shell
// cmdLine e.g. "ls -al"
func ShellExec(cmdLine string, shells ...string) (string, error) {
	// shell := "/bin/sh"
	shell := "sh"
	if len(shells) > 0 {
		shell = shells[0]
	}

	cmd := exec.Command(shell, "-c", cmdLine)
	bs, err := cmd.Output()
	return string(bs), err
}

// curShellCache value
var curShellCache string

// CurrentShell get current used shell env file.
//
// return like: "/bin/zsh" "/bin/bash". if onlyName=true, will return "zsh", "bash"
func CurrentShell(onlyName bool, fallbackShell ...string) (binPath string) {
	var err error

	fbShell := ""
	if len(fallbackShell) > 0 {
		fbShell = fallbackShell[0]
	}

	if curShellCache == "" {
		// 检查父进程名称
		parentProcess := os.Getenv("GOPROCESS")
		if parentProcess != "" {
			return parentProcess
		}

		binPath = os.Getenv("SHELL") // 适用于 Unix-like 系统
		if len(binPath) == 0 {
			// TODO check on Windows
			binPath, err = ShellExec("echo $SHELL")
			if err != nil {
				return fbShell
			}
		}

		binPath = strings.TrimSpace(binPath)
		// cache result
		curShellCache = binPath
	} else {
		binPath = curShellCache
	}

	if onlyName && len(binPath) > 0 {
		binPath = filepath.Base(binPath)
	} else if len(binPath) == 0 {
		binPath = fbShell
	}
	return
}

func checkWinCurrentShell() string {
	// 在 Windows 上，可以检查 COMSPEC 环境变量
	comSpec := os.Getenv("COMSPEC")
	// 没法检查 pwsh, 返回的还是 cmd
	return comSpec
}

// HasShellEnv has shell env check.
//
// Usage:
//
//	HasShellEnv("sh")
//	HasShellEnv("bash")
func HasShellEnv(shell string) bool {
	// can also use: "echo $0"
	out, err := ShellExec("echo OK", shell)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "OK"
}
