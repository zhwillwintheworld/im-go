package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器实现
type MessageHandler struct {
	messageService *service.MessageService
	userService    *service.UserService
	groupService   *service.GroupService
	routerService  *service.RouterService
	logger         *slog.Logger
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	messageService *service.MessageService,
	userService *service.UserService,
	groupService *service.GroupService,
	routerService *service.RouterService,
) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		userService:    userService,
		groupService:   groupService,
		routerService:  routerService,
		logger:         slog.Default(),
	}
}

// HandleUserMessage 处理用户消息
func (h *MessageHandler) HandleUserMessage(ctx context.Context, msg *proto.UserMessage, accessNodeId string) {
	// 1. 消息存储
	serverMsgId, err := h.messageService.SaveMessage(ctx, msg)
	if err != nil {
		h.logger.Error("Failed to save message", "error", err)
		return
	}

	// 2. 发送 ACK 给发送者
	if err := h.routerService.SendAckToUser(ctx, msg.FromUserId, msg.ClientMsgId, serverMsgId); err != nil {
		h.logger.Error("Failed to send ack", "error", err)
	}

	// 3. 路由消息给接收者
	if msg.ToUserId > 0 {
		// 单聊消息
		if err := h.routerService.RouteMessage(ctx, msg.ToUserId, msg, serverMsgId); err != nil {
			h.logger.Error("Failed to route message to user", "toUserId", msg.ToUserId, "error", err)
		}
	} else if msg.ToGroupId > 0 {
		// 群聊消息
		members, err := h.groupService.GetGroupMembers(ctx, msg.ToGroupId)
		if err != nil {
			h.logger.Error("Failed to get group members", "groupId", msg.ToGroupId, "error", err)
			return
		}
		// 过滤发送者
		filteredMembers := filterOut(members, msg.FromUserId)
		if err := h.routerService.RouteToMultiple(ctx, filteredMembers, msg, serverMsgId); err != nil {
			h.logger.Error("Failed to route message to group", "groupId", msg.ToGroupId, "error", err)
		}
	}
}

// HandleUserOnline 处理用户上线
func (h *MessageHandler) HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string) {
	err := h.userService.RegisterUserLocation(ctx, event.UserId, accessNodeId, event.ConnId, event.DeviceId, event.Platform)
	if err != nil {
		h.logger.Error("Failed to register user location", "userId", event.UserId, "error", err)
		return
	}
	h.logger.Info("User online", "userId", event.UserId, "accessNodeId", accessNodeId)
}

// HandleUserOffline 处理用户下线
func (h *MessageHandler) HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string) {
	err := h.userService.UnregisterUserLocation(ctx, event.UserId, event.ConnId, accessNodeId)
	if err != nil {
		h.logger.Error("Failed to unregister user location", "userId", event.UserId, "error", err)
		return
	}
	h.logger.Info("User offline", "userId", event.UserId, "accessNodeId", accessNodeId)
}

// filterOut 过滤掉指定用户
func filterOut(members []int64, excludeId int64) []int64 {
	result := make([]int64, 0, len(members))
	for _, m := range members {
		if m != excludeId {
			result = append(result, m)
		}
	}
	return result
}
