package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"sudooom.im.web/internal/jwt"
	"sudooom.im.web/pkg/response"
)

// JWTAuth JWT 认证中间件
func JWTAuth(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c.GetHeader("Authorization"))
		if token == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				response.Error(c, response.CodeTokenExpired)
			} else {
				response.Error(c, response.CodeTokenInvalid)
			}
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("device_id", claims.DeviceID)
		c.Next()
	}
}

// extractToken 从 Authorization header 提取 token
func extractToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
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
