package service

import (
	"context"

	"sudooom.im.web/internal/model"
	"sudooom.im.web/internal/repository"
)

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" example:"张三"`                           // 昵称
	Avatar   string `json:"avatar" example:"https://example.com/avatar.png"` // 头像URL
}

// UserService 用户服务
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetByID 通过 ID 获取用户
func (s *UserService) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateProfile 更新用户资料
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req *UpdateProfileRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	return s.userRepo.Update(ctx, user)
}

// Search 搜索用户
func (s *UserService) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.userRepo.Search(ctx, keyword, pageSize, offset)
}
