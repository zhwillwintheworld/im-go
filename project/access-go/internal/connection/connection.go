package connection

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/webtransport-go"
)

var connIDCounter int64

// Connection 表示一个客户端连接
type Connection struct {
	id         int64
	userID     int64
	deviceID   string
	platform   string
	session    *webtransport.Session
	sessInfo   *SessionInfo
	logger     *slog.Logger
	writeChan  chan []byte
	closeChan  chan struct{}
	closeOnce  sync.Once
	createTime time.Time
}

// SessionInfo 表示会话状态
type SessionInfo struct {
	UserID         int64
	DeviceID       string
	Platform       string
	LoginTime      time.Time
	LastActiveTime time.Time
}

func NewFromWebTransport(session *webtransport.Session, logger *slog.Logger) *Connection {
	id := atomic.AddInt64(&connIDCounter, 1)
	c := &Connection{
		id:         id,
		session:    session,
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

func (c *Connection) DeviceID() string {
	return c.deviceID
}

func (c *Connection) Platform() string {
	return c.platform
}

func (c *Connection) BindSession(sessInfo *SessionInfo) {
	c.sessInfo = sessInfo
	c.userID = sessInfo.UserID
	c.deviceID = sessInfo.DeviceID
	c.platform = sessInfo.Platform
	sessInfo.LoginTime = time.Now()
	sessInfo.LastActiveTime = time.Now()
}

func (c *Connection) SessionInfo() *SessionInfo {
	return c.sessInfo
}

func (c *Connection) WebTransportSession() *webtransport.Session {
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
			stream, err := c.session.OpenStream()
			if err != nil {
				c.logger.Error("Failed to open stream", "error", err)
				continue
			}
			if _, err := stream.Write(data); err != nil {
				c.logger.Error("Failed to write to stream", "error", err)
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
		c.session.CloseWithError(0, "connection closed")
	})
}

func (c *Connection) UpdateActive() {
	if c.sessInfo != nil {
		c.sessInfo.LastActiveTime = time.Now()
	}
}

func (c *Connection) CreateTime() time.Time {
	return c.createTime
}
