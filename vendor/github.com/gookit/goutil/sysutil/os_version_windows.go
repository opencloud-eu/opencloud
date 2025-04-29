package sysutil

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/strutil"
)

// OSVersionInfo 结构体用于存储操作系统版本信息
//
// NOTE: Windows 10 和 Windows 11 在主版本号和次版本号上是相同的，因此需要通过构建号（win11: DwBuildNumber>=22000）来进一步区分
type OSVersionInfo struct {
	// 主版本号
	MajorVersion uint16
	// 次版本号
	MinorVersion uint16
	// 构建号
	BuildNumber uint32
	// 修订号
	RevisionNumber uint32
}

// FetchOsVersion Get Windows system version information
//
// 还可用使用dll获取：
//
//	通过 GetVersion, GetVersionEx 函数获取的信息不准确. win11获取到 6.2.9200, 实际是 10.0.22631
func FetchOsVersion() (*OSVersionInfo, error) {
	// Windows cmd 执行 ver 命令
	out, err := ShellExec("ver", "cmd")
	if err != nil {
		return nil, err
	}

	return parseOsVersionString(out)
}

// IsLtWindows7 判断是否小于 Windows 7
func (ov *OSVersionInfo) IsLtWindows7() bool {
	return ov.MajorVersion < 6 || (ov.MajorVersion == 6 && ov.MinorVersion < 1)
}

// IsWindows7 判断是否为 Windows 7
func (ov *OSVersionInfo) IsWindows7() bool {
	return ov.MajorVersion == 6 && ov.MinorVersion == 1
}

// IsWindows8 判断是否为 Windows 8
func (ov *OSVersionInfo) IsWindows8() bool {
	return ov.MajorVersion == 6 && ov.MinorVersion == 2
}

// IsWindows10 判断是否为 Windows 10
func (ov *OSVersionInfo) IsWindows10() bool {
	return ov.MajorVersion == 10 && ov.MinorVersion == 0 && ov.BuildNumber < 22000
}

// IsWindows11 判断是否为 Windows 11
func (ov *OSVersionInfo) IsWindows11() bool {
	return ov.MajorVersion == 10 && ov.MinorVersion == 0 && ov.BuildNumber >= 22000
}

// Name 获取 Windows 通用的版本名称. eg: xp, win7, win8, win10, win11, unknown
func (ov *OSVersionInfo) Name() string {
	switch ov.MajorVersion {
	case 10:
		if ov.BuildNumber < 22000 {
			return "win10"
		} else {
			return "win11"
		}
	case 6:
		switch ov.MinorVersion {
		case 0:
			return "vista"
		case 1:
			return "win7"
		case 2:
			return "win8"
		case 3:
			return "win8.1"
		}
	case 5:
		switch ov.MinorVersion {
		case 1:
			return "xp"
		case 2:
			return "ws2003" // win server 2003
		}
	}
	return "unknown"
}

// String format
func (ov *OSVersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d", ov.MajorVersion, ov.MinorVersion, ov.BuildNumber)
}

func parseOsVersionString(out string) (*OSVersionInfo, error) {
	// out eg: Microsoft Windows [Version 10.0.22631.4391] => 10.0.22631.4391
	// 部分系统会输出中文 eg: Microsoft Windows [版本 10.0.22631.4391]
	out = strings.TrimSpace(out)
	ns := strings.SplitN(out, "[", 2)
	if len(ns) < 2 {
		return nil, errorx.Rawf("cannot parse version info: %s", out)
	}

	ns = strutil.SplitByWhitespace(strings.Trim(ns[1], "]"))
	if len(ns) < 2 {
		return nil, errorx.Rawf("cannot parse version info2: %s", out)
	}

	var err error
	var ovi OSVersionInfo
	// get like: 10.0.22631.4391 or 10.0.22631
	verStr := ns[1]

	if strings.Count(verStr, ".") >= 3 {
		_, err = fmt.Sscanf(verStr, "%d.%d.%d.%d", &ovi.MajorVersion, &ovi.MinorVersion, &ovi.BuildNumber, &ovi.RevisionNumber)
	} else {
		_, err = fmt.Sscanf(verStr, "%d.%d.%d", &ovi.MajorVersion, &ovi.MinorVersion, &ovi.BuildNumber)
	}

	if err != nil {
		return nil, errorx.Rawf("parse version info %q error: %v", verStr, err)
	}
	return &ovi, nil
}

// 全局变量
var stdOv, stdErr = FetchOsVersion()

// OsVersion Get operating system version information
func OsVersion() *OSVersionInfo {
	return stdOv
}

// OvParseError error on parse os version info
func OvParseError() error {
	return stdErr
}
