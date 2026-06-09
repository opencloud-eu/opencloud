package notification

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New(
	validator.WithPrivateFieldValidation(),
	validator.WithRequiredStructEnabled(),
)
