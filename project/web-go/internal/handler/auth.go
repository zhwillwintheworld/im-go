package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"sudooom.im.web/internal/middleware"
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
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body service.RegisterRequest true "注册信息"
// @Success      200  {object}  response.Response{data=object{user_id=int64,username=string,nickname=string}}
// @Failure      200  {object}  response.Response
// @Router       /auth/register [post]
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
// @Summary      用户登录
// @Description  用户登录获取 Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body service.LoginRequest true "登录信息"
// @Success      200  {object}  response.Response{data=service.LoginResponse}
// @Failure      200  {object}  response.Response
// @Router       /auth/login [post]
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
// @Summary      用户登出
// @Description  用户登出，Token 失效
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	platform := middleware.GetPlatform(c)
	accessToken := middleware.GetAccessToken(c)

	if err := h.authService.Logout(c.Request.Context(), userID, platform, accessToken); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}
