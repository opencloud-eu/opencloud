package sysutil

import (
	"os"
	"os/user"

	"github.com/gookit/goutil/internal/comfunc"
)

// MustFindUser must find a system user by name
func MustFindUser(uname string) *user.User {
	u, err := user.Lookup(uname)
	if err != nil {
		panic(err)
	}
	return u
}

// LoginUser must get current user, will panic if error
func LoginUser() *user.User {
	return CurrentUser()
}

// CurrentUser must get current user, will panic if error
func CurrentUser() *user.User {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u
}

// UHomeDir get user home dir path, ignore error. (by user.Current)
func UHomeDir() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.HomeDir
}

// homeDir cache
var _homeDir string

// UserHomeDir is alias of os.UserHomeDir, but ignore error.(by os.UserHomeDir)
func UserHomeDir() string {
	if _homeDir == "" {
		_homeDir, _ = os.UserHomeDir()
	}
	return _homeDir
}

// HomeDir get user home dir path.
func HomeDir() string { return UserHomeDir() }

// UserDir will prepend user home dir to subPaths
func UserDir(subPaths ...string) string {
	return comfunc.JoinPaths2(UserHomeDir(), subPaths)
}

// UserCacheDir will prepend user `$HOME/.cache` to subPaths
func UserCacheDir(subPaths ...string) string {
	return comfunc.JoinPaths3(UserHomeDir(), ".cache", subPaths)
}

// UserConfigDir will prepend user `$HOME/.config` to subPath
func UserConfigDir(subPaths ...string) string {
	return comfunc.JoinPaths3(UserHomeDir(), ".config", subPaths)
}

// ExpandPath will parse `~` as user home dir path.
func ExpandPath(path string) string { return comfunc.ExpandHome(path) }

// ExpandHome will parse `~` as user home dir path.
func ExpandHome(path string) string { return comfunc.ExpandHome(path) }
