package time

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"github.com/shamaton/msgpack/v2/def"
	"github.com/shamaton/msgpack/v2/ext"
)

var zero = time.Unix(0, 0)

var Decoder = new(timeDecoder)

type timeDecoder struct {
	ext.DecoderCommon
}

var _ ext.Decoder = (*timeDecoder)(nil)

func (td *timeDecoder) Code() int8 {
	return def.TimeStamp
}

func (td *timeDecoder) readSize1Safe(index int, d *[]byte) (byte, int, bool) {
	if len(*d) < index+def.Byte1 {
		return 0, 0, false
	}
	v, next := td.ReadSize1(index, d)
	return v, next, true
}

func (td *timeDecoder) readSize4Safe(index int, d *[]byte) ([]byte, int, bool) {
	if len(*d) < index+def.Byte4 {
		return nil, 0, false
	}
	v, next := td.ReadSize4(index, d)
	return v, next, true
}

func (td *timeDecoder) readSize8Safe(index int, d *[]byte) ([]byte, int, bool) {
	if len(*d) < index+def.Byte8 {
		return nil, 0, false
	}
	v, next := td.ReadSize8(index, d)
	return v, next, true
}

func (td *timeDecoder) IsType(offset int, d *[]byte) bool {
	code, offset, ok := td.readSize1Safe(offset, d)
	if !ok {
		return false
	}

	switch code {
	case def.Fixext4:
		t, _, ok := td.readSize1Safe(offset, d)
		if !ok || int8(t) != td.Code() {
			return false
		}
		_, _, ok = td.readSize4Safe(offset+def.Byte1, d)
		return ok
	case def.Fixext8:
		t, _, ok := td.readSize1Safe(offset, d)
		if !ok || int8(t) != td.Code() {
			return false
		}
		_, _, ok = td.readSize8Safe(offset+def.Byte1, d)
		return ok
	case def.Ext8:
		l, offset, ok := td.readSize1Safe(offset, d)
		if !ok {
			return false
		}
		t, _, ok := td.readSize1Safe(offset, d)
		if !ok || l != 12 || int8(t) != td.Code() {
			return false
		}
		_, _, ok = td.readSize4Safe(offset+def.Byte1, d)
		if !ok {
			return false
		}
		_, _, ok = td.readSize8Safe(offset+def.Byte1+def.Byte4, d)
		return ok
	}
	return false
}

func (td *timeDecoder) AsValue(offset int, k reflect.Kind, d *[]byte) (interface{}, int, error) {
	code, offset, ok := td.readSize1Safe(offset, d)
	if !ok {
		return zero, 0, def.ErrTooShortBytes
	}

	switch code {
	case def.Fixext4:
		_, offset, ok = td.readSize1Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		bs, offset, ok := td.readSize4Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		v := time.Unix(int64(binary.BigEndian.Uint32(bs)), 0)
		if decodeAsLocal {
			return v, offset, nil
		}
		return v.UTC(), offset, nil

	case def.Fixext8:
		_, offset, ok = td.readSize1Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		bs, offset, ok := td.readSize8Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		data64 := binary.BigEndian.Uint64(bs)
		nano := int64(data64 >> 34)
		if nano > 999999999 {
			return zero, 0, fmt.Errorf("in timestamp 64 formats, nanoseconds must not be larger than 999999999 : %d", nano)
		}
		v := time.Unix(int64(data64&0x00000003ffffffff), nano)
		if decodeAsLocal {
			return v, offset, nil
		}
		return v.UTC(), offset, nil

	case def.Ext8:
		_, offset, ok = td.readSize1Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		_, offset, ok = td.readSize1Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		nanobs, offset, ok := td.readSize4Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		secbs, offset, ok := td.readSize8Safe(offset, d)
		if !ok {
			return zero, 0, def.ErrTooShortBytes
		}
		nano := binary.BigEndian.Uint32(nanobs)
		if nano > 999999999 {
			return zero, 0, fmt.Errorf("in timestamp 96 formats, nanoseconds must not be larger than 999999999 : %d", nano)
		}
		sec := binary.BigEndian.Uint64(secbs)
		v := time.Unix(int64(sec), int64(nano))
		if decodeAsLocal {
			return v, offset, nil
		}
		return v.UTC(), offset, nil
	}

	return zero, 0, fmt.Errorf("should not reach this line!! code %x decoding %v", code, k)
}
