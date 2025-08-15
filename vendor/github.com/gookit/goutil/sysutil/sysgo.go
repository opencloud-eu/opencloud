package sysutil

import (
	"runtime"

	"github.com/gookit/goutil/x/goinfo"
)

// GoVersion get go runtime version. eg: "1.18.2"
func GoVersion() string {
	return runtime.Version()[2:]
}

// GoInfo define. alias of goinfo.GoInfo
type GoInfo = goinfo.GoInfo

// CallerInfo define. alias of goinfo.CallerInfo
type CallerInfo = goinfo.CallerInfo

// ParseGoVersion get info by parse `go version` results. alias of goinfo.ParseGoVersion()
//
// Examples:
//
//		line, err := sysutil.ExecLine("go version")
//		if err != nil {
//			return err
//		}
//
//		info, err := sysutil.ParseGoVersion()
//	 	dump.P(info)
func ParseGoVersion(line string) (*GoInfo, error) {
	return goinfo.ParseGoVersion(line)
}

// OsGoInfo fetch and parse. alias of goinfo.OsGoInfo()
func OsGoInfo() (*GoInfo, error) {
	return goinfo.OsGoInfo()
}

// CallersInfos returns an array of the CallerInfo. can with filters
func CallersInfos(skip, num int, filters ...goinfo.CallerFilterFunc) []*CallerInfo {
	return goinfo.CallersInfos(skip+1, num, filters...)
}
