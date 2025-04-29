//go:build windows
// +build windows

package sysutil

// ChangeUserByName change work user by new username.
func ChangeUserByName(newUname string) (err error) {
	return ChangeUserUIDGid(0, 0)
}

// ChangeUserUidGid change work user by new username uid,gid
//
// Deprecated: use ChangeUserUIDGid instead
func ChangeUserUidGid(newUid int, newGid int) (err error) {
	return ChangeUserUIDGid(newUid, newGid)
}

// ChangeUserUIDGid change work user by new username uid,gid
func ChangeUserUIDGid(newUid int, newGid int) (err error) {
	return nil
}

// IsAdmin Determine whether the current user is an administrator
func IsAdmin() bool {
	// 执行 net session 判断
	_, err := ExecCmd("net", []string{"session"})
	return err == nil
}
