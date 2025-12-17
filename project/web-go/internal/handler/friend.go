package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"sudooom.im.web/internal/middleware"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/internal/service"
	"sudooom.im.web/pkg/response"
)

// FriendHandler 好友处理器
type FriendHandler struct {
	friendService *service.FriendService
}

// NewFriendHandler 创建好友处理器
func NewFriendHandler(friendService *service.FriendService) *FriendHandler {
	return &FriendHandler{friendService: friendService}
}

// GetFriendList 获取好友列表
// GET /api/v1/friends
func (h *FriendHandler) GetFriendList(c *gin.Context) {
	userID := middleware.GetUserID(c)

	friends, err := h.friendService.GetFriends(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	var result []gin.H
	for _, f := range friends {
		result = append(result, gin.H{
			"id":         f.ID,
			"friend_id":  f.FriendID,
			"username":   f.Username,
			"nickname":   f.Nickname,
			"avatar":     f.Avatar,
			"remark":    f.Remark,
			"create_at": f.CreateAt,
		})
	}

	response.Success(c, gin.H{"list": result})
}

// SendRequest 发送好友请求
// POST /api/v1/friends/request
func (h *FriendHandler) SendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req service.FriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
		return
	}

	err := h.friendService.SendRequest(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrCannotAddSelf) {
			response.Error(c, response.CodeCannotAddSelf)
			return
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		if errors.Is(err, repository.ErrAlreadyFriends) {
			response.Error(c, response.CodeAlreadyFriends)
			return
		}
		if errors.Is(err, repository.ErrRequestPending) {
			response.Error(c, response.CodeRequestPending)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}

// GetPendingRequests 获取待处理的好友请求
// GET /api/v1/friends/requests
func (h *FriendHandler) GetPendingRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)

	requests, err := h.friendService.GetPendingRequests(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	var result []gin.H
	for _, r := range requests {
		result = append(result, gin.H{
			"id":            r.ID,
			"from_user_id":  r.FromUserID,
			"from_username": r.FromUsername,
			"from_nickname": r.FromNickname,
			"from_avatar":   r.FromAvatar,
			"message":   r.Message,
			"create_at": r.CreateAt,
		})
	}

	response.Success(c, gin.H{"list": result})
}

// AcceptRequest 接受好友请求
// POST /api/v1/friends/accept/:id
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, "invalid request id")
		return
	}

	if err := h.friendService.AcceptRequest(c.Request.Context(), userID, requestID); err != nil {
		if errors.Is(err, repository.ErrFriendRequestNotFound) {
			response.Error(c, response.CodeFriendRequestNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}

// RejectRequest 拒绝好友请求
// POST /api/v1/friends/reject/:id
func (h *FriendHandler) RejectRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, "invalid request id")
		return
	}

	if err := h.friendService.RejectRequest(c.Request.Context(), userID, requestID); err != nil {
		if errors.Is(err, repository.ErrFriendRequestNotFound) {
			response.Error(c, response.CodeFriendRequestNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}

// DeleteFriend 删除好友
// DELETE /api/v1/friends/:id
func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	userID := middleware.GetUserID(c)

	friendID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, "invalid friend id")
		return
	}

	if err := h.friendService.DeleteFriend(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}
