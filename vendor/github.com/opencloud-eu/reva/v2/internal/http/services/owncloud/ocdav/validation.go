package ocdav

import (
	"errors"
	"fmt"
	"strings"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
)

// Validator validates strings
type Validator func(string) error

// ValidatorsFromConfig returns the configured Validators
func ValidatorsFromConfig(c *config.Config) []Validator {
	vals := []Validator{
		notEmpty(), // we always want to exclude empty names
		doesNotContain(c.NameValidation.InvalidChars), // forbidden characters
		isShorterThan(c.NameValidation.MaxLength),     // max length
	}

	return vals
}

// ValidateName will validate a file or folder name, returning an error when it is not accepted
func ValidateName(name string, validators []Validator) error {
	return ValidateDestination(name, append(validators, notReserved()))
}

// ValidateDestination will validate a file or folder destination name (which can be . or ..), returning an error when it is not accepted
func ValidateDestination(name string, validators []Validator) error {
	for _, v := range validators {
		if err := v(name); err != nil {
			return fmt.Errorf("name validation failed: %w", err)
		}
	}
	return nil
}

func notReserved() Validator {
	return func(s string) error {
		if s == ".." || s == "." {
			return errors.New(". and .. are reserved names")
		}
		return nil
	}
}

func notEmpty() Validator {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("must not be empty")
		}
		return nil
	}
}

func doesNotContain(bad []string) Validator {
	return func(s string) error {
		for _, b := range bad {
			if strings.Contains(s, b) {
				return fmt.Errorf("must not contain %s", b)
			}
		}
		return nil
	}
}

func isShorterThan(maxLength int) Validator {
	return func(s string) error {
		if len(s) > maxLength {
			return fmt.Errorf("must be shorter than %d", maxLength)
		}
		return nil
	}
}
