//go:build darwin || freebsd || netbsd || openbsd

package timemanager

import "syscall"

// StatCtime returns the creation time
func StatCTime(st *syscall.Stat_t) syscall.Timespec {
	return st.Ctimespec
}
