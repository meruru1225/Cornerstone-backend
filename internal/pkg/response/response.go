package response

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	"errors"
	log "log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
)

const (
	Ok                  = 200
	BadRequest          = 400
	Unauthorized        = 401
	Forbidden           = 403
	NotFound            = 404
	InternalServerError = 500
)

// Success 成功返回封装
func Success(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, dto.Response{
		Code:    Ok,
		Message: "success",
		Data:    data,
	})
}

// Fail 失败返回封装
func Fail(c *gin.Context, businessCode int, message string) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    businessCode,
		Message: message,
		Data:    nil,
	})
}

// Error 处理错误
func Error(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		Fail(c, BadRequest, "参数错误")
		return
	}

	var vew *util.ValidationErrorWrapper
	if errors.As(err, &vew) {
		Fail(c, BadRequest, vew.Message)
		return
	}

	var unmarshalTypeError *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeError) {
		Fail(c, BadRequest, "Json错误")
		return
	}

	if strings.Contains(err.Error(), "strconv") || strings.Contains(err.Error(), "parsing") {
		Fail(c, BadRequest, "查询参数类型错误")
		return
	}

	code, ok := service.ErrorMap[err]
	if !ok {
		code = InternalServerError
		log.Error("Error", "err", err)
	}
	Fail(c, code, err.Error())
}
