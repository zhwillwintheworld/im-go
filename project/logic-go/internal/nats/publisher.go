package nats

import (
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

// MessagePublisher 消息发布器
type MessagePublisher struct {
	nc     *nats.Conn
	logger *slog.Logger
}

// NewMessagePublisher 创建消息发布器
func NewMessagePublisher(nc *nats.Conn) *MessagePublisher {
	return &MessagePublisher{
		nc:     nc,
		logger: slog.Default(),
	}
}

// PublishToAccess 推送消息到指定 Access 节点
func (p *MessagePublisher) PublishToAccess(accessNodeId string, message *proto.DownstreamMessage) error {
	subject := sharedNats.BuildAccessDownstreamSubject(accessNodeId)
	data, err := json.Marshal(message)
	if err != nil {
		p.logger.Error("Failed to marshal message", "error", err)
		return err
	}

	if err := p.nc.Publish(subject, data); err != nil {
		p.logger.Error("Failed to publish to access", "accessNodeId", accessNodeId, "error", err)
		return err
	}

	p.logger.Debug("Published message to access node", "accessNodeId", accessNodeId, "subject", subject)
	return nil
}

// Broadcast 广播消息到所有 Access 节点
func (p *MessagePublisher) Broadcast(message *proto.DownstreamMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		p.logger.Error("Failed to marshal broadcast message", "error", err)
		return err
	}

	if err := p.nc.Publish(sharedNats.SubjectAccessBroadcast, data); err != nil {
		p.logger.Error("Failed to broadcast message", "error", err)
		return err
	}

	p.logger.Debug("Broadcasted message to all access nodes")
	return nil
}
