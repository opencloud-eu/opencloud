package algorithm

import (
	"errors"
)

var (
	// ErrEncodedHashInvalidFormat is an error returned when an encoded hash has an invalid format.
	ErrEncodedHashInvalidFormat = errors.New("provided encoded hash has an invalid format")

	// ErrEncodedHashInvalidIdentifier is an error returned when an encoded hash has an invalid identifier for the
	// given digest.
	ErrEncodedHashInvalidIdentifier = errors.New("provided encoded hash has an invalid identifier")

	// ErrEncodedHashInvalidVersion is an error returned when an encoded hash has an unsupported or otherwise invalid
	// version.
	ErrEncodedHashInvalidVersion = errors.New("provided encoded hash has an invalid version")

	// ErrEncodedHashInvalidOption is an error returned when an encoded hash has an unsupported or otherwise invalid
	// option in the option field.
	ErrEncodedHashInvalidOption = errors.New("provided encoded hash has an invalid option")

	// ErrEncodedHashInvalidOptionKey is an error returned when an encoded hash has an unknown or otherwise invalid
	// option key in the option field.
	ErrEncodedHashInvalidOptionKey = errors.New("provided encoded hash has an invalid option key")

	// ErrEncodedHashInvalidOptionValue is an error returned when an encoded hash has an unknown or otherwise invalid
	// option value in the option field.
	ErrEncodedHashInvalidOptionValue = errors.New("provided encoded hash has an invalid option value")

	// ErrEncodedHashKeyEncoding is an error returned when an encoded hash has a salt with an invalid or unsupported
	// encoding.
	ErrEncodedHashKeyEncoding = errors.New("provided encoded hash has a key value that can't be decoded")

	// ErrEncodedHashSaltEncoding is an error returned when an encoded hash has a salt with an invalid or unsupported
	// encoding.
	ErrEncodedHashSaltEncoding = errors.New("provided encoded hash has a salt value that can't be decoded")

	// ErrKeyDerivation is returned when a Key function returns an error.
	ErrKeyDerivation = errors.New("failed to derive the key with the provided parameters")

	// ErrSaltEncoding is an error returned when a salt has an invalid or unsupported encoding.
	ErrSaltEncoding = errors.New("provided salt has a value that can't be decoded")

	// ErrPasswordInvalid is an error returned when a password has an invalid or unsupported properties. It is NOT
	// returned on password mismatches.
	ErrPasswordInvalid = errors.New("password is invalid")

	// ErrSaltInvalid is an error returned when a salt has an invalid or unsupported properties.
	ErrSaltInvalid = errors.New("salt is invalid")

	// ErrSaltReadRandomBytes is an error returned when generating the random bytes for salt resulted in an error.
	ErrSaltReadRandomBytes = errors.New("could not read random bytes for salt")

	// ErrParameterInvalid is an error returned when a parameter has an invalid value.
	ErrParameterInvalid = errors.New("parameter is invalid")
)

// Error format strings.
const (
	ErrFmtInvalidIntParameter = "%w: parameter '%s' must be between %d%s and %d but is set to '%d'"
	ErrFmtDigestDecode        = "%s decode error: %w"
	ErrFmtDigestMatch         = "%s match error: %w"
	ErrFmtHasherHash          = "%s hashing error: %w"
	ErrFmtHasherValidation    = "%s validation error: %w"
)
