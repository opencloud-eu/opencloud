package gofakeit

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// RecursiveDepth controls the maximum recursion depth when populating structs.
// Increase if your data structures are deeply nested; decrease to be more conservative.
var RecursiveDepth = 10

// Struct fills in exported fields of a struct with random data
// based on the value of `fake` tag of exported fields
// or with the result of a call to the Fake() method
// if the field type implements `Fakeable`.
// Use `fake:"skip"` to explicitly skip an element.
// All built-in types are supported, with templating support
// for string types.
func Struct(v any) error { return structFunc(GlobalFaker, v) }

// Struct fills in exported fields of a struct with random data
// based on the value of `fake` tag of exported fields.
// Use `fake:"skip"` to explicitly skip an element.
// All built-in types are supported, with templating support
// for string types.
func (f *Faker) Struct(v any) error { return structFunc(f, v) }

func structFunc(f *Faker, v any) error {
	return r(f, reflect.TypeOf(v), reflect.ValueOf(v), "", 0, 0)
}

func r(f *Faker, t reflect.Type, v reflect.Value, tag string, size int, depth int) error {
	// Handle special types

	if t.PkgPath() == "encoding/json" {
		// encoding/json has two special types:
		// - RawMessage
		// - Number

		switch t.Name() {
		case "RawMessage":
			return rJsonRawMessage(f, v, tag)
		case "Number":
			return rJsonNumber(f, v, tag)
		default:
			return errors.New("unknown encoding/json type: " + t.Name())
		}
	}

	// Handle generic types
	switch t.Kind() {
	case reflect.Ptr:
		return rPointer(f, t, v, tag, size, depth)
	case reflect.Struct:
		return rStruct(f, t, v, tag, depth)
	case reflect.String:
		return rString(f, t, v, tag)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rUint(f, t, v, tag)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rInt(f, t, v, tag)
	case reflect.Float32, reflect.Float64:
		return rFloat(f, t, v, tag)
	case reflect.Bool:
		return rBool(f, t, v, tag)
	case reflect.Array, reflect.Slice:
		return rSlice(f, t, v, tag, size, depth)
	case reflect.Map:
		return rMap(f, t, v, tag, size, depth)
	}

	return nil
}

func rCustom(f *Faker, v reflect.Value, tag string) error {
	// If tag is empty return error
	if tag == "" {
		return errors.New("tag is empty")
	}

	fName, fParams := parseNameAndParamsFromTag(tag)
	info := GetFuncLookup(fName)

	// Check to see if it's a replaceable lookup function
	if info == nil {
		return fmt.Errorf("function %q not found", tag)
	}

	// Parse map params
	mapParams, err := parseMapParams(info, fParams)
	if err != nil {
		return err
	}

	// Call function
	fValue, err := info.Generate(f, mapParams, info)
	if err != nil {
		return err
	}

	// Create new element of expected type
	field := reflect.New(reflect.TypeOf(fValue))
	field.Elem().Set(reflect.ValueOf(fValue))

	// Check if element is pointer if so
	// grab the underlying value
	fieldElem := field.Elem()
	if fieldElem.Kind() == reflect.Ptr {
		fieldElem = fieldElem.Elem()
	}

	// Check if field kind is the same as the expected type
	if fieldElem.Kind() != v.Kind() {
		// return error saying the field and kinds that do not match
		return errors.New("field kind " + fieldElem.Kind().String() + " does not match expected kind " + v.Kind().String())
	}

	// Set the value
	v.Set(fieldElem.Convert(v.Type()))

	// If a function is called to set the struct
	// stop from going through sub fields
	return nil
}

func rStruct(f *Faker, t reflect.Type, v reflect.Value, tag string, depth int) error {
	// Prevent recursing deeper than configured levels
	if depth >= RecursiveDepth {
		return nil
	}

	// Check if tag exists, if so run custom function
	if t.Name() != "" && tag != "" {
		return rCustom(f, v, tag)
	}

	// Check if struct is fakeable
	if isFakeable(t) {
		value, err := callFake(f, v, reflect.Struct)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(value))
		return nil
	}

	// Loop through all the fields of the struct
	n := t.NumField()
	for i := 0; i < n; i++ {
		elementT := t.Field(i)
		elementV := v.Field(i)
		fakeTag, ok := elementT.Tag.Lookup("fake")

		// Check whether or not to skip this field
		if ok && fakeTag == "skip" || fakeTag == "-" {
			// Do nothing, skip it
			continue
		}

		// Check to make sure you can set it or that it's an embedded(anonymous) field
		if !elementV.CanSet() && !elementT.Anonymous {
			continue
		}

		// Check if reflect type is of values we can specifically set
		elemStr := elementT.Type.String()
		switch elemStr {
		case "time.Time", "*time.Time":
			// Check if element is a pointer
			elemV := elementV
			if elemStr == "*time.Time" {
				elemV = reflect.New(elementT.Type.Elem()).Elem()
			}

			// Run rTime on the element
			err := rTime(f, elementT, elemV, fakeTag)
			if err != nil {
				return err
			}

			if elemStr == "*time.Time" {
				elementV.Set(elemV.Addr())
			}

			continue
		}

		// Check if fakesize is set
		size := -1 // Set to -1 to indicate fakesize was not set
		fs, ok := elementT.Tag.Lookup("fakesize")
		if ok {
			var err error

			// Check if size has params separated by ,
			if strings.Contains(fs, ",") {
				sizeSplit := strings.SplitN(fs, ",", 2)
				if len(sizeSplit) == 2 {
					var sizeMin int
					var sizeMax int

					sizeMin, err = strconv.Atoi(sizeSplit[0])
					if err != nil {
						return err
					}
					sizeMax, err = strconv.Atoi(sizeSplit[1])
					if err != nil {
						return err
					}

					size = f.IntN(sizeMax-sizeMin+1) + sizeMin
				}
			} else {
				size, err = strconv.Atoi(fs)
				if err != nil {
					return err
				}
			}
		}

		// Recursively call r() to fill in the struct
		err := r(f, elementT.Type, elementV, fakeTag, size, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func rPointer(f *Faker, t reflect.Type, v reflect.Value, tag string, size int, depth int) error {
	elemT := t.Elem()
	// Prevent recursing deeper than configured levels
	if depth >= RecursiveDepth {
		return nil
	}

	if v.IsNil() {
		nv := reflect.New(elemT).Elem()
		err := r(f, elemT, nv, tag, size, depth+1)
		if err != nil {
			return err
		}

		v.Set(nv.Addr())
	} else {
		err := r(f, elemT, v.Elem(), tag, size, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func rSlice(f *Faker, t reflect.Type, v reflect.Value, tag string, size int, depth int) error {
	// If you cant even set it dont even try
	if !v.CanSet() {
		return errors.New("cannot set slice")
	}

	// Prevent recursing deeper than configured levels
	if depth >= RecursiveDepth {
		return nil
	}

	// Check if tag exists, if so run custom function
	if t.Name() != "" && tag != "" {
		// Check to see if custom function works if not continue to normal loop of values
		err := rCustom(f, v, tag)
		if err == nil {
			return nil
		}
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.Slice, reflect.Array)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(value))
		return nil
	}

	// Grab original size to use if needed for sub arrays
	ogSize := size

	// If the value has a len and is less than the size
	// use that instead of the requested size
	elemLen := v.Len()
	if elemLen == 0 && size == -1 {
		size = number(f, 1, 10)
	} else if elemLen != 0 && (size == -1 || elemLen < size) {
		size = elemLen
	}

	// Get the element type
	elemT := t.Elem()

	// Loop through the elements length and set based upon the index
	for i := 0; i < size; i++ {
		nv := reflect.New(elemT)
		err := r(f, elemT, nv.Elem(), tag, ogSize, depth+1)
		if err != nil {
			return err
		}

		// If values are already set fill them up, otherwise append
		if elemLen != 0 {
			v.Index(i).Set(reflect.Indirect(nv))
		} else {
			v.Set(reflect.Append(reflect.Indirect(v), reflect.Indirect(nv)))
		}
	}

	return nil
}

func rMap(f *Faker, t reflect.Type, v reflect.Value, tag string, size int, depth int) error {
	// If you cant even set it dont even try
	if !v.CanSet() {
		return errors.New("cannot set slice")
	}

	// Prevent recursing deeper than configured levels
	if depth >= RecursiveDepth {
		return nil
	}

	// Check if tag exists, if so run custom function
	if tag != "" {
		return rCustom(f, v, tag)
	} else if isFakeable(t) && size <= 0 {
		// Only call custom function if no fakesize is specified (size <= 0)
		value, err := callFake(f, v, reflect.Map)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(value))
		return nil
	}

	// Set a size
	newSize := size
	if newSize == -1 {
		newSize = number(f, 1, 10)
	}

	// Create new map based upon map key value type
	mapType := reflect.MapOf(t.Key(), t.Elem())
	newMap := reflect.MakeMap(mapType)

	for i := 0; i < newSize; i++ {
		// Create new key
		mapIndex := reflect.New(t.Key())
		err := r(f, t.Key(), mapIndex.Elem(), "", -1, depth+1)
		if err != nil {
			return err
		}

		// Create new value
		mapValue := reflect.New(t.Elem())
		err = r(f, t.Elem(), mapValue.Elem(), "", -1, depth+1)
		if err != nil {
			return err
		}

		newMap.SetMapIndex(mapIndex.Elem(), mapValue.Elem())
	}

	// Set newMap into struct field
	if t.Kind() == reflect.Ptr {
		v.Set(newMap.Elem())
	} else {
		v.Set(newMap)
	}

	return nil
}

func rString(f *Faker, t reflect.Type, v reflect.Value, tag string) error {
	if tag != "" {
		genStr, err := generate(f, tag)
		if err != nil {
			return err
		}

		v.SetString(genStr)
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.String)
		if err != nil {
			return err
		}

		valueStr, ok := value.(string)
		if !ok {
			return errors.New("call to Fake method did not return a string")
		}
		v.SetString(valueStr)
	} else {
		genStr, err := generate(f, strings.Repeat("?", number(f, 4, 10)))
		if err != nil {
			return err
		}

		v.SetString(genStr)
	}

	return nil
}

func rInt(f *Faker, t reflect.Type, v reflect.Value, tag string) error {
	if tag != "" {
		genStr, err := generate(f, tag)
		if err != nil {
			return err
		}

		i, err := strconv.ParseInt(genStr, 10, 64)
		if err != nil {
			return err
		}

		v.SetInt(i)
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)
		if err != nil {
			return err
		}

		switch i := value.(type) {
		case int:
			v.SetInt(int64(i))
		case int8:
			v.SetInt(int64(i))
		case int16:
			v.SetInt(int64(i))
		case int32:
			v.SetInt(int64(i))
		case int64:
			v.SetInt(int64(i))
		default:
			return errors.New("call to Fake method did not return an integer")
		}
	} else {
		// If no tag or error converting to int, set with random value
		switch t.Kind() {
		case reflect.Int:
			v.SetInt(int64Func(f))
		case reflect.Int8:
			v.SetInt(int64(int8Func(f)))
		case reflect.Int16:
			v.SetInt(int64(int16Func(f)))
		case reflect.Int32:
			v.SetInt(int64(int32Func(f)))
		case reflect.Int64:
			v.SetInt(int64Func(f))
		}
	}

	return nil
}

func rUint(f *Faker, t reflect.Type, v reflect.Value, tag string) error {
	if tag != "" {
		genStr, err := generate(f, tag)
		if err != nil {
			return err
		}

		u, err := strconv.ParseUint(genStr, 10, 64)
		if err != nil {
			return err
		}

		v.SetUint(u)
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64)
		if err != nil {
			return err
		}

		switch i := value.(type) {
		case uint:
			v.SetUint(uint64(i))
		case uint8:
			v.SetUint(uint64(i))
		case uint16:
			v.SetUint(uint64(i))
		case uint32:
			v.SetUint(uint64(i))
		case uint64:
			v.SetUint(uint64(i))
		default:
			return errors.New("call to Fake method did not return an unsigned integer")
		}
	} else {
		// If no tag or error converting to uint, set with random value
		switch t.Kind() {
		case reflect.Uint:
			v.SetUint(f.Uint64())
		case reflect.Uint8:
			v.SetUint(uint64(uint8Func(f)))
		case reflect.Uint16:
			v.SetUint(uint64(uint16Func(f)))
		case reflect.Uint32:
			v.SetUint(uint64(uint32Func(f)))
		case reflect.Uint64:
			v.SetUint(f.Uint64())
		}
	}

	return nil
}

func rFloat(f *Faker, t reflect.Type, v reflect.Value, tag string) error {
	if tag != "" {
		genStr, err := generate(f, tag)
		if err != nil {
			return err
		}

		f, err := strconv.ParseFloat(genStr, 64)
		if err != nil {
			return err
		}

		v.SetFloat(f)
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.Float32, reflect.Float64)
		if err != nil {
			return err
		}

		switch i := value.(type) {
		case float32:
			v.SetFloat(float64(i))
		case float64:
			v.SetFloat(float64(i))
		default:
			return errors.New("call to Fake method did not return a float")
		}
	} else {
		// If no tag or error converting to float, set with random value
		switch t.Kind() {
		case reflect.Float64:
			v.SetFloat(float64Func(f))
		case reflect.Float32:
			v.SetFloat(float64(float32Func(f)))
		}
	}

	return nil
}

func rBool(f *Faker, t reflect.Type, v reflect.Value, tag string) error {
	if tag != "" {
		genStr, err := generate(f, tag)
		if err != nil {
			return err
		}

		b, err := strconv.ParseBool(genStr)
		if err != nil {
			return err
		}

		v.SetBool(b)
	} else if isFakeable(t) {
		value, err := callFake(f, v, reflect.Bool)
		if err != nil {
			return err
		}

		switch i := value.(type) {
		case bool:
			v.SetBool(bool(i))
		default:
			return errors.New("call to Fake method did not return a boolean")
		}
	} else {
		// If no tag or error converting to boolean, set with random value
		v.SetBool(boolFunc(f))
	}

	return nil
}

// rTime will set a time.Time field the best it can from either the default date tag or from the generate tag
func rTime(f *Faker, t reflect.StructField, v reflect.Value, tag string) error {
	if tag != "" {
		// Generate time
		timeOutput, err := generate(f, tag)
		if err != nil {
			return err
		}

		// Check to see if timeOutput has monotonic clock reading
		// if so, remove it. This is because time.Parse() does not
		// support parsing the monotonic clock reading
		if strings.Contains(timeOutput, " m=") {
			timeOutput = strings.Split(timeOutput, " m=")[0]
		}

		// Check to see if they are passing in a format	to parse the time
		timeFormat, timeFormatOK := t.Tag.Lookup("format")
		if timeFormatOK {
			timeFormat = javaDateTimeFormatToGolangFormat(timeFormat)
		} else {
			// If tag == "{date}" use time.RFC3339
			// They are attempting to use the default date lookup
			if tag == "{date}" {
				timeFormat = time.RFC3339
			} else {
				// Default format of time.Now().String()
				timeFormat = "2006-01-02 15:04:05.999999999 -0700 MST"
			}
		}

		// If output is larger than format cut the output
		// This helps us avoid errors from time.Parse
		if len(timeOutput) > len(timeFormat) {
			timeOutput = timeOutput[:len(timeFormat)]
		}

		// Attempt to parse the time
		timeStruct, err := time.Parse(timeFormat, timeOutput)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(timeStruct))
		return nil
	}

	v.Set(reflect.ValueOf(date(f)))
	return nil
}
