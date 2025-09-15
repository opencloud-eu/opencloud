package encoding

import (
	"math"
	"reflect"

	"github.com/shamaton/msgpack/v2/def"
)

func (e *encoder) calcFixedSlice(rv reflect.Value) (int, bool) {
	// calcLength formally returns (int, error), but for map lengths in Go
	// the error case is unreachable. The error value is always nil and is
	// intentionally ignored with `_`.
	switch sli := rv.Interface().(type) {
	case []int:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcInt(int64(v))
		}
		return size, true

	case []uint:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcUint(uint64(v))
		}
		return size, true

	case []string:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcString(v)
		}
		return size, true

	case []float32:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcFloat32(float64(v))
		}
		return size, true

	case []float64:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcFloat64(v)
		}
		return size, true

	case []bool:
		size, _ := e.calcLength(len(sli))
		size += def.Byte1 * len(sli)
		return size, true

	case []int8:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcInt(int64(v))
		}
		return size, true

	case []int16:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcInt(int64(v))
		}
		return size, true

	case []int32:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcInt(int64(v))
		}
		return size, true

	case []int64:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcInt(v)
		}
		return size, true

	case []uint8:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcUint(uint64(v))
		}
		return size, true

	case []uint16:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcUint(uint64(v))
		}
		return size, true

	case []uint32:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcUint(uint64(v))
		}
		return size, true

	case []uint64:
		size, _ := e.calcLength(len(sli))
		for _, v := range sli {
			size += e.calcUint(v)
		}
		return size, true
	}

	return 0, false
}

func (e *encoder) writeSliceLength(l int, offset int) int {
	// format size
	if l <= 0x0f {
		offset = e.setByte1Int(def.FixArray+l, offset)
	} else if l <= math.MaxUint16 {
		offset = e.setByte1Int(def.Array16, offset)
		offset = e.setByte2Int(l, offset)
	} else if uint(l) <= math.MaxUint32 {
		offset = e.setByte1Int(def.Array32, offset)
		offset = e.setByte4Int(l, offset)
	}
	return offset
}

func (e *encoder) writeFixedSlice(rv reflect.Value, offset int) (int, bool) {

	switch sli := rv.Interface().(type) {
	case []int:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeInt(int64(v), offset)
		}
		return offset, true

	case []uint:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeUint(uint64(v), offset)
		}
		return offset, true

	case []string:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeString(v, offset)
		}
		return offset, true

	case []float32:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeFloat32(float64(v), offset)
		}
		return offset, true

	case []float64:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeFloat64(float64(v), offset)
		}
		return offset, true

	case []bool:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeBool(v, offset)
		}
		return offset, true

	case []int8:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeInt(int64(v), offset)
		}
		return offset, true

	case []int16:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeInt(int64(v), offset)
		}
		return offset, true

	case []int32:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeInt(int64(v), offset)
		}
		return offset, true

	case []int64:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeInt(v, offset)
		}
		return offset, true

	case []uint8:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeUint(uint64(v), offset)
		}
		return offset, true

	case []uint16:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeUint(uint64(v), offset)
		}
		return offset, true

	case []uint32:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeUint(uint64(v), offset)
		}
		return offset, true

	case []uint64:
		offset = e.writeSliceLength(len(sli), offset)
		for _, v := range sli {
			offset = e.writeUint(v, offset)
		}
		return offset, true
	}

	return offset, false
}
