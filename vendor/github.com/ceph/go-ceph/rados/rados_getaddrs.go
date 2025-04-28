package rados

// #cgo LDFLAGS: -lrados
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import (
	"unsafe"
)

// GetAddrs returns the addresses of the RADOS session,
// suitable for blocklisting.
//
// Implements:
//
//	int rados_getaddrs(rados_t cluster, char **addrs)
func (c *Conn) GetAddrs() (string, error) {
	var cAddrs *C.char
	defer C.free(unsafe.Pointer(cAddrs))

	ret := C.rados_getaddrs(c.cluster, &cAddrs)
	if ret < 0 {
		return "", getError(ret)
	}

	return C.GoString(cAddrs), nil
}
