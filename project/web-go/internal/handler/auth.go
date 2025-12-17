package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"sudooom.im.web/internal/jwt"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/internal/service"
	"sudooom.im.web/pkg/response"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, repository.ErrUsernameExists) {
			response.Error(c, response.CodeUsernameExists)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, response.CodeInvalidCredentials)
			return
		}
		if errors.Is(err, service.ErrUserDisabled) {
			response.Error(c, response.CodeUserDisabled)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, resp)
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: 将 Token 加入黑名单
	response.Success(c, nil)
}

// RefreshToken 刷新 Token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenInvalid) {
			response.Error(c, response.CodeTokenInvalid)
			return
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			response.Error(c, response.CodeTokenExpired)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, resp)
}
