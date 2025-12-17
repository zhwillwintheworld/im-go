package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
func CORS(allowedOrigins, allowedMethods []string, allowCredentials bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查 origin 是否在允许列表中
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", joinStrings(allowedMethods))
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		gin.DefaultWriter.Write([]byte(
			formatLog(status, latency, clientIP, method, path),
		))
	}
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func formatLog(status int, latency time.Duration, clientIP, method, path string) string {
	return "[GIN] " + time.Now().Format("2006/01/02 - 15:04:05") +
		" | " + string(rune(status)) +
		" | " + latency.String() +
		" | " + clientIP +
		" | " + method +
		" | " + path + "\n"
}
