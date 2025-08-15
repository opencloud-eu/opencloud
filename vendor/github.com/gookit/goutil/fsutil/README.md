# FileSystem Util

`fsutil` Provide some commonly file system util functions.

## Install

```shell
go get github.com/gookit/goutil/fsutil
```

## Go docs

- [Go docs](https://pkg.go.dev/github.com/gookit/goutil/fsutil)

## Find files

More see [./finder](./finder)

```go
// find all files in dir
fsutil.FindInDir("./", func(filePath string, de fs.DirEntry) error {
    fmt.Println(filePath)
    return nil
})

// find files with filters
fsutil.FindInDir("./", func(filePath string, de fs.DirEntry) error {
    fmt.Println(filePath)
    return nil
}, fsutil.ExcludeDotFile)
```

## Functions API

> **Note**: doc by run `go doc ./fsutil`

```go
func ApplyFilters(fPath string, ent fs.DirEntry, filters []FilterFunc) bool
func CopyFile(srcPath, dstPath string) error
func CreateFile(fpath string, filePerm, dirPerm os.FileMode, fileFlag ...int) (*os.File, error)
func DeleteIfExist(fPath string) error
func DeleteIfFileExist(fPath string) error
func Dir(fpath string) string
func DiscardReader(src io.Reader)
func ExcludeDotFile(_ string, ent fs.DirEntry) bool
func Expand(pathStr string) string
func ExpandPath(pathStr string) string
func Extname(fpath string) string
func FileExists(path string) bool
func FileExt(fpath string) string
func FindInDir(dir string, handleFn HandleFunc, filters ...FilterFunc) (e error)
func GetContents(in any) []byte
func GlobWithFunc(pattern string, fn func(filePath string) error) (err error)
func IsAbsPath(aPath string) bool
func IsDir(path string) bool
func IsFile(path string) bool
func IsImageFile(path string) bool
func IsZipFile(filepath string) bool
func JoinPaths(elem ...string) string
func JoinSubPaths(basePath string, elem ...string) string
func LineScanner(in any) *bufio.Scanner
func MimeType(path string) (mime string)
func MkDirs(perm os.FileMode, dirPaths ...string) error
func MkParentDir(fpath string) error
func MkSubDirs(perm os.FileMode, parentDir string, subDirs ...string) error
func Mkdir(dirPath string, perm os.FileMode) error
func MustCopyFile(srcPath, dstPath string)
func MustCreateFile(filePath string, filePerm, dirPerm os.FileMode) *os.File
func MustReadFile(filePath string) []byte
func MustReadReader(r io.Reader) []byte
func MustRemove(fPath string)
func Name(fpath string) string
func NewIOReader(in any) (r io.Reader, err error)
func OSTempDir(pattern string) (string, error)
func OSTempFile(pattern string) (*os.File, error)
func OnlyFindDir(_ string, ent fs.DirEntry) bool
func OnlyFindFile(_ string, ent fs.DirEntry) bool
func OpenAppendFile(filepath string) (*os.File, error)
func OpenFile(filepath string, flag int, perm os.FileMode) (*os.File, error)
func OpenReadFile(filepath string) (*os.File, error)
func OpenTruncFile(filepath string) (*os.File, error)
func PathExists(path string) bool
func PathMatch(pattern, s string) bool
func PathName(fpath string) string
func PutContents(filePath string, data any, fileFlag ...int) (int, error)
func QuickOpenFile(filepath string, fileFlag ...int) (*os.File, error)
func QuietRemove(fPath string)
func ReadAll(in any) []byte
func ReadExistFile(filePath string) []byte
func ReadFile(filePath string) []byte
func ReadOrErr(in any) ([]byte, error)
func ReadReader(r io.Reader) []byte
func ReadString(in any) string
func ReadStringOrErr(in any) (string, error)
func ReaderMimeType(r io.Reader) (mime string)
func Realpath(pathStr string) string
func Remove(fPath string) error
func ResolvePath(pathStr string) string
func RmFileIfExist(fPath string) error
func RmIfExist(fPath string) error
func SearchNameUp(dirPath, name string) string
func SearchNameUpx(dirPath, name string) (string, bool)
func SlashPath(path string) string
func SplitPath(pathStr string) (dir, name string)
func Suffix(fpath string) string
func TempDir(dir, pattern string) (string, error)
func TempFile(dir, pattern string) (*os.File, error)
func TextScanner(in any) *scanner.Scanner
func ToAbsPath(p string) string
func UnixPath(path string) string
func Unzip(archive, targetDir string) (err error)
func WalkDir(dir string, fn fs.WalkDirFunc) error
func WriteFile(filePath string, data any, perm os.FileMode, fileFlag ...int) error
func WriteOSFile(f *os.File, data any) (n int, err error)
type FilterFunc func(fPath string, ent fs.DirEntry) bool
func ExcludeSuffix(ss ...string) FilterFunc
func IncludeSuffix(ss ...string) FilterFunc
type HandleFunc func(fPath string, ent fs.DirEntry) error
```

## Code Check & Testing

```bash
gofmt -w -l ./
golint ./...
```

**Testing**:

```shell
go test -v ./fsutil/...
```

**Test limit by regexp**:

```shell
go test -v -run ^TestSetByKeys ./fsutil/...
```
