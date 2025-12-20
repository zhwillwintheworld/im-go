package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token has expired")
)

// TokenType Token 类型
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Platform 平台类型
type Platform string

const (
	PlatformUnknown Platform = "unknown" // 未知
	PlatformAndroid Platform = "android" // Android
	PlatformIOS     Platform = "ios"     // iOS
	PlatformWeb     Platform = "web"     // Web 网页
	PlatformDesktop Platform = "desktop" // 桌面应用
	PlatformWechat  Platform = "wechat"  // 微信小程序
)

// Claims JWT 声明
type Claims struct {
	UserID    int64     `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	Platform  Platform  `json:"platform"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair Token 对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Service JWT 服务
type Service struct {
	secretKey     []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewService 创建 JWT 服务
func NewService(secretKey string, accessExpire, refreshExpire time.Duration) *Service {
	return &Service{
		secretKey:     []byte(secretKey),
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// GenerateTokenPair 生成 Token 对
func (s *Service) GenerateTokenPair(userID int64, deviceID string, platform Platform) (*TokenPair, error) {
	now := time.Now()
	accessExpiresAt := now.Add(s.accessExpire)
	refreshExpiresAt := now.Add(s.refreshExpire)

	// 生成 Access Token
	accessToken, err := s.generateToken(userID, deviceID, platform, AccessToken, accessExpiresAt)
	if err != nil {
		return nil, err
	}

	// 生成 Refresh Token
	refreshToken, err := s.generateToken(userID, deviceID, platform, RefreshToken, refreshExpiresAt)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt.Unix(),
	}, nil
}

// generateToken 生成单个 Token
func (s *Service) generateToken(userID int64, deviceID string, platform Platform, tokenType TokenType, expiresAt time.Time) (string, error) {
	claims := &Claims{
		UserID:    userID,
		DeviceID:  deviceID,
		Platform:  platform,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "im-web",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateAccessToken 验证 Access Token
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, AccessToken)
}

// ValidateRefreshToken 验证 Refresh Token
func (s *Service) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, RefreshToken)
}

// GetAccessExpire 获取 AccessToken 过期时长
func (s *Service) GetAccessExpire() time.Duration {
	return s.accessExpire
}

// validateToken 验证 Token
func (s *Service) validateToken(tokenString string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.TokenType != expectedType {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ParseTokenExpireTime 解析 Token 获取过期时间（不验证签名，用于快速获取过期时间）
func ParseTokenExpireTime(tokenString string) (time.Time, error) {
	// 使用 ParseUnverified 不验证签名，只解析 claims
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return time.Time{}, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.ExpiresAt == nil {
		return time.Time{}, ErrTokenInvalid
	}

	return claims.ExpiresAt.Time, nil
}
