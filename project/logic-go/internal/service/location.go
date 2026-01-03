package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
	sharedModel "sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// AllPlatforms 支持的所有平台列表
var AllPlatforms = []string{"android", "ios", "web", "desktop", "wechat"}

// cachedUserLocation 缓存的用户位置信息
type cachedUserLocation struct {
	Locations []sharedModel.UserLocation
}

// LocationService 用户位置管理服务
type LocationService struct {
	redisClient   *redis.Client
	logger        *slog.Logger
	locationCache sync.Map // map[int64]*cachedUserLocation
}

// NewLocationService 创建位置服务
func NewLocationService(redisClient *redis.Client) *LocationService {
	return &LocationService{
		redisClient: redisClient,
		logger:      slog.Default(),
	}

}

// GetUserLocations 获取用户所有平台的位置（带缓存）
func (s *LocationService) GetUserLocations(ctx context.Context, userId int64) ([]sharedModel.UserLocation, error) {
	// 1. 尝试从缓存读取
	if cached, ok := s.locationCache.Load(userId); ok {
		entry := cached.(*cachedUserLocation)
		// 缓存命中，直接返回
		return entry.Locations, nil
	}

	// 2. 缓存未命中，查询 Redis
	locations, err := s.getUserLocationsFromRedis(ctx, userId, AllPlatforms)
	if err != nil {
		return nil, err
	}

	// 3. 写入缓存（只要有位置信息就缓存）
	if len(locations) > 0 {
		s.locationCache.Store(userId, &cachedUserLocation{
			Locations: locations,
		})
	}

	return locations, nil
}

// GetUserLocationsByPlatforms 获取用户在指定平台的位置
func (s *LocationService) GetUserLocationsByPlatforms(ctx context.Context, userId int64, platforms []string) ([]sharedModel.UserLocation, error) {
	if len(platforms) == 0 {
		return nil, nil
	}
	return s.getUserLocationsFromRedis(ctx, userId, platforms)
}

// getUserLocationsFromRedis 从 Redis 获取用户位置（私有方法）
func (s *LocationService) getUserLocationsFromRedis(ctx context.Context, userId int64, platforms []string) ([]sharedModel.UserLocation, error) {
	locations := make([]sharedModel.UserLocation, 0, len(platforms))

	// 构建所有平台的 key
	keys := make([]string, len(platforms))
	for i, platform := range platforms {
		keys[i] = sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)
	}

	// 批量获取
	results, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, result := range results {
		if result == nil {
			continue
		}
		jsonStr, ok := result.(string)
		if !ok || jsonStr == "" {
			continue
		}

		// 解析 JSON 格式的 UserLocation
		var loc sharedModel.UserLocation
		if err := json.Unmarshal([]byte(jsonStr), &loc); err != nil {
			s.logger.Warn("Failed to unmarshal user location",
				"userId", userId,
				"platform", platforms[i],
				"error", err)
			continue
		}
		locations = append(locations, loc)
	}

	return locations, nil
}

// InvalidateCache 失效用户缓存
func (s *LocationService) InvalidateCache(userId int64) {
	s.locationCache.Delete(userId)
}

// InvalidateCacheBatch 批量失效用户缓存
func (s *LocationService) InvalidateCacheBatch(userIds []int64) {
	for _, userId := range userIds {
		s.locationCache.Delete(userId)
	}
}
