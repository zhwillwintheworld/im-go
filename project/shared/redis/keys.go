package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sudooom.im.shared/model"
)

const (
	// UserLocationKeyPrefix 用户位置 Redis Key 前缀
	UserLocationKeyPrefix = "im:user:location:"

	// LocationTTL 用户位置 TTL
	LocationTTL = 24 * time.Hour
)

// BuildUserLocationKey 构建用户位置 Key（旧版，用于 Hash 模式）
func BuildUserLocationKey(userId int64) string {
	return fmt.Sprintf("%s%d", UserLocationKeyPrefix, userId)
}

// BuildUserLocationKeyWithPlatform 构建用户位置 Key（按平台）
// Key: im:user:location:{userId}:{platform}
func BuildUserLocationKeyWithPlatform(userId int64, platform string) string {
	return fmt.Sprintf("%s%d:%s", UserLocationKeyPrefix, userId, platform)
}

// UserLocationStore 用户位置存储接口
type UserLocationStore interface {
	Register(ctx context.Context, location *model.UserLocation) error
	Unregister(ctx context.Context, userId int64, accessNodeId string, connId int64) error
	Get(ctx context.Context, userId int64) ([]model.UserLocation, error)
	Refresh(ctx context.Context, userId int64) error
}

// ParseUserLocation 解析用户位置 JSON
func ParseUserLocation(data string) (*model.UserLocation, error) {
	var loc model.UserLocation
	if err := json.Unmarshal([]byte(data), &loc); err != nil {
		return nil, err
	}
	return &loc, nil
}
