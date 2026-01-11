package util

import (
	"Cornerstone/internal/api/dto"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type ValidationErrorWrapper struct {
	Message string
	Raw     validator.ValidationErrors
}

func (e *ValidationErrorWrapper) Error() string {
	return e.Message
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func ValidateDTO(dto any) error {
	if err := validate.Struct(dto); err != nil {
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) {
			firstError := vErrs[0]
			msg := fmt.Sprintf("参数[%s]错误", firstError.Field())
			return &ValidationErrorWrapper{
				Message: msg,
				Raw:     vErrs,
			}
		}
	}
	return nil
}

func ValidateRegDTO(dto *dto.RegisterDTO) bool {
	if dto.Username != nil && dto.Password != nil {
		if len(*dto.Username) < 4 || len(*dto.Password) < 6 {
			return false
		}
		if len(*dto.Username) > 20 || len(*dto.Password) > 20 {
			return false
		}
		return true
	}
	if dto.Phone != nil {
		if len(*dto.Phone) != 11 || len(*dto.PhoneToken) == 0 {
			return false
		}
		return true
	}
	return false
}

func ValidateLoginDTO(dto *dto.CredentialDTO) bool {
	if dto.Username != nil && dto.Password != nil {
		return true
	}
	if dto.Phone != nil {
		return true
	}
	return false
}
