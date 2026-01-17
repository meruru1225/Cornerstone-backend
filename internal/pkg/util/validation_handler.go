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

func ValidateRegDTO(dto *dto.RegisterDTO) error {
	var vErrs validator.ValidationErrors
	if dto.Username != nil && dto.Password != nil {
		if len(*dto.Username) < 4 || len(*dto.Password) < 6 {
			return &ValidationErrorWrapper{
				Message: "用户名或密码长度错误",
				Raw:     vErrs,
			}
		}
		if len(*dto.Username) > 20 || len(*dto.Password) > 20 {
			return &ValidationErrorWrapper{
				Message: "用户名或密码长度错误",
				Raw:     vErrs,
			}
		}
	}
	if dto.Phone != nil {
		if len(*dto.Phone) != 11 || len(*dto.PhoneToken) == 0 {
			return &ValidationErrorWrapper{
				Message: "手机号或验证码错误",
				Raw:     vErrs,
			}
		}
	}
	return ValidateDTO(dto)
}
