package connection

import (
	"errors"
	"testing"
)

func TestSendReturnsBackpressureWhenWriteQueueIsFull(t *testing.T) {
	conn := &Connection{
		id:        1,
		writeChan: make(chan []byte, 1),
		closeChan: make(chan struct{}),
	}

	if err := conn.Send([]byte("first")); err != nil {
		t.Fatalf("第一次发送应进入写队列，实际错误: %v", err)
	}

	err := conn.Send([]byte("second"))
	if !errors.Is(err, ErrConnectionBackpressure) {
		t.Fatalf("写队列满应快速返回背压错误，实际: %v", err)
	}
	if got := conn.WriteQueueLength(); got != 1 {
		t.Fatalf("写队列长度 = %d，期望 1", got)
	}
	if got := conn.WriteQueueCapacity(); got != 1 {
		t.Fatalf("写队列容量 = %d，期望 1", got)
	}
}

func TestSendReturnsClosedWhenConnectionClosed(t *testing.T) {
	conn := &Connection{
		id:        1,
		writeChan: make(chan []byte, 1),
		closeChan: make(chan struct{}),
	}
	close(conn.closeChan)

	err := conn.Send([]byte("message"))
	if !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("连接关闭后发送应返回关闭错误，实际: %v", err)
	}
}
