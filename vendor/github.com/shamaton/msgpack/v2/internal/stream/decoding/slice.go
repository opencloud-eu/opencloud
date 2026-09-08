package decoding

import (
	"encoding/binary"
	"reflect"

	"github.com/shamaton/msgpack/v2/def"
)

var (
	typeIntSlice   = reflect.TypeOf([]int{})
	typeInt8Slice  = reflect.TypeOf([]int8{})
	typeInt16Slice = reflect.TypeOf([]int16{})
	typeInt32Slice = reflect.TypeOf([]int32{})
	typeInt64Slice = reflect.TypeOf([]int64{})

	typeUintSlice   = reflect.TypeOf([]uint{})
	typeUint8Slice  = reflect.TypeOf([]uint8{})
	typeUint16Slice = reflect.TypeOf([]uint16{})
	typeUint32Slice = reflect.TypeOf([]uint32{})
	typeUint64Slice = reflect.TypeOf([]uint64{})

	typeFloat32Slice = reflect.TypeOf([]float32{})
	typeFloat64Slice = reflect.TypeOf([]float64{})

	typeStringSlice = reflect.TypeOf([]string{})

	typeBoolSlice = reflect.TypeOf([]bool{})
)

func (d *decoder) isFixSlice(v byte) bool {
	return def.FixArray <= v && v <= def.FixArray+0x0f
}

func (d *decoder) sliceLength(code byte, k reflect.Kind) (int, error) {
	switch {
	case d.isFixSlice(code):
		return int(code - def.FixArray), nil
	case code == def.Array16:
		bs, err := d.readSize2()
		if err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(bs)), nil
	case code == def.Array32:
		bs, err := d.readSize4()
		if err != nil {
			return 0, err
		}
		return lengthFromUint32(binary.BigEndian.Uint32(bs))
	}
	return 0, d.errorTemplate(code, k)
}

func (d *decoder) asFixedSlice(rv reflect.Value, l int) (bool, error) {
	t := rv.Type()
	k := t.Elem().Kind()

	switch t {
	case typeIntSlice:
		sli := make([]int, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asInt(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, int(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeUintSlice:
		sli := make([]uint, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asUint(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, uint(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeStringSlice:
		sli := make([]string, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asString(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeBoolSlice:
		sli := make([]bool, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asBool(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeFloat32Slice:
		sli := make([]float32, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asFloat32(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeFloat64Slice:
		sli := make([]float64, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asFloat64(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeInt8Slice:
		sli := make([]int8, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asInt(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, int8(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeInt16Slice:
		sli := make([]int16, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asInt(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, int16(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeInt32Slice:
		sli := make([]int32, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asInt(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, int32(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeInt64Slice:
		sli := make([]int64, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asInt(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeUint8Slice:
		sli := make([]uint8, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asUint(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, uint8(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeUint16Slice:
		sli := make([]uint16, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asUint(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, uint16(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeUint32Slice:
		sli := make([]uint32, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asUint(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, uint32(v))
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil

	case typeUint64Slice:
		sli := make([]uint64, 0, initialSliceCap(l, t.Elem()))
		for i := 0; i < l; i++ {
			v, err := d.asUint(k)
			if err != nil {
				return false, err
			}
			sli = append(sli, v)
		}
		rv.Set(reflect.ValueOf(sli))
		return true, nil
	}

	return false, nil
}
