package room

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// UserInfoProvider 用户信息提供者接口
type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, userId int64) *model.User
}

// RedisUserInfoProvider 基于 Redis 的用户信息提供者
type RedisUserInfoProvider struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

// NewRedisUserInfoProvider 创建 Redis 用户信息提供者
func NewRedisUserInfoProvider(redisClient *redis.Client) *RedisUserInfoProvider {
	return &RedisUserInfoProvider{
		redisClient: redisClient,
		logger:      slog.Default(),
	}
}

// GetUserInfo 从 Redis 获取用户信息
func (p *RedisUserInfoProvider) GetUserInfo(ctx context.Context, userId int64) *model.User {
	userInfoKey := sharedRedis.BuildUserInfoKey(userId)
	data, err := p.redisClient.Get(ctx, userInfoKey).Result()
	if err != nil {
		p.logger.Warn("Failed to get user info from Redis, using default", "userId", userId, "error", err)
		return p.getDefaultUserInfo(userId)
	}

	var user model.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		p.logger.Warn("Failed to unmarshal user info", "userId", userId, "error", err)
		return p.getDefaultUserInfo(userId)
	}

	return &user
}

// getDefaultUserInfo 获取默认用户信息（降级策略）
func (p *RedisUserInfoProvider) getDefaultUserInfo(userId int64) *model.User {
	return &model.User{
		UserID:   userId,
		Username: fmt.Sprintf("user_%d", userId),
		Nickname: fmt.Sprintf("玩家%d", userId),
		Avatar:   "",
	}
}
