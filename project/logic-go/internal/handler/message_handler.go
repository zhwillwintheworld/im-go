package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器实现
type MessageHandler struct {
	messageBatcher      *service.MessageBatcher // 批量写入器
	messageService      *service.MessageService // 保留用于查询
	userService         *service.UserService
	groupService        *service.GroupService
	routerService       *service.RouterService
	conversationService *service.ConversationService // 会话服务
	logger              *slog.Logger
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	messageBatcher *service.MessageBatcher,
	messageService *service.MessageService,
	userService *service.UserService,
	groupService *service.GroupService,
	routerService *service.RouterService,
	conversationService *service.ConversationService,
) *MessageHandler {
	return &MessageHandler{
		messageBatcher:      messageBatcher,
		messageService:      messageService,
		userService:         userService,
		groupService:        groupService,
		routerService:       routerService,
		conversationService: conversationService,
		logger:              slog.Default(),
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

		// 更新发送者会话
		h.conversationService.UpdateConversationForSender(ctx, msg.FromUserId, msg.ToUserId, 0, serverMsgId)
		// 更新接收者会话
		h.conversationService.UpdateConversationForReceiver(ctx, msg.ToUserId, msg.FromUserId, 0, serverMsgId)

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

		// 更新所有群成员会话
		h.conversationService.UpdateConversationForGroupMembers(ctx, members, msg.FromUserId, msg.ToGroupId, serverMsgId)
	}

	// 4. 多端同步：同步消息给发送者的其他设备
	if err := h.routerService.SyncToSenderOtherDevices(ctx, platform, msg.FromUserId, msg, serverMsgId); err != nil {
		h.logger.Error("Failed to sync to sender other devices", "error", err)
	}
}

// HandleConversationRead 处理会话已读
func (h *MessageHandler) HandleConversationRead(ctx context.Context, event *proto.ConversationRead) {
	if err := h.conversationService.MarkRead(ctx, event.UserId, event.PeerID, event.GroupID, event.LastReadMsgID); err != nil {
		h.logger.Error("Failed to mark conversation read", "userId", event.UserId, "error", err)
	}
	h.logger.Debug("Conversation marked read", "userId", event.UserId, "peerId", event.PeerID, "groupId", event.GroupID)
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
