package service

import (
	"context"
	"log/slog"
	"time"

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

// RegisterUserLocation 注册用户位置
func (s *UserService) RegisterUserLocation(ctx context.Context, userId int64, accessNodeId string, connId int64, deviceId string, platform string) error {
	location := &sharedModel.UserLocation{
		UserId:       userId,
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		DeviceId:     deviceId,
		Platform:     platform,
		LoginTime:    time.Now(),
	}

	key := sharedRedis.BuildUserLocationKey(userId)
	field := sharedRedis.BuildUserLocationField(accessNodeId, connId)

	value, err := sharedRedis.SerializeUserLocation(location)
	if err != nil {
		return err
	}

	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, key, field, value)
	pipe.Expire(ctx, key, sharedRedis.LocationTTL)
	_, err = pipe.Exec(ctx)

	if err == nil {
		s.logger.Info("User location registered",
			"userId", userId,
			"accessNodeId", accessNodeId,
			"connId", connId,
			"platform", platform)
	}

	return err
}

// UnregisterUserLocation 移除用户位置
func (s *UserService) UnregisterUserLocation(ctx context.Context, userId int64, connId int64, accessNodeId string) error {
	key := sharedRedis.BuildUserLocationKey(userId)
	field := sharedRedis.BuildUserLocationField(accessNodeId, connId)

	err := s.redisClient.HDel(ctx, key, field).Err()

	if err == nil {
		s.logger.Info("User location unregistered",
			"userId", userId,
			"connId", connId)
	}

	return err
}

// GetUserLocations 获取用户所有位置
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
