package structs

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// NewWriter create a struct writer
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

// SetOptions for set values to struct
type SetOptions struct {
	// FieldTagName get field name for read value. default tag: json
	FieldTagName string
	// ValueHook before set value hook TODO
	ValueHook func(val any) any

	// ParseDefault init default value by DefaultValTag tag value.
	// default: false
	//
	// see InitDefaults()
	ParseDefault bool

	// DefaultValTag name. tag: default
	DefaultValTag string

	// ParseDefaultEnv parse env var on default tag. eg: `default:"${APP_ENV}"`
	//
	// default: false
	ParseDefaultEnv bool

	// DefaultEnvPrefixTag name. tag: defaultenvprefix
	DefaultEnvPrefixTag string

	// StopOnError if true, will stop set value on error happened. default: false
	// StopOnError bool
}

// WithParseDefault value by tag "default"
func WithParseDefault(opt *SetOptions) {
	opt.ParseDefault = true
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
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		name := ft.Name
		// skip don't exported field
		if name[0] >= 'a' && name[0] <= 'z' {
			continue
		}

		// get field name
		tagVal, ok := ft.Tag.Lookup(opt.FieldTagName)
		if ok {
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
			if err := initDefaultValue(fv, defVal, opt.ParseDefaultEnv, envPrefix); err != nil {
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

		// field is struct
		if fv.Kind() == reflect.Struct {
			// up: special handle time.Time struct
			if _, ok := fv.Interface().(time.Time); ok {
				tm, er := strutil.ToTime(strutil.StringOr(val, ""))
				if er != nil {
					es = append(es, er)
					continue
				}
				if er = reflects.SetValue(fv, tm); er != nil {
					es = append(es, er)
				}
				continue
			}

			asMp, err := maputil.TryAnyMap(val)
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
		if err := reflects.SetValue(fv, val); err != nil {
			es = append(es, err)
			continue
		}
	}

	return es.ErrOrNil()
}
