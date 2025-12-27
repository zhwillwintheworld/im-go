package model

import "time"

// MessageType 消息类型
type MessageType int

const (
	MessageTypeText  MessageType = 1 // 文本
	MessageTypeImage MessageType = 2 // 图片
	MessageTypeVoice MessageType = 3 // 语音
	MessageTypeVideo MessageType = 4 // 视频
	MessageTypeFile  MessageType = 5 // 文件
)

// MessageStatus 消息状态
const (
	MessageStatusNormal   = 0 // 正常
	MessageStatusRecalled = 1 // 已撤回
	MessageStatusDeleted  = 2 // 已删除
)

// Message 消息实体
type Message struct {
	Id          int64       `json:"id" db:"id"`
	ObjectCode  string      `json:"objectCode" db:"object_code"`
	ClientMsgId string      `json:"clientMsgId" db:"client_msg_id"`
	FromUserId  int64       `json:"fromUserId" db:"from_user_id"`
	ToUserId    *int64      `json:"toUserId" db:"to_user_id"`
	ToGroupId   *int64      `json:"toGroupId" db:"to_group_id"`
	MsgType     MessageType `json:"msgType" db:"msg_type"`
	Content     []byte      `json:"content" db:"content"`
	Status      int         `json:"status" db:"status"`
	CreateAt    time.Time   `json:"createAt" db:"create_at"`
	UpdateAt    time.Time   `json:"updateAt" db:"update_at"`
	Deleted     int         `json:"-" db:"deleted"`
}
