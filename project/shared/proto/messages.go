package proto

// ============== 上行消息 (Access -> Logic) ==============

// UpstreamMessage 上行消息封装
type UpstreamMessage struct {
	AccessNodeId     string            `json:"AccessNodeId"`
	Platform         string            `json:"Platform,omitempty"` // 发送消息的平台
	UserMessage      *UserMessage      `json:"UserMessage,omitempty"`
	UserOnline       *UserOnline       `json:"UserOnline,omitempty"`
	UserOffline      *UserOffline      `json:"UserOffline,omitempty"`
	ConversationRead *ConversationRead `json:"ConversationRead,omitempty"` // 会话已读
}

// UserMessage 用户消息
type UserMessage struct {
	ClientMsgId string `json:"ClientMsgId"`
	FromUserId  int64  `json:"FromUserId,string"`
	ToUserId    int64  `json:"ToUserId,string"`
	ToGroupId   int64  `json:"ToGroupId,string"`
	MsgType     int32  `json:"MsgType"`
	Content     []byte `json:"Content"`
	Timestamp   int64  `json:"Timestamp"`
}

// UserOnline 用户上线事件
type UserOnline struct {
	UserId   int64  `json:"UserId,string"`
	ConnId   int64  `json:"ConnId,string"`
	DeviceId string `json:"DeviceId"`
	Platform string `json:"Platform"`
}

// UserOffline 用户下线事件
type UserOffline struct {
	UserId int64 `json:"UserId,string"`
	ConnId int64 `json:"ConnId,string"`
}

// ConversationRead 会话已读请求
type ConversationRead struct {
	UserId        int64 `json:"UserId,string"`            // 发起已读的用户ID
	PeerID        int64 `json:"PeerID,string,omitempty"`  // 私聊对方ID
	GroupID       int64 `json:"GroupID,string,omitempty"` // 群聊ID
	LastReadMsgID int64 `json:"LastReadMsgID,string"`     // 最后已读消息ID
}

// ============== 下行消息 (Logic -> Access) ==============

// DownstreamMessage 下行消息封装
type DownstreamMessage struct {
	Payload DownstreamPayload `json:"Payload"`
}

// DownstreamPayload 下行消息载荷
type DownstreamPayload struct {
	PushMessage *PushMessage `json:"PushMessage,omitempty"`
	MessageAck  *MessageAck  `json:"MessageAck,omitempty"`
}

// PushMessage 推送消息
type PushMessage struct {
	ServerMsgId int64  `json:"ServerMsgId,string"`
	FromUserId  int64  `json:"FromUserId,string"`
	ToUserId    int64  `json:"ToUserId,string"`
	ToGroupId   int64  `json:"ToGroupId,string"`
	MsgType     int32  `json:"MsgType"`
	Content     []byte `json:"Content"`
	Timestamp   int64  `json:"Timestamp"`
}

// MessageAck 消息确认
type MessageAck struct {
	ClientMsgId string `json:"ClientMsgId"`
	ServerMsgId int64  `json:"ServerMsgId,string"`
	ToUserId    int64  `json:"ToUserId,string"` // 接收 ACK 的用户 ID
	Timestamp   int64  `json:"Timestamp"`
}
