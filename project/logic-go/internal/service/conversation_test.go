package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	sharedRedis "sudooom.im.shared/redis"
)

// 注意：这些测试需要一个运行中的 Redis 实例
// 如果没有 Redis，测试将被跳过

func getTestRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试专用数据库
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("跳过测试：无法连接 Redis: %v", err)
	}

	// 清理测试数据库
	client.FlushDB(ctx)

	return client
}

func TestConversationService_UpdateConversationForSender(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)
	peerId := int64(2001)
	msgId := int64(3001)

	// 测试更新发送者会话
	err := svc.UpdateConversationForSender(ctx, userId, peerId, 0, msgId)
	if err != nil {
		t.Fatalf("UpdateConversationForSender failed: %v", err)
	}

	// 验证会话索引已创建
	idxKey := sharedRedis.BuildConversationIndexKey(userId)
	members, err := client.ZRange(ctx, idxKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 member in index, got %d", len(members))
	}

	expectedMember := sharedRedis.BuildConversationPeerMember(peerId)
	if members[0] != expectedMember {
		t.Errorf("Expected member '%s', got '%s'", expectedMember, members[0])
	}

	// 验证会话详情已创建
	convKey := sharedRedis.BuildConversationPeerKey(userId, peerId)
	lastMsgId, err := client.HGet(ctx, convKey, "last_msg_id").Int64()
	if err != nil {
		t.Fatalf("Failed to get last_msg_id: %v", err)
	}
	if lastMsgId != msgId {
		t.Errorf("Expected last_msg_id %d, got %d", msgId, lastMsgId)
	}
}

func TestConversationService_UpdateConversationForReceiver(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)
	peerId := int64(2001)
	msgId := int64(3001)

	// 测试更新接收者会话
	err := svc.UpdateConversationForReceiver(ctx, userId, peerId, 0, msgId)
	if err != nil {
		t.Fatalf("UpdateConversationForReceiver failed: %v", err)
	}

	// 验证未读数递增
	convKey := sharedRedis.BuildConversationPeerKey(userId, peerId)
	unreadCount, err := client.HGet(ctx, convKey, "unread_count").Int64()
	if err != nil {
		t.Fatalf("Failed to get unread_count: %v", err)
	}
	if unreadCount != 1 {
		t.Errorf("Expected unread_count 1, got %d", unreadCount)
	}

	// 再次接收消息
	err = svc.UpdateConversationForReceiver(ctx, userId, peerId, 0, msgId+1)
	if err != nil {
		t.Fatalf("Second UpdateConversationForReceiver failed: %v", err)
	}

	unreadCount, _ = client.HGet(ctx, convKey, "unread_count").Int64()
	if unreadCount != 2 {
		t.Errorf("Expected unread_count 2, got %d", unreadCount)
	}
}

func TestConversationService_MarkRead(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)
	peerId := int64(2001)
	msgId := int64(3001)

	// 先创建一个有未读消息的会话
	err := svc.UpdateConversationForReceiver(ctx, userId, peerId, 0, msgId)
	if err != nil {
		t.Fatalf("UpdateConversationForReceiver failed: %v", err)
	}

	// 标记已读
	err = svc.MarkRead(ctx, userId, peerId, 0, msgId)
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	// 验证未读数清零
	convKey := sharedRedis.BuildConversationPeerKey(userId, peerId)
	unreadCount, err := client.HGet(ctx, convKey, "unread_count").Int64()
	if err != nil {
		t.Fatalf("Failed to get unread_count: %v", err)
	}
	if unreadCount != 0 {
		t.Errorf("Expected unread_count 0, got %d", unreadCount)
	}

	// 验证 last_read_msg_id 已更新
	lastReadMsgId, err := client.HGet(ctx, convKey, "last_read_msg_id").Int64()
	if err != nil {
		t.Fatalf("Failed to get last_read_msg_id: %v", err)
	}
	if lastReadMsgId != msgId {
		t.Errorf("Expected last_read_msg_id %d, got %d", msgId, lastReadMsgId)
	}
}

func TestConversationService_GetUserConversations(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)

	// 创建多个会话
	for i := int64(1); i <= 3; i++ {
		peerId := int64(2000 + i)
		msgId := int64(3000 + i)
		err := svc.UpdateConversationForSender(ctx, userId, peerId, 0, msgId)
		if err != nil {
			t.Fatalf("UpdateConversationForSender failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	}

	// 获取会话列表
	conversations, err := svc.GetUserConversations(ctx, userId, 0, 10)
	if err != nil {
		t.Fatalf("GetUserConversations failed: %v", err)
	}

	if len(conversations) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(conversations))
	}

	// 验证倒序排列（最新的在前）
	if len(conversations) >= 2 && conversations[0].UpdateAt < conversations[1].UpdateAt {
		t.Error("Conversations should be sorted by update_at descending")
	}
}

func TestConversationService_GetTotalUnreadCount(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)

	// 创建多个有未读消息的会话
	for i := int64(1); i <= 3; i++ {
		peerId := int64(2000 + i)
		msgId := int64(3000 + i)
		err := svc.UpdateConversationForReceiver(ctx, userId, peerId, 0, msgId)
		if err != nil {
			t.Fatalf("UpdateConversationForReceiver failed: %v", err)
		}
	}

	// 获取总未读数
	totalUnread, err := svc.GetTotalUnreadCount(ctx, userId)
	if err != nil {
		t.Fatalf("GetTotalUnreadCount failed: %v", err)
	}

	if totalUnread != 3 {
		t.Errorf("Expected total unread count 3, got %d", totalUnread)
	}
}

func TestConversationService_GroupConversation(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	svc := NewConversationService(client)
	ctx := context.Background()

	userId := int64(1001)
	groupId := int64(5001)
	msgId := int64(3001)

	// 测试群聊会话更新
	err := svc.UpdateConversationForSender(ctx, userId, 0, groupId, msgId)
	if err != nil {
		t.Fatalf("UpdateConversationForSender (group) failed: %v", err)
	}

	// 验证会话索引已创建（群聊）
	idxKey := sharedRedis.BuildConversationIndexKey(userId)
	members, err := client.ZRange(ctx, idxKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	expectedMember := sharedRedis.BuildConversationGroupMember(groupId)
	if len(members) != 1 || members[0] != expectedMember {
		t.Errorf("Expected member '%s', got %v", expectedMember, members)
	}

	// 验证群聊会话详情已创建
	convKey := sharedRedis.BuildConversationGroupKey(userId, groupId)
	lastMsgId, err := client.HGet(ctx, convKey, "last_msg_id").Int64()
	if err != nil {
		t.Fatalf("Failed to get last_msg_id: %v", err)
	}
	if lastMsgId != msgId {
		t.Errorf("Expected last_msg_id %d, got %d", msgId, lastMsgId)
	}
}
