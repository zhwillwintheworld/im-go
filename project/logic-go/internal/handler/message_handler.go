package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器实现
type MessageHandler struct {
	messageBatcher *service.MessageBatcher // 改用批量写入器
	messageService *service.MessageService // 保留用于查询
	userService    *service.UserService
	groupService   *service.GroupService
	routerService  *service.RouterService
	logger         *slog.Logger
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	messageBatcher *service.MessageBatcher,
	messageService *service.MessageService,
	userService *service.UserService,
	groupService *service.GroupService,
	routerService *service.RouterService,
) *MessageHandler {
	return &MessageHandler{
		messageBatcher: messageBatcher,
		messageService: messageService,
		userService:    userService,
		groupService:   groupService,
		routerService:  routerService,
		logger:         slog.Default(),
	}
}

// HandleUserMessage 处理用户消息
func (h *MessageHandler) HandleUserMessage(ctx context.Context, msg *proto.UserMessage, accessNodeId string, platform string) {
	// 1. 异步批量消息存储（立即返回 serverMsgId）
	serverMsgId, err := h.messageBatcher.SaveMessage(msg)
	if err != nil {
		h.logger.Error("Failed to queue message for saving", "error", err)
		return
	}

	// 2. 发送 ACK 给发送者（直接回复到消息来源的 access 节点）
	if err := h.routerService.SendAckToUserDirect(ctx, accessNodeId, msg.FromUserId, msg.ClientMsgId, serverMsgId); err != nil {
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

	// 4. 多端同步：同步消息给发送者的其他设备
	if err := h.routerService.SyncToSenderOtherDevices(ctx, platform, msg.FromUserId, msg, serverMsgId); err != nil {
		h.logger.Error("Failed to sync to sender other devices", "error", err)
	}
}

// HandleUserOnline 处理用户上线
func (h *MessageHandler) HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string) {
	// location 由 access-go 管理，这里只记录日志
	h.logger.Info("User online", "userId", event.UserId, "accessNodeId", accessNodeId, "platform", event.Platform)
}

// HandleUserOffline 处理用户下线
func (h *MessageHandler) HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string) {
	// location 由 access-go 管理，这里只记录日志
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
