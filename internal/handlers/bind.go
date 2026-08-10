package handlers

import (
	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/go-playground/validator/v10"
)

func bindError(err error) error {
	if _, ok := err.(validator.ValidationErrors); ok {
		return apperr.Validation("invalid request body")
	}
	return apperr.Validation("invalid request body")
}
