package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"sudooom.im.shared/jwt"
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
	// TODO: 将 Token 加入黑名单
	response.Success(c, nil)
}

// RefreshToken 刷新 Token
// @Summary      刷新 Token
// @Description  使用 refresh_token 获取新的 access_token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body object{refresh_token=string} true "刷新 Token"
// @Success      200  {object}  response.Response{data=service.LoginResponse}
// @Failure      200  {object}  response.Response
// @Router       /auth/refresh [post]
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
