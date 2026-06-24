package http

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"salespilot/internal/http/httperr"
)

type Validator struct {
	v *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{v: validator.New()}
}

func (val *Validator) Validate(i interface{}) error {
	if err := val.v.Struct(i); err != nil {
		var msgs []string
		for _, fe := range err.(validator.ValidationErrors) {
			msgs = append(msgs, fmt.Sprintf("%s: %s", fe.Field(), fe.Tag()))
		}
		return httperr.NewValidation(strings.Join(msgs, "; "))
	}
	return nil
}
