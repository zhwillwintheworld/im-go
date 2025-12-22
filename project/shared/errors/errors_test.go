package errors

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	err := NewError(10001, "test error")

	if err.Code != 10001 {
		t.Errorf("Expected code 10001, got %d", err.Code)
	}
	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Error("Expected Err to be nil")
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "without wrapped error",
			err:      NewError(10001, "test error"),
			expected: "[10001] test error",
		},
		{
			name:     "with wrapped error",
			err:      NewError(10001, "test error").Wrap(errors.New("original error")),
			expected: "[10001] test error: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestAppError_Wrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := ErrUserNotFound.Wrap(originalErr)

	if appErr.Code != ErrUserNotFound.Code {
		t.Errorf("Expected code %d, got %d", ErrUserNotFound.Code, appErr.Code)
	}
	if appErr.Message != ErrUserNotFound.Message {
		t.Errorf("Expected message '%s', got '%s'", ErrUserNotFound.Message, appErr.Message)
	}
	if appErr.Err != originalErr {
		t.Error("Expected wrapped error to be the original error")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := ErrUserNotFound.Wrap(originalErr)

	unwrapped := errors.Unwrap(appErr)
	if unwrapped != originalErr {
		t.Error("Expected unwrapped error to be the original error")
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   *AppError
		expected bool
	}{
		{
			name:     "same error",
			err:      ErrUserNotFound,
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "wrapped same error",
			err:      ErrUserNotFound.Wrap(errors.New("wrapped")),
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "different error",
			err:      ErrInvalidCredentials,
			target:   ErrUserNotFound,
			expected: false,
		},
		{
			name:     "non-app error",
			err:      errors.New("standard error"),
			target:   ErrUserNotFound,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "app error",
			err:      ErrUserNotFound,
			expected: CodeUserNotFound,
		},
		{
			name:     "wrapped app error",
			err:      ErrInvalidCredentials.Wrap(errors.New("wrapped")),
			expected: CodeInvalidCredentials,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: CodeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "app error",
			err:      ErrUserNotFound,
			expected: "用户不存在",
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: "服务器内部错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessage(tt.err); got != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	// 验证预定义错误的 Code 是否正确
	predefinedErrors := map[*AppError]int{
		ErrUsernameExists:        CodeUsernameExists,
		ErrInvalidCredentials:    CodeInvalidCredentials,
		ErrTokenInvalid:          CodeTokenInvalid,
		ErrTokenExpired:          CodeTokenExpired,
		ErrUserDisabled:          CodeUserDisabled,
		ErrUserNotFound:          CodeUserNotFound,
		ErrInvalidParams:         CodeInvalidParams,
		ErrFriendRequestNotFound: CodeFriendRequestNotFound,
		ErrAlreadyFriends:        CodeAlreadyFriends,
		ErrCannotAddSelf:         CodeCannotAddSelf,
		ErrRequestPending:        CodeRequestPending,
		ErrServerError:           CodeServerError,
		ErrDBError:               CodeDBError,
		ErrTooManyRequest:        CodeTooManyReqest,
	}

	for err, expectedCode := range predefinedErrors {
		if err.Code != expectedCode {
			t.Errorf("Error %s: expected code %d, got %d", err.Message, expectedCode, err.Code)
		}
	}
}
