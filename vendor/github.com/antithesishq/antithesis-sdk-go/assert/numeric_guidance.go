//go:build enable_antithesis_sdk

package assert

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// --------------------------------------------------------------------------------
// IntegerGap is used for:
// - int, int8, int16, int32, int64:
// - uint, uint8, uint16, uint32, uint64, uintptr:
//
// FloatGap is used for:
// - float32, float64
// --------------------------------------------------------------------------------
type numericGapType int

const (
	integerGapType numericGapType = iota
	floatGapType
)

func gapTypeForOperand[T Number](num T) numericGapType {
	gapType := integerGapType

	switch any(num).(type) {
	case float32, float64:
		gapType = floatGapType
	}
	return gapType
}

// --------------------------------------------------------------------------------
// numericGuidanceTracker - Tracking Info for Numeric Guidance
//
// For GuidanceFnMaximize:
//   - gap is the largest value sent so far
//
// For GuidanceFnMinimize:
//   - gap is the most negative value sent so far
//
// --------------------------------------------------------------------------------
type numericGuidanceInfo struct {
	descriminator numericGapType
	maximize      bool
	// The best gap sent so far (a gapValue[uint64] or gapValue[float64],
	// fixed per entry). Read lock-free on every evaluation; updated under mu.
	gap atomic.Value
	// Serializes improvements (compare, update, emit) so guidance emissions
	// form an ordered monotone chain; the no-improvement fast path never
	// takes it.
	mu sync.Mutex
}

var numericGuidanceTracker sync.Map // message key -> *numericGuidanceInfo

func getNumericGuidanceEntry(messageKey string, trackerType numericGapType, maximize bool) *numericGuidanceInfo {
	if entry, ok := numericGuidanceTracker.Load(messageKey); ok {
		return entry.(*numericGuidanceInfo)
	}
	entry, _ := numericGuidanceTracker.LoadOrStore(messageKey,
		newNumericGuidanceInfo(trackerType, maximize))
	return entry.(*numericGuidanceInfo)
}

// Create an numeric guidance entry
func newNumericGuidanceInfo(trackerType numericGapType, maximize bool) *numericGuidanceInfo {

	var gap any
	if trackerType == integerGapType {
		gap = newGapValue(uint64(math.MaxUint64), maximize)
	} else {
		gap = newGapValue(float64(math.MaxFloat64), maximize)
	}
	trackerInfo := numericGuidanceInfo{
		maximize:      maximize,
		descriminator: trackerType,
	}
	trackerInfo.gap.Store(gap)
	return &trackerInfo
}

func (tI *numericGuidanceInfo) should_maximize() bool {
	return tI.maximize
}

func (tI *numericGuidanceInfo) is_integer_gap() bool {
	return tI.descriminator == integerGapType
}

// --------------------------------------------------------------------------------
// Represents integral and floating point extremes
// --------------------------------------------------------------------------------
type gapValue[T numConstraint] struct {
	gap_size        T
	gap_is_negative bool
}

func newGapValue[T numConstraint](sz T, is_neg bool) any {
	switch any(sz).(type) {
	case uint64:
		return gapValue[uint64]{gap_size: uint64(sz), gap_is_negative: is_neg}

	case float64:
		return gapValue[float64]{gap_size: float64(sz), gap_is_negative: is_neg}
	}
	return nil
}

func is_same_sign(left_val int64, right_val int64) bool {
	same_sign := false
	if left_val < 0 {
		// left is negative
		if right_val < 0 {
			same_sign = true
		}
	} else {
		// left is non-negative
		if right_val >= 0 {
			same_sign = true
		}
	}
	return same_sign
}

func abs_int64(val int64) uint64 {
	if val >= 0 {
		return uint64(val)
	}
	return uint64(0 - val)
}

func is_greater_than[T numConstraint](left gapValue[T], right gapValue[T]) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size > right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return true // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size > left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return false // any negative is less than a positive
	}
	return false
}

func is_less_than[T numConstraint](left gapValue[T], right gapValue[T]) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size < right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return false // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size < left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return true // any negative is less than a positive
	}
	return true
}

// The candidate gap for a pair of guidance operands, in the tracker's
// representation (nil when the operand type resolves to none).
func candidateGap(data any) any {
	// Needs to have individual case statements to assist
	// the compiler to infer the actual type of the var named 'operands'
	switch operands := data.(type) {
	case numericOperands[int32]:
		return makeGap(operands)
	case numericOperands[int64]:
		return makeGap(operands)
	case numericOperands[uint64]:
		return makeGap(operands)
	case numericOperands[float64]:
		return makeFloatGap(operands)
	}
	return nil
}

// Whether candidate improves on the entry's tracked extremum.
func (tI *numericGuidanceInfo) improves(candidate any) bool {
	maximize := tI.should_maximize()
	switch prev := tI.gap.Load().(type) {
	case gapValue[uint64]:
		if gap, ok := candidate.(gapValue[uint64]); ok {
			if maximize {
				return is_greater_than(gap, prev)
			}
			return is_less_than(gap, prev)
		}
	case gapValue[float64]:
		if gap, ok := candidate.(gapValue[float64]); ok {
			if maximize {
				return is_greater_than(gap, prev)
			}
			return is_less_than(gap, prev)
		}
	}
	return false
}

func send_value_if_needed(tI *numericGuidanceInfo, gI *guidanceInfo) {
	if tI == nil {
		return
	}

	// if this is a catalog entry (gI.hit is false)
	// do not update the reference gap in the tracker (tI *numericGuidanceInfo)
	if !gI.Hit {
		emitGuidance(gI)
		return
	}

	// Most evaluations do not improve on the tracked extremum; that path is
	// a lock-free load and compare. The mutex serializes only improvements.
	candidate := candidateGap(gI.Data)
	if !tI.improves(candidate) {
		return
	}
	tI.mu.Lock()
	defer tI.mu.Unlock()
	if !tI.improves(candidate) {
		return
	}
	tI.gap.Store(candidate)
	emitGuidance(gI)
}

func emitGuidance(gI *guidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": gI})
}

// When left and right are the same sign (both negative, or both non-negative)
// Calculate: <result> = (left - right).  The gap_size is abs(<result>) and
// gap_is_negative is (right > left)
func makeGap[Op operandConstraint](operand numericOperands[Op]) gapValue[uint64] {

	var gap_size uint64
	var gap_is_negative bool

	switch this_op := any(operand).(type) {
	case numericOperands[int32]:
		result := int64(this_op.Left) - int64(this_op.Right)
		gap_size = abs_int64(result)
		gap_is_negative = result < 0

	case numericOperands[int64]:
		if is_same_sign(this_op.Left, this_op.Right) {
			result := int64(this_op.Left) - int64(this_op.Right)
			gap_size = abs_int64(result)
			gap_is_negative = result < 0
			break
		}

		// Otherwise left and right are opposite signs
		// gap = abs(left) + abs(right)
		// gap_is_negative = left < right
		left_gap_size := abs_int64(this_op.Left)
		right_gap_size := abs_int64(this_op.Right)
		gap_size = left_gap_size + right_gap_size
		gap_is_negative = this_op.Left < this_op.Right

	case numericOperands[uint64]:
		left_val := this_op.Left
		right_val := this_op.Right
		gap_is_negative = false
		if left_val < right_val {
			gap_is_negative = true
			gap_size = right_val - left_val
		} else {
			gap_size = left_val - right_val
		}

	default:
		zero_gap, _ := newGapValue(uint64(0), false).(gapValue[uint64])
		return zero_gap
	}

	this_gap, _ := newGapValue(gap_size, gap_is_negative).(gapValue[uint64])
	return this_gap
} // MakeGap

func makeFloatGap[Op operandConstraint](operand numericOperands[Op]) gapValue[float64] {
	switch this_op := any(operand).(type) {
	case numericOperands[float64]:
		left_val := this_op.Left
		right_val := this_op.Right
		gap_is_negative := false
		var gap_size float64
		if left_val < right_val {
			gap_is_negative = true
			gap_size = right_val - left_val
		} else {
			gap_size = left_val - right_val
		}

		this_gap, _ := newGapValue(gap_size, gap_is_negative).(gapValue[float64])
		return this_gap

	default:
		zero_gap, _ := newGapValue(float64(0.0), false).(gapValue[float64])
		return zero_gap
	}
} // MakeFloatGap
