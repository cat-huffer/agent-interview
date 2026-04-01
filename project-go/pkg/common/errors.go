package common

import "fmt"

// 业务错误码常量
const (
	ErrCodeInternal     = 10000 // 内部错误
	ErrCodeInvalidParam = 10001 // 参数错误
	ErrCodeUnauthorized = 10002 // 未授权
	ErrCodeNotFound     = 10003 // 资源不存在
	ErrCodeRateLimit    = 10004 // 限流
	ErrCodeLLMFailed    = 20001 // LLM 调用失败
	ErrCodeLLMTimeout   = 20002 // LLM 调用超时
	ErrCodeCircuitOpen  = 20003 // 熔断器打开
	ErrCodeToolFailed   = 30001 // 工具执行失败
	ErrCodeToolNotFound = 30002 // 工具不存在
	ErrCodeRAGFailed    = 40001 // RAG 检索失败
	ErrCodeETLFailed    = 50001 // ETL 处理失败
)

// AppError 应用级错误，携带业务错误码
type AppError struct {
	Code    int    // 业务错误码
	Message string // 面向用户的错误消息
	Cause   error  // 原始错误（内部使用）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is / errors.As 链式解包
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError 创建一个新的应用错误
func NewAppError(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapError 将标准错误包装为应用错误
func WrapError(code int, message string, err error) error {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// ErrInternal 创建内部错误
func ErrInternal(err error) *AppError {
	return NewAppError(ErrCodeInternal, "内部服务器错误", err)
}

// ErrInvalidParam 创建参数错误
func ErrInvalidParam(msg string) *AppError {
	return NewAppError(ErrCodeInvalidParam, msg, nil)
}

// ErrLLMFailed 创建 LLM 调用失败错误
func ErrLLMFailed(err error) *AppError {
	return NewAppError(ErrCodeLLMFailed, "大模型调用失败", err)
}

// ErrCircuitOpen 创建熔断器打开错误
func ErrCircuitOpen(model string) *AppError {
	return NewAppError(ErrCodeCircuitOpen, fmt.Sprintf("模型 %s 熔断器已打开，请稍后重试", model), nil)
}

// ErrToolNotFound 创建工具不存在错误
func ErrToolNotFound(name string) *AppError {
	return NewAppError(ErrCodeToolNotFound, fmt.Sprintf("工具 %s 不存在", name), nil)
}
