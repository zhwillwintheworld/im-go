package nats

import (
	"hash/fnv"
	"strconv"
)

// NATS Subject 常量定义
const (
	// SubjectLogicUpstream Access -> Logic 上行消息
	SubjectLogicUpstream = "im.logic.upstream"

	// SubjectLogicRoomShardPrefix Access -> Logic 房间请求分片上行消息
	// 完整格式: im.logic.room.{shard}
	SubjectLogicRoomShardPrefix = "im.logic.room."

	// SubjectLogicGameShardPrefix Access -> Logic 游戏请求分片上行消息
	// 完整格式: im.logic.game.{shard}
	SubjectLogicGameShardPrefix = "im.logic.game."

	// SubjectAccessDownstreamPrefix Logic -> Access 下行消息前缀
	// 完整格式: im.access.{node_id}.downstream
	SubjectAccessDownstreamPrefix = "im.access."
	SubjectAccessDownstreamSuffix = ".downstream"

	// SubjectAccessBroadcast Logic -> All Access 广播消息
	SubjectAccessBroadcast = "im.access.broadcast"

	// QueueGroupLogic Logic 服务队列组名称
	QueueGroupLogic = "logic-group"
)

const DefaultRoomShardCount = 1

// BuildAccessDownstreamSubject 构建 Access 节点下行 Subject
func BuildAccessDownstreamSubject(nodeID string) string {
	return SubjectAccessDownstreamPrefix + nodeID + SubjectAccessDownstreamSuffix
}

// BuildLogicRoomShardSubject 构建房间请求 shard subject。
func BuildLogicRoomShardSubject(shard int) string {
	return SubjectLogicRoomShardPrefix + strconv.Itoa(normalizeShard(shard))
}

// BuildLogicGameShardSubject 构建游戏请求 shard subject。
func BuildLogicGameShardSubject(shard int) string {
	return SubjectLogicGameShardPrefix + strconv.Itoa(normalizeShard(shard))
}

// ShardForRoomID 根据 roomId 计算稳定 shard。roomId 为空时使用 fallbackUserID 分配创建请求。
func ShardForRoomID(roomID string, fallbackUserID int64, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	key := roomID
	if key == "" {
		key = strconv.FormatInt(fallbackUserID, 10)
	}
	return ShardForKey(key, shardCount)
}

// ShardForKey 根据任意字符串计算稳定 shard。
func ShardForKey(key string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(shardCount))
}

func normalizeShard(shard int) int {
	if shard < 0 {
		return 0
	}
	return shard
}
