package redis

import (
	"fmt"
	"strings"
	"time"
)

const (
	// UserLocationKeyPrefix 用户位置 Redis Key 前缀
	UserLocationKeyPrefix = "im:user:location:"

	// LocationTTL 用户位置 TTL
	LocationTTL = 24 * time.Hour
)

// BuildUserLocationKeyWithPlatform 构建用户位置 Key（按平台）
// Key: im:user:location:{userId}:{platform}
func BuildUserLocationKeyWithPlatform(userId int64, platform string) string {
	return fmt.Sprintf("%s%d:%s", UserLocationKeyPrefix, userId, strings.ToLower(platform))
}

// ============== 会话相关 Key ==============

const (
	// ConversationKeyPrefix 会话 Key 前缀
	ConversationKeyPrefix = "conv:"
)

// BuildConversationIndexKey 构建会话索引 Key (ZSet)
// Key: conv:{userId}:idx
func BuildConversationIndexKey(userId int64) string {
	return fmt.Sprintf("%s%d:idx", ConversationKeyPrefix, userId)
}

// BuildConversationPeerKey 构建私聊会话详情 Key (Hash)
// Key: conv:{userId}:p:{peerId}
func BuildConversationPeerKey(userId, peerId int64) string {
	return fmt.Sprintf("%s%d:p:%d", ConversationKeyPrefix, userId, peerId)
}

// BuildConversationGroupKey 构建群聊会话详情 Key (Hash)
// Key: conv:{userId}:g:{groupId}
func BuildConversationGroupKey(userId, groupId int64) string {
	return fmt.Sprintf("%s%d:g:%d", ConversationKeyPrefix, userId, groupId)
}

// BuildConversationMember 构建会话索引 Member
// Private: "p:{peerId}", Group: "g:{groupId}"
func BuildConversationPeerMember(peerId int64) string {
	return fmt.Sprintf("p:%d", peerId)
}

func BuildConversationGroupMember(groupId int64) string {
	return fmt.Sprintf("g:%d", groupId)
}

// ============== 用户信息相关 Key ==============

// BuildUserInfoKey 构建用户基本信息 Key（与平台无关，永久存储）
// Key: user:info:{userId}
// Value: JSON{"user_id", "username", "nickname", "avatar"}
func BuildUserInfoKey(userId int64) string {
	return fmt.Sprintf("user:info:%d", userId)
}

// ============== Token 相关 Key ==============

// BuildUserTokenKey 构建用户 Token key
// Key: user:token:{userId}:{platform}
// Value: accessToken
func BuildUserTokenKey(userId int64, platform string) string {
	return fmt.Sprintf("user:token:%d:%s", userId, strings.ToLower(platform))
}

// BuildTokenInfoKey 构建 Token 信息 key
// Key: token:info:{accessToken}
// Value: JSON{user_id, device_id, platform}
func BuildTokenInfoKey(accessToken string) string {
	return fmt.Sprintf("token:info:%s", accessToken)
}
