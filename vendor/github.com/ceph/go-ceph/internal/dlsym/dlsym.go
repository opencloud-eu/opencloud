package dlsym

// #cgo LDFLAGS: -ldl
//
// #define _GNU_SOURCE
//
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrUndefinedSymbol is returned by LookupSymbol when the requested symbol
// could not be found.
var ErrUndefinedSymbol = errors.New("symbol not found")

// LookupSymbol resolves the named symbol from the already dynamically loaded
// libraries. If the symbol is found, a pointer to it is returned, in case of a
// failure, the message provided by dlerror() is included in the error message.
func LookupSymbol(symbol string) (unsafe.Pointer, error) {
	cSymName := C.CString(symbol)
	defer C.free(unsafe.Pointer(cSymName))

	// clear dlerror before looking up the symbol
	C.dlerror()
	// resolve the address of the symbol
	sym := C.dlsym(C.RTLD_DEFAULT, cSymName)
	e := C.dlerror()
	dlerr := C.GoString(e)
	if dlerr != "" {
		return nil, fmt.Errorf("%w: %s", ErrUndefinedSymbol, dlerr)
	}

	return sym, nil
}
