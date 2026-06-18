package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

func TestRouteToMultipleEnqueuesLargeFanout(t *testing.T) {
	s := &RouterService{
		config: normalizeRouterConfig(RouterConfig{
			LargeGroupThreshold: 2,
			FanoutBufferSize:    1,
		}),
		fanoutQueue: make(chan fanoutTask, 1),
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	msg := &proto.UserMessage{
		FromUserId: 1001,
		ToGroupId:  9001,
		MsgType:    1,
		Content:    []byte("hello"),
	}

	if err := s.RouteToMultiple(context.Background(), []int64{2001, 2002}, msg, 3001); err != nil {
		t.Fatalf("RouteToMultiple failed: %v", err)
	}
	msg.Content[0] = 'x'

	select {
	case task := <-s.fanoutQueue:
		if len(task.userIds) != 2 {
			t.Fatalf("fanout user count = %d, 期望 2", len(task.userIds))
		}
		if got := string(task.payload.PushMessage.Content); got != "hello" {
			t.Fatalf("fanout payload content = %q, 期望 hello", got)
		}
	default:
		t.Fatal("大群 fan-out 应进入异步队列")
	}
}

func TestRouteToMultipleReturnsQueueFullWhenFanoutQueueFull(t *testing.T) {
	s := &RouterService{
		config: normalizeRouterConfig(RouterConfig{
			LargeGroupThreshold: 2,
			FanoutBufferSize:    1,
		}),
		fanoutQueue: make(chan fanoutTask, 1),
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s.fanoutQueue <- fanoutTask{}

	err := s.RouteToMultiple(context.Background(), []int64{2001, 2002}, &proto.UserMessage{}, 3001)
	if !errors.Is(err, ErrFanoutQueueFull) {
		t.Fatalf("队列满应返回 ErrFanoutQueueFull，实际 %v", err)
	}
	stats := s.Stats()
	if stats.FanoutDroppedCount != 1 {
		t.Fatalf("fan-out 丢弃计数 = %d，期望 1", stats.FanoutDroppedCount)
	}
	if stats.FanoutQueueSize != 1 || stats.FanoutQueueCapacity != 1 {
		t.Fatalf("fan-out 队列快照异常，stats=%+v", stats)
	}
}

func TestDispatchUserLocationsHonorsConcurrencyLimit(t *testing.T) {
	var current int64
	var maxSeen int64
	var calls int64

	s := &RouterService{
		config: RouterConfig{DispatchConcurrency: 2},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s.dispatch = func(userId int64, locations []sharedModel.UserLocation, payload proto.DownstreamPayload) error {
		now := atomic.AddInt64(&current, 1)
		updateMaxInt64(&maxSeen, now)
		atomic.AddInt64(&calls, 1)
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		return nil
	}

	results := []userLocationResult{
		{userId: 1, locations: []sharedModel.UserLocation{{UserId: 1}}},
		{userId: 2, locations: []sharedModel.UserLocation{{UserId: 2}}},
		{userId: 3, locations: []sharedModel.UserLocation{{UserId: 3}}},
		{userId: 4, locations: []sharedModel.UserLocation{{UserId: 4}}},
		{userId: 5, locations: []sharedModel.UserLocation{{UserId: 5}}},
	}

	s.dispatchUserLocations(results, proto.DownstreamPayload{})

	if got := atomic.LoadInt64(&calls); got != int64(len(results)) {
		t.Fatalf("dispatch calls = %d, 期望 %d", got, len(results))
	}
	if got := atomic.LoadInt64(&maxSeen); got > 2 {
		t.Fatalf("最大并发 = %d，超过限制 2", got)
	}
}

func updateMaxInt64(target *int64, value int64) {
	for {
		current := atomic.LoadInt64(target)
		if value <= current {
			return
		}
		if atomic.CompareAndSwapInt64(target, current, value) {
			return
		}
	}
}
