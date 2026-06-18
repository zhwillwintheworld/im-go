package game

import (
	"io"
	"log/slog"
	"testing"
)

func TestGameManagerTriggerSnapshotEnqueuesRoomID(t *testing.T) {
	m := &GameManager{
		snapshotChan: make(chan string, 1),
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	m.TriggerSnapshot("room-1")

	select {
	case got := <-m.snapshotChan:
		if got != "room-1" {
			t.Fatalf("snapshot roomId = %q, 期望 room-1", got)
		}
	default:
		t.Fatal("期望快照触发进入队列")
	}
}

func TestGameManagerTriggerSnapshotIgnoresEmptyRoomID(t *testing.T) {
	m := &GameManager{
		snapshotChan: make(chan string, 1),
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	m.TriggerSnapshot("")

	select {
	case got := <-m.snapshotChan:
		t.Fatalf("空 roomId 不应进入快照队列，实际收到 %q", got)
	default:
	}
}

func TestGameManagerTriggerSnapshotDropsWhenQueueFull(t *testing.T) {
	m := &GameManager{
		snapshotChan: make(chan string, 1),
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	m.snapshotChan <- "existing"
	m.TriggerSnapshot("room-2")

	if got := len(m.snapshotChan); got != 1 {
		t.Fatalf("快照队列满时不应继续入队，当前长度 %d", got)
	}
}
