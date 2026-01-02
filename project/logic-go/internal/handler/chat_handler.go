package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// ChatHandler 聊天消息处理器
type ChatHandler struct {
	messageBatcher      *service.MessageBatcher
	messageService      *service.MessageService
	groupService        *service.GroupService
	routerService       *service.RouterService
	conversationService *service.ConversationService
	logger              *slog.Logger
}

// NewChatHandler 创建聊天消息处理器
func NewChatHandler(
	messageBatcher *service.MessageBatcher,
	messageService *service.MessageService,
	groupService *service.GroupService,
	routerService *service.RouterService,
	conversationService *service.ConversationService,
) *ChatHandler {
	return &ChatHandler{
		messageBatcher:      messageBatcher,
		messageService:      messageService,
		groupService:        groupService,
		routerService:       routerService,
		conversationService: conversationService,
		logger:              slog.Default(),
	}
}

// Handle 处理聊天消息
func (h *ChatHandler) Handle(ctx context.Context, msg *proto.UserMessage, accessNodeId string, connId int64, platform string) {
	// 1. 异步批量消息存储（立即返回 serverMsgId）
	serverMsgId, err := h.messageBatcher.SaveMessage(msg)
	if err != nil {
		h.logger.Error("Failed to queue message for saving", "error", err)
		return
	}

	// 直接回 ACK 给发送者（使用 connId 避免查询 Redis）
	if err := h.routerService.SendAckToUserDirect(ctx, accessNodeId, connId, msg.FromUserId, msg.ClientMsgId, serverMsgId); err != nil {
		h.logger.Error("Failed to send ack", "error", err)
	}

	// 3. 路由消息给接收者
	if msg.ToUserId > 0 {
		// 单聊消息
		if err := h.routerService.RouteMessage(ctx, msg.ToUserId, msg, serverMsgId); err != nil {
			h.logger.Error("Failed to route message to user", "toUserId", msg.ToUserId, "error", err)
		}

		// 异步更新会话（非关键路径）
		go func() {
			if err := h.conversationService.UpdateConversationForSender(context.Background(), msg.FromUserId, msg.ToUserId, 0, serverMsgId); err != nil {
				h.logger.Error("Failed to update conversation for sender", "error", err, "fromUserId", msg.FromUserId, "toUserId", msg.ToUserId)
			}
			if err := h.conversationService.UpdateConversationForReceiver(context.Background(), msg.ToUserId, msg.FromUserId, 0, serverMsgId); err != nil {
				h.logger.Error("Failed to update conversation for receiver", "error", err, "toUserId", msg.ToUserId, "fromUserId", msg.FromUserId)
			}
		}()

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

		// 异步更新所有群成员会话（非关键路径）
		go func() {
			h.conversationService.UpdateConversationForGroupMembers(context.Background(), members, msg.FromUserId, msg.ToGroupId, serverMsgId)
		}()
	}

	// 4. 异步多端同步：同步消息给发送者的其他设备（非关键路径）
	go func() {
		if err := h.routerService.SyncToSenderOtherDevices(context.Background(), platform, msg.FromUserId, msg, serverMsgId); err != nil {
			h.logger.Error("Failed to sync to sender other devices", "error", err)
		}
	}()
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
