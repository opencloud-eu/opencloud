# System Utils

Provide some system util functions. eg: sysenv, exec, user, process

- quick exec a command line string.
- quick build a command for run

## Install

```bash
go get github.com/gookit/goutil/sysutil
```

## Usage

```go
out, err := sysutil.ExecCmd("ls", []string{"-al"})
```

## Clipboard

Package `clipboard` provide a simple clipboard read and write operations.

### Install

```bash
go get github.com/gookit/goutil/sysutil/clipboard
```

### Usage

Examples:

```go
src := "hello, this is clipboard"
err = clipboard.WriteString(src)
assert.NoErr(t, err)

// str: "hello, this is clipboard"
str, err = clipboard.ReadString()
assert.NoErr(t, err)
assert.NotEmpty(t, str)
assert.Eq(t, src, str)
```

## Cmd run

### Install

```bash
go get github.com/gookit/goutil/sysutil/cmdr
```
### Usage

**Cmd Builder**:

```go
c := cmdr.NewCmd("ls").
    WithArg("-l").
    WithArgs([]string{"-h"}).
    AddArg("-a").
    AddArgf("%s", "./")

c.OnBefore(func(c *cmdr.Cmd) {
    assert.Eq(t, "ls -l -h -a ./", c.Cmdline())
})

out := c.SafeOutput()
fmt.Println(out)
```

**Batch Cmd Tasks**:

Can use `cmdr.Runner` run multi cmd tasks at once.

```go
buf := new(bytes.Buffer)
rr := cmdr.NewRunner()

rr.Add(&cmdr.Task{
    ID:  "task1",
    Cmd: cmdr.NewCmd("id", "-F").WithOutput(buf, buf),
})
rr.AddCmd(cmdr.NewCmd("ls").AddArgs([]string{"-l", "-h"}).WithOutput(buf, buf))

err = rr.Run()
```

### Functions API

> generate by: `go doc ./sysutil`

```go
func BinDir() string
func BinFile() string
func ChangeUserByName(newUname string) (err error)
func ChangeUserUidGid(newUid int, newGid int) (err error)
func CurrentShell(onlyName bool) (path string)
func CurrentUser() *user.User
func EnvPaths() []string
func ExecCmd(binName string, args []string, workDir ...string) (string, error)
func ExecLine(cmdLine string, workDir ...string) (string, error)
func Executable(binName string) (string, error)
func ExpandPath(path string) string
func FindExecutable(binName string) (string, error)
func FlushExec(bin string, args ...string) error
func GoVersion() string
func HasExecutable(binName string) bool
func HasShellEnv(shell string) bool
func HomeDir() string
func Hostname() string
func IsConsole(out io.Writer) bool
func IsDarwin() bool
func IsLinux() bool
func IsMSys() bool
func IsMac() bool
func IsShellSpecialVar(c uint8) bool
func IsTerminal(fd uintptr) bool
func IsWin() bool
func IsWindows() bool
func Kill(pid int, signal syscall.Signal) error
func LoginUser() *user.User
func MustFindUser(uname string) *user.User
func NewCmd(bin string, args ...string) *cmdr.Cmd
func OpenBrowser(URL string) error
func ProcessExists(pid int) bool
func QuickExec(cmdLine string, workDir ...string) (string, error)
func SearchPath(keywords string) []string
func ShellExec(cmdLine string, shells ...string) (string, error)
func StdIsTerminal() bool
func UHomeDir() string
func UserCacheDir(subPath string) string
func UserConfigDir(subPath string) string
func UserDir(subPath string) string
func UserHomeDir() string
func Workdir() string
type CallerInfo struct{ ... }
    func CallersInfos(skip, num int, filters ...func(file string, fc *runtime.Func) bool) []*CallerInfo
type GoInfo struct{ ... }
    func OsGoInfo() (*GoInfo, error)
    func ParseGoVersion(line string) (*GoInfo, error)
```
