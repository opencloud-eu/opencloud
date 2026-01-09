package cephfs

// Fd returns the integer open file descriptor in cephfs.
// NOTE: It doesn't make sense to consume the returned integer fd anywhere
// outside CephFS and is recommended not to do so given the undefined behaviour.
// Also, as seen with the Go standard library, the fd is only valid as long as
// the corresponding File object is intact in the sense that an fd from a closed
// File object is invalid.
func (f *File) Fd() int {
	if f == nil || f.mount == nil {
		return -1
	}

	return int(f.fd)
}
