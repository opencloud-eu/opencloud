package rados

// #cgo LDFLAGS: -lrados
// #include <errno.h>
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/ceph/go-ceph/internal/log"
)

const (
	// AllNamespaces is used to reset a selected namespace to all
	// namespaces. See the IOContext SetNamespace function.
	AllNamespaces = C.LIBRADOS_ALL_NSPACES

	// FIXME: for backwards compatibility

	// RadosAllNamespaces is used to reset a selected namespace to all
	// namespaces. See the IOContext SetNamespace function.
	//
	// Deprecated: use AllNamespaces instead
	RadosAllNamespaces = AllNamespaces
)

// OpFlags are flags that can be set on a per-op basis.
type OpFlags uint

const (
	// OpFlagNone can be use to not set any flags.
	OpFlagNone = OpFlags(0)
	// OpFlagExcl marks an op to fail a create operation if the object
	// already exists.
	OpFlagExcl = OpFlags(C.LIBRADOS_OP_FLAG_EXCL)
	// OpFlagFailOk allows the transaction to succeed even if the flagged
	// op fails.
	OpFlagFailOk = OpFlags(C.LIBRADOS_OP_FLAG_FAILOK)
	// OpFlagFAdviseRandom indicates read/write op random.
	OpFlagFAdviseRandom = OpFlags(C.LIBRADOS_OP_FLAG_FADVISE_RANDOM)
	// OpFlagFAdviseSequential indicates read/write op sequential.
	OpFlagFAdviseSequential = OpFlags(C.LIBRADOS_OP_FLAG_FADVISE_SEQUENTIAL)
	// OpFlagFAdviseWillNeed indicates read/write data will be accessed in
	// the near future (by someone).
	OpFlagFAdviseWillNeed = OpFlags(C.LIBRADOS_OP_FLAG_FADVISE_WILLNEED)
	// OpFlagFAdviseDontNeed indicates read/write data will not accessed in
	// the near future (by anyone).
	OpFlagFAdviseDontNeed = OpFlags(C.LIBRADOS_OP_FLAG_FADVISE_DONTNEED)
	// OpFlagFAdviseNoCache indicates read/write data will not accessed
	// again (by *this* client).
	OpFlagFAdviseNoCache = OpFlags(C.LIBRADOS_OP_FLAG_FADVISE_NOCACHE)
)

// Version returns the major, minor, and patch components of the version of
// the RADOS library linked against.
func Version() (int, int, int) {
	var cMajor, cMinor, cPatch C.int
	C.rados_version(&cMajor, &cMinor, &cPatch)
	return int(cMajor), int(cMinor), int(cPatch)
}

func makeConn() *Conn {
	return &Conn{connected: false}
}

func newConn(user *C.char) (*Conn, error) {
	conn := makeConn()
	ret := C.rados_create(&conn.cluster, user)

	if ret != 0 {
		return nil, getError(ret)
	}

	runtime.SetFinalizer(conn, freeConn)
	return conn, nil
}

// NewConn creates a new connection object. It returns the connection and an
// error, if any.
func NewConn() (*Conn, error) {
	return newConn(nil)
}

// NewConnWithUser creates a new connection object with a custom username.
// The supplied username should not include a 'client.' prefix. It returns
// the connection and an error, if any.
func NewConnWithUser(user string) (*Conn, error) {
	cUser := C.CString(user)
	defer C.free(unsafe.Pointer(cUser))
	return newConn(cUser)
}

// NewConnWithClusterAndUser creates a new connection object for a specific
// cluster and username. The supplied username should be in the 'client.<user>'
// format. It returns the connection and an error, if any.
func NewConnWithClusterAndUser(clusterName string, userName string) (*Conn, error) {
	cClusterName := C.CString(clusterName)
	defer C.free(unsafe.Pointer(cClusterName))

	cName := C.CString(userName)
	defer C.free(unsafe.Pointer(cName))

	conn := makeConn()
	ret := C.rados_create2(&conn.cluster, cClusterName, cName, 0)
	if ret != 0 {
		return nil, getError(ret)
	}

	runtime.SetFinalizer(conn, freeConn)
	return conn, nil
}

// freeConn releases resources that are allocated while configuring the
// connection to the cluster. rados_shutdown() should only be needed after a
// successful call to rados_connect(), however if the connection has been
// configured with non-default parameters, some of the parameters may be
// allocated before connecting. rados_shutdown() will free the allocated
// resources, even if there has not been a connection yet.
//
// This function is setup as a destructor/finalizer when rados_create() is
// called.
func freeConn(conn *Conn) {
	if conn.cluster != nil {
		log.Warnf("unreachable Conn object has not been shut down. Cleaning up.")
		C.rados_shutdown(conn.cluster)
		// prevent calling rados_shutdown() more than once
		conn.cluster = nil
	}
}
