package nats

import "testing"

func TestBuildLogicShardSubjects(t *testing.T) {
	if got := BuildLogicRoomShardSubject(3); got != "im.logic.room.3" {
		t.Fatalf("room shard subject = %q", got)
	}
	if got := BuildLogicGameShardSubject(4); got != "im.logic.game.4" {
		t.Fatalf("game shard subject = %q", got)
	}
	if got := BuildLogicRoomShardSubject(-1); got != "im.logic.room.0" {
		t.Fatalf("负数 shard 应归一到 0，实际 %q", got)
	}
}

func TestShardForRoomIDStable(t *testing.T) {
	first := ShardForRoomID("room-1", 1001, 16)
	for i := 0; i < 10; i++ {
		if got := ShardForRoomID("room-1", 2000+int64(i), 16); got != first {
			t.Fatalf("同一 roomId 必须稳定落到同一 shard，第一次 %d，本次 %d", first, got)
		}
	}
}

func TestShardForRoomIDUsesFallbackForCreateRequest(t *testing.T) {
	first := ShardForRoomID("", 1001, 16)
	second := ShardForRoomID("", 1001, 16)
	if first != second {
		t.Fatalf("创建请求 fallback userId 应稳定，第一次 %d，第二次 %d", first, second)
	}
}

func TestShardForRoomIDDefaultsToZeroWhenShardCountIsOne(t *testing.T) {
	if got := ShardForRoomID("room-1", 1001, 1); got != 0 {
		t.Fatalf("shard_count <= 1 时应返回 0，实际 %d", got)
	}
}
