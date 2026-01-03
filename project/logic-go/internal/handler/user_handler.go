package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// UserHandler 用户事件处理器
type UserHandler struct {
	conversationService *service.ConversationService
	routerService       *service.RouterService
	logger              *slog.Logger
}

// NewUserHandler 创建用户事件处理器
func NewUserHandler(conversationService *service.ConversationService, routerService *service.RouterService) *UserHandler {
	return &UserHandler{
		conversationService: conversationService,
		routerService:       routerService,
		logger:              slog.Default(),
	}
}

// HandleUserOnline 处理用户上线
func (h *UserHandler) HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string) {
	// location 由 access-go 管理，这里只记录日志
	h.logger.Info("User online",
		"userId", event.UserId,
		"platform", event.Platform,
		"deviceId", event.DeviceId,
		"accessNodeId", accessNodeId)
}

// HandleUserOffline 处理用户下线
func (h *UserHandler) HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string) {
	// 清除位置缓存
	h.routerService.InvalidateUserCache(event.UserId)

	h.logger.Info("User offline",
		"userId", event.UserId,
		"accessNodeId", accessNodeId)
}

// HandleConversationRead 处理会话已读
func (h *UserHandler) HandleConversationRead(ctx context.Context, event *proto.ConversationRead) {
	if err := h.conversationService.MarkRead(ctx, event.UserId, event.PeerID, event.GroupID, event.LastReadMsgID); err != nil {
		h.logger.Error("Failed to mark conversation read", "userId", event.UserId, "error", err)
		return
	}
	h.logger.Debug("Conversation marked read",
		"userId", event.UserId,
		"peerId", event.PeerID,
		"groupId", event.GroupID,
		"lastReadMsgId", event.LastReadMsgID)
}
