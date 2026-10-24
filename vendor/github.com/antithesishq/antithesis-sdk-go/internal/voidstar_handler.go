//go:build enable_antithesis_sdk && linux && amd64 && cgo

package internal

import (
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"
)

// --------------------------------------------------------------------------------
// To build and run an executable with this package
//
// CC=clang CGO_ENABLED=1 go run ./main.go
// --------------------------------------------------------------------------------

// \/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/
//
// The commented lines below, and the `import "C"` line which must directly follow
// the commented lines are used by CGO.  They are load-bearing, and should not be
// changed without first understanding how CGO uses them.
//
// \/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/\/

// #cgo LDFLAGS: -ldl
//
// #include <dlfcn.h>
// #include <stdbool.h>
// #include <stdint.h>
// #include <stdlib.h>
//
// typedef void (*go_fuzz_json_data_fn)(const char *data, size_t size);
// void
// go_fuzz_json_data(void *f, const char *data, size_t size) {
//   ((go_fuzz_json_data_fn)f)(data, size);
// }
//
// typedef void (*go_fuzz_flush_fn)(void);
// void
// go_fuzz_flush(void *f) {
//   ((go_fuzz_flush_fn)f)();
// }
//
// typedef uint64_t (*go_fuzz_get_random_fn)(void);
// uint64_t
// go_fuzz_get_random(void *f) {
//   return ((go_fuzz_get_random_fn)f)();
// }
//
// typedef bool (*go_notify_coverage_fn)(size_t);
// int
// go_notify_coverage(void *f, size_t edges) {
//   bool b = ((go_notify_coverage_fn)f)(edges);
//   return b ? 1 : 0;
// }
//
// typedef uint64_t (*go_init_coverage_fn)(size_t num_edges, const char *symbols);
// uint64_t
// go_init_coverage(void *f, size_t num_edges, const char *symbols) {
//   return ((go_init_coverage_fn)f)(num_edges, symbols);
// }
//
// typedef uint64_t (*go_notify_coverage_v2_fn)(size_t, uint64_t);
// uint64_t
// go_notify_coverage_v2(void *f, size_t edge, uint64_t hits) {
//   return ((go_notify_coverage_v2_fn)f)(edge, hits);
// }
//
// typedef const uint64_t *(*go_lease_generation_addr_fn)(void);
// const uint64_t *
// go_lease_generation_addr(void *f) {
//   return ((go_lease_generation_addr_fn)f)();
// }
//
// typedef uint64_t (*go_request_abi_version_fn)(uint64_t);
// uint64_t
// go_request_abi_version(void *f, uint64_t v) {
//   return ((go_request_abi_version_fn)f)(v);
// }
//
import "C"

type voidstarHandler struct {
	fuzzJsonData   unsafe.Pointer
	fuzzFlush      unsafe.Pointer
	fuzzGetRandom  unsafe.Pointer
	initCoverage   unsafe.Pointer
	notifyCoverage unsafe.Pointer
	// Coverage-lease support (linkage ABI v2, instrumentation.h). Nil/zero
	// when the loaded library predates the mechanism or negotiation failed.
	notifyCoverageV2 unsafe.Pointer
	leaseGeneration  *uint64
}

func (h *voidstarHandler) output(message string) {
	msg_len := len(message)
	if msg_len == 0 {
		return
	}
	cstrMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cstrMessage))
	C.go_fuzz_json_data(h.fuzzJsonData, cstrMessage, C.ulong(msg_len))
	C.go_fuzz_flush(h.fuzzFlush)
}

func (h *voidstarHandler) random() uint64 {
	return uint64(C.go_fuzz_get_random(h.fuzzGetRandom))
}

func (h *voidstarHandler) init_coverage(num_edge uint64, symbols string) uint64 {
	cstrSymbols := C.CString(symbols)
	defer C.free(unsafe.Pointer(cstrSymbols))
	return uint64(C.go_init_coverage(h.initCoverage, C.ulong(num_edge), cstrSymbols))
}

func (h *voidstarHandler) notify(edge uint64) bool {
	ival := int(C.go_notify_coverage(h.notifyCoverage, C.ulong(edge)))
	return ival == 1
}

func (h *voidstarHandler) notify_v2(edge uint64, hits uint64) (uint64, bool) {
	if h.notifyCoverageV2 == nil {
		return 0, false
	}
	return uint64(C.go_notify_coverage_v2(h.notifyCoverageV2, C.ulong(edge), C.uint64_t(hits))), true
}

func (h *voidstarHandler) lease_generation() (uint64, bool) {
	if h.leaseGeneration == nil {
		return 0, false
	}
	// The header requires this read to be non-hoistable out of hot loops;
	// an atomic load has the required opaque semantics. The word lives for
	// the life of the process and is only ever written by the fuzzer.
	return atomic.LoadUint64(h.leaseGeneration), true
}

// Attempt to load libvoidstar and some symbols from `path`
func openSharedLib(path string) (*voidstarHandler, error) {
	cstrPath := C.CString(path)
	defer C.free(unsafe.Pointer(cstrPath))

	dlError := func(message string) error {
		return fmt.Errorf("%s: (%s)", message, C.GoString(C.dlerror()))
	}

	sharedLib := C.dlopen(cstrPath, C.int(C.RTLD_NOW))
	if sharedLib == nil {
		return nil, dlError("Can not load the Antithesis native library")
	}

	loadFunc := func(name string) (symbol unsafe.Pointer, err error) {
		cstrName := C.CString(name)
		defer C.free(unsafe.Pointer(cstrName))
		if symbol = C.dlsym(sharedLib, cstrName); symbol == nil {
			err = dlError(fmt.Sprintf("Can not access symbol %s", name))
		}
		return
	}

	// Symbols absent from older library vintages must resolve softly: no
	// library version may fail the SDK's load (instrumentation.h, "Linkage
	// ABI versioning").
	loadOptional := func(name string) unsafe.Pointer {
		cstrName := C.CString(name)
		defer C.free(unsafe.Pointer(cstrName))
		symbol := C.dlsym(sharedLib, cstrName)
		C.dlerror() // clear any lookup error
		return symbol
	}

	fuzzJsonData, err := loadFunc("fuzz_json_data")
	if err != nil {
		return nil, err
	}
	fuzzFlush, err := loadFunc("fuzz_flush")
	if err != nil {
		return nil, err
	}
	fuzzGetRandom, err := loadFunc("fuzz_get_random")
	if err != nil {
		return nil, err
	}
	notifyCoverage, err := loadFunc("notify_coverage")
	if err != nil {
		return nil, err
	}
	initCoverage, err := loadFunc("init_coverage_module")
	if err != nil {
		return nil, err
	}
	handler := &voidstarHandler{
		fuzzJsonData:   fuzzJsonData,
		fuzzFlush:      fuzzFlush,
		fuzzGetRandom:  fuzzGetRandom,
		initCoverage:   initCoverage,
		notifyCoverage: notifyCoverage,
	}

	// Negotiate for coverage leases: request ABI v2 and gate on the granted
	// version (a grant above the request means "cannot negotiate").
	requestAbi := loadOptional("instrumentation_request_abi_version")
	notifyV2 := loadOptional("notify_coverage_v2")
	leaseAddr := loadOptional("coverage_lease_generation_addr")
	if requestAbi != nil && notifyV2 != nil && leaseAddr != nil {
		if granted := uint64(C.go_request_abi_version(requestAbi, 2)); granted == 2 {
			if addr := C.go_lease_generation_addr(leaseAddr); addr != nil {
				handler.notifyCoverageV2 = notifyV2
				handler.leaseGeneration = (*uint64)(unsafe.Pointer(addr))
			}
		}
	}
	return handler, nil
}

const sdkRunningInDegradedMode = false

// If we have a file at `defaultNativeLibraryPath`, we load the shared library
// (and panic on any error encountered during load).
func init_in_antithesis() libHandler {
	if _, err := os.Stat(defaultNativeLibraryPath); err == nil {
		handler, err := openSharedLib(defaultNativeLibraryPath)
		if err != nil {
			panic(err)
		}
		return handler
	}
	return nil
}
