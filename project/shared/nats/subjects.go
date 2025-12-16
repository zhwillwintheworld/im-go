package nats

// NATS Subject 常量定义
const (
	// SubjectLogicUpstream Access -> Logic 上行消息
	SubjectLogicUpstream = "im.logic.upstream"

	// SubjectAccessDownstreamPrefix Logic -> Access 下行消息前缀
	// 完整格式: im.access.{node_id}.downstream
	SubjectAccessDownstreamPrefix = "im.access."
	SubjectAccessDownstreamSuffix = ".downstream"

	// SubjectAccessBroadcast Logic -> All Access 广播消息
	SubjectAccessBroadcast = "im.access.broadcast"

	// QueueGroupLogic Logic 服务队列组名称
	QueueGroupLogic = "logic-group"
)

// BuildAccessDownstreamSubject 构建 Access 节点下行 Subject
func BuildAccessDownstreamSubject(nodeID string) string {
	return SubjectAccessDownstreamPrefix + nodeID + SubjectAccessDownstreamSuffix
}
