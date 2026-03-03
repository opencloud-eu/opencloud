package vips

// #include <vips/vips.h>
import "C"

import "unsafe"

// NewInterpolate creates a VipsInterpolate from a name string.
// Common names: "nearest", "bilinear", "bicubic", "nohalo", "vsqbs", "lbb".
// The caller should call g_object_unref on the result when done, or pass it
// to a generated operation (which does not take ownership).
func NewInterpolate(name string) (*C.VipsInterpolate, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	interp := C.vips_interpolate_new(cName)
	if interp == nil {
		return nil, handleVipsError()
	}
	return interp, nil
}
