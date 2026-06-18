package handler

import "testing"

func TestLogicRoomShardSubjectStableForSameRoom(t *testing.T) {
	h := &Handler{roomShardCount: 16}

	first := h.logicRoomShardSubject("room-1", 1001)
	for i := 0; i < 10; i++ {
		if got := h.logicRoomShardSubject("room-1", 2000+int64(i)); got != first {
			t.Fatalf("同一 roomId 应落到同一 room subject，第一次 %q，本次 %q", first, got)
		}
	}
}

func TestLogicGameShardSubjectUsesSameShardAsRoom(t *testing.T) {
	h := &Handler{roomShardCount: 16}

	roomSubject := h.logicRoomShardSubject("room-1", 1001)
	gameSubject := h.logicGameShardSubject("room-1", 1001)
	roomShard := roomSubject[len("im.logic.room."):]
	gameShard := gameSubject[len("im.logic.game."):]
	if roomShard != gameShard {
		t.Fatalf("同一 roomId 的 room/game subject 应使用相同 shard，room=%q game=%q", roomSubject, gameSubject)
	}
}

func TestLogicRoomShardSubjectDefaultsToZeroForCreateRequest(t *testing.T) {
	h := &Handler{roomShardCount: 1}
	if got := h.logicRoomShardSubject("", 1001); got != "im.logic.room.0" {
		t.Fatalf("默认单 shard 创建请求应落到 im.logic.room.0，实际 %q", got)
	}
}
