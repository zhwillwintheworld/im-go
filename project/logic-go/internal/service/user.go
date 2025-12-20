package service

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
	sharedModel "sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// UserService 用户服务
type UserService struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

// NewUserService 创建用户服务
func NewUserService(redisClient *redis.Client) *UserService {
	return &UserService{
		redisClient: redisClient,
		logger:      slog.Default(),
	}
}

// GetUserLocations 获取用户所有位置（读取 access-go 写入的数据）
func (s *UserService) GetUserLocations(ctx context.Context, userId int64) ([]sharedModel.UserLocation, error) {
	key := sharedRedis.BuildUserLocationKey(userId)

	entries, err := s.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	locations := make([]sharedModel.UserLocation, 0, len(entries))
	for _, value := range entries {
		loc, err := sharedRedis.ParseUserLocation(value)
		if err != nil {
			continue
		}
		locations = append(locations, *loc)
	}

	return locations, nil
}

// IsUserOnline 检查用户是否在线
func (s *UserService) IsUserOnline(ctx context.Context, userId int64) (bool, error) {
	key := sharedRedis.BuildUserLocationKey(userId)
	count, err := s.redisClient.HLen(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
