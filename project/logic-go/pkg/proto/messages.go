package pb

// 这是 Protobuf 生成代码的占位实现
// 实际项目中应该使用 protoc 生成

// UpstreamMessage 上行消息（Access -> Logic）
type UpstreamMessage struct {
	AccessNodeId string
	UserMessage  *UserMessage
	UserOnline   *UserOnline
	UserOffline  *UserOffline
}

func (m *UpstreamMessage) GetAccessNodeId() string {
	if m != nil {
		return m.AccessNodeId
	}
	return ""
}

func (m *UpstreamMessage) GetUserMessage() *UserMessage {
	if m != nil {
		return m.UserMessage
	}
	return nil
}

func (m *UpstreamMessage) GetUserOnline() *UserOnline {
	if m != nil {
		return m.UserOnline
	}
	return nil
}

func (m *UpstreamMessage) GetUserOffline() *UserOffline {
	if m != nil {
		return m.UserOffline
	}
	return nil
}

func (m *UpstreamMessage) Reset()         {}
func (m *UpstreamMessage) String() string { return "" }
func (m *UpstreamMessage) ProtoMessage()  {}

// UserMessage 用户消息
type UserMessage struct {
	ClientMsgId string
	FromUserId  int64
	ToUserId    int64
	ToGroupId   int64
	MsgType     int32
	Content     []byte
	Timestamp   int64
}

func (m *UserMessage) GetClientMsgId() string {
	if m != nil {
		return m.ClientMsgId
	}
	return ""
}

func (m *UserMessage) GetFromUserId() int64 {
	if m != nil {
		return m.FromUserId
	}
	return 0
}

func (m *UserMessage) GetToUserId() int64 {
	if m != nil {
		return m.ToUserId
	}
	return 0
}

func (m *UserMessage) GetToGroupId() int64 {
	if m != nil {
		return m.ToGroupId
	}
	return 0
}

func (m *UserMessage) GetMsgType() int32 {
	if m != nil {
		return m.MsgType
	}
	return 0
}

func (m *UserMessage) GetContent() []byte {
	if m != nil {
		return m.Content
	}
	return nil
}

func (m *UserMessage) Reset()         {}
func (m *UserMessage) String() string { return "" }
func (m *UserMessage) ProtoMessage()  {}

// UserOnline 用户上线事件
type UserOnline struct {
	UserId   int64
	ConnId   int64
	DeviceId string
	Platform string
}

func (m *UserOnline) GetUserId() int64 {
	if m != nil {
		return m.UserId
	}
	return 0
}

func (m *UserOnline) GetConnId() int64 {
	if m != nil {
		return m.ConnId
	}
	return 0
}

func (m *UserOnline) GetDeviceId() string {
	if m != nil {
		return m.DeviceId
	}
	return ""
}

func (m *UserOnline) GetPlatform() string {
	if m != nil {
		return m.Platform
	}
	return ""
}

func (m *UserOnline) Reset()         {}
func (m *UserOnline) String() string { return "" }
func (m *UserOnline) ProtoMessage()  {}

// UserOffline 用户下线事件
type UserOffline struct {
	UserId int64
	ConnId int64
}

func (m *UserOffline) GetUserId() int64 {
	if m != nil {
		return m.UserId
	}
	return 0
}

func (m *UserOffline) GetConnId() int64 {
	if m != nil {
		return m.ConnId
	}
	return 0
}

func (m *UserOffline) Reset()         {}
func (m *UserOffline) String() string { return "" }
func (m *UserOffline) ProtoMessage()  {}

// DownstreamMessage 下行消息（Logic -> Access）
type DownstreamMessage struct {
	Payload isDownstreamMessage_Payload
}

type isDownstreamMessage_Payload interface {
	isDownstreamMessage_Payload()
}

type DownstreamMessage_PushMessage struct {
	PushMessage *PushMessage
}

type DownstreamMessage_MessageAck struct {
	MessageAck *MessageAck
}

func (*DownstreamMessage_PushMessage) isDownstreamMessage_Payload() {}
func (*DownstreamMessage_MessageAck) isDownstreamMessage_Payload()  {}

func (m *DownstreamMessage) GetPushMessage() *PushMessage {
	if x, ok := m.GetPayload().(*DownstreamMessage_PushMessage); ok {
		return x.PushMessage
	}
	return nil
}

func (m *DownstreamMessage) GetMessageAck() *MessageAck {
	if x, ok := m.GetPayload().(*DownstreamMessage_MessageAck); ok {
		return x.MessageAck
	}
	return nil
}

func (m *DownstreamMessage) GetPayload() isDownstreamMessage_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *DownstreamMessage) Reset()         {}
func (m *DownstreamMessage) String() string { return "" }
func (m *DownstreamMessage) ProtoMessage()  {}

// PushMessage 推送消息
type PushMessage struct {
	ServerMsgId int64
	FromUserId  int64
	ToUserId    int64
	ToGroupId   int64
	MsgType     int32
	Content     []byte
	Timestamp   int64
}

func (m *PushMessage) Reset()         {}
func (m *PushMessage) String() string { return "" }
func (m *PushMessage) ProtoMessage()  {}

// MessageAck 消息确认
type MessageAck struct {
	ClientMsgId string
	ServerMsgId int64
	Timestamp   int64
}

func (m *MessageAck) Reset()         {}
func (m *MessageAck) String() string { return "" }
func (m *MessageAck) ProtoMessage()  {}
