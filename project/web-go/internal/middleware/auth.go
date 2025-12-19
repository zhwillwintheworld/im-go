package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"sudooom.im.shared/jwt"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/pkg/response"
)

// TokenAuth Token 认证中间件（基于 Redis）
func TokenAuth(tokenRepo *repository.TokenRepository, accessExpire, autoRenewThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 从 Redis 获取用户信息
		userInfo, err := tokenRepo.GetUserInfoByToken(c.Request.Context(), authHeader)
		if err != nil {
			response.Error(c, response.CodeServerError)
			c.Abort()
			return
		}
		if userInfo == nil {
			// token 不存在或已过期
			response.Error(c, response.CodeTokenInvalid)
			c.Abort()
			return
		}

		// 从 JWT 解析过期时间，检查是否需要自动续期（避免额外 Redis 请求）
		expireTime, err := jwt.ParseTokenExpireTime(authHeader)
		if err == nil && !expireTime.IsZero() {
			remaining := time.Until(expireTime)
			if remaining > 0 && remaining < autoRenewThreshold {
				// 自动续期
				_ = tokenRepo.RefreshTokenExpire(c.Request.Context(), userInfo, authHeader, accessExpire)
			}
		}

		c.Set("user_id", userInfo.UserID)
		c.Set("device_id", userInfo.DeviceID)
		c.Set("platform", userInfo.Platform)
		c.Set("access_token", authHeader)
		c.Next()
	}
}

// GetUserID 从 context 获取 user_id
func GetUserID(c *gin.Context) int64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(int64)
}

// GetDeviceID 从 context 获取 device_id
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get("device_id")
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetPlatform 从 context 获取 platform
func GetPlatform(c *gin.Context) string {
	platform, exists := c.Get("platform")
	if !exists {
		return ""
	}
	return platform.(string)
}

// GetAccessToken 从 context 获取 access_token
func GetAccessToken(c *gin.Context) string {
	token, exists := c.Get("access_token")
	if !exists {
		return ""
	}
	return token.(string)
}
