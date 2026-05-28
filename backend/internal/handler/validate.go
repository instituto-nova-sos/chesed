package handler

import (
	"github.com/go-playground/validator/v10"
)

var requestValidator = validator.New()

func validateStruct(s any) error {
	return requestValidator.Struct(s)
}
