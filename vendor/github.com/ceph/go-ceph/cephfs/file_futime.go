package cephfs

// Futime changes file/directory last access and modification times.
//
// Implements:
//
//	int ceph_futime(struct ceph_mount_info *cmount, int fd, struct utimbuf *buf);
func (f *File) Futime(times *Utime) error {
	if err := f.validate(); err != nil {
		return err
	}

	return f.mount.Futime(int(f.fd), times)
}

// Futimens changes file/directory last access and modification times, here times param
// is an array of Timespec struct having length 2, where times[0] represents the access time
// and times[1] represents the modification time.
//
// Implements:
//
//	int ceph_futimens(struct ceph_mount_info *cmount, int fd, struct timespec times[2]);
func (f *File) Futimens(times []Timespec) error {
	if err := f.validate(); err != nil {
		return err
	}

	return f.mount.Futimens(int(f.fd), times)
}

// Futimes changes file/directory last access and modification times, here times param
// is an array of Timeval struct type having length 2, where times[0] represents the access time
// and times[1] represents the modification time.
//
// Implements:
//
//	int ceph_futimes(struct ceph_mount_info *cmount, int fd, struct timeval times[2]);
func (f *File) Futimes(times []Timeval) error {
	if err := f.validate(); err != nil {
		return err
	}

	return f.mount.Futimes(int(f.fd), times)
}
