package connection

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

var connIDCounter int64

// Connection 表示一个客户端连接
type Connection struct {
	id         int64
	userID     int64
	deviceID   string
	platform   string
	quicConn   quic.Connection
	session    *Session
	logger     *zap.Logger
	writeChan  chan []byte
	closeChan  chan struct{}
	closeOnce  sync.Once
	createTime time.Time
}

// Session 表示会话状态
type Session struct {
	UserID         int64
	DeviceID       string
	Platform       string
	LoginTime      time.Time
	LastActiveTime time.Time
}

func New(quicConn quic.Connection, logger *zap.Logger) *Connection {
	id := atomic.AddInt64(&connIDCounter, 1)
	c := &Connection{
		id:         id,
		quicConn:   quicConn,
		logger:     logger,
		writeChan:  make(chan []byte, 256),
		closeChan:  make(chan struct{}),
		createTime: time.Now(),
	}
	go c.writeLoop()
	return c
}

func (c *Connection) ID() int64 {
	return c.id
}

func (c *Connection) UserID() int64 {
	return c.userID
}

func (c *Connection) BindSession(session *Session) {
	c.session = session
	c.userID = session.UserID
	c.deviceID = session.DeviceID
	c.platform = session.Platform
}

func (c *Connection) Session() *Session {
	return c.session
}

func (c *Connection) Send(data []byte) error {
	select {
	case c.writeChan <- data:
		return nil
	case <-c.closeChan:
		return ErrConnectionClosed
	}
}

func (c *Connection) writeLoop() {
	for {
		select {
		case data := <-c.writeChan:
			stream, err := c.quicConn.OpenUniStream()
			if err != nil {
				c.logger.Error("Failed to open stream", zap.Error(err))
				continue
			}
			if _, err := stream.Write(data); err != nil {
				c.logger.Error("Failed to write to stream", zap.Error(err))
			}
			stream.Close()
		case <-c.closeChan:
			return
		}
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		c.quicConn.CloseWithError(0, "connection closed")
	})
}

func (c *Connection) UpdateActive() {
	if c.session != nil {
		c.session.LastActiveTime = time.Now()
	}
}
