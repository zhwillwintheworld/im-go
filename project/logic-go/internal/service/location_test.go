package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	sharedModel "sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

func TestLocationServiceGetUsersLocationsBatch(t *testing.T) {
	client := getTestRedisClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("Failed to close Redis client: %v", err)
		}
	}()

	ctx := context.Background()
	svc := NewLocationService(client)

	storeLocation(t, ctx, client, sharedModel.UserLocation{
		UserId:       1001,
		Platform:     "web",
		AccessNodeId: "access-a",
		ConnId:       11,
	})
	key := sharedRedis.BuildUserLocationKeyWithPlatform(1002, "desktop")
	if err := client.Set(ctx, key, `{"accessNodeId":"access-b","connId":22}`, time.Minute).Err(); err != nil {
		t.Fatalf("Set location failed: %v", err)
	}
	invalidKey := sharedRedis.BuildUserLocationKeyWithPlatform(1002, "ios")
	if err := client.Set(ctx, invalidKey, `{bad-json`, time.Minute).Err(); err != nil {
		t.Fatalf("Set invalid location failed: %v", err)
	}

	locationsByUser, err := svc.GetUsersLocations(ctx, []int64{1001, 1002, 1001, 0})
	if err != nil {
		t.Fatalf("GetUsersLocations failed: %v", err)
	}

	if got := len(locationsByUser[1001]); got != 1 {
		t.Fatalf("user 1001 locations = %d, 期望 1", got)
	}
	if got := len(locationsByUser[1002]); got != 1 {
		t.Fatalf("user 1002 locations = %d, 期望 1", got)
	}
	if locationsByUser[1002][0].UserId != 1002 {
		t.Fatalf("缺失 userId 时应按查询 key 补齐，实际 %d", locationsByUser[1002][0].UserId)
	}
	if locationsByUser[1002][0].Platform != "desktop" {
		t.Fatalf("缺失 platform 时应按查询 key 补齐，实际 %q", locationsByUser[1002][0].Platform)
	}
}

func TestLocationServiceGetUsersLocationsReturnsCacheCopy(t *testing.T) {
	client := getTestRedisClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("Failed to close Redis client: %v", err)
		}
	}()

	ctx := context.Background()
	svc := NewLocationService(client)
	storeLocation(t, ctx, client, sharedModel.UserLocation{
		UserId:       1001,
		Platform:     "web",
		AccessNodeId: "access-a",
		ConnId:       11,
	})

	first, err := svc.GetUsersLocations(ctx, []int64{1001})
	if err != nil {
		t.Fatalf("GetUsersLocations failed: %v", err)
	}
	first[1001][0].Platform = "mutated"

	second, err := svc.GetUsersLocations(ctx, []int64{1001})
	if err != nil {
		t.Fatalf("GetUsersLocations second call failed: %v", err)
	}
	if second[1001][0].Platform != "web" {
		t.Fatalf("缓存返回值应为副本，实际 platform=%q", second[1001][0].Platform)
	}
}

func storeLocation(t *testing.T, ctx context.Context, client *redis.Client, loc sharedModel.UserLocation) {
	t.Helper()

	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("Marshal location failed: %v", err)
	}
	key := sharedRedis.BuildUserLocationKeyWithPlatform(loc.UserId, loc.Platform)
	if err := client.Set(ctx, key, string(data), time.Minute).Err(); err != nil {
		t.Fatalf("Set location failed: %v", err)
	}
}
