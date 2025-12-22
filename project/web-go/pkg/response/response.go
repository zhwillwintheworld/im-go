package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	sharedErrors "sudooom.im.shared/errors"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 错误码常量（使用 shared/errors 包的定义）
const (
	CodeSuccess = sharedErrors.CodeSuccess

	// 认证相关 10000-10999
	CodeUsernameExists     = sharedErrors.CodeUsernameExists
	CodeInvalidCredentials = sharedErrors.CodeInvalidCredentials
	CodeTokenInvalid       = sharedErrors.CodeTokenInvalid
	CodeTokenExpired       = sharedErrors.CodeTokenExpired
	CodeUserDisabled       = sharedErrors.CodeUserDisabled

	// 用户相关 11000-11999
	CodeUserNotFound  = sharedErrors.CodeUserNotFound
	CodeInvalidParams = sharedErrors.CodeInvalidParams

	// 好友相关 12000-12999
	CodeFriendRequestNotFound = sharedErrors.CodeFriendRequestNotFound
	CodeAlreadyFriends        = sharedErrors.CodeAlreadyFriends
	CodeCannotAddSelf         = sharedErrors.CodeCannotAddSelf
	CodeRequestPending        = sharedErrors.CodeRequestPending

	// 系统错误 50000-50999
	CodeServerError = sharedErrors.CodeServerError
	CodeDBError     = sharedErrors.CodeDBError
)

// 错误信息（保留用于向后兼容）
var codeMessages = map[int]string{
	CodeSuccess:               "success",
	CodeUsernameExists:        "用户名已存在",
	CodeInvalidCredentials:    "用户名或密码错误",
	CodeTokenInvalid:          "Token 无效",
	CodeTokenExpired:          "Token 已过期",
	CodeUserDisabled:          "用户已被禁用",
	CodeUserNotFound:          "用户不存在",
	CodeInvalidParams:         "参数校验失败",
	CodeFriendRequestNotFound: "好友请求不存在",
	CodeAlreadyFriends:        "已经是好友关系",
	CodeCannotAddSelf:         "不能添加自己为好友",
	CodeRequestPending:        "好友请求待处理中",
	CodeServerError:           "服务器内部错误",
	CodeDBError:               "数据库错误",
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int) {
	message := codeMessages[code]
	if message == "" {
		message = "unknown error"
	}
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorWithMsg 自定义错误消息
func ErrorWithMsg(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorFromAppError 从 AppError 生成错误响应
func ErrorFromAppError(c *gin.Context, err error) {
	code := sharedErrors.GetCode(err)
	message := sharedErrors.GetMessage(err)
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// Unauthorized 未认证
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    CodeTokenInvalid,
		Message: codeMessages[CodeTokenInvalid],
		Data:    nil,
	})
}

// TooManyRequests 请求过多
func TooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, Response{
		Code:    sharedErrors.CodeTooManyReqest,
		Message: "请求过于频繁，请稍后再试",
		Data:    nil,
	})
}
