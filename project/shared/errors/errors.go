package errors

import (
	"errors"
	"fmt"
)

// AppError 应用错误类型
// 用于统一管理业务错误，包含错误码和错误消息
type AppError struct {
	Code    int    // 错误码
	Message string // 用户可见的错误消息
	Err     error  // 原始错误（可选，用于调试）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError 创建新错误
func NewError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装原始错误
func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// Is 判断是否为指定错误
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// GetCode 获取错误码，如果不是 AppError 返回默认错误码
func GetCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeServerError // 默认返回服务器错误
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return "服务器内部错误"
}

// ============== 错误码定义 ==============

const (
	CodeSuccess = 0

	// 认证相关 10000-10999
	CodeUsernameExists     = 10001
	CodeInvalidCredentials = 10002
	CodeTokenInvalid       = 10003
	CodeTokenExpired       = 10004
	CodeUserDisabled       = 10005

	// 用户相关 11000-11999
	CodeUserNotFound  = 11001
	CodeInvalidParams = 11002

	// 好友相关 12000-12999
	CodeFriendRequestNotFound = 12001
	CodeAlreadyFriends        = 12002
	CodeCannotAddSelf         = 12003
	CodeRequestPending        = 12004

	// 系统错误 50000-50999
	CodeServerError   = 50001
	CodeDBError       = 50002
	CodeTooManyReqest = 50003
)

// ============== 预定义错误 ==============

// 认证相关
var (
	ErrUsernameExists     = NewError(CodeUsernameExists, "用户名已存在")
	ErrInvalidCredentials = NewError(CodeInvalidCredentials, "用户名或密码错误")
	ErrTokenInvalid       = NewError(CodeTokenInvalid, "Token 无效")
	ErrTokenExpired       = NewError(CodeTokenExpired, "Token 已过期")
	ErrUserDisabled       = NewError(CodeUserDisabled, "用户已被禁用")
)

// 用户相关
var (
	ErrUserNotFound  = NewError(CodeUserNotFound, "用户不存在")
	ErrInvalidParams = NewError(CodeInvalidParams, "参数校验失败")
)

// 好友相关
var (
	ErrFriendRequestNotFound = NewError(CodeFriendRequestNotFound, "好友请求不存在")
	ErrAlreadyFriends        = NewError(CodeAlreadyFriends, "已经是好友关系")
	ErrCannotAddSelf         = NewError(CodeCannotAddSelf, "不能添加自己为好友")
	ErrRequestPending        = NewError(CodeRequestPending, "好友请求待处理中")
)

// 系统相关
var (
	ErrServerError    = NewError(CodeServerError, "服务器内部错误")
	ErrDBError        = NewError(CodeDBError, "数据库错误")
	ErrTooManyRequest = NewError(CodeTooManyReqest, "请求过于频繁，请稍后再试")
)
