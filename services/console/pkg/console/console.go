package console

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var (
	Validate          = validator.New()
	ErrValidation     = errors.New("failed to validate")
	ErrOptionsInvalid = errors.New("options are invalid")
	ErrRequest        = errors.New("request failed")
)
