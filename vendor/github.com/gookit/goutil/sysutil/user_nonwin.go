//go:build !windows

package sysutil

import (
	"os"
	"syscall"

	"github.com/gookit/goutil/strutil"
)

// IsAdmin Determine whether the current user is an administrator(root)
func IsAdmin() bool {
	return os.Getuid() == 0
}

// ChangeUserByName change work user by new username.
func ChangeUserByName(newUname string) error {
	u := MustFindUser(newUname)
	// syscall.Setlogin(newUname)
	return ChangeUserUIDGid(strutil.IntOrPanic(u.Uid), strutil.IntOrPanic(u.Gid))
}

// ChangeUserUidGid change work user by new username uid,gid
//
// Deprecated: use ChangeUserUIDGid instead
func ChangeUserUidGid(newUID int, newGid int) error {
	return ChangeUserUIDGid(newUID, newGid)
}

// ChangeUserUIDGid change work user by new username uid,gid
func ChangeUserUIDGid(newUID int, newGid int) (err error) {
	if newUID > 0 {
		err = syscall.Setuid(newUID)

		// update group id
		if err == nil && newGid > 0 {
			err = syscall.Setgid(newGid)
		}
	}
	return
}
