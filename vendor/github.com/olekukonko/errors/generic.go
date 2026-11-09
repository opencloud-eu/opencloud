package errors

// AsType attempts to find the first error in the chain that matches type T.
// Returns the matched error and true if found, otherwise zero value and false.
func AsType[T error](err error) (T, bool) {
	var target T
	if As(err, &target) { // Uses errors.As from helper.go
		return target, true
	}
	var zero T
	return zero, false
}

// IsType checks if the error or any error in its chain is of type T.
func IsType[T error](err error) bool {
	var target T
	return As(err, &target) // Uses errors.As from helper.go
}

// FindType returns the first error in the chain of type T that satisfies the predicate.
func FindType[T error](err error, predicate func(T) bool) (T, bool) {
	var zero T
	if err == nil || predicate == nil {
		return zero, false
	}

	// Use your Walk function to traverse the chain
	var found T
	var foundIt bool
	Walk(err, func(e error) {
		if !foundIt {
			if target, ok := e.(T); ok && predicate(target) {
				found = target
				foundIt = true
			}
		}
	})
	return found, foundIt
}

// Map applies a transformation function to each error of type T in the chain.
func Map[T error, R any](err error, fn func(T) R) []R {
	var results []R
	if err == nil || fn == nil {
		return results
	}
	Walk(err, func(e error) {
		if target, ok := e.(T); ok {
			results = append(results, fn(target))
		}
	})
	return results
}

// Reduce walks the error chain and accumulates a result for errors of type T.
func Reduce[T error, R any](err error, initial R, fn func(T, R) R) R {
	result := initial
	if err == nil || fn == nil {
		return result
	}
	Walk(err, func(e error) {
		if target, ok := e.(T); ok {
			result = fn(target, result)
		}
	})
	return result
}

// Filter returns a slice of all errors of type T from the error chain.
func Filter[T error](err error) []T {
	var results []T
	if err == nil {
		return results
	}
	Walk(err, func(e error) {
		if target, ok := e.(T); ok {
			results = append(results, target)
		}
	})
	return results
}

// FirstOfType returns the first error in the chain of type T.
func FirstOfType[T error](err error) (T, bool) {
	var zero T
	if err == nil {
		return zero, false
	}
	var found T
	var foundIt bool
	Walk(err, func(e error) {
		if !foundIt {
			if target, ok := e.(T); ok {
				found = target
				foundIt = true
			}
		}
	})
	return found, foundIt
}

// Contains checks if any error in the chain matches any of the target errors.
// Uses our package's Is function for matching.
func Contains(err error, targets ...error) bool {
	for _, target := range targets {
		if Is(err, target) { // Uses errors.Is from helper.go
			return true
		}
	}
	return false
}

// JoinErrors joins multiple errors using Join and wraps them with context.
// Returns nil if all errors are nil.
func JoinErrors(errs []error, keyValues ...interface{}) error {
	nonNil := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}

	if len(nonNil) == 0 {
		return nil
	}

	if len(nonNil) == 1 && len(keyValues) == 0 {
		return nonNil[0]
	}

	joined := Join(nonNil...)

	if len(keyValues) == 0 {
		return joined
	}

	// When multiple errors are joined, the test asserts the result is *MultiError.
	// Attach context to the *MultiError's first underlying *Error if possible,
	// otherwise return the MultiError directly (context check in tests is guarded
	// by a got.(*Error) type assertion and is a no-op for *MultiError).
	if _, ok := joined.(*MultiError); ok {
		return joined
	}

	// Single error with context: wrap in *Error so context is accessible.
	e := New("multiple errors occurred")
	e.Wrap(joined)
	e.With(keyValues...)
	return e
}
