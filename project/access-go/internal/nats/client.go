package nats

import (
	"github.com/example/im-access/internal/config"
	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
}

func NewClient(cfg config.NATSConfig) (*Client, error) {
	opts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			// 断开连接处理
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			// 重连处理
		}),
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
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

func (c *Client) Close() {
	c.conn.Close()
}
