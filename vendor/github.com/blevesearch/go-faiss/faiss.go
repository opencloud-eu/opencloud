// Package faiss provides bindings to Faiss, a library for vector similarity
// search.
// More detailed documentation can be found at the Faiss wiki:
// https://github.com/facebookresearch/faiss/wiki.
package faiss

/*
#cgo LDFLAGS: -lfaiss_c

#include <faiss/c_api/Index_c.h>
#include <faiss/c_api/utils/distances_c.h>
#include <faiss/c_api/utils/utils_c.h>
*/
import "C"

// Metric type
const (
	MetricInnerProduct  = C.METRIC_INNER_PRODUCT
	MetricL2            = C.METRIC_L2
	MetricL1            = C.METRIC_L1
	MetricLinf          = C.METRIC_Linf
	MetricLp            = C.METRIC_Lp
	MetricCanberra      = C.METRIC_Canberra
	MetricBrayCurtis    = C.METRIC_BrayCurtis
	MetricJensenShannon = C.METRIC_JensenShannon
)

// In-place normalization of provided vector (single)
func NormalizeVector(vector []float32) []float32 {
	C.faiss_fvec_renorm_L2(
		C.size_t(len(vector)),
		1, // number of vectors
		(*C.float)(&vector[0]))

	return vector
}

// RealToBinary converts n real-valued vectors into binary vectors.
// Each output bit is 1 if the corresponding input value is > 0,
// and 0 otherwise. d must be a multiple of 8.
// The returned slice has length n * (d / 8).
func RealToBinary(x []float32, d int) []uint8 {
	n := len(x) / d
	out := make([]uint8, n*(d/8))
	C.faiss_real_to_binary(
		C.size_t(n),
		C.size_t(d),
		(*C.float)(&x[0]),
		(*C.uint8_t)(&out[0]),
	)
	return out
}
