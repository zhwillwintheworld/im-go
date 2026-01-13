package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	sharedRedis "sudooom.im.shared/redis"
)

// UserTokenInfo 存储在Redis中的 Token 信息（仅认证字段）
type UserTokenInfo struct {
	UserID   int64  `json:"userId"`
	DeviceID string `json:"deviceId"`
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

// SaveToken 保存Token到Redis
// 1. user:token:{user_id}:{platform} -> accessToken
// 2. token:info:{accessToken} -> userInfo JSON
func (r *TokenRepository) SaveToken(ctx context.Context, userInfo *UserTokenInfo, accessToken string, expiration time.Duration) error {
	// 构建 Keys
	userTokenKey := sharedRedis.BuildUserTokenKey(userInfo.UserID, userInfo.Platform)
	tokenInfoKey := sharedRedis.BuildTokenInfoKey(accessToken)

	// 序列化用户信息
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	// 使用 Pipeline 批量执行
	pipe := r.rdb.Pipeline()

	// 存储 user:token:{user_id}:{platform} -> accessToken
	pipe.Set(ctx, userTokenKey, accessToken, expiration)

	// 存储 token:info:{accessToken} -> userInfo JSON
	pipe.Set(ctx, tokenInfoKey, userInfoJSON, expiration)

	// 执行 Pipeline
	_, err = pipe.Exec(ctx)
	return err
}

// GetTokenByUserPlatform 根据用户ID和平台获取Token
func (r *TokenRepository) GetTokenByUserPlatform(ctx context.Context, userID int64, platform string) (string, error) {
	key := sharedRedis.BuildUserTokenKey(userID, platform)
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
	key := sharedRedis.BuildTokenInfoKey(accessToken)
	data, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var userInfo UserTokenInfo
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// DeleteToken 删除Token（登出时使用）
func (r *TokenRepository) DeleteToken(ctx context.Context, userID int64, platform, accessToken string) error {
	tokenInfoKey := sharedRedis.BuildTokenInfoKey(accessToken)

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, tokenInfoKey)
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteOldToken 删除旧Token（重新登录时清理旧Token）
func (r *TokenRepository) DeleteOldToken(ctx context.Context, userID int64, platform string) error {
	// 先获取旧Token
	userTokenKey := sharedRedis.BuildUserTokenKey(userID, platform)
	oldToken, err := r.rdb.Get(ctx, userTokenKey).Result()
	if err == redis.Nil {
		// 没有旧Token，无需删除
		return nil
	}
	if err != nil {
		return err
	}

	// 删除旧Token的用户信息
	oldTokenInfoKey := sharedRedis.BuildTokenInfoKey(oldToken)
	return r.rdb.Del(ctx, oldTokenInfoKey).Err()
}

// GetTokenTTL 获取Token的剩余过期时间
func (r *TokenRepository) GetTokenTTL(ctx context.Context, accessToken string) (time.Duration, error) {
	key := sharedRedis.BuildTokenInfoKey(accessToken)
	ttl, err := r.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return ttl, nil
}

// RefreshTokenExpire 刷新Token的过期时间
func (r *TokenRepository) RefreshTokenExpire(ctx context.Context, userInfo *UserTokenInfo, accessToken string, expiration time.Duration) error {
	userTokenKey := sharedRedis.BuildUserTokenKey(userInfo.UserID, userInfo.Platform)
	tokenInfoKey := sharedRedis.BuildTokenInfoKey(accessToken)

	pipe := r.rdb.Pipeline()
	pipe.Expire(ctx, userTokenKey, expiration)
	pipe.Expire(ctx, tokenInfoKey, expiration)
	_, err := pipe.Exec(ctx)
	return err
}

// SaveUserInfo 保存用户基本信息到 Redis（永久存储，与平台无关）
func (r *TokenRepository) SaveUserInfo(ctx context.Context, userID int64, username, nickname, avatar string) error {
	userInfoKey := sharedRedis.BuildUserInfoKey(userID)
	userInfo := map[string]interface{}{
		"userId":   userID,
		"username": username,
		"nickname": nickname,
		"avatar":   avatar,
	}
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}
	// 不设置过期时间，永久保存
	return r.rdb.Set(ctx, userInfoKey, string(userInfoJSON), 0).Err()
}
