package sysutil

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gookit/goutil/internal/checkfn"
	"github.com/gookit/goutil/internal/comfunc"
	"golang.org/x/term"
)

// IsMSys msys(MINGW64) env，不一定支持颜色
func IsMSys() bool {
	// "MSYSTEM=MINGW64"
	return len(os.Getenv("MSYSTEM")) > 0
}

// IsConsole check out is in stderr/stdout/stdin
//
// Usage:
//
//	sysutil.IsConsole(os.Stdout)
func IsConsole(out io.Writer) bool {
	o, ok := out.(*os.File)
	if !ok {
		return false
	}

	fd := o.Fd()

	// fix: cannot use 'o == os.Stdout' to compare
	return fd == uintptr(syscall.Stdout) || fd == uintptr(syscall.Stdin) || fd == uintptr(syscall.Stderr)
}

// IsTerminal isatty check
//
// Usage:
//
//	sysutil.IsTerminal(os.Stdout.Fd())
func IsTerminal(fd uintptr) bool {
	// return isatty.IsTerminal(fd) // "github.com/mattn/go-isatty"
	return term.IsTerminal(int(fd))
}

// StdIsTerminal os.Stdout is terminal
func StdIsTerminal() bool {
	return IsTerminal(os.Stdout.Fd())
}

// Hostname is alias of os.Hostname, but ignore error
func Hostname() string {
	name, _ := os.Hostname()
	return name
}

// CurrentShell get current used shell env file.
//
// eg "/bin/zsh" "/bin/bash".
// if onlyName=true, will return "zsh", "bash"
func CurrentShell(onlyName bool) (path string) {
	return comfunc.CurrentShell(onlyName)
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

// IsShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func IsShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// FindExecutable in the system
//
// Usage:
//
//	sysutil.FindExecutable("bash")
func FindExecutable(binName string) (string, error) {
	return exec.LookPath(binName)
}

// Executable find in the system, alias of FindExecutable()
//
// Usage:
//
//	sysutil.Executable("bash")
func Executable(binName string) (string, error) {
	return exec.LookPath(binName)
}

// HasExecutable in the system
//
// Usage:
//
//	HasExecutable("bash")
func HasExecutable(binName string) bool {
	_, err := exec.LookPath(binName)
	return err == nil
}

// Getenv get ENV value by key name, can with default value
func Getenv(name string, def ...string) string {
	val := os.Getenv(name)
	if val == "" && len(def) > 0 {
		val = def[0]
	}
	return val
}

// Environ like os.Environ, but will returns key-value map[string]string data.
func Environ() map[string]string { return comfunc.Environ() }

// EnvMapWith like os.Environ, but will return key-value map[string]string data.
func EnvMapWith(newEnv map[string]string) map[string]string {
	envMp := comfunc.Environ()
	for name, value := range newEnv {
		envMp[name] = value
	}
	return envMp
}

// EnvPaths get and split $PATH to []string
func EnvPaths() []string {
	return filepath.SplitList(os.Getenv("PATH"))
}

// SearchPathOption settings for SearchPath
type SearchPathOption struct {
	// 限制的扩展名
	LimitExt []string
}

// SearchPath search executable files in the system $PATH
//
// Usage:
//
//	sysutil.SearchPath("go")
func SearchPath(keywords string, limit int) []string {
	path := os.Getenv("PATH")
	ptn := "*" + keywords + "*"
	list := make([]string, 0)

	// if windows, will limit with .exe, .bat, .cmd
	isWindows := IsWindows()
	winExts := []string{".exe", ".bat", ".cmd"}
	checked := make(map[string]bool)
	for _, dir := range filepath.SplitList(path) {
		// Unix shell semantics: path element "" means "."
		if dir == "" {
			dir = "."
		}

		// mark dir is checked
		if _, ok := checked[dir]; ok {
			continue
		}

		checked[dir] = true
		matches, err := filepath.Glob(filepath.Join(dir, ptn))
		if err == nil && len(matches) > 0 {
			if isWindows {
				// if windows, will limit with .exe, .bat, .cmd
				for _, fPath := range matches {
					fExt := filepath.Ext(fPath)
					if checkfn.StringsContains(winExts, fExt) {
						continue
					}
					list = append(list, fPath)
				}
			} else {
				list = append(list, matches...)
			}

			// limit result size
			size := len(list)
			if limit > 0 && size >= limit {
				list = list[:limit]
				break
			}
		}
	}

	return list
}
