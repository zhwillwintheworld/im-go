package service

import (
	"context"
	"testing"
	"time"
)

func TestGroupServiceGetGroupMembersUsesCacheCopy(t *testing.T) {
	svc := NewGroupServiceWithCacheTTL(nil, time.Minute)
	groupId := int64(9001)
	svc.memberCache.Store(groupId, &cachedGroupMembers{
		Members:   []int64{1001, 1002},
		ExpiresAt: time.Now().Add(time.Minute),
	})

	first, err := svc.GetGroupMembers(context.Background(), groupId)
	if err != nil {
		t.Fatalf("GetGroupMembers cache hit failed: %v", err)
	}
	first[0] = 9999

	second, err := svc.GetGroupMembers(context.Background(), groupId)
	if err != nil {
		t.Fatalf("GetGroupMembers second cache hit failed: %v", err)
	}
	if second[0] != 1001 {
		t.Fatalf("群成员缓存应返回副本，实际第一个成员 %d", second[0])
	}
}

func TestGroupServiceInvalidateGroupMemberCache(t *testing.T) {
	svc := NewGroupServiceWithCacheTTL(nil, time.Minute)
	groupId := int64(9001)
	svc.memberCache.Store(groupId, &cachedGroupMembers{
		Members:   []int64{1001},
		ExpiresAt: time.Now().Add(time.Minute),
	})

	svc.InvalidateGroupMemberCache(groupId)

	if _, err := svc.GetGroupMembers(context.Background(), groupId); err == nil {
		t.Fatal("缓存失效后无数据库连接应返回错误")
	}
}
