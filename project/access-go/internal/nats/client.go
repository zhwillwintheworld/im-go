package nats

import (
	"log/slog"

	"github.com/nats-io/nats.go"
	"sudooom.im.access/internal/config"
)

type Client struct {
	conn   *nats.Conn
	logger *slog.Logger
}

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

func (c *Client) Publish(subject string, data []byte) error {
	return c.conn.Publish(subject, data)
}

func (c *Client) Subscribe(subject string, handler func(data []byte)) error {
	_, err := c.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

func (c *Client) QueueSubscribe(subject, queue string, handler func(data []byte)) error {
	_, err := c.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
