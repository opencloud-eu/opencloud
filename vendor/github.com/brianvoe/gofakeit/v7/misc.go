package gofakeit

import (
	"encoding/hex"
	"reflect"

	"github.com/brianvoe/gofakeit/v7/data"
)

// Bool will generate a random boolean value
func Bool() bool { return boolFunc(GlobalFaker) }

// Bool will generate a random boolean value
func (f *Faker) Bool() bool { return boolFunc(f) }

func boolFunc(f *Faker) bool { return randIntRange(f, 0, 1) == 1 }

// UUID (version 4) will generate a random unique identifier based upon random numbers
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func UUID() string { return uuid(GlobalFaker) }

// UUID (version 4) will generate a random unique identifier based upon random numbers
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx 8-4-4-4-12
func (f *Faker) UUID() string { return uuid(f) }

func uuid(f *Faker) string {
	version := byte(4)
	uuid := make([]byte, 16)

	// Read 16 random bytes
	for i := 0; i < 16; i++ {
		uuid[i] = byte(f.IntN(256))
	}

	// Set version
	uuid[6] = (uuid[6] & 0x0f) | (version << 4)

	// Set variant
	uuid[8] = (uuid[8] & 0xbf) | 0x80

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = dash
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = dash
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = dash
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = dash
	hex.Encode(buf[24:], uuid[10:])

	return string(buf)
}

// ShuffleAnySlice takes in a slice and outputs it in a random order
func ShuffleAnySlice(v any) { shuffleAnySlice(GlobalFaker, v) }

// ShuffleAnySlice takes in a slice and outputs it in a random order
func (f *Faker) ShuffleAnySlice(v any) { shuffleAnySlice(f, v) }

func shuffleAnySlice(f *Faker, v any) {
	if v == nil {
		return
	}

	// Check type of passed in value, if not a slice return with no action taken
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Slice {
		return
	}

	s := reflect.ValueOf(v)
	n := s.Len()

	if n <= 1 {
		return
	}

	swap := func(i, j int) {
		tmp := reflect.ValueOf(s.Index(i).Interface())
		s.Index(i).Set(s.Index(j))
		s.Index(j).Set(tmp)
	}

	//if size is > int32 probably it will never finish, or ran out of entropy
	i := n - 1
	for ; i > 0; i-- {
		j := int(int32NFunc(f, int32(i+1)))
		swap(i, j)
	}
}

// FlipACoin will return a random value of Heads or Tails
func FlipACoin() string { return flipACoin(GlobalFaker) }

// FlipACoin will return a random value of Heads or Tails
func (f *Faker) FlipACoin() string { return flipACoin(f) }

func flipACoin(f *Faker) string {
	if boolFunc(f) {
		return "Heads"
	}

	return "Tails"
}

// RandomMapKey will return a random key from a map
func RandomMapKey(mapI any) any { return randomMapKey(GlobalFaker, mapI) }

// RandomMapKey will return a random key from a map
func (f *Faker) RandomMapKey(mapI any) any { return randomMapKey(f, mapI) }

func randomMapKey(f *Faker, mapI any) any {
	keys := reflect.ValueOf(mapI).MapKeys()
	return keys[f.IntN(len(keys))].Interface()
}

// Categories will return a map string array of available data categories and sub categories
func Categories() map[string][]string {
	types := make(map[string][]string)
	for category, subCategoriesMap := range data.Data {
		subCategories := make([]string, 0)
		for subType := range subCategoriesMap {
			subCategories = append(subCategories, subType)
		}
		types[category] = subCategories
	}
	return types
}

func addMiscLookup() {
	AddFuncLookup("uuid", Info{
		Display:     "UUID",
		Category:    "misc",
		Description: "128-bit identifier used to uniquely identify objects or entities in computer systems",
		Example:     "590c1440-9888-45b0-bd51-a817ee07c3f2",
		Output:      "string",
		Aliases:     []string{"identifier", "unique", "guid", "id", "128-bit"},
		Keywords:    []string{"uuid", "computer", "system", "objects", "entities", "identify", "version", "hexadecimal"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return uuid(f), nil
		},
	})

	AddFuncLookup("bool", Info{
		Display:     "Boolean",
		Category:    "misc",
		Description: "Data type that represents one of two possible values, typically true or false",
		Example:     "true",
		Output:      "bool",
		Aliases:     []string{"boolean", "true", "false", "logic", "binary"},
		Keywords:    []string{"bool", "data", "type", "represents", "values", "typically", "two", "possible"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return boolFunc(f), nil
		},
	})

	AddFuncLookup("flipacoin", Info{
		Display:     "Flip A Coin",
		Category:    "misc",
		Description: "Decision-making method involving the tossing of a coin to determine outcomes",
		Example:     "Tails",
		Output:      "string",
		Aliases:     []string{"coin", "flip", "heads", "tails", "decision", "random"},
		Keywords:    []string{"decision-making", "method", "tossing", "determine", "outcomes", "chance", "probability"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return flipACoin(f), nil
		},
	})
}
