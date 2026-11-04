package decoding

import (
	"fmt"
	"io"
	"reflect"

	"github.com/shamaton/msgpack/v2/def"
	"github.com/shamaton/msgpack/v2/internal/common"
	"github.com/shamaton/msgpack/v2/internal/common/decodingutil"
)

type decoder struct {
	r       io.Reader
	buf     *common.Buffer
	asArray bool
	depth   int32
	common.Common
}

// Decode analyzes the MessagePack-encoded data and stores
// the result into the pointer of v.
func Decode(r io.Reader, v interface{}, asArray bool) error {
	if r == nil {
		return def.ErrNoData
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%w. v.(type): %T", def.ErrReceiverNotPointer, v)
	}

	rv = rv.Elem()

	d := decoder{
		r:       r,
		buf:     common.GetBuffer(),
		asArray: asArray,
	}
	err := d.decode(rv)
	common.PutBuffer(d.buf)
	return err
}

func (d *decoder) decode(rv reflect.Value) error {
	code, err := d.readSize1()
	if err != nil {
		return err
	}
	return d.decodeWithCode(code, rv)
}

func (d *decoder) decodeWithCode(code byte, rv reflect.Value) error {
	k := rv.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := d.asIntWithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := d.asUintWithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetUint(v)

	case reflect.Float32:
		v, err := d.asFloat32WithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetFloat(float64(v))

	case reflect.Float64:
		v, err := d.asFloat64WithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetFloat(v)

	case reflect.String:
		// byte slice
		if d.isCodeBin(code) {
			v, err := d.asBinStringWithCode(code, k)
			if err != nil {
				return err
			}
			rv.SetString(v)
			return nil
		}
		v, err := d.asStringWithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetString(v)

	case reflect.Bool:
		v, err := d.asBoolWithCode(code, k)
		if err != nil {
			return err
		}
		rv.SetBool(v)

	case reflect.Slice:
		// nil
		if d.isCodeNil(code) {
			return nil
		}
		// byte slice
		if d.isCodeBin(code) {
			bs, err := d.asBinWithCode(code, k)
			if err != nil {
				return err
			}
			rv.SetBytes(bs)
			return nil
		}
		// string to bytes
		if d.isCodeString(code) {
			l, err := d.stringByteLength(code, k)
			if err != nil {
				return err
			}
			bs, err := d.asStringByteByLength(l, k)
			if err != nil {
				return err
			}
			rv.SetBytes(bs)
			return nil
		}

		// get slice length
		l, err := d.sliceLength(code, k)
		if err != nil {
			return err
		}

		// check fixed type
		found, err := d.asFixedSlice(rv, l)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		// create slice dynamically
		if err = d.enter(); err != nil {
			return err
		}
		tmpSlice := reflect.MakeSlice(rv.Type(), 0, initialSliceCap(l, rv.Type().Elem()))
		for i := 0; i < l; i++ {
			tmpSlice = reflect.Append(tmpSlice, reflect.Zero(rv.Type().Elem()))
			v := tmpSlice.Index(i)
			if v.Kind() == reflect.Struct {
				structCode, err := d.readSize1()
				if err != nil {
					return err
				}
				if err = d.setStruct(structCode, v, k); err != nil {
					return err
				}
			} else {
				if err = d.decode(v); err != nil {
					return err
				}
			}
		}
		rv.Set(tmpSlice)
		d.depth--

	case reflect.Complex64:
		v, err := d.asComplex64(code, k)
		if err != nil {
			return err
		}
		rv.SetComplex(complex128(v))

	case reflect.Complex128:
		v, err := d.asComplex128(code, k)
		if err != nil {
			return err
		}
		rv.SetComplex(v)

	case reflect.Array:
		// nil
		if d.isCodeNil(code) {
			return nil
		}
		// byte slice
		if d.isCodeBin(code) {
			bs, err := d.asBinWithCode(code, k)
			if err != nil {
				return err
			}
			if len(bs) > rv.Len() {
				return fmt.Errorf("%v len is %d, but msgpack has %d elements, %w", rv.Type(), rv.Len(), len(bs), def.ErrNotMatchArrayElement)
			}
			for i, b := range bs {
				rv.Index(i).SetUint(uint64(b))
			}
			return nil
		}
		// string to bytes
		if d.isCodeString(code) {
			l, err := d.stringByteLength(code, k)
			if err != nil {
				return err
			}
			if l > rv.Len() {
				return fmt.Errorf("%v len is %d, but msgpack has %d elements, %w", rv.Type(), rv.Len(), l, def.ErrNotMatchArrayElement)
			}
			bs, err := d.asStringByteByLength(l, k)
			if err != nil {
				return err
			}
			for i, b := range bs {
				rv.Index(i).SetUint(uint64(b))
			}
			return nil
		}

		// get slice length
		l, err := d.sliceLength(code, k)
		if err != nil {
			return err
		}

		if l > rv.Len() {
			return fmt.Errorf("%v len is %d, but msgpack has %d elements, %w", rv.Type(), rv.Len(), l, def.ErrNotMatchArrayElement)
		}

		// create array dynamically
		if err = d.enter(); err != nil {
			return err
		}
		for i := 0; i < l; i++ {
			err = d.decode(rv.Index(i))
			if err != nil {
				return err
			}
		}
		d.depth--

	case reflect.Map:
		// nil
		if d.isCodeNil(code) {
			return nil
		}

		// get map length
		l, err := d.mapLength(code, k)
		if err != nil {
			return err
		}

		// check fixed type
		found, err := d.asFixedMap(rv, l)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		// create dynamically
		if err = d.enter(); err != nil {
			return err
		}
		key := rv.Type().Key()
		value := rv.Type().Elem()
		ifaceKey := key.Kind() == reflect.Interface
		checkKey := decodingutil.MapKeyNeedsCheck(key)
		if rv.IsNil() {
			rv.Set(reflect.MakeMapWithSize(rv.Type(), initialMapCapForType(l, key, value)))
		}
		for i := 0; i < l; i++ {
			k := reflect.New(key).Elem()
			v := reflect.New(value).Elem()
			var keyCode byte
			keyCode, err = d.readSize1()
			if err != nil {
				return err
			}
			extKey := false
			if ifaceKey {
				if err = d.canSetAsMapKey(keyCode); err != nil {
					return err
				}
				extKey = decodingutil.IsExtCode(keyCode)
			}
			err = d.decodeWithCode(keyCode, k)
			if err != nil {
				return err
			}
			err = d.decode(v)
			if err != nil {
				return err
			}

			if checkKey || extKey {
				if err = decodingutil.CheckMapKeyValue(k); err != nil {
					return err
				}
			}
			rv.SetMapIndex(k, v)
		}
		d.depth--

	case reflect.Struct:
		err := d.setStruct(code, rv, k)
		if err != nil {
			return err
		}

	case reflect.Ptr:
		// nil
		if d.isCodeNil(code) {
			return nil
		}

		if rv.Elem().Kind() == reflect.Invalid {
			n := reflect.New(rv.Type().Elem())
			rv.Set(n)
		}

		err := d.decodeWithCode(code, rv.Elem())
		if err != nil {
			return err
		}

	case reflect.Interface:
		if rv.Elem().Kind() == reflect.Ptr {
			err := d.decode(rv.Elem())
			if err != nil {
				return err
			}
		} else {
			v, err := d.asInterfaceWithCode(code, k)
			if err != nil {
				return err
			}
			if v != nil {
				rv.Set(reflect.ValueOf(v))
			}
		}

	default:
		return fmt.Errorf("%v is %w type", rv.Kind(), def.ErrUnsupportedType)
	}
	return nil
}

// enter increments the nesting depth. Callers decrement d.depth on success;
// on error the whole decode is aborted, so no decrement is needed.
func (d *decoder) enter() error {
	d.depth++
	if d.depth > decodingutil.MaxDepth {
		return decodingutil.DepthError()
	}
	return nil
}

func (d *decoder) errorTemplate(code byte, k reflect.Kind) error {
	return fmt.Errorf("%w %x decoding as %v", def.ErrCanNotDecode, code, k)
}
