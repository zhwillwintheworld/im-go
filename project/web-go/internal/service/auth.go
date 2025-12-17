package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"sudooom.im.web/internal/jwt"
	"sudooom.im.web/internal/model"
	"sudooom.im.web/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user is disabled")
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       int64  `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// AuthService 认证服务
type AuthService struct {
	userRepo   *repository.UserRepository
	jwtService *jwt.Service
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo *repository.UserRepository, jwtService *jwt.Service) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtService: jwtService,
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

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		Nickname:     req.Nickname,
		Phone:        req.Phone,
		Email:        req.Email,
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
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, req.DeviceID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID:       user.ID,
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
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, claims.DeviceID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID:       user.ID,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}
