package gofakeit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// FuncLookups is the primary map array with mapping to all available data
var FuncLookups map[string]Info
var lockFuncLookups sync.Mutex

// MapParams is the values to pass into a lookup generate
type MapParams map[string]MapParamsValue

type MapParamsValue []string

// Info structures fields to better break down what each one generates
type Info struct {
	Display     string                                                `json:"display"`      // display name
	Category    string                                                `json:"category"`     // category
	Description string                                                `json:"description"`  // description
	Example     string                                                `json:"example"`      // example
	Output      string                                                `json:"output"`       // output type
	Aliases     []string                                              `json:"aliases"`      // alt names users might type
	Keywords    []string                                              `json:"keywords"`     // free words and domain terms
	ContentType string                                                `json:"content_type"` // content type
	Params      []Param                                               `json:"params"`       // params
	Any         any                                                   `json:"any"`          // any
	Generate    func(f *Faker, m *MapParams, info *Info) (any, error) `json:"-"`            // generate function
}

// Param is a breakdown of param requirements and type definition
type Param struct {
	Field       string   `json:"field"`
	Display     string   `json:"display"`
	Type        string   `json:"type"`
	Optional    bool     `json:"optional"`
	Default     string   `json:"default"`
	Options     []string `json:"options"`
	Description string   `json:"description"`
}

// Field is used for defining what name and function you to generate for file outuputs
type Field struct {
	Name     string    `json:"name"`
	Function string    `json:"function"`
	Params   MapParams `json:"params"`
}

func init() { initLookup() }

// init will add all the functions to MapLookups
func initLookup() {
	addAddressLookup()
	addAnimalLookup()
	addAppLookup()
	addAuthLookup()
	addBeerLookup()
	addBookLookup()
	addCarLookup()
	addCelebrityLookup()
	addColorLookup()
	addCompanyLookup()
	addDatabaseSQLLookup()
	addDateTimeLookup()
	addEmojiLookup()
	addErrorLookup()
	addFileCSVLookup()
	addFileJSONLookup()
	addFileLookup()
	addFileXMLLookup()
	addFinanceLookup()
	addFoodLookup()
	addGameLookup()
	addGenerateLookup()
	addHackerLookup()
	addHipsterLookup()
	addHtmlLookup()
	addImageLookup()
	addInternetLookup()
	addLanguagesLookup()
	addMinecraftLookup()
	addMiscLookup()
	addMovieLookup()
	addNumberLookup()
	addPaymentLookup()
	addPersonLookup()
	addProductLookup()
	addSchoolLookup()
	addSongLookup()
	addStringLookup()
	addTemplateLookup()
	addWeightedLookup()
	addWordAdjectiveLookup()
	addWordAdverbLookup()
	addWordConnectiveLookup()
	addWordGeneralLookup()
	addWordGrammerLookup()
	addWordNounLookup()
	addWordPrepositionLookup()
	addWordPronounLookup()
	addWordVerbLookup()
	addWordMiscLookup()
	addTextLookup()
}

// internalFuncLookups is the internal map array with mapping to all available data
var internalFuncLookups map[string]Info = map[string]Info{
	"fields": {
		Description: "Example fields for generating csv, json, xml, etc",
		Output:      "gofakeit.Field",
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			function, _ := GetRandomSimpleFunc(f)
			return Field{
				Name:     function,
				Function: function,
			}, nil
		},
	},
}

// NewMapParams will create a new MapParams
func NewMapParams() *MapParams {
	return &MapParams{}
}

// Add will take in a field and value and add it to the map params type
func (m *MapParams) Add(field string, value string) {
	_, ok := (*m)[field]
	if !ok {
		(*m)[field] = []string{value}
		return
	}

	(*m)[field] = append((*m)[field], value)
}

// Get will return the array of string from the provided field
func (m *MapParams) Get(field string) []string {
	return (*m)[field]
}

// Size will return the total size of the underlying map
func (m *MapParams) Size() int {
	size := 0
	for range *m {
		size++
	}
	return size
}

// UnmarshalJSON will unmarshal the json into the []string
func (m *MapParamsValue) UnmarshalJSON(data []byte) error {
	// check if the data is an array
	// if so, marshal it into m
	if data[0] == '[' {
		var values []any
		err := json.Unmarshal(data, &values)
		if err != nil {
			return err
		}

		// convert the values to array of strings
		for _, value := range values {
			typeOf := reflect.TypeOf(value).Kind().String()

			if typeOf == "map" {
				v, err := json.Marshal(value)
				if err != nil {
					return err
				}
				*m = append(*m, string(v))
			} else {
				*m = append(*m, fmt.Sprintf("%v", value))
			}
		}
		return nil
	}

	// if not, then convert into a string and add it to m
	var s any
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	*m = append(*m, fmt.Sprintf("%v", s))
	return nil
}

func GetRandomSimpleFunc(f *Faker) (string, Info) {
	// Loop through all the functions and add them to a slice
	var keys []string
	for k, info := range FuncLookups {
		// Only grab simple functions
		if info.Params == nil {
			keys = append(keys, k)
		}
	}

	// Randomly grab a function from the slice
	randomKey := randomString(f, keys)

	// Return the function name and info
	return randomKey, FuncLookups[randomKey]
}

// AddFuncLookup takes a field and adds it to map
func AddFuncLookup(functionName string, info Info) {
	if FuncLookups == nil {
		FuncLookups = make(map[string]Info)
	}

	// Check content type
	if info.ContentType == "" {
		info.ContentType = "text/plain"
	}

	lockFuncLookups.Lock()
	FuncLookups[functionName] = info
	lockFuncLookups.Unlock()
}

// GetFuncLookup will lookup
func GetFuncLookup(functionName string) *Info {
	var info Info
	var ok bool

	// Check internal functions first
	info, ok = internalFuncLookups[functionName]
	if ok {
		return &info
	}

	info, ok = FuncLookups[functionName]
	if ok {
		return &info
	}

	return nil
}

// RemoveFuncLookup will remove a function from lookup
func RemoveFuncLookup(functionName string) {
	_, ok := FuncLookups[functionName]
	if !ok {
		return
	}

	lockFuncLookups.Lock()
	delete(FuncLookups, functionName)
	lockFuncLookups.Unlock()
}

// GetAny will retrieve Any field from Info
func (i *Info) GetAny(m *MapParams, field string) (any, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	// Make sure value[0] exists
	if len(value) == 0 {
		return nil, fmt.Errorf("could not find field: %s", field)
	}

	var anyValue any

	// Try to convert to int
	valueInt, err := strconv.ParseInt(value[0], 10, 64)
	if err == nil {
		return int(valueInt), nil
	}

	// Try to convert to float
	valueFloat, err := strconv.ParseFloat(value[0], 64)
	if err == nil {
		return valueFloat, nil
	}

	// Try to convert to boolean
	valueBool, err := strconv.ParseBool(value[0])
	if err == nil {
		return valueBool, nil
	}

	err = json.Unmarshal([]byte(value[0]), &anyValue)
	if err == nil {
		return valueBool, nil
	}

	return value[0], nil
}

// GetMap will retrieve map[string]any field from data
func (i *Info) GetMap(m *MapParams, field string) (map[string]any, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	var mapValue map[string]any
	err = json.Unmarshal([]byte(value[0]), &mapValue)
	if err != nil {
		return nil, fmt.Errorf("%s field could not parse to map[string]any", field)
	}

	return mapValue, nil
}

// GetField will retrieve field from data
func (i *Info) GetField(m *MapParams, field string) (*Param, []string, error) {
	// Get param
	var p *Param
	for _, param := range i.Params {
		if param.Field == field {
			p = &param
			break
		}
	}
	if p == nil {
		return nil, nil, fmt.Errorf("could not find param field %s", field)
	}

	// Get value from map
	if m != nil {
		value, ok := (*m)[field]
		if !ok {
			// If default isnt empty use default
			if p.Default != "" {
				return p, []string{p.Default}, nil
			}

			return nil, nil, fmt.Errorf("could not find field: %s", field)
		}

		return p, value, nil
	} else if p.Default != "" {
		// If p.Type is []uint, then we need to convert it to []string
		if strings.HasPrefix(p.Default, "[") {
			// Remove [] from type
			defaultClean := p.Default[1 : len(p.Default)-1]

			// Split on comma
			defaultSplit := strings.Split(defaultClean, ",")

			return p, defaultSplit, nil
		}

		// If default isnt empty use default
		return p, []string{p.Default}, nil
	}

	return nil, nil, fmt.Errorf("could not find field: %s", field)
}

// GetBool will retrieve boolean field from data
func (i *Info) GetBool(m *MapParams, field string) (bool, error) {
	p, value, err := i.GetField(m, field)
	if err != nil {
		return false, err
	}

	// Try to convert to boolean
	valueBool, err := strconv.ParseBool(value[0])
	if err != nil {
		return false, fmt.Errorf("%s field could not parse to bool value", p.Field)
	}

	return valueBool, nil
}

// GetInt will retrieve int field from data
func (i *Info) GetInt(m *MapParams, field string) (int, error) {
	p, value, err := i.GetField(m, field)
	if err != nil {
		return 0, err
	}

	// Try to convert to int
	valueInt, err := strconv.ParseInt(value[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s field could not parse to int value", p.Field)
	}

	return int(valueInt), nil
}

// GetUint will retrieve uint field from data
func (i *Info) GetUint(m *MapParams, field string) (uint, error) {
	p, value, err := i.GetField(m, field)
	if err != nil {
		return 0, err
	}

	// Try to convert to int
	valueUint, err := strconv.ParseUint(value[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s field could not parse to int value", p.Field)
	}

	return uint(valueUint), nil
}

// GetFloat32 will retrieve int field from data
func (i *Info) GetFloat32(m *MapParams, field string) (float32, error) {
	p, value, err := i.GetField(m, field)
	if err != nil {
		return 0, err
	}

	// Try to convert to float
	valueFloat, err := strconv.ParseFloat(value[0], 32)
	if err != nil {
		return 0, fmt.Errorf("%s field could not parse to float value", p.Field)
	}

	return float32(valueFloat), nil
}

// GetFloat64 will retrieve int field from data
func (i *Info) GetFloat64(m *MapParams, field string) (float64, error) {
	p, value, err := i.GetField(m, field)
	if err != nil {
		return 0, err
	}

	// Try to convert to float
	valueFloat, err := strconv.ParseFloat(value[0], 64)
	if err != nil {
		return 0, fmt.Errorf("%s field could not parse to float value", p.Field)
	}

	return valueFloat, nil
}

// GetString will retrieve string field from data
func (i *Info) GetString(m *MapParams, field string) (string, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return "", err
	}

	return value[0], nil
}

// GetStringArray will retrieve []string field from data
func (i *Info) GetStringArray(m *MapParams, field string) ([]string, error) {
	_, values, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// GetIntArray will retrieve []int field from data
func (i *Info) GetIntArray(m *MapParams, field string) ([]int, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	var ints []int
	for i := 0; i < len(value); i++ {
		valueInt, err := strconv.ParseInt(value[i], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s value could not parse to int", value[i])
		}
		ints = append(ints, int(valueInt))
	}

	return ints, nil
}

// GetUintArray will retrieve []uint field from data
func (i *Info) GetUintArray(m *MapParams, field string) ([]uint, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	var uints []uint
	for i := 0; i < len(value); i++ {
		valueUint, err := strconv.ParseUint(value[i], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s value could not parse to uint", value[i])
		}
		uints = append(uints, uint(valueUint))
	}

	return uints, nil
}

// GetFloat32Array will retrieve []float field from data
func (i *Info) GetFloat32Array(m *MapParams, field string) ([]float32, error) {
	_, value, err := i.GetField(m, field)
	if err != nil {
		return nil, err
	}

	var floats []float32
	for i := 0; i < len(value); i++ {
		valueFloat, err := strconv.ParseFloat(value[i], 32)
		if err != nil {
			return nil, fmt.Errorf("%s value could not parse to float", value[i])
		}
		floats = append(floats, float32(valueFloat))
	}

	return floats, nil
}
