// Package common 提供全局公共工具，包括统一的 API 返回格式和错误定义。
package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Result 统一 API 返回结构
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

// OK 返回成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    0,
		Message: "success",
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

// OKWithMessage 返回成功响应（自定义消息）
func OKWithMessage(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    0,
		Message: msg,
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

// Fail 返回失败响应
func Fail(c *gin.Context, httpCode int, err error) {
	code := http.StatusInternalServerError
	msg := "内部服务器错误"

	if ae, ok := err.(*AppError); ok {
		code = ae.Code
		msg = ae.Message
	} else if err != nil {
		msg = err.Error()
	}

	c.JSON(httpCode, Result{
		Code:    code,
		Message: msg,
		TraceID: c.GetString("trace_id"),
	})
}

// FailWithCode 返回带有业务错误码的失败响应
func FailWithCode(c *gin.Context, httpCode int, bizCode int, msg string) {
	c.JSON(httpCode, Result{
		Code:    bizCode,
		Message: msg,
		TraceID: c.GetString("trace_id"),
	})
}
