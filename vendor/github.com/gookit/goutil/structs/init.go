package structs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gookit/goutil/internal/varexpr"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

const defaultInitTag = "default"
const defaultEnvPrefixTag = "defaultenvprefix"

// InitOptFunc define
type InitOptFunc func(opt *InitOptions)

// InitOptions struct
type InitOptions struct {
	// TagName default value tag name. tag: default
	TagName string
	// EnvPrefix default ENV prefix name.
	EnvPrefix string
	// EnvPrefixTagName default value tag name. tag: defaultenvprefix
	EnvPrefixTagName string
	// ParseEnv var name on default value. eg: `default:"${APP_ENV}"`
	//
	// default: false
	ParseEnv bool
	// ParseTime parse string to `time.Duration`, `time.Time`. default: false
	//
	// eg: default:"10s", default:"2025-04-23 15:04:05"
	ParseTime bool
	// ValueHook before set value hook TODO
	ValueHook func(val string) any
}

// WithParseTime set parse time string on default value.
func (opt *InitOptions) WithParseTime(val bool) *InitOptions {
	opt.ParseTime = val
	return opt
}

// WithParseEnv set parse env var on default value.
func (opt *InitOptions) WithParseEnv(val bool) *InitOptions {
	opt.ParseEnv = val
	return opt
}

// Init struct default value by field "default" tag.
func Init(ptr any, optFns ...InitOptFunc) error {
	return InitDefaults(ptr, optFns...)
}

// InitDefaults init struct default value by field "default" tag.
//
// TIPS:
//
//	Support init field types: string, bool, intX, uintX, floatX, array, slice
//
// Example:
//
//	type User1 struct {
//		Name string `default:"inhere"`
//		Age  int32  `default:"30"`
//	}
//
//	u1 := &User1{}
//	err = structs.InitDefaults(u1)
//	fmt.Printf("%+v\n", u1) // Output: {Name:inhere Age:30}
func InitDefaults(ptr any, optFns ...InitOptFunc) error {
	rv := reflect.ValueOf(ptr)
	if !reflects.IsValidPtr(rv) {
		return errors.New("must be provider an pointer value")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("must be provider an struct value")
	}

	opt := &InitOptions{TagName: defaultInitTag, EnvPrefixTagName: defaultEnvPrefixTag}
	for _, fn := range optFns {
		fn(opt)
	}

	return initDefaults(rv, opt, "")
}

// type InitBuilder struct {
// 	opt InitOptions
// }

func initDefaults(rv reflect.Value, opt *InitOptions, envPrefix string) error {
	rt := rv.Type()
	opt.EnvPrefix = envPrefix

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		// skip doesn't exported field
		if IsUnexported(sf.Name) {
			continue
		}

		val, hasTag := sf.Tag.Lookup(opt.TagName)
		if !hasTag || val == "-" {
			continue
		}

		var childPrefixVar string
		prefixVar, _ := sf.Tag.Lookup(opt.EnvPrefixTagName)
		if prefixVar != "" {
			childPrefixVar = fmt.Sprintf("%s%s", envPrefix, prefixVar)
			// opt.EnvPrefix = childPrefixVar
		}

		fv := rv.Field(i)
		if fv.Kind() == reflect.Struct {
			// special: struct is time.Time type
			if reflects.IsTimeType(fv.Type()) {
				if err := initDefaultValue(fv, val, opt); err != nil {
					return err
				}
			} else if err := initDefaults(fv, opt, childPrefixVar); err != nil {
				return err
			}
			continue
		}

		// Skip init on field has value. but will check slice and pointer field
		if !fv.IsZero() {
			// special: handle for pointer struct field
			if fv.Kind() == reflect.Pointer {
				fv = fv.Elem()
				if fv.Kind() == reflect.Struct {
					if err := initDefaults(fv, opt, childPrefixVar); err != nil {
						return err
					}
				}
			} else if fv.Kind() == reflect.Slice {
				el := sf.Type.Elem()
				if el.Kind() == reflect.Pointer {
					el = el.Elem()
				}

				// init sub struct in slice. like `[]SubStruct` or `[]*SubStruct`
				if el.Kind() == reflect.Struct && fv.Len() > 0 {
					for i := 0; i < fv.Len(); i++ {
						subFv := reflect.Indirect(fv.Index(i))
						if err := initDefaults(subFv, opt, childPrefixVar); err != nil {
							return err
						}
					}
				}
			}
			continue
		}

		// handle for pointer field
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}

			fv = fv.Elem()
			if fv.Kind() == reflect.Struct {
				if err := initDefaults(fv, opt, childPrefixVar); err != nil {
					return err
				}
				continue
			}
		} else if fv.Kind() == reflect.Slice {
			el := sf.Type.Elem()
			isPtr := el.Kind() == reflect.Pointer
			if isPtr {
				el = el.Elem()
			}

			// init sub struct in slice. like `[]SubStruct` or `[]*SubStruct`
			if el.Kind() == reflect.Struct {
				// up: if slice elem is struct and slice len=0, will be skip init default value
				if fv.Len() == 0 {
					continue
				}

				// make sub-struct and init. like: `SubStruct`
				subFv := reflect.New(el)
				subFvE := subFv.Elem()
				if err := initDefaults(subFvE, opt, childPrefixVar); err != nil {
					return err
				}

				// make new slice and set value.
				newFv := reflect.MakeSlice(reflect.SliceOf(sf.Type.Elem()), 0, 1)
				if isPtr {
					newFv = reflect.Append(newFv, subFv)
				} else {
					newFv = reflect.Append(newFv, subFvE)
				}
				fv.Set(newFv)
				continue
			}
		}

		if err := initDefaultValue(fv, val, opt); err != nil {
			return err
		}
	}

	return nil
}

func enhanceDefaultVar(val string, envPrefix string) string {
	if len(envPrefix) == 0 {
		return val
	}
	cleanedVar := strings.TrimSpace(val)

	if strings.HasPrefix(cleanedVar, "${") && strings.HasSuffix(cleanedVar, "}") {
		parts := strings.SplitN(cleanedVar[2:len(cleanedVar)-1], "|", 2)
		if len(parts) > 0 {
			env := strings.TrimSpace(parts[0])
			if env != "" {
				if len(parts) == 1 {
					return fmt.Sprintf("${%s%s}", envPrefix, env)
				} else {
					return fmt.Sprintf("${%s%s|%s}", envPrefix, env, parts[1])
				}
			}
		}
	}
	return val
}

func initDefaultValue(fv reflect.Value, val string, opt *InitOptions) error {
	if val == "" || !fv.CanSet() {
		return nil
	}

	// parse env var
	if opt.ParseEnv {
		val = varexpr.SafeParse(enhanceDefaultVar(val, opt.EnvPrefix))
	}

	var anyVal any = val

	// enhance: parse special value type. eg: time.Duration, time.Time
	if opt.ParseTime {
		newVal, err := reflects.ToTimeOrDuration(val, fv.Type())
		if err != nil {
			return err
		}
		anyVal = newVal
	}

	// simple slice: convert simple kind(string,intX,uintX,...) to slice. eg: "1,2,3" => []int{1,2,3}
	if reflects.IsArrayOrSlice(fv.Kind()) && reflects.IsSimpleKind(reflects.SliceElemKind(fv.Type())) {
		ss := strutil.SplitTrimmed(val, ",")
		valRv, err := reflects.ConvSlice(reflect.ValueOf(ss), fv.Type().Elem())
		if err == nil {
			if fv.Kind() == reflect.Array {
				for i := 0; i < valRv.Len(); i++ {
					fv.Index(i).Set(valRv.Index(i))
				}
			} else {
				reflects.SetRValue(fv, valRv)
			}
		}
		return err
	}

	// set value
	return reflects.SetValue(fv, anyVal)
}
