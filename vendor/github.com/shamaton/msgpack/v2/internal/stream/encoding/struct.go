package encoding

import (
	"math"
	"reflect"
	"sync"

	"github.com/shamaton/msgpack/v2/def"
	"github.com/shamaton/msgpack/v2/ext"
	"github.com/shamaton/msgpack/v2/internal/common"
)

type structCache struct {
	indexes []int
	names   []string
	omits   []bool
	noOmit  bool
	common.Common
}

var cachemap = sync.Map{}

type structWriteFunc func(rv reflect.Value) error

func (e *encoder) getStructWriter(typ reflect.Type) structWriteFunc {

	for i := range extCoders {
		if extCoders[i].Type() == typ {
			return func(rv reflect.Value) error {
				w := ext.CreateStreamWriter(e.w, e.buf)
				return extCoders[i].Write(w, rv)
			}
		}
	}

	if e.asArray {
		return e.writeStructArray
	}
	return e.writeStructMap
}

func (e *encoder) writeStruct(rv reflect.Value) error {

	for i := range extCoders {
		if extCoders[i].Type() == rv.Type() {
			w := ext.CreateStreamWriter(e.w, e.buf)
			return extCoders[i].Write(w, rv)
		}
	}

	if e.asArray {
		return e.writeStructArray(rv)
	}
	return e.writeStructMap(rv)
}

func (e *encoder) writeStructArray(rv reflect.Value) error {
	c := e.getStructCache(rv)

	// write format
	num := len(c.indexes)
	if num <= 0x0f {
		if err := e.setByte1Int(def.FixArray + num); err != nil {
			return err
		}
	} else if num <= math.MaxUint16 {
		if err := e.setByte1Int(def.Array16); err != nil {
			return err
		}
		if err := e.setByte2Int(num); err != nil {
			return err
		}
	} else if uint(num) <= math.MaxUint32 {
		if err := e.setByte1Int(def.Array32); err != nil {
			return err
		}
		if err := e.setByte4Int(num); err != nil {
			return err
		}
	}

	for i := 0; i < num; i++ {
		if err := e.create(rv.Field(c.indexes[i])); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) writeStructMap(rv reflect.Value) error {
	c := e.getStructCache(rv)

	num := len(c.indexes)
	l := 0
	if c.noOmit {
		l = num
	} else {
		for i := 0; i < num; i++ {
			irv := rv.Field(c.indexes[i])
			if !c.omits[i] || !irv.IsZero() {
				l++
			}
		}
	}

	// format size
	if l <= 0x0f {
		if err := e.setByte1Int(def.FixMap + l); err != nil {
			return err
		}
	} else if l <= math.MaxUint16 {
		if err := e.setByte1Int(def.Map16); err != nil {
			return err
		}
		if err := e.setByte2Int(l); err != nil {
			return err
		}
	} else if uint(l) <= math.MaxUint32 {
		if err := e.setByte1Int(def.Map32); err != nil {
			return err
		}
		if err := e.setByte4Int(l); err != nil {
			return err
		}
	}

	for i := 0; i < num; i++ {
		irv := rv.Field(c.indexes[i])
		if !c.omits[i] || !irv.IsZero() {
			if err := e.writeString(c.names[i]); err != nil {
				return err
			}
			if err := e.create(irv); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *encoder) getStructCache(rv reflect.Value) *structCache {
	t := rv.Type()
	cache, find := cachemap.Load(t)
	if find {
		return cache.(*structCache)
	}

	num := rv.NumField()
	c := &structCache{
		indexes: make([]int, 0, num),
		names:   make([]string, 0, num),
		omits:   make([]bool, 0, num),
	}
	omitCount := 0
	for i := 0; i < num; i++ {
		if ok, omit, name := e.CheckField(rv.Type().Field(i)); ok {
			c.indexes = append(c.indexes, i)
			c.names = append(c.names, name)
			c.omits = append(c.omits, omit)
			if omit {
				omitCount++
			}
		}
	}
	c.noOmit = omitCount == 0
	cachemap.Store(t, c)
	return c
}
