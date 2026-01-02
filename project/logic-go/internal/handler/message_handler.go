package handler

import (
	"context"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器组合器
// 实现 nats.MessageHandler 接口,将请求委托给各个子 handler
type MessageHandler struct {
	chatHandler *ChatHandler
	roomHandler *RoomHandler
	gameHandler *GameHandler
	userHandler *UserHandler
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	messageBatcher *service.MessageBatcher,
	messageService *service.MessageService,
	groupService *service.GroupService,
	routerService *service.RouterService,
	conversationService *service.ConversationService,
	redisClient *redis.Client,
	roomService *service.RoomService,
) *MessageHandler {
	return &MessageHandler{
		chatHandler: NewChatHandler(messageBatcher, messageService, groupService, routerService, conversationService),
		roomHandler: NewRoomHandler(redisClient, roomService),
		gameHandler: NewGameHandler(redisClient),
		userHandler: NewUserHandler(conversationService),
	}
}

// HandleUserMessage 处理用户消息
func (h *MessageHandler) HandleUserMessage(ctx context.Context, msg *proto.UserMessage, accessNodeId string, connId int64, platform string) {
	h.chatHandler.Handle(ctx, msg, accessNodeId, connId, platform)
}

// HandleConversationRead 处理会话已读
func (h *MessageHandler) HandleConversationRead(ctx context.Context, event *proto.ConversationRead) {
	h.userHandler.HandleConversationRead(ctx, event)
}

// HandleUserOnline 处理用户上线
func (h *MessageHandler) HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string) {
	h.userHandler.HandleUserOnline(ctx, event, accessNodeId)
}

// HandleUserOffline 处理用户下线
func (h *MessageHandler) HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string) {
	h.userHandler.HandleUserOffline(ctx, event, accessNodeId)
}

// HandleRoomRequest 处理房间请求
func (h *MessageHandler) HandleRoomRequest(ctx context.Context, req *proto.RoomRequest, accessNodeId string) {
	_ = h.roomHandler.Handle(ctx, req, accessNodeId)
}

// HandleGameRequest 处理游戏请求
func (h *MessageHandler) HandleGameRequest(ctx context.Context, req *proto.GameRequest, accessNodeId string) {
	_ = h.gameHandler.Handle(ctx, req, accessNodeId)
}
