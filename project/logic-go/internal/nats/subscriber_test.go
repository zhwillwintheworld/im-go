package nats

import (
	"reflect"
	"testing"
)

func TestNewMessageSubscriberDefaultsRoomShardConfig(t *testing.T) {
	sub := NewMessageSubscriber(nil, nil, SubscriberConfig{})

	if sub.config.RoomShardCount != 1 {
		t.Fatalf("默认 RoomShardCount 应为 1，实际 %d", sub.config.RoomShardCount)
	}
	if sub.config.RoomShardIndex != 0 {
		t.Fatalf("默认 RoomShardIndex 应为 0，实际 %d", sub.config.RoomShardIndex)
	}
	if sub.config.RoomWorkerCount <= 0 || sub.config.RoomBufferSize <= 0 {
		t.Fatalf("room/game worker 和 buffer 应有默认值，worker=%d buffer=%d", sub.config.RoomWorkerCount, sub.config.RoomBufferSize)
	}
}

func TestNewMessageSubscriberNormalizesInvalidRoomShardIndex(t *testing.T) {
	sub := NewMessageSubscriber(nil, nil, SubscriberConfig{
		RoomShardCount: 4,
		RoomShardIndex: 8,
	})

	if sub.config.RoomShardIndex != 0 {
		t.Fatalf("非法 RoomShardIndex 应归一到 0，实际 %d", sub.config.RoomShardIndex)
	}
}

func TestRoomGameShardSubjects(t *testing.T) {
	sub := NewMessageSubscriber(nil, nil, SubscriberConfig{
		RoomShardCount: 8,
		RoomShardIndex: 3,
	})

	expected := []string{"im.logic.room.3", "im.logic.game.3"}
	if got := sub.roomGameShardSubjects(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("room/game shard subjects = %#v，期望 %#v", got, expected)
	}
}
