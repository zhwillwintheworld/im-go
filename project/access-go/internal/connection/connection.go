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

	// 流复用优化：使用客户端创建的双向流
	clientStream *webtransport.Stream // 客户端创建的双向流，用于发送消息
	streamMutex  sync.Mutex
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

// SetClientStream 设置客户端创建的双向流用于发送消息
func (c *Connection) SetClientStream(stream *webtransport.Stream) {
	c.streamMutex.Lock()
	defer c.streamMutex.Unlock()
	c.clientStream = stream
	// Client stream set
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
			c.streamMutex.Lock()

			// 使用客户端创建的双向流发送消息
			if c.clientStream == nil {
				c.logger.Error("Client stream not set, cannot send message", "conn_id", c.id)
				c.streamMutex.Unlock()
				continue
			}

			// 直接写入客户端的流，不关闭流（通过帧头区分消息）
			if _, err := c.clientStream.Write(data); err != nil {
				c.logger.Error("Failed to write to client stream", "error", err)
				// 出错时重置流
				c.clientStream = nil
			}

			c.streamMutex.Unlock()
		case <-c.closeChan:
			// 关闭时清理流（不需要close，已经由 HandleStream defer 处理）
			c.streamMutex.Lock()
			c.clientStream = nil
			c.streamMutex.Unlock()
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

// LastActiveTime 返回最后活跃时间
func (c *Connection) LastActiveTime() time.Time {
	if c.sessInfo != nil {
		return c.sessInfo.LastActiveTime
	}
	return c.createTime
}
