package workerpool

import (
	"io"
	"log/slog"
	"testing"
)

func TestSubmitReturnsFalseImmediatelyWhenQueueFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool := New(0, 1, logger)
	defer pool.Shutdown()

	if !pool.Submit(func() {}) {
		t.Fatal("第一次提交应进入空队列")
	}

	if pool.Submit(func() {}) {
		t.Fatal("队列已满时 Submit 应快速返回 false")
	}
}
