package service

import (
	"context"
	"errors"

	"sudooom.im.shared/snowflake"
	"sudooom.im.web/internal/model"
	"sudooom.im.web/internal/repository"
)

var (
	ErrCannotAddSelf = errors.New("cannot add yourself as friend")
)

// FriendRequestRequest 好友请求
type FriendRequestRequest struct {
	FriendID int64  `json:"friendId" binding:"required" example:"2"` // 好友用户ID
	Message  string `json:"message" example:"你好，我是张三"`               // 验证消息
}

// FriendService 好友服务
type FriendService struct {
	friendRepo *repository.FriendRepository
	userRepo   *repository.UserRepository
	snowflake  *snowflake.Node
}

// NewFriendService 创建好友服务
func NewFriendService(friendRepo *repository.FriendRepository, userRepo *repository.UserRepository, sf *snowflake.Node) *FriendService {
	return &FriendService{
		friendRepo: friendRepo,
		userRepo:   userRepo,
		snowflake:  sf,
	}
}

// SendRequest 发送好友请求
func (s *FriendService) SendRequest(ctx context.Context, userID int64, req *FriendRequestRequest) error {
	// 不能添加自己
	if userID == req.FriendID {
		return ErrCannotAddSelf
	}

	// 检查目标用户是否存在
	_, err := s.userRepo.GetByID(ctx, req.FriendID)
	if err != nil {
		return err
	}

	// 检查是否已是好友
	isFriend, err := s.friendRepo.IsFriend(ctx, userID, req.FriendID)
	if err != nil {
		return err
	}
	if isFriend {
		return repository.ErrAlreadyFriends
	}

	// 检查是否有待处理的请求
	existingReq, err := s.friendRepo.GetPendingRequest(ctx, userID, req.FriendID)
	if err != nil {
		return err
	}
	if existingReq != nil {
		return repository.ErrRequestPending
	}

	// 创建好友请求
	request := &model.FriendRequest{
		ID:         s.snowflake.Generate().Int64(),
		FromUserID: userID,
		ToUserID:   req.FriendID,
		Message:    req.Message,
		Status:     model.FriendRequestStatusPending,
	}
	return s.friendRepo.CreateRequest(ctx, request)
}

// AcceptRequest 接受好友请求
func (s *FriendService) AcceptRequest(ctx context.Context, userID, requestID int64) error {
	// 获取请求
	request, err := s.friendRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}

	// 验证请求归属
	if request.ToUserID != userID {
		return repository.ErrFriendRequestNotFound
	}

	// 验证状态
	if request.Status != model.FriendRequestStatusPending {
		return repository.ErrFriendRequestNotFound
	}

	// 更新请求状态
	if err := s.friendRepo.UpdateRequestStatus(ctx, requestID, model.FriendRequestStatusAccepted); err != nil {
		return err
	}

	// 创建好友关系（双向，需要两个 ID）
	userFriendID := s.snowflake.Generate().Int64()
	friendFriendID := s.snowflake.Generate().Int64()
	return s.friendRepo.CreateFriendship(ctx, userFriendID, friendFriendID, userID, request.FromUserID)
}

// RejectRequest 拒绝好友请求
func (s *FriendService) RejectRequest(ctx context.Context, userID, requestID int64) error {
	// 获取请求
	request, err := s.friendRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}

	// 验证请求归属
	if request.ToUserID != userID {
		return repository.ErrFriendRequestNotFound
	}

	// 验证状态
	if request.Status != model.FriendRequestStatusPending {
		return repository.ErrFriendRequestNotFound
	}

	return s.friendRepo.UpdateRequestStatus(ctx, requestID, model.FriendRequestStatusRejected)
}

// DeleteFriend 删除好友
func (s *FriendService) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	return s.friendRepo.DeleteFriendship(ctx, userID, friendID)
}

// GetFriends 获取好友列表
func (s *FriendService) GetFriends(ctx context.Context, userID int64) ([]*model.FriendWithUser, error) {
	return s.friendRepo.GetFriends(ctx, userID)
}

// GetPendingRequests 获取待处理的好友请求
func (s *FriendService) GetPendingRequests(ctx context.Context, userID int64) ([]*model.FriendRequestWithUser, error) {
	return s.friendRepo.GetPendingRequestsForUser(ctx, userID)
}
