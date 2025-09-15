package encoding

import (
	"fmt"
	"math"
	"reflect"

	"github.com/shamaton/msgpack/v2/def"
	"github.com/shamaton/msgpack/v2/internal/common"
)

type encoder struct {
	d       []byte
	asArray bool
	common.Common
	mk map[uintptr][]reflect.Value
	mv map[uintptr][]reflect.Value
}

// Encode returns the MessagePack-encoded byte array of v.
func Encode(v interface{}, asArray bool) (b []byte, err error) {
	e := encoder{asArray: asArray}
	/*
		defer func() {
			e := recover()
			if e != nil {
				b = nil
				err = fmt.Errorf("unexpected error!! \n%s", stackTrace())
			}
		}()
	*/

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
	}
	size, err := e.calcSize(rv)
	if err != nil {
		return nil, err
	}

	e.d = make([]byte, size)
	last := e.create(rv, 0)
	if size != last {
		return nil, fmt.Errorf("%w size=%d, lastIdx=%d", def.ErrNotMatchLastIndex, size, last)
	}
	return e.d, err
}

//func stackTrace() string {
//	msg := ""
//	for depth := 0; ; depth++ {
//		_, file, line, ok := runtime.Caller(depth)
//		if !ok {
//			break
//		}
//		msg += fmt.Sprintln(depth, ": ", file, ":", line)
//	}
//	return msg
//}

func (e *encoder) calcSize(rv reflect.Value) (int, error) {
	switch rv.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v := rv.Uint()
		return e.calcUint(v), nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		v := rv.Int()
		return e.calcInt(int64(v)), nil

	case reflect.Float32:
		return e.calcFloat32(0), nil

	case reflect.Float64:
		return e.calcFloat64(0), nil

	case reflect.String:
		return e.calcString(rv.String()), nil

	case reflect.Bool:
		return def.Byte1, nil

	case reflect.Complex64:
		return e.calcComplex64(), nil

	case reflect.Complex128:
		return e.calcComplex128(), nil

	case reflect.Slice:
		if rv.IsNil() {
			return def.Byte1, nil
		}
		// bin format
		if e.isByteSlice(rv) {
			size, err := e.calcByteSlice(rv.Len())
			if err != nil {
				return 0, err
			}
			return size, nil
		}

		if size, find := e.calcFixedSlice(rv); find {
			return size, nil
		}

		// func
		elem := rv.Type().Elem()
		var f structCalcFunc
		if elem.Kind() == reflect.Struct {
			f = e.getStructCalc(elem)
		} else {
			f = e.calcSize
		}

		l := rv.Len()
		size, err := e.calcLength(l)
		if err != nil {
			return 0, err
		}

		// objects size
		for i := 0; i < l; i++ {
			s, err := f(rv.Index(i))
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil

	case reflect.Array:
		// bin format
		if e.isByteSlice(rv) {
			size, err := e.calcByteSlice(rv.Len())
			if err != nil {
				return 0, err
			}
			return size, nil
		}

		// func
		elem := rv.Type().Elem()
		var f structCalcFunc
		if elem.Kind() == reflect.Struct {
			f = e.getStructCalc(elem)
		} else {
			f = e.calcSize
		}

		l := rv.Len()
		size, err := e.calcLength(l)
		if err != nil {
			return 0, err
		}

		// objects size
		for i := 0; i < l; i++ {
			s, err := f(rv.Index(i))
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil

	case reflect.Map:
		if rv.IsNil() {
			return def.Byte1, nil
		}

		if size, find := e.calcFixedMap(rv); find {
			return size, nil
		}

		if e.mk == nil {
			e.mk = map[uintptr][]reflect.Value{}
			e.mv = map[uintptr][]reflect.Value{}
		}

		keys := rv.MapKeys()
		size, err := e.calcLength(len(keys))
		if err != nil {
			return 0, err
		}

		// key-value
		mv := make([]reflect.Value, len(keys))
		i := 0
		for _, k := range keys {
			keySize, err := e.calcSize(k)
			if err != nil {
				return 0, err
			}
			value := rv.MapIndex(k)
			valueSize, err := e.calcSize(value)
			if err != nil {
				return 0, err
			}
			size += keySize + valueSize
			mv[i] = value
			i++
		}
		e.mk[rv.Pointer()], e.mv[rv.Pointer()] = keys, mv
		return size, nil

	case reflect.Struct:
		size, err := e.calcStruct(rv)
		if err != nil {
			return 0, err
		}
		return size, nil

	case reflect.Ptr:
		if rv.IsNil() {
			return def.Byte1, nil
		}
		size, err := e.calcSize(rv.Elem())
		if err != nil {
			return 0, err
		}
		return size, nil

	case reflect.Interface:
		size, err := e.calcSize(rv.Elem())
		if err != nil {
			return 0, err
		}
		return size, nil

	case reflect.Invalid:
		// do nothing (return nil)
		return def.Byte1, nil

	default:
		return 0, fmt.Errorf("%v is %w type", rv.Kind(), def.ErrUnsupportedType)
	}
}

func (e *encoder) calcLength(l int) (int, error) {
	if l <= 0x0f {
		return def.Byte1, nil
	} else if l <= math.MaxUint16 {
		return def.Byte1 + def.Byte2, nil
	} else if uint(l) <= math.MaxUint32 {
		return def.Byte1 + def.Byte4, nil
	}
	// not supported error
	return 0, fmt.Errorf("array length %d is %w", l, def.ErrUnsupportedLength)
}

func (e *encoder) create(rv reflect.Value, offset int) int {

	switch rv.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v := rv.Uint()
		offset = e.writeUint(v, offset)

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		v := rv.Int()
		offset = e.writeInt(v, offset)

	case reflect.Float32:
		offset = e.writeFloat32(rv.Float(), offset)

	case reflect.Float64:
		offset = e.writeFloat64(rv.Float(), offset)

	case reflect.Bool:
		offset = e.writeBool(rv.Bool(), offset)

	case reflect.String:
		offset = e.writeString(rv.String(), offset)

	case reflect.Complex64:
		offset = e.writeComplex64(complex64(rv.Complex()), offset)

	case reflect.Complex128:
		offset = e.writeComplex128(rv.Complex(), offset)

	case reflect.Slice:
		if rv.IsNil() {
			return e.writeNil(offset)
		}

		// bin format
		if e.isByteSlice(rv) {
			offset = e.writeByteSliceLength(rv.Len(), offset)
			offset = e.setBytes(rv.Bytes(), offset)
			return offset
		}

		if offset, find := e.writeFixedSlice(rv, offset); find {
			return offset
		}

		// func
		elem := rv.Type().Elem()
		var f structWriteFunc
		if elem.Kind() == reflect.Struct {
			f = e.getStructWriter(elem)
		} else {
			f = e.create
		}

		// objects
		l := rv.Len()
		offset = e.writeSliceLength(l, offset)
		for i := 0; i < l; i++ {
			offset = f(rv.Index(i), offset)
		}

	case reflect.Array:
		l := rv.Len()
		// bin format
		if e.isByteSlice(rv) {
			offset = e.writeByteSliceLength(l, offset)
			// objects
			for i := 0; i < l; i++ {
				offset = e.setByte1Uint64(rv.Index(i).Uint(), offset)
			}
			return offset
		}

		// format
		offset = e.writeSliceLength(l, offset)

		// func
		elem := rv.Type().Elem()
		var f structWriteFunc
		if elem.Kind() == reflect.Struct {
			f = e.getStructWriter(elem)
		} else {
			f = e.create
		}

		// objects
		for i := 0; i < l; i++ {
			offset = f(rv.Index(i), offset)
		}

	case reflect.Map:
		if rv.IsNil() {
			return e.writeNil(offset)
		}

		l := rv.Len()
		offset = e.writeMapLength(l, offset)

		if offset, find := e.writeFixedMap(rv, offset); find {
			return offset
		}

		// key-value
		p := rv.Pointer()
		for i := range e.mk[p] {
			offset = e.create(e.mk[p][i], offset)
			offset = e.create(e.mv[p][i], offset)
		}

	case reflect.Struct:
		offset = e.writeStruct(rv, offset)

	case reflect.Ptr:
		if rv.IsNil() {
			return e.writeNil(offset)
		}

		offset = e.create(rv.Elem(), offset)

	case reflect.Interface:
		offset = e.create(rv.Elem(), offset)

	case reflect.Invalid:
		return e.writeNil(offset)

	}
	return offset
}
