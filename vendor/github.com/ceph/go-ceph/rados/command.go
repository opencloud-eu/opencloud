package rados

// #cgo LDFLAGS: -lrados
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import (
	"unsafe"

	"github.com/ceph/go-ceph/internal/cutil"
)

func radosBufferFree(p unsafe.Pointer) {
	C.rados_buffer_free((*C.char)(p))
}

// MonCommand sends a command to one of the monitors
func (c *Conn) MonCommand(args []byte) ([]byte, string, error) {
	return c.MonCommandWithInputBuffer(args, nil)
}

// MonCommandWithInputBuffer sends a command to one of the monitors, with an input buffer
func (c *Conn) MonCommandWithInputBuffer(args, inputBuffer []byte) ([]byte, string, error) {
	ci := cutil.NewCommandInput([][]byte{args}, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(radosBufferFree)
	defer co.Free()

	ret := C.rados_mon_command(
		c.cluster,
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}

// PGCommand sends a command to one of the PGs
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
//
// Implements:
//
//	int rados_pg_command(rados_t cluster, const char *pgstr,
//	                     const char **cmd, size_t cmdlen,
//	                     const char *inbuf, size_t inbuflen,
//	                     char **outbuf, size_t *outbuflen,
//	                     char **outs, size_t *outslen);
func (c *Conn) PGCommand(pgid []byte, args [][]byte) ([]byte, string, error) {
	return c.PGCommandWithInputBuffer(pgid, args, nil)
}

// PGCommandWithInputBuffer sends a command to one of the PGs, with an input buffer
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
//
// Implements:
//
//	int rados_pg_command(rados_t cluster, const char *pgstr,
//	                     const char **cmd, size_t cmdlen,
//	                     const char *inbuf, size_t inbuflen,
//	                     char **outbuf, size_t *outbuflen,
//	                     char **outs, size_t *outslen);
func (c *Conn) PGCommandWithInputBuffer(pgid []byte, args [][]byte, inputBuffer []byte) ([]byte, string, error) {
	name := C.CString(string(pgid))
	defer C.free(unsafe.Pointer(name))
	ci := cutil.NewCommandInput(args, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(radosBufferFree)
	defer co.Free()

	ret := C.rados_pg_command(
		c.cluster,
		name,
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}

// MgrCommand sends a command to a ceph-mgr.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
func (c *Conn) MgrCommand(args [][]byte) ([]byte, string, error) {
	return c.MgrCommandWithInputBuffer(args, nil)
}

// MgrCommandWithInputBuffer sends a command, with an input buffer, to a ceph-mgr.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
//
// Implements:
//
//	int rados_mgr_command(rados_t cluster, const char **cmd,
//	                      size_t cmdlen, const char *inbuf,
//	                      size_t inbuflen, char **outbuf,
//	                      size_t *outbuflen, char **outs,
//	                      size_t *outslen);
func (c *Conn) MgrCommandWithInputBuffer(args [][]byte, inputBuffer []byte) ([]byte, string, error) {
	ci := cutil.NewCommandInput(args, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(radosBufferFree)
	defer co.Free()

	ret := C.rados_mgr_command(
		c.cluster,
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}

// OsdCommand sends a command to the specified ceph OSD.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
func (c *Conn) OsdCommand(osd int, args [][]byte) ([]byte, string, error) {
	return c.OsdCommandWithInputBuffer(osd, args, nil)
}

// OsdCommandWithInputBuffer sends a command, with an input buffer, to the
// specified ceph OSD.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
//
// Implements:
//
//	int rados_osd_command(rados_t cluster, int osdid,
//	                                     const char **cmd, size_t cmdlen,
//	                                     const char *inbuf, size_t inbuflen,
//	                                     char **outbuf, size_t *outbuflen,
//	                                     char **outs, size_t *outslen);
func (c *Conn) OsdCommandWithInputBuffer(
	osd int, args [][]byte, inputBuffer []byte) ([]byte, string, error) {

	ci := cutil.NewCommandInput(args, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(radosBufferFree)
	defer co.Free()

	ret := C.rados_osd_command(
		c.cluster,
		C.int(osd),
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}

// MonCommandTarget sends a command to a specified monitor.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
func (c *Conn) MonCommandTarget(name string, args [][]byte) ([]byte, string, error) {
	return c.MonCommandTargetWithInputBuffer(name, args, nil)
}

// MonCommandTargetWithInputBuffer sends a command, with an input buffer, to a specified monitor.
//
// The args parameter takes a slice of byte slices but typically a single
// slice element is sufficient. The use of two slices exists to best match
// the structure of the underlying C call which is often a legacy interface
// in Ceph.
//
// Implements:
//
//	int rados_mon_command_target(rados_t cluster, const char *name,
//	                             const char **cmd, size_t cmdlen,
//	                             const char *inbuf, size_t inbuflen,
//	                             char **outbuf, size_t *outbuflen,
//	                             char **outs, size_t *outslen);
func (c *Conn) MonCommandTargetWithInputBuffer(
	name string, args [][]byte, inputBuffer []byte) ([]byte, string, error) {

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	ci := cutil.NewCommandInput(args, inputBuffer)
	defer ci.Free()
	co := cutil.NewCommandOutput().SetFreeFunc(radosBufferFree)
	defer co.Free()

	ret := C.rados_mon_command_target(
		c.cluster,
		cName,
		(**C.char)(ci.Cmd()),
		C.size_t(ci.CmdLen()),
		(*C.char)(ci.InBuf()),
		C.size_t(ci.InBufLen()),
		(**C.char)(co.OutBuf()),
		(*C.size_t)(co.OutBufLen()),
		(**C.char)(co.Outs()),
		(*C.size_t)(co.OutsLen()))
	buf, status := co.GoValues()
	return buf, status, getError(ret)
}
