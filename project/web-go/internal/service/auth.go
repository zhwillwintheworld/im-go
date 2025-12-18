package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"sudooom.im.shared/jwt"
	"sudooom.im.shared/snowflake"
	"sudooom.im.web/internal/model"
	"sudooom.im.web/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user is disabled")
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"zhangsan"` // 用户名
	Password string `json:"password" binding:"required,min=6,max=50" example:"123456"`   // 密码
	Nickname string `json:"nickname" binding:"required,min=1,max=50" example:"张三"`       // 昵称
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"zhangsan"`    // 用户名
	Password string `json:"password" binding:"required" example:"123456"`      // 密码
	DeviceID string `json:"device_id" example:"device-uuid-123"`               // 设备ID
	Platform string `json:"platform" example:"pc" enums:"pc,mini_program,app"` // 平台类型
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       int64  `json:"user_id" example:"1"`                                     // 用户ID
	ObjectCode   string `json:"object_code" example:"1234567890123456789"`               // 用户唯一标识
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6..."`  // 访问令牌
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6..."` // 刷新令牌
	ExpiresAt    int64  `json:"expires_at" example:"1702915200"`                         // 过期时间戳
}

// AuthService 认证服务
type AuthService struct {
	userRepo   *repository.UserRepository
	tokenRepo  *repository.TokenRepository
	jwtService *jwt.Service
	snowflake  *snowflake.Node
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository, jwtService *jwt.Service, sf *snowflake.Node) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtService: jwtService,
		snowflake:  sf,
	}
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	// 检查用户名是否存在
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrUsernameExists
	}

	// 密码加密
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 生成雪花ID
	objectCode := s.snowflake.Generate().String()

	user := &model.User{
		ObjectCode:   objectCode,
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		Nickname:     req.Nickname,
		Status:       model.UserStatusNormal,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 查询用户
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if user.Status != model.UserStatusNormal {
		return nil, ErrUserDisabled
	}

	// 生成 Token
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, req.DeviceID, jwt.Platform(req.Platform))
	if err != nil {
		return nil, err
	}

	// 删除旧Token（同一用户同一平台只保留一个Token）
	if err := s.tokenRepo.DeleteOldToken(ctx, user.ID, req.Platform); err != nil {
		return nil, err
	}

	// 存储Token到Redis
	userTokenInfo := &repository.UserTokenInfo{
		UserID:     user.ID,
		ObjectCode: user.ObjectCode,
		Username:   user.Username,
		Nickname:   user.Nickname,
		Avatar:     user.Avatar,
		DeviceID:   req.DeviceID,
		Platform:   req.Platform,
	}
	if err := s.tokenRepo.SaveToken(ctx, userTokenInfo, tokenPair.AccessToken, s.jwtService.GetAccessExpire()); err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID:       user.ID,
		ObjectCode:   user.ObjectCode,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// RefreshToken 刷新 Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 验证 Refresh Token
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// 检查用户状态
	if user.Status != model.UserStatusNormal {
		return nil, ErrUserDisabled
	}

	// 生成新的 Token Pair
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, claims.DeviceID, claims.Platform)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID:       user.ID,
		ObjectCode:   user.ObjectCode,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}
