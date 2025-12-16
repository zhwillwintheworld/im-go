package nats

import (
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"sudooom.im.logic/internal/config"
)

// Client NATS 客户端封装
type Client struct {
	conn   *nats.Conn
	logger *slog.Logger
}

// NewClient 创建 NATS 客户端
func NewClient(cfg config.NATSConfig) (*Client, error) {
	opts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.Warn("Disconnected from NATS", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("Reconnected to NATS", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			slog.Info("NATS connection closed")
		}),
		nats.Timeout(10 * time.Second),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		logger: slog.Default(),
	}, nil
}

// Conn 返回底层 NATS 连接
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// Close 关闭连接
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}
