package service

import (
	"errors"
	"testing"
	"time"

	"sudooom.im.shared/proto"
	"sudooom.im.shared/snowflake"
)

func TestMessageBatcherSaveMessageReturnsDuplicateForSameClientMsgID(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{
		BatchSize:     10,
		FlushInterval: time.Hour,
	})

	msg := &proto.UserMessage{
		ClientMsgId: "client-1",
		FromUserId:  1001,
		ToUserId:    2001,
		MsgType:     1,
		Content:     []byte("hello"),
	}

	firstServerMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, nil)
	if err != nil {
		t.Fatalf("第一次入队失败: %v", err)
	}
	if duplicate {
		t.Fatal("第一次入队不应标记为重复消息")
	}

	secondServerMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, nil)
	if err != nil {
		t.Fatalf("重复消息不应返回错误: %v", err)
	}
	if !duplicate {
		t.Fatal("相同 fromUserId + clientMsgId 应标记为重复消息")
	}
	if secondServerMsgID != firstServerMsgID {
		t.Fatalf("重复消息应返回相同 serverMsgId，第一次 %d，第二次 %d", firstServerMsgID, secondServerMsgID)
	}
	if got := batcher.GetQueueSize(); got != 1 {
		t.Fatalf("重复消息不应再次入队，队列长度=%d", got)
	}
}

func TestMessageBatcherDuplicateMessageRegistersPersistedCallback(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{
		BatchSize:     10,
		FlushInterval: time.Hour,
	})

	msg := &proto.UserMessage{
		ClientMsgId: "client-1",
		FromUserId:  1001,
		ToUserId:    2001,
		MsgType:     1,
		Content:     []byte("hello"),
	}

	firstPersisted := 0
	secondPersisted := 0
	serverMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, func(serverMsgId int64) {
		if serverMsgId == 0 {
			t.Fatal("persisted 回调必须携带 serverMsgId")
		}
		firstPersisted++
	})
	if err != nil {
		t.Fatalf("第一次入队失败: %v", err)
	}
	if duplicate {
		t.Fatal("第一次入队不应标记为重复消息")
	}

	duplicateServerMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, func(serverMsgId int64) {
		if serverMsgId != serverMsgID {
			t.Fatalf("重复消息 persisted 回调应携带相同 serverMsgId，期望 %d，实际 %d", serverMsgID, serverMsgId)
		}
		secondPersisted++
	})
	if err != nil {
		t.Fatalf("重复消息不应返回错误: %v", err)
	}
	if !duplicate {
		t.Fatal("第二次相同消息应标记为重复")
	}
	if duplicateServerMsgID != serverMsgID {
		t.Fatalf("重复消息应复用 serverMsgId，第一次 %d，第二次 %d", serverMsgID, duplicateServerMsgID)
	}

	batcher.notifyPersisted(messageIdempotencyKey(msg), serverMsgID)
	if firstPersisted != 1 || secondPersisted != 1 {
		t.Fatalf("persisted 通知应触发所有等待回调，first=%d second=%d", firstPersisted, secondPersisted)
	}
}

func TestMessageBatcherDuplicateAfterPersistedTriggersCallback(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{
		BatchSize:     10,
		FlushInterval: time.Hour,
	})

	msg := &proto.UserMessage{
		ClientMsgId: "client-1",
		FromUserId:  1001,
		ToUserId:    2001,
		MsgType:     1,
		Content:     []byte("hello"),
	}

	serverMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, nil)
	if err != nil {
		t.Fatalf("第一次入队失败: %v", err)
	}
	if duplicate {
		t.Fatal("第一次入队不应标记为重复消息")
	}
	batcher.notifyPersisted(messageIdempotencyKey(msg), serverMsgID)

	persisted := make(chan int64, 1)
	duplicateServerMsgID, duplicate, err := batcher.SaveMessageWithCallback(msg, func(serverMsgId int64) {
		persisted <- serverMsgId
	})
	if err != nil {
		t.Fatalf("已 persisted 的重复消息不应返回错误: %v", err)
	}
	if !duplicate {
		t.Fatal("已 persisted 的相同消息应标记为重复")
	}
	if duplicateServerMsgID != serverMsgID {
		t.Fatalf("重复消息应复用 serverMsgId，第一次 %d，第二次 %d", serverMsgID, duplicateServerMsgID)
	}

	select {
	case got := <-persisted:
		if got != serverMsgID {
			t.Fatalf("persisted 补发回调 serverMsgId=%d，期望 %d", got, serverMsgID)
		}
	case <-time.After(time.Second):
		t.Fatal("已 persisted 的重复消息应异步补发 persisted 回调")
	}
}

func TestMessageBatcherSaveMessageReturnsQueueFullAndReleasesIdempotency(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{
		BatchSize:     1,
		FlushInterval: time.Hour,
	})

	for i := 0; i < cap(batcher.msgChan); i++ {
		_, duplicate, err := batcher.SaveMessageWithCallback(&proto.UserMessage{
			ClientMsgId: "client-fill-" + snowflake.Int64ToString(int64(i)),
			FromUserId:  1001,
			ToUserId:    2001,
			MsgType:     1,
			Content:     []byte("fill"),
		}, nil)
		if err != nil {
			t.Fatalf("填充队列第 %d 条失败: %v", i, err)
		}
		if duplicate {
			t.Fatalf("填充队列第 %d 条不应重复", i)
		}
	}

	fullMsg := &proto.UserMessage{
		ClientMsgId: "client-full",
		FromUserId:  1001,
		ToUserId:    2001,
		MsgType:     1,
		Content:     []byte("full"),
	}
	_, duplicate, err := batcher.SaveMessageWithCallback(fullMsg, nil)
	if !errors.Is(err, ErrMessageBatchQueueFull) {
		t.Fatalf("队列满应返回 ErrMessageBatchQueueFull，实际: %v", err)
	}
	if duplicate {
		t.Fatal("队列满失败不应标记为重复消息")
	}

	key := messageIdempotencyKey(fullMsg)
	if _, ok := batcher.getIdempotentMsgID(key); ok {
		t.Fatal("队列满失败后必须释放幂等占位，允许客户端后续重试")
	}

	stats := batcher.Stats()
	if stats.QueueFullCount != 1 {
		t.Fatalf("队列满计数 = %d，期望 1", stats.QueueFullCount)
	}
	if stats.QueueSize != cap(batcher.msgChan) || stats.QueueCapacity != cap(batcher.msgChan) {
		t.Fatalf("队列快照异常，size=%d capacity=%d", stats.QueueSize, stats.QueueCapacity)
	}
}

func TestMessageBatcherDeleteIdempotentMsgIDOnlyDeletesMatchingServerMsgID(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{})
	const key = "1001:client-1"
	batcher.setIdempotentMsgID(key, 10, nil)

	batcher.deleteIdempotentMsgID(key, 20)
	if got, ok := batcher.getIdempotentMsgID(key); !ok || got != 10 {
		t.Fatalf("serverMsgId 不匹配时不应删除幂等记录，got=%d ok=%v", got, ok)
	}

	batcher.deleteIdempotentMsgID(key, 10)
	if _, ok := batcher.getIdempotentMsgID(key); ok {
		t.Fatal("serverMsgId 匹配时应删除幂等记录")
	}
}

func TestMessageBatcherFailedFlushCountDefaultsToZero(t *testing.T) {
	batcher := newTestMessageBatcher(t, MessageBatcherConfig{})
	if got := batcher.GetFailedFlushCount(); got != 0 {
		t.Fatalf("新 batcher 的失败计数应为 0，实际 %d", got)
	}
	if stats := batcher.Stats(); stats.LastFlushDuration != 0 || stats.LastFlushBatchSize != 0 {
		t.Fatalf("新 batcher 不应有批写耗时，stats=%+v", stats)
	}
}

func newTestMessageBatcher(t *testing.T, config MessageBatcherConfig) *MessageBatcher {
	t.Helper()

	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("创建 snowflake 节点失败: %v", err)
	}
	return NewMessageBatcher(nil, node, config)
}
