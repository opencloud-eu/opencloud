package gofakeit

import (
	"errors"
)

// Weighted will take in an array of options and weights and return a random selection based upon its indexed weight
func Weighted(options []any, weights []float32) (any, error) {
	return weighted(GlobalFaker, options, weights)
}

// Weighted will take in an array of options and weights and return a random selection based upon its indexed weight
func (f *Faker) Weighted(options []any, weights []float32) (any, error) {
	return weighted(f, options, weights)
}

// Weighted will take in an array of options and weights and return a random selection based upon its indexed weight
func weighted(f *Faker, options []any, weights []float32) (any, error) {
	ol := len(options)
	wl := len(weights)

	// If options length is 1 just return it back
	if ol == 1 {
		return options[0], nil
	}

	// Make sure they are passing in options
	if ol == 0 {
		return nil, errors.New("didnt pass options")
	}

	// Make sure they are passing in weights
	if wl == 0 {
		return nil, errors.New("didnt pass weights")
	}

	// Make sure they are passing in the same length
	if ol != wl {
		return nil, errors.New("options and weights need to be the same length")
	}

	// Compute the discrete cumulative density from the sum of the weights
	cdf := make([]float32, wl)
	var sumOfWeights float32 = 0.0
	for i, weight := range weights {
		if i > 0 {
			cdf[i] = cdf[i-1] + weight
			sumOfWeights += weight
			continue
		}

		cdf[i] = weight
		sumOfWeights += weight
	}

	// Get rand value from a multple of sumOfWeights
	randSumOfWeights := f.Float32() * sumOfWeights

	var l int = 0
	var h int = wl - 1
	for l <= h {
		m := l + (h-l)/2
		if randSumOfWeights <= cdf[m] {
			if m == 0 || (m > 0 && randSumOfWeights > cdf[m-1]) {
				return options[m], nil
			}
			h = m - 1
		} else {
			l = m + 1
		}
	}

	return nil, errors.New("end of function")
}

func addWeightedLookup() {
	AddFuncLookup("weighted", Info{
		Display:     "Weighted",
		Category:    "misc",
		Description: "Randomly select a given option based upon an equal amount of weights",
		Example:     "[hello, 2, 6.9],[1, 2, 3] => 6.9",
		Output:      "any",
		Aliases: []string{
			"weighted choice", "probabilistic pick", "random weight", "distribution choice", "chance selection", "ratio selection", "stochastic option",
		},
		Keywords: []string{
			"randomly", "select", "probability", "distribution", "likelihood", "chance", "statistical", "outcome", "bias", "ratio",
		},
		Params: []Param{
			{Field: "options", Display: "Options", Type: "[]string", Description: "Array of any values"},
			{Field: "weights", Display: "Weights", Type: "[]float", Description: "Array of weights"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			options, err := info.GetStringArray(m, "options")
			if err != nil {
				return nil, err
			}

			weights, err := info.GetFloat32Array(m, "weights")
			if err != nil {
				return nil, err
			}

			optionsInterface := make([]any, len(options))
			for i, o := range options {
				optionsInterface[i] = o
			}

			return weighted(f, optionsInterface, weights)
		},
	})
}
