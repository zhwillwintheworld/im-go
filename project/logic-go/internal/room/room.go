package room

import (
	"sync"
	"time"

	"sudooom.im.shared/model"
)

// RoomInstance 房间实例对象
// 在内存中管理房间状态，使用 RWMutex 保证并发安全
// 与 model.Room（数据传输对象）区分，RoomInstance 负责并发控制和生命周期管理
//
// 使用示例：
//
//	room := NewRoom("room123", createParams)
//	err := room.Join(joinParams)
//	err := room.Ready(userId)
type RoomInstance struct {
	mu         sync.RWMutex // 读写锁，保护房间状态
	roomInfo   *model.Room  // 房间数据
	lastActive time.Time    // 最后活跃时间（用于淘汰策略）
}

// NewRoom 创建房间实例
func NewRoom(roomID string, creatorID int64, config *model.RoomConfig, gameType string) *RoomInstance {
	now := time.Now()
	return &RoomInstance{
		roomInfo: &model.Room{
			RoomID:       roomID,
			RoomName:     config.RoomName,
			RoomPassword: config.RoomPassword,
			RoomType:     config.RoomType,
			MaxPlayers:   config.MaxPlayers,
			GameType:     gameType,
			GameSettings: config.GameSettings,
			CreatorID:    creatorID,
			Status:       "waiting",
			Players:      make([]model.RoomPlayer, 0),
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		lastActive: now,
	}
}

// GetSnapshot 获取房间快照（只读）
func (r *RoomInstance) GetSnapshot() *model.Room {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 深拷贝房间信息
	snapshot := *r.roomInfo
	snapshot.Players = append([]model.RoomPlayer{}, r.roomInfo.Players...)
	return &snapshot
}

// Join 加入房间
func (r *RoomInstance) Join(userId int64, seatIndex int32, userInfo *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证房间状态
	if r.roomInfo.Status != "waiting" {
		return ErrGameStarted
	}

	// 检查是否已在房间
	for _, p := range r.roomInfo.Players {
		if p.UserID == userId {
			return ErrAlreadyInRoom
		}
	}

	// 检查座位
	if seatIndex != -1 {
		for _, p := range r.roomInfo.Players {
			if p.SeatIndex == seatIndex {
				return ErrSeatOccupied
			}
		}
	}

	// 加入玩家
	player := model.RoomPlayer{
		UserID:    userId,
		SeatIndex: seatIndex,
		IsReady:   false,
		IsHost:    userId == r.roomInfo.CreatorID,
		UserInfo:  userInfo,
	}
	r.roomInfo.Players = append(r.roomInfo.Players, player)

	r.lastActive = time.Now()

	return nil
}

// Leave 离开房间
func (r *RoomInstance) Leave(userId int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 查找玩家
	index := -1
	for i, p := range r.roomInfo.Players {
		if p.UserID == userId {
			index = i
			break
		}
	}

	if index == -1 {
		return ErrNotInRoom
	}

	// 移除玩家
	r.roomInfo.Players = append(r.roomInfo.Players[:index], r.roomInfo.Players[index+1:]...)
	r.lastActive = time.Now()

	return nil
}

// Ready 准备/取消准备
func (r *RoomInstance) Ready(userId int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 查找玩家
	for i := range r.roomInfo.Players {
		if r.roomInfo.Players[i].UserID == userId {
			r.roomInfo.Players[i].IsReady = !r.roomInfo.Players[i].IsReady
			r.lastActive = time.Now()
			return nil
		}
	}

	return ErrNotInRoom
}

// ChangeSeat 换座位
func (r *RoomInstance) ChangeSeat(userId int64, targetSeat int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.roomInfo.Status != "waiting" {
		return ErrGameStarted
	}

	// 查找玩家
	playerIndex := -1
	for i, p := range r.roomInfo.Players {
		if p.UserID == userId {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return ErrNotInRoom
	}

	// 检查是否已准备
	if r.roomInfo.Players[playerIndex].IsReady {
		return ErrPlayerReady
	}

	// 检查目标座位
	if targetSeat != -1 {
		for _, p := range r.roomInfo.Players {
			if p.SeatIndex == targetSeat {
				return ErrSeatOccupied
			}
		}
	}

	r.roomInfo.Players[playerIndex].SeatIndex = targetSeat
	r.lastActive = time.Now()

	return nil
}

// StartGame 开始游戏
func (r *RoomInstance) StartGame(userId int64, strategy GameTypeStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.roomInfo.Status != "waiting" {
		return ErrGameStarted
	}

	// 检查是否是房主
	isHost := false
	for _, p := range r.roomInfo.Players {
		if p.UserID == userId && p.IsHost {
			isHost = true
			break
		}
	}
	if !isHost {
		return ErrNotRoomHost
	}

	// 使用策略验证玩家（已持有写锁，直接传递 roomInfo）
	if err := strategy.ValidatePlayers(r.roomInfo); err != nil {
		return err
	}

	r.roomInfo.Status = "playing"
	r.lastActive = time.Now()

	return nil
}

// LastActiveTime 获取最后活跃时间
func (r *RoomInstance) LastActiveTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastActive
}
