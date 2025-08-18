package goinfo

import (
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"github.com/gookit/goutil/strutil"
)

// FullFcName struct.
type FullFcName struct {
	// FullName eg: "github.com/gookit/goutil/x/goinfo.PanicIf"
	FullName string
	pkgPath  string // "github.com/gookit/goutil/x/goinfo"
	pkgName  string // "goinfo"
	funcName string // "PanicIf"
}

// Parse the full func name.
func (ffn *FullFcName) Parse() {
	if ffn.funcName != "" {
		return
	}

	i := strings.LastIndex(ffn.FullName, "/")

	ffn.pkgPath = ffn.FullName[:i+1]
	// spilt get pkg and func name
	ffn.pkgName, ffn.funcName = strutil.MustCut(ffn.FullName[i+1:], ".")
	ffn.pkgPath += ffn.pkgName
}

// PkgPath string get. eg: github.com/gookit/goutil/x/goinfo
func (ffn *FullFcName) PkgPath() string {
	ffn.Parse()
	return ffn.pkgPath
}

// PkgName string get. eg: goinfo
func (ffn *FullFcName) PkgName() string {
	ffn.Parse()
	return ffn.pkgName
}

// FuncName get short func name. eg: PanicIf
func (ffn *FullFcName) FuncName() string {
	ffn.Parse()
	return ffn.funcName
}

// String get full func name string, pkg path and func name.
func (ffn *FullFcName) String() string {
	return ffn.FullName
}

// FuncName get full func name, contains pkg path.
//
// eg:
//
//	// OUTPUT: github.com/gookit/goutil/x/goinfo.PkgName
//	goinfo.FuncName(goinfo.PkgName)
func FuncName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

// CutFuncName get pkg path and short func name
// eg:
//
//	"github.com/gookit/goutil/x/goinfo.FuncName" => [github.com/gookit/goutil/x/goinfo, FuncName]
func CutFuncName(fullFcName string) (pkgPath, shortFnName string) {
	ffn := FullFcName{FullName: fullFcName}
	return ffn.PkgPath(), ffn.FuncName()
}

// PkgName get current package name
//
// Usage:
//
//	fullFcName := goinfo.FuncName(fn)
//	pgkName := goinfo.PkgName(fullFcName)
func PkgName(fullFcName string) string {
	for {
		lastPeriod := strings.LastIndex(fullFcName, ".")
		lastSlash := strings.LastIndex(fullFcName, "/")

		if lastPeriod > lastSlash {
			fullFcName = fullFcName[:lastPeriod]
		} else {
			break
		}
	}
	return fullFcName
}

// GoodFuncName reports whether the function name is a valid identifier.
func GoodFuncName(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		switch {
		case r == '_':
		case i == 0 && !unicode.IsLetter(r):
			return false
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			return false
		}
	}
	return true
}
