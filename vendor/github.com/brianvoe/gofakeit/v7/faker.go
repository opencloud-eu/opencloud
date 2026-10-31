package gofakeit

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"sync"

	"github.com/brianvoe/gofakeit/v7/source"
)

// Create global variable to deal with global function call
var GlobalFaker *Faker = New(0)

// Faker struct is the primary struct for using localized
type Faker struct {
	Rand rand.Source

	// Lock to make thread safe
	Locked bool
	mu     sync.Mutex
}

// New creates and returns a new Faker struct seeded with a given seed
// using the PCG algorithm in lock mode for thread safety
func New(seed uint64) *Faker {
	// If seed is 0, use a random crypto seed
	if seed == 0 {
		faker := NewFaker(source.NewCrypto(), false)
		seed = faker.Uint64()
	}

	return &Faker{
		Rand:   rand.NewPCG(seed, seed),
		Locked: true,
	}
}

// NewFaker takes in a rand.Source and thread lock state and returns a new Faker struct
func NewFaker(src rand.Source, lock bool) *Faker {
	return &Faker{
		Rand:   src,
		Locked: lock,
	}
}

// Seed attempts to seed the Faker with the given seed
func (f *Faker) Seed(args ...any) error {
	// Lock if locked
	if f.Locked {
		f.mu.Lock()
		defer f.mu.Unlock()
	}

	// Ensure GlobalFaker is not nil and Rand is initialized
	if GlobalFaker == nil || GlobalFaker.Rand == nil {
		return errors.New("GlobalFaker or GlobalFaker.Rand is nil")
	}

	// If args is empty or 0, seed with a random crypto seed
	if len(args) == 0 {
		faker := NewFaker(source.NewCrypto(), false)
		args = append(args, faker.Uint64())
	}

	if args[0] == 0 {
		faker := NewFaker(source.NewCrypto(), false)
		args[0] = faker.Uint64()
	}

	// Retrieve the Seed method
	method := reflect.ValueOf(GlobalFaker.Rand).MethodByName("Seed")
	if !method.IsValid() {
		return errors.New("Seed method not found")
	}

	// Adjust args if method requires exactly 2 args but only 1 was provided
	if method.Type().NumIn() == 2 && len(args) == 1 {
		args = append(args, args[0]) // Duplicate the first value if only one is provided
	}

	// Get array of function argument types and prepare converted arguments
	argTypes := make([]reflect.Type, method.Type().NumIn())
	convertedArgs := make([]reflect.Value, len(args))
	for i := 0; i < method.Type().NumIn(); i++ {
		argTypes[i] = method.Type().In(i)
	}

	// Convert args to the expected type by the Seed method
	for i, arg := range args {
		if i < len(argTypes) { // Ensure arg index is within argTypes bounds
			argValue := reflect.ValueOf(arg)
			// Check if conversion is necessary
			if argValue.Type().ConvertibleTo(argTypes[i]) {
				convertedArgs[i] = argValue.Convert(argTypes[i])
			} else {
				// If not convertible, use the argument as is (reflectively)
				convertedArgs[i] = argValue
			}
		}
	}

	// Dynamically call the Seed method with converted arguments
	method.Call(convertedArgs)

	return nil
}

// Seed attempts to seed the GlobalFaker with the given seed
func Seed(args ...any) error {
	return GlobalFaker.Seed(args...)
}
