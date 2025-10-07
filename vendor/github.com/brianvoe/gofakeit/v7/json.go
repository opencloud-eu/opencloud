package gofakeit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// JSONOptions defines values needed for json generation
type JSONOptions struct {
	Type     string  `json:"type" xml:"type" fake:"{randomstring:[array,object]}"` // array or object
	RowCount int     `json:"row_count" xml:"row_count" fake:"{number:1,10}"`
	Indent   bool    `json:"indent" xml:"indent"`
	Fields   []Field `json:"fields" xml:"fields" fake:"{fields}"`
}

type jsonKeyVal struct {
	Key   string
	Value any
}

type jsonOrderedKeyVal []*jsonKeyVal

func (okv jsonOrderedKeyVal) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("{")
	for i, kv := range okv {
		// Add comma to all except last one
		if i != 0 {
			buf.WriteString(",")
		}

		// Marshal key and write
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)

		// Write colon separator
		buf.WriteString(":")

		// Marshal value and write
		val, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("}")

	return buf.Bytes(), nil
}

// JSON generates an object or an array of objects in json format.
// A nil JSONOptions returns a randomly structured JSON.
func JSON(jo *JSONOptions) ([]byte, error) { return jsonFunc(GlobalFaker, jo) }

// JSON generates an object or an array of objects in json format.
// A nil JSONOptions returns a randomly structured JSON.
func (f *Faker) JSON(jo *JSONOptions) ([]byte, error) { return jsonFunc(f, jo) }

// JSON generates an object or an array of objects in json format
func jsonFunc(f *Faker, jo *JSONOptions) ([]byte, error) {
	if jo == nil {
		// We didn't get a JSONOptions, so create a new random one
		err := f.Struct(&jo)
		if err != nil {
			return nil, err
		}
	}

	// Check to make sure they passed in a type
	if jo.Type != "array" && jo.Type != "object" {
		return nil, errors.New("invalid type, must be array or object")
	}

	if len(jo.Fields) <= 0 {
		return nil, errors.New("must pass fields in order to build json object(s)")
	}

	if jo.Type == "object" {
		v := make(jsonOrderedKeyVal, len(jo.Fields))

		// Loop through fields and add to them to map[string]any
		for i, field := range jo.Fields {
			if field.Function == "autoincrement" {
				// Object only has one
				v[i] = &jsonKeyVal{Key: field.Name, Value: 1}
				continue
			}

			// Get function info
			funcInfo := GetFuncLookup(field.Function)
			if funcInfo == nil {
				return nil, errors.New("invalid function, " + field.Function + " does not exist")
			}

			// Call function value
			value, err := funcInfo.Generate(f, &field.Params, funcInfo)
			if err != nil {
				return nil, err
			}

			if _, ok := value.([]byte); ok {
				// If it's a slice, unmarshal it into an interface
				var val any
				err := json.Unmarshal(value.([]byte), &val)
				if err != nil {
					return nil, err
				}
				value = val
			}

			v[i] = &jsonKeyVal{Key: field.Name, Value: value}

		}

		// Marshal into bytes
		if jo.Indent {
			j, _ := json.MarshalIndent(v, "", "    ")
			return j, nil
		}

		j, _ := json.Marshal(v)
		return j, nil
	}

	if jo.Type == "array" {
		// Make sure you set a row count
		if jo.RowCount <= 0 {
			return nil, errors.New("must have row count")
		}

		v := make([]jsonOrderedKeyVal, jo.RowCount)

		for i := 0; i < int(jo.RowCount); i++ {
			vr := make(jsonOrderedKeyVal, len(jo.Fields))

			// Loop through fields and add to them to map[string]any
			for ii, field := range jo.Fields {
				if field.Function == "autoincrement" {
					vr[ii] = &jsonKeyVal{Key: field.Name, Value: i + 1} // +1 because index starts with 0
					continue
				}

				// Get function info
				funcInfo := GetFuncLookup(field.Function)
				if funcInfo == nil {
					return nil, errors.New("invalid function, " + field.Function + " does not exist")
				}

				// Call function value
				value, err := funcInfo.Generate(f, &field.Params, funcInfo)
				if err != nil {
					return nil, err
				}

				if _, ok := value.([]byte); ok {
					// If it's a slice, unmarshal it into an interface
					var val any
					err := json.Unmarshal(value.([]byte), &val)
					if err != nil {
						return nil, err
					}
					value = val
				}

				vr[ii] = &jsonKeyVal{Key: field.Name, Value: value}
			}

			v[i] = vr
		}

		// Marshal into bytes
		if jo.Indent {
			j, _ := json.MarshalIndent(v, "", "    ")
			return j, nil
		}

		j, _ := json.Marshal(v)
		return j, nil
	}

	return nil, errors.New("invalid type, must be array or object")
}

func addFileJSONLookup() {
	AddFuncLookup("json", Info{
		Display:     "JSON",
		Category:    "file",
		Description: "Format for structured data interchange used in programming, returns an object or an array of objects",
		Example: `[
			{ "first_name": "Markus", "last_name": "Moen", "password": "Dc0VYXjkWABx" },
			{ "first_name": "Osborne", "last_name": "Hilll", "password": "XPJ9OVNbs5lm" },
			{ "first_name": "Mertie", "last_name": "Halvorson", "password": "eyl3bhwfV8wA" }
		]`,
		Output:      "[]byte",
		ContentType: "application/json",
		Aliases:     []string{"data", "interchange", "structured", "format", "serialization", "api"},
		Keywords:    []string{"json", "object", "array", "fields", "indent", "rowcount", "type", "serialize", "deserialize", "marshal", "unmarshal"},
		Params: []Param{
			{Field: "type", Display: "Type", Type: "string", Default: "object", Options: []string{"object", "array"}, Description: "Type of JSON, object or array"},
			{Field: "rowcount", Display: "Row Count", Type: "int", Default: "100", Description: "Number of rows in JSON array"},
			{Field: "indent", Display: "Indent", Type: "bool", Default: "false", Description: "Whether or not to add indents and newlines"},
			{Field: "fields", Display: "Fields", Type: "[]Field", Description: "Fields containing key name and function to run in json format"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			jo := JSONOptions{}

			typ, err := info.GetString(m, "type")
			if err != nil {
				return nil, err
			}
			jo.Type = typ

			rowcount, err := info.GetInt(m, "rowcount")
			if err != nil {
				return nil, err
			}
			jo.RowCount = rowcount

			indent, err := info.GetBool(m, "indent")
			if err != nil {
				return nil, err
			}
			jo.Indent = indent

			fieldsStr, err := info.GetStringArray(m, "fields")
			if err != nil {
				return nil, err
			}

			// Check to make sure fields has length
			if len(fieldsStr) > 0 {
				jo.Fields = make([]Field, len(fieldsStr))

				for i, f := range fieldsStr {
					// Unmarshal fields string into fields array
					err = json.Unmarshal([]byte(f), &jo.Fields[i])
					if err != nil {
						return nil, err
					}
				}
			}

			return jsonFunc(f, &jo)
		},
	})
}

// encoding/json.RawMessage is a special case of []byte
// it cannot be handled as a reflect.Array/reflect.Slice
// because it needs additional structure in the output
func rJsonRawMessage(f *Faker, v reflect.Value, tag string) error {
	if tag != "" {
		err := rCustom(f, v, tag)
		if err == nil {
			jsonData := v.Bytes()
			if !json.Valid(jsonData) {
				fName, _ := parseNameAndParamsFromTag(tag)
				return errors.New("custom function " + fName + " returned invalid json data: " + string(jsonData))
			}
		}
		return err
	}

	b, err := f.JSON(nil)
	if err != nil {
		return err
	}

	v.SetBytes(b)
	return nil
}

// encoding/json.Number is a special case of string
// that represents a JSON number literal.
// It cannot be handled as a string because it needs to
// represent an integer or a floating-point number.
func rJsonNumber(f *Faker, v reflect.Value, tag string) error {
	var ret json.Number

	var numberType string

	if tag == "" {
		numberType = f.RandomString([]string{"int", "float"})

		switch numberType {
		case "int":
			retInt := f.Int16()
			ret = json.Number(strconv.Itoa(int(retInt)))
		case "float":
			retFloat := f.Float64()
			ret = json.Number(strconv.FormatFloat(retFloat, 'f', -1, 64))
		}
	} else {
		fName, fParams := parseNameAndParamsFromTag(tag)
		info := GetFuncLookup(fName)
		if info == nil {
			return fmt.Errorf("invalid function, %s does not exist", fName)
		}

		// Parse map params
		mapParams, err := parseMapParams(info, fParams)
		if err != nil {
			return err
		}

		valueIface, err := info.Generate(f, mapParams, info)
		if err != nil {
			return err
		}

		switch value := valueIface.(type) {
		case int:
			ret = json.Number(strconv.FormatInt(int64(value), 10))
		case int8:
			ret = json.Number(strconv.FormatInt(int64(value), 10))
		case int16:
			ret = json.Number(strconv.FormatInt(int64(value), 10))
		case int32:
			ret = json.Number(strconv.FormatInt(int64(value), 10))
		case int64:
			ret = json.Number(strconv.FormatInt(int64(value), 10))
		case uint:
			ret = json.Number(strconv.FormatUint(uint64(value), 10))
		case uint8:
			ret = json.Number(strconv.FormatUint(uint64(value), 10))
		case uint16:
			ret = json.Number(strconv.FormatUint(uint64(value), 10))
		case uint32:
			ret = json.Number(strconv.FormatUint(uint64(value), 10))
		case uint64:
			ret = json.Number(strconv.FormatUint(uint64(value), 10))
		case float32:
			ret = json.Number(strconv.FormatFloat(float64(value), 'f', -1, 64))
		case float64:
			ret = json.Number(strconv.FormatFloat(float64(value), 'f', -1, 64))
		default:
			return fmt.Errorf("invalid type, %s is not a valid type for json.Number", reflect.TypeOf(value))
		}
	}
	v.Set(reflect.ValueOf(ret))
	return nil
}
