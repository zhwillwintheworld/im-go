package jwt

import (
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)
	if service == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}
	if tokenPair.ExpiresAt <= time.Now().Unix() {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.UserID != 12345 {
		t.Errorf("Expected UserID 12345, got %d", claims.UserID)
	}
	if claims.DeviceID != "device-123" {
		t.Errorf("Expected DeviceID device-123, got %s", claims.DeviceID)
	}
	if claims.Platform != PlatformWeb {
		t.Errorf("Expected Platform web, got %s", claims.Platform)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("Expected TokenType access, got %s", claims.TokenType)
	}
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformIOS)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	claims, err := service.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	if claims.UserID != 12345 {
		t.Errorf("Expected UserID 12345, got %d", claims.UserID)
	}
	if claims.TokenType != RefreshToken {
		t.Errorf("Expected TokenType refresh, got %s", claims.TokenType)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	_, err := service.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateAccessToken_WrongType(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 尝试用 Refresh Token 验证为 Access Token
	_, err = service.ValidateAccessToken(tokenPair.RefreshToken)
	if err == nil {
		t.Error("Expected error when validating refresh token as access token")
	}
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// 创建一个过期时间非常短的 service
	service := NewService("test-secret-key", -time.Hour, 24*time.Hour) // 已过期

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	_, err = service.ValidateAccessToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected error for expired token")
	}
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateAccessToken_WrongSecretKey(t *testing.T) {
	service1 := NewService("secret-key-1", time.Hour, 24*time.Hour)
	service2 := NewService("secret-key-2", time.Hour, 24*time.Hour)

	tokenPair, err := service1.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 使用不同的 secret key 验证
	_, err = service2.ValidateAccessToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected error for wrong secret key")
	}
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestGetAccessExpire(t *testing.T) {
	expire := 2 * time.Hour
	service := NewService("test-secret-key", expire, 24*time.Hour)

	if service.GetAccessExpire() != expire {
		t.Errorf("Expected %v, got %v", expire, service.GetAccessExpire())
	}
}

func TestParseTokenExpireTime(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	tokenPair, err := service.GenerateTokenPair(12345, "device-123", PlatformWeb)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	expireTime, err := ParseTokenExpireTime(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to parse token expire time: %v", err)
	}

	// 检查过期时间是否在大约 1 小时后
	expectedExpire := time.Now().Add(time.Hour)
	diff := expireTime.Sub(expectedExpire)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("Expire time difference too large: %v", diff)
	}
}

func TestAllPlatforms(t *testing.T) {
	service := NewService("test-secret-key", time.Hour, 24*time.Hour)

	platforms := []Platform{
		PlatformUnknown,
		PlatformAndroid,
		PlatformIOS,
		PlatformWeb,
		PlatformDesktop,
		PlatformWechat,
	}

	for _, platform := range platforms {
		t.Run(string(platform), func(t *testing.T) {
			tokenPair, err := service.GenerateTokenPair(12345, "device-123", platform)
			if err != nil {
				t.Fatalf("Failed to generate token pair for platform %s: %v", platform, err)
			}

			claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
			if err != nil {
				t.Fatalf("Failed to validate token for platform %s: %v", platform, err)
			}

			if claims.Platform != platform {
				t.Errorf("Expected platform %s, got %s", platform, claims.Platform)
			}
		})
	}
}
