package web

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var (
	validate          = validator.New()
	ErrOptionsInvalid = errors.New("options are invalid")
	ErrRequest        = errors.New("request failed")
)
