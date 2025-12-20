package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// tokenUserPrefix 用户Token前缀: user:token:{user_id}:{platform} -> accessToken
	tokenUserPrefix = "user:token:"
	// tokenInfoPrefix Token信息前缀: token:info:{accessToken} -> userInfo JSON
	tokenInfoPrefix = "token:info:"
)

// UserTokenInfo 存储在Redis中的用户信息
type UserTokenInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

// TokenRepository Token 数据访问层
type TokenRepository struct {
	rdb *redis.Client
}

// NewTokenRepository 创建 Token Repository
func NewTokenRepository(rdb *redis.Client) *TokenRepository {
	return &TokenRepository{rdb: rdb}
}

// buildUserTokenKey 构建用户Token的Key: user:token:{user_id}:{platform}
func buildUserTokenKey(userID int64, platform string) string {
	return fmt.Sprintf("%s%d:%s", tokenUserPrefix, userID, platform)
}

// buildTokenInfoKey 构建Token信息的Key: token:info:{accessToken}
func buildTokenInfoKey(accessToken string) string {
	return tokenInfoPrefix + accessToken
}

// SaveToken 保存Token到Redis
// 1. user:token:{user_id}:{platform} -> accessToken
// 2. token:info:{accessToken} -> userInfo JSON
func (r *TokenRepository) SaveToken(ctx context.Context, userInfo *UserTokenInfo, accessToken string, expiration time.Duration) error {
	// 构建 Keys
	userTokenKey := buildUserTokenKey(userInfo.UserID, userInfo.Platform)
	tokenInfoKey := buildTokenInfoKey(accessToken)

	// 序列化用户信息
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	// 使用 Pipeline 批量执行
	pipe := r.rdb.Pipeline()

	// 存储 user:token:{user_id}:{platform} -> accessToken
	pipe.Set(ctx, userTokenKey, accessToken, expiration)

	// 存储 token:info:{accessToken} -> userInfo JSON
	pipe.Set(ctx, tokenInfoKey, userInfoJSON, expiration)

	// 执行 Pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// GetTokenByUserPlatform 根据用户ID和平台获取Token
func (r *TokenRepository) GetTokenByUserPlatform(ctx context.Context, userID int64, platform string) (string, error) {
	key := buildUserTokenKey(userID, platform)
	token, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetUserInfoByToken 根据Token获取用户信息
func (r *TokenRepository) GetUserInfoByToken(ctx context.Context, accessToken string) (*UserTokenInfo, error) {
	key := buildTokenInfoKey(accessToken)
	data, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
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

// DeleteToken 删除Token（登出时使用）
func (r *TokenRepository) DeleteToken(ctx context.Context, userID int64, platform, accessToken string) error {
	tokenInfoKey := buildTokenInfoKey(accessToken)

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, tokenInfoKey)
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteOldToken 删除旧Token（重新登录时清理旧Token）
func (r *TokenRepository) DeleteOldToken(ctx context.Context, userID int64, platform string) error {
	// 先获取旧Token
	userTokenKey := buildUserTokenKey(userID, platform)
	oldToken, err := r.rdb.Get(ctx, userTokenKey).Result()
	if err == redis.Nil {
		// 没有旧Token，无需删除
		return nil
	}
	if err != nil {
		return err
	}

	// 删除旧Token的用户信息
	oldTokenInfoKey := buildTokenInfoKey(oldToken)
	return r.rdb.Del(ctx, oldTokenInfoKey).Err()
}

// GetTokenTTL 获取Token的剩余过期时间
func (r *TokenRepository) GetTokenTTL(ctx context.Context, accessToken string) (time.Duration, error) {
	key := buildTokenInfoKey(accessToken)
	ttl, err := r.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return ttl, nil
}

// RefreshTokenExpire 刷新Token的过期时间
func (r *TokenRepository) RefreshTokenExpire(ctx context.Context, userInfo *UserTokenInfo, accessToken string, expiration time.Duration) error {
	userTokenKey := buildUserTokenKey(userInfo.UserID, userInfo.Platform)
	tokenInfoKey := buildTokenInfoKey(accessToken)

	pipe := r.rdb.Pipeline()
	pipe.Expire(ctx, userTokenKey, expiration)
	pipe.Expire(ctx, tokenInfoKey, expiration)
	_, err := pipe.Exec(ctx)
	return err
}
