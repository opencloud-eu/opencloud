package structs

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// NewWriter create a struct writer
//
// TIP: must be pointer for set field value
func NewWriter(ptr any) *Wrapper {
	rv := reflect.ValueOf(ptr)

	if rv.Kind() != reflect.Pointer {
		panic("must be provider an pointer value")
	}
	return WrapValue(rv)
}

/*************************************************************
 * set values to a struct
 *************************************************************/

// SetOptFunc define
type SetOptFunc func(opt *SetOptions)

// BeforeSetFunc hook type.
type BeforeSetFunc func(fieldName string, value any, fv reflect.Value) any

// SetOptions for set values to struct
type SetOptions struct {
	// FieldTagName get field name for read value. default tag: json
	FieldTagName string
	// BeforeSetFn hook func. will fire on before set value.
	//  - you can modify value here.
	//  - returns nil will skip set value.
	//  - returns value will be set to field value.
	BeforeSetFn BeforeSetFunc

	// ParseTime parse string to `time.Duration`, `time.Time`. default: false
	//
	// eg: default:"10s", default:"2025-04-23 15:04:05"
	ParseTime bool
	// ParseDefault init default value by DefaultValTag tag value. default: false
	//
	// see InitDefaults()
	ParseDefault bool
	// DefaultValTag name. tag: default
	DefaultValTag string

	// ParseDefaultEnv parse env var on default tag. eg: `default:"${APP_ENV}"` default: false
	ParseDefaultEnv bool
	// DefaultEnvPrefixTag name. tag: defaultenvprefix
	DefaultEnvPrefixTag string

	// StopOnError if true, will stop set value on error happened. default: false
	// StopOnError bool
}

// WithParseDefault value by tag "default"
func WithParseDefault(opt *SetOptions) { opt.ParseDefault = true }

// WithBeforeSetFn value by tag "default"
func WithBeforeSetFn(fn BeforeSetFunc) SetOptFunc {
	return func(opt *SetOptions) { opt.BeforeSetFn = fn }
}

// TODO refactoring SetValues to the struct
type ValuesSetter struct {
	src any // source struct
	// raw reflect.Value of source struct
	rv reflect.Value

	option  *SetOptions
	initOpt *InitOptions
}

// BindData set values to struct ptr from map data.
func BindData(ptr any, data map[string]any, optFns ...SetOptFunc) error {
	return SetValues(ptr, data, optFns...)
}

// SetValues set values to struct ptr from map data.
//
// TIPS:
//
//	Only support set: string, bool, intX, uintX, floatX
func SetValues(ptr any, data map[string]any, optFns ...SetOptFunc) error {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Ptr {
		return errors.New("must be provider an pointer value")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("must be provider an struct value")
	}

	opt := &SetOptions{
		FieldTagName:        defaultFieldTag,
		DefaultValTag:       defaultInitTag,
		DefaultEnvPrefixTag: defaultEnvPrefixTag,
	}

	for _, fn := range optFns {
		fn(opt)
	}
	return setValues(rv, data, opt, "")
}

func setValues(rv reflect.Value, data map[string]any, opt *SetOptions, envPrefix string) error {
	if len(data) == 0 {
		return nil
	}

	var es comdef.Errors
	initOpt := &InitOptions{
		EnvPrefix: envPrefix,
		ParseEnv:  opt.ParseDefaultEnv,
		ParseTime: opt.ParseTime,
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		name := ft.Name
		// skip don't exported field
		if name[0] >= 'a' && name[0] <= 'z' {
			continue
		}

		// get field name
		tagVal, ok0 := ft.Tag.Lookup(opt.FieldTagName)
		if ok0 {
			info, err := ParseTagValueDefault(name, tagVal)
			if err != nil {
				es = append(es, err)
				continue
			}
			name = info.Get("name")
		}

		fv := rv.Field(i)
		val, ok := data[name]

		// set field value by default tag.
		if !ok && opt.ParseDefault && fv.IsZero() {
			defVal := ft.Tag.Get(opt.DefaultValTag)
			if err := initDefaultValue(fv, defVal, initOpt); err != nil {
				es = append(es, err)
			}
			continue
		}

		// handle for pointer field
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			fv = fv.Elem()
		}

		// hook: call BeforeSet func
		if opt.BeforeSetFn != nil {
			val = opt.BeforeSetFn(ft.Name, val, fv)
			if val == nil {
				continue
			}
			ok = true // on value is not nil
		}

		// field is struct
		if fv.Kind() == reflect.Struct {
			valRf := reflects.Indirect(reflect.ValueOf(val))

			// up: special handle time.Time field
			if reflects.IsTimeType(fv.Type()) {
				// maybe val is time.Time
				if reflects.IsTimeType(valRf.Type()) {
					if err := reflects.SetValue(fv, valRf); err != nil {
						es = append(es, err)
					}
					continue
				}

				// val is datetime string
				tm, err := strutil.ToTime(strutil.StringOr(val, ""))
				if err != nil {
					es = append(es, err)
				} else if err = reflects.SetValue(fv, tm); err != nil {
					es = append(es, err)
				}
				continue
			}

			// val as map
			asMp, err := reflects.TryAnyMap(valRf)
			if err != nil {
				err = fmt.Errorf("must provide map for set struct field %q, err=%v", ft.Name, err)
				es = append(es, err)
				continue
			}

			defEnvPrefixVal := ft.Tag.Get(opt.DefaultEnvPrefixTag)
			childEnvPrefix := fmt.Sprintf("%s%s", envPrefix, defEnvPrefixVal)

			// recursive processing sub-struct
			if err = setValues(fv, asMp, opt, childEnvPrefix); err != nil {
				es = append(es, err)
			}
			continue
		}

		// set field value
		if ok && val != nil {
			if err := reflects.SetValue(fv, val); err != nil {
				es = append(es, err)
			}
		}
	}

	return es.ErrOrNil()
}
