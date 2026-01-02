package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.access/internal/config"
	sharedRedis "sudooom.im.shared/redis"
)

const (
	// 用户位置 TTL: 2 分钟，心跳续期
	locationTTL = 2 * time.Minute
)

// UserTokenInfo 存储在 Redis 中的用户 Token 信息（仅认证字段）
type UserTokenInfo struct {
	UserID   int64  `json:"user_id"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

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
// Key: im:user:location:{userId}:{platform}, Value: JSON{accessNodeId, connId}
// 一个 platform 只维持一个连接，新连接会覆盖旧连接
func (c *Client) RegisterUserLocation(ctx context.Context, userId int64, platform string, connId int64) error {
	key := sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)

	// 存储完整的路由信息
	location := UserLocation{
		UserId:       userId,
		AccessNodeId: c.nodeID,
		ConnId:       connId,
		Platform:     platform,
		LoginTime:    time.Now(),
	}

	data, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	err = c.client.Set(ctx, key, data, locationTTL).Err()

	if err == nil {
		c.logger.Debug("Registered user location",
			"userId", userId,
			"platform", platform,
			"connId", connId,
			"nodeId", c.nodeID)
	}

	return err
}

// UnregisterUserLocation 移除用户位置
func (c *Client) UnregisterUserLocation(ctx context.Context, userId int64, platform string) error {
	key := sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)

	err := c.client.Del(ctx, key).Err()

	if err == nil {
		// User location unregistered
	}

	return err
}

// RefreshUserLocation 刷新用户位置 TTL（心跳时调用）
func (c *Client) RefreshUserLocation(ctx context.Context, userId int64, platform string) error {
	key := sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)
	return c.client.Expire(ctx, key, locationTTL).Err()
}

// GetUserLocation 获取用户某平台的位置（返回完整路由信息）
func (c *Client) GetUserLocation(ctx context.Context, userId int64, platform string) (*UserLocation, error) {
	key := sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)
	data, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // 用户不在线
	}
	if err != nil {
		return nil, err
	}

	// 解析 JSON
	var location UserLocation
	if err := json.Unmarshal([]byte(data), &location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %w", err)
	}

	return &location, nil
}

// GetUserInfoByToken 从 Redis 获取 token 对应的用户信息
func (c *Client) GetUserInfoByToken(ctx context.Context, token string) (*UserTokenInfo, error) {
	key := sharedRedis.BuildTokenInfoKey(token)
	data, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // token 不存在
	}
	if err != nil {
		return nil, err
	}

	var userInfo UserTokenInfo
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}

// GetCurrentToken 获取用户在该 platform 的当前有效 token
func (c *Client) GetCurrentToken(ctx context.Context, userId int64, platform string) (string, error) {
	key := sharedRedis.BuildUserTokenKey(userId, platform)
	token, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return token, err
}

// IsTokenCurrent 检查传入的 token 是否是该用户该 platform 当前有效的 token
func (c *Client) IsTokenCurrent(ctx context.Context, userId int64, platform, token string) (bool, error) {
	currentToken, err := c.GetCurrentToken(ctx, userId, platform)
	if err != nil {
		return false, err
	}
	return currentToken == token, nil
}

// Ping 检查 Redis 连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}
