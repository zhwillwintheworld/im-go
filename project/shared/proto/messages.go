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
	RoomRequest      *RoomRequest      `json:"RoomRequest,omitempty"`      // 房间请求
	GameRequest      *GameRequest      `json:"GameRequest,omitempty"`      // 游戏请求
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

// RoomRequest 房间请求
type RoomRequest struct {
	UserId     int64  `json:"UserId,string"`
	ReqId      string `json:"ReqId"`
	Action     string `json:"Action"`               // CREATE, JOIN, LEAVE, READY, CHANGE_SEAT, START_GAME
	RoomId     string `json:"RoomId"`               // 房间ID
	GameType   string `json:"GameType"`             // 游戏类型：HT_MAHJONG
	RoomConfig string `json:"RoomConfig,omitempty"` // 房间配置（JSON）
	SeatIndex  int32  `json:"SeatIndex,omitempty"`  // 座位索引（-1表示不指定）
}

// GameRequest 游戏请求
type GameRequest struct {
	UserId      int64  `json:"UserId,string"`
	ReqId       string `json:"ReqId"`
	RoomId      string `json:"RoomId"`
	GameType    string `json:"GameType"`
	GamePayload []byte `json:"GamePayload"` // FlatBuffers 游戏请求数据
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
	RoomPush    *RoomPush    `json:"RoomPush,omitempty"` // 房间推送
	GamePush    *GamePush    `json:"GamePush,omitempty"` // 游戏推送
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

// RoomPush 房间推送
type RoomPush struct {
	Event    string `json:"Event"`                     // USER_JOINED, USER_LEFT, USER_READY, GAME_START, GAME_OVER, ROOM_DISMISSED
	RoomId   string `json:"RoomId"`                    // 房间ID
	UserId   int64  `json:"UserId,string,omitempty"`   // 触发事件的用户ID
	RoomInfo []byte `json:"RoomInfo"`                  // FlatBuffers RoomInfo 数据
	ToUserId int64  `json:"ToUserId,string,omitempty"` // 目标用户ID（可选，为空则广播给房间所有人）
}

// GamePush 游戏推送
type GamePush struct {
	RoomId      string `json:"RoomId"`
	GameType    string `json:"GameType"`
	GamePayload []byte `json:"GamePayload"`               // FlatBuffers 游戏推送数据
	ToUserId    int64  `json:"ToUserId,string,omitempty"` // 目标用户ID（可选，为空则广播）
}
