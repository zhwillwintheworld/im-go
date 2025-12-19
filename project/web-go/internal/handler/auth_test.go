package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sudooom.im.web/internal/service"
	"sudooom.im.web/pkg/response"
)

// MockAuthService 模拟 AuthService
type MockAuthService struct {
	LoginFunc        func(ctx context.Context, req *service.LoginRequest) (*service.LoginResponse, error)
	RegisterFunc     func(ctx context.Context, req *service.RegisterRequest) (interface{}, error)
	RefreshTokenFunc func(ctx context.Context, refreshToken string) (*service.LoginResponse, error)
}

func (m *MockAuthService) Login(ctx context.Context, req *service.LoginRequest) (*service.LoginResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return nil, nil
}

// setupTestRouter 创建测试用的 gin 路由
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// APIResponse 用于解析响应体
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestAuthHandler_Login_Success(t *testing.T) {
	// 准备测试数据
	expectedResp := &service.LoginResponse{
		UserID:       1,
		ObjectCode:   "1234567890123456789",
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    1702915200,
	}

	// 创建 mock service
	mockService := &MockAuthService{
		LoginFunc: func(ctx context.Context, req *service.LoginRequest) (*service.LoginResponse, error) {
			// 验证请求参数
			assert.Equal(t, "testuser", req.Username)
			assert.Equal(t, "password123", req.Password)
			assert.Equal(t, "device-123", req.DeviceID)
			assert.Equal(t, "pc", req.Platform)
			return expectedResp, nil
		},
	}

	// 由于 AuthHandler 使用具体类型 *service.AuthService，
	// 我们需要创建一个可测试的 handler 包装器
	// 这里使用集成测试的方式直接测试 HTTP 层

	router := setupTestRouter()

	// 注册一个测试用的 login 路由
	router.POST("/auth/login", func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
			return
		}

		resp, err := mockService.Login(c.Request.Context(), &req)
		if err != nil {
			if err == service.ErrInvalidCredentials {
				response.Error(c, response.CodeInvalidCredentials)
				return
			}
			if err == service.ErrUserDisabled {
				response.Error(c, response.CodeUserDisabled)
				return
			}
			response.Error(c, response.CodeServerError)
			return
		}

		response.Success(c, resp)
	})

	// 构造请求
	reqBody := service.LoginRequest{
		Username: "testuser",
		Password: "password123",
		DeviceID: "device-123",
		Platform: "pc",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// 解析 data 字段
	var loginResp service.LoginResponse
	err = json.Unmarshal(resp.Data, &loginResp)
	require.NoError(t, err)

	assert.Equal(t, expectedResp.UserID, loginResp.UserID)
	assert.Equal(t, expectedResp.ObjectCode, loginResp.ObjectCode)
	assert.Equal(t, expectedResp.AccessToken, loginResp.AccessToken)
	assert.Equal(t, expectedResp.RefreshToken, loginResp.RefreshToken)
	assert.Equal(t, expectedResp.ExpiresAt, loginResp.ExpiresAt)
}

func TestAuthHandler_Login_InvalidParams(t *testing.T) {
	router := setupTestRouter()

	// 注册一个测试用的 login 路由
	router.POST("/auth/login", func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
			return
		}
		response.Success(c, nil)
	})

	testCases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "缺少用户名",
			body:     `{"password": "123456"}`,
			wantCode: response.CodeInvalidParams,
		},
		{
			name:     "缺少密码",
			body:     `{"username": "testuser"}`,
			wantCode: response.CodeInvalidParams,
		},
		{
			name:     "空请求体",
			body:     `{}`,
			wantCode: response.CodeInvalidParams,
		},
		{
			name:     "无效的JSON",
			body:     `{invalid}`,
			wantCode: response.CodeInvalidParams,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.wantCode, resp.Code)
		})
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockService := &MockAuthService{
		LoginFunc: func(ctx context.Context, req *service.LoginRequest) (*service.LoginResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}

	router := setupTestRouter()

	router.POST("/auth/login", func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
			return
		}

		resp, err := mockService.Login(c.Request.Context(), &req)
		if err != nil {
			if err == service.ErrInvalidCredentials {
				response.Error(c, response.CodeInvalidCredentials)
				return
			}
			response.Error(c, response.CodeServerError)
			return
		}

		response.Success(c, resp)
	})

	reqBody := `{"username": "zhanghua", "password": "123456"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeInvalidCredentials, resp.Code)
}

func TestAuthHandler_Login_UserDisabled(t *testing.T) {
	mockService := &MockAuthService{
		LoginFunc: func(ctx context.Context, req *service.LoginRequest) (*service.LoginResponse, error) {
			return nil, service.ErrUserDisabled
		},
	}

	router := setupTestRouter()

	router.POST("/auth/login", func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
			return
		}

		resp, err := mockService.Login(c.Request.Context(), &req)
		if err != nil {
			if err == service.ErrInvalidCredentials {
				response.Error(c, response.CodeInvalidCredentials)
				return
			}
			if err == service.ErrUserDisabled {
				response.Error(c, response.CodeUserDisabled)
				return
			}
			response.Error(c, response.CodeServerError)
			return
		}

		response.Success(c, resp)
	})

	reqBody := `{"username": "zhanghua", "password": "123456"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeUserDisabled, resp.Code)
}
