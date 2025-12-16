package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.access/internal/config"
)

const (
	userLocationKeyPrefix = "im:user:location:"
	locationTTL           = 24 * time.Hour
)

// UserLocation 用户位置信息
type UserLocation struct {
	UserId       int64     `json:"userId"`
	AccessNodeId string    `json:"accessNodeId"`
	ConnId       int64     `json:"connId"`
	DeviceId     string    `json:"deviceId"`
	Platform     string    `json:"platform"`
	LoginTime    time.Time `json:"loginTime"`
}

// Client Redis 客户端
type Client struct {
	client *redis.Client
	nodeID string
	logger *slog.Logger
}

// NewClient 创建 Redis 客户端
func NewClient(cfg config.RedisConfig, nodeID string) *Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return &Client{
		client: client,
		nodeID: nodeID,
		logger: slog.Default(),
	}
}

// RegisterUserLocation 注册用户位置
func (c *Client) RegisterUserLocation(ctx context.Context, userId, connId int64, deviceId, platform string) error {
	location := &UserLocation{
		UserId:       userId,
		AccessNodeId: c.nodeID,
		ConnId:       connId,
		DeviceId:     deviceId,
		Platform:     platform,
		LoginTime:    time.Now(),
	}

	key := fmt.Sprintf("%s%d", userLocationKeyPrefix, userId)
	field := fmt.Sprintf("%s:%d", c.nodeID, connId)

	value, err := json.Marshal(location)
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	pipe.HSet(ctx, key, field, value)
	pipe.Expire(ctx, key, locationTTL)
	_, err = pipe.Exec(ctx)

	if err == nil {
		c.logger.Debug("Registered user location",
			"userId", userId,
			"connId", connId)
	}

	return err
}

// UnregisterUserLocation 移除用户位置
func (c *Client) UnregisterUserLocation(ctx context.Context, userId, connId int64) error {
	key := fmt.Sprintf("%s%d", userLocationKeyPrefix, userId)
	field := fmt.Sprintf("%s:%d", c.nodeID, connId)

	err := c.client.HDel(ctx, key, field).Err()

	if err == nil {
		c.logger.Debug("Unregistered user location",
			"userId", userId,
			"connId", connId)
	}

	return err
}

// RefreshUserLocation 刷新用户位置 TTL
func (c *Client) RefreshUserLocation(ctx context.Context, userId int64) error {
	key := fmt.Sprintf("%s%d", userLocationKeyPrefix, userId)
	return c.client.Expire(ctx, key, locationTTL).Err()
}

// Ping 检查 Redis 连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}
