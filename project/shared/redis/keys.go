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

// BuildUserLocationKey 构建用户位置 Key
func BuildUserLocationKey(userId int64) string {
	return fmt.Sprintf("%s%d", UserLocationKeyPrefix, userId)
}

// BuildUserLocationField 构建用户位置 Field
func BuildUserLocationField(accessNodeId string, connId int64) string {
	return fmt.Sprintf("%s:%d", accessNodeId, connId)
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

// SerializeUserLocation 序列化用户位置为 JSON
func SerializeUserLocation(location *model.UserLocation) (string, error) {
	data, err := json.Marshal(location)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
