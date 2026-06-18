package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	sharedModel "sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

const locationCacheTTL = 500 * time.Millisecond

// AllPlatforms 支持的所有平台列表
var AllPlatforms = []string{"android", "ios", "web", "desktop", "wechat"}

// cachedUserLocation 缓存的用户位置信息
type cachedUserLocation struct {
	Locations []sharedModel.UserLocation
	ExpiresAt time.Time
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
	locationsByUser, err := s.GetUsersLocations(ctx, []int64{userId})
	if err != nil {
		return nil, err
	}
	return locationsByUser[userId], nil
}

// GetUsersLocations 批量获取多个用户所有平台的位置，优先使用短 TTL 缓存，未命中用户使用 Redis MGET。
func (s *LocationService) GetUsersLocations(ctx context.Context, userIds []int64) (map[int64][]sharedModel.UserLocation, error) {
	locationsByUser := make(map[int64][]sharedModel.UserLocation, len(userIds))
	missingUserIds := make([]int64, 0, len(userIds))
	seen := make(map[int64]struct{}, len(userIds))
	now := time.Now()

	for _, userId := range userIds {
		if userId <= 0 {
			continue
		}
		if _, ok := seen[userId]; ok {
			continue
		}
		seen[userId] = struct{}{}

		if cached, ok := s.locationCache.Load(userId); ok {
			entry := cached.(*cachedUserLocation)
			if now.Before(entry.ExpiresAt) {
				locationsByUser[userId] = cloneLocations(entry.Locations)
				continue
			}
			s.locationCache.Delete(userId)
		}

		missingUserIds = append(missingUserIds, userId)
	}

	if len(missingUserIds) == 0 {
		return locationsByUser, nil
	}

	fetched, err := s.getUsersLocationsFromRedis(ctx, missingUserIds, AllPlatforms)
	if err != nil {
		return nil, err
	}

	for userId, locations := range fetched {
		if len(locations) == 0 {
			continue
		}
		cloned := cloneLocations(locations)
		locationsByUser[userId] = cloneLocations(cloned)
		s.locationCache.Store(userId, &cachedUserLocation{
			Locations: cloned,
			ExpiresAt: time.Now().Add(locationCacheTTL),
		})
	}

	return locationsByUser, nil
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
	locationsByUser, err := s.getUsersLocationsFromRedis(ctx, []int64{userId}, platforms)
	if err != nil {
		return nil, err
	}
	return locationsByUser[userId], nil
}

// getUsersLocationsFromRedis 从 Redis 批量获取多个用户的位置。
func (s *LocationService) getUsersLocationsFromRedis(ctx context.Context, userIds []int64, platforms []string) (map[int64][]sharedModel.UserLocation, error) {
	type lookupMeta struct {
		userId   int64
		platform string
	}

	locationsByUser := make(map[int64][]sharedModel.UserLocation, len(userIds))
	if len(userIds) == 0 || len(platforms) == 0 {
		return locationsByUser, nil
	}

	keys := make([]string, 0, len(userIds)*len(platforms))
	metas := make([]lookupMeta, 0, len(userIds)*len(platforms))
	for _, userId := range userIds {
		if userId <= 0 {
			continue
		}
		for _, platform := range platforms {
			keys = append(keys, sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform))
			metas = append(metas, lookupMeta{userId: userId, platform: platform})
		}
	}
	if len(keys) == 0 {
		return locationsByUser, nil
	}

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
				"userId", metas[i].userId,
				"platform", metas[i].platform,
				"error", err)
			continue
		}
		if loc.UserId == 0 {
			loc.UserId = metas[i].userId
		}
		if loc.Platform == "" {
			loc.Platform = metas[i].platform
		}
		locationsByUser[metas[i].userId] = append(locationsByUser[metas[i].userId], loc)
	}

	return locationsByUser, nil
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

func cloneLocations(locations []sharedModel.UserLocation) []sharedModel.UserLocation {
	if len(locations) == 0 {
		return nil
	}
	cloned := make([]sharedModel.UserLocation, len(locations))
	copy(cloned, locations)
	return cloned
}
