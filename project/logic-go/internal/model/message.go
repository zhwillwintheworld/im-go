package model

import "time"

// MessageType 消息类型
type MessageType int

const (
	MessageTypeText  MessageType = 1
	MessageTypeImage MessageType = 2
	MessageTypeFile  MessageType = 3
	MessageTypeVoice MessageType = 4
	MessageTypeVideo MessageType = 5
)

// Message 消息实体
type Message struct {
	Id          int64       `json:"id"`
	ClientMsgId string      `json:"clientMsgId"`
	FromUserId  int64       `json:"fromUserId"`
	ToUserId    int64       `json:"toUserId"`
	ToGroupId   int64       `json:"toGroupId"`
	MsgType     MessageType `json:"msgType"`
	Content     string      `json:"content"`
	Status      int         `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
}

// OfflineMessage 离线消息
type OfflineMessage struct {
	UserId    int64     `json:"userId"`
	MessageId int64     `json:"messageId"`
	CreatedAt time.Time `json:"createdAt"`
}
