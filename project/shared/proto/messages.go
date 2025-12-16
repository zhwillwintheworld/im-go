package proto

// ============== 上行消息 (Access -> Logic) ==============

// UpstreamMessage 上行消息封装
type UpstreamMessage struct {
	AccessNodeId string       `json:"AccessNodeId"`
	UserMessage  *UserMessage `json:"UserMessage,omitempty"`
	UserOnline   *UserOnline  `json:"UserOnline,omitempty"`
	UserOffline  *UserOffline `json:"UserOffline,omitempty"`
}

// UserMessage 用户消息
type UserMessage struct {
	ClientMsgId string `json:"ClientMsgId"`
	FromUserId  int64  `json:"FromUserId"`
	ToUserId    int64  `json:"ToUserId"`
	ToGroupId   int64  `json:"ToGroupId"`
	MsgType     int32  `json:"MsgType"`
	Content     []byte `json:"Content"`
	Timestamp   int64  `json:"Timestamp"`
}

// UserOnline 用户上线事件
type UserOnline struct {
	UserId   int64  `json:"UserId"`
	ConnId   int64  `json:"ConnId"`
	DeviceId string `json:"DeviceId"`
	Platform string `json:"Platform"`
}

// UserOffline 用户下线事件
type UserOffline struct {
	UserId int64 `json:"UserId"`
	ConnId int64 `json:"ConnId"`
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
	ServerMsgId int64  `json:"ServerMsgId"`
	FromUserId  int64  `json:"FromUserId"`
	ToUserId    int64  `json:"ToUserId"`
	ToGroupId   int64  `json:"ToGroupId"`
	MsgType     int32  `json:"MsgType"`
	Content     []byte `json:"Content"`
	Timestamp   int64  `json:"Timestamp"`
}

// MessageAck 消息确认
type MessageAck struct {
	ClientMsgId string `json:"ClientMsgId"`
	ServerMsgId int64  `json:"ServerMsgId"`
	ToUserId    int64  `json:"ToUserId"` // 接收 ACK 的用户 ID
	Timestamp   int64  `json:"Timestamp"`
}
