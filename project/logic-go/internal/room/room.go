package room

import (
	"context"
	"sync"
	"time"

	"sudooom.im.shared/model"
)

// Room 房间对象
// 直接在内存中管理房间状态，使用 RWMutex 保证并发安全
//
// 使用示例：
//
//	room := NewRoom("room123", createParams)
//	err := room.Join(ctx, joinParams)
//	err := room.Ready(ctx, userId)
type Room struct {
	mu sync.RWMutex // 读写锁，保护房间状态

	// 基本信息
	roomID       string
	roomName     string
	roomPassword string
	roomType     string
	maxPlayers   int
	gameType     string
	gameSettings map[string]string
	creatorID    int64
	status       string // waiting, playing, finished

	// 玩家列表
	players []model.RoomPlayer

	// 时间戳
	createdAt  time.Time
	updatedAt  time.Time
	lastActive time.Time
}

// NewRoom 创建房间
func NewRoom(roomID string, creatorID int64, config *model.RoomConfig) *Room {
	now := time.Now()
	return &Room{
		roomID:       roomID,
		roomName:     config.RoomName,
		roomPassword: config.RoomPassword,
		roomType:     config.RoomType,
		maxPlayers:   config.MaxPlayers,
		gameType:     "", // 游戏类型在创建时由外部指定
		gameSettings: config.GameSettings,
		creatorID:    creatorID,
		status:       "waiting",
		players:      make([]model.RoomPlayer, 0),
		createdAt:    now,
		updatedAt:    now,
		lastActive:   now,
	}
}

// GetSnapshot 获取房间快照（只读）
func (r *Room) GetSnapshot() *model.Room {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &model.Room{
		RoomID:       r.roomID,
		RoomName:     r.roomName,
		RoomPassword: r.roomPassword,
		RoomType:     r.roomType,
		MaxPlayers:   r.maxPlayers,
		GameType:     r.gameType,
		GameSettings: r.gameSettings,
		CreatorID:    r.creatorID,
		Status:       r.status,
		Players:      append([]model.RoomPlayer{}, r.players...), // 复制切片
		CreatedAt:    r.createdAt,
		UpdatedAt:    r.updatedAt,
	}
}

// Join 加入房间
func (r *Room) Join(ctx context.Context, userId int64, seatIndex int32, userInfo *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证房间状态
	if r.status != "waiting" {
		return ErrGameStarted
	}

	// 检查是否已在房间
	for _, p := range r.players {
		if p.UserID == userId {
			return ErrAlreadyInRoom
		}
	}

	// 检查座位
	if seatIndex != -1 {
		for _, p := range r.players {
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
		IsHost:    userId == r.creatorID,
		UserInfo:  userInfo,
	}
	r.players = append(r.players, player)

	r.lastActive = time.Now()

	return nil
}

// Leave 离开房间
func (r *Room) Leave(ctx context.Context, userId int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 查找玩家
	index := -1
	for i, p := range r.players {
		if p.UserID == userId {
			index = i
			break
		}
	}

	if index == -1 {
		return ErrNotInRoom
	}

	// 移除玩家
	r.players = append(r.players[:index], r.players[index+1:]...)
	r.lastActive = time.Now()

	return nil
}

// Ready 准备/取消准备
func (r *Room) Ready(ctx context.Context, userId int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 查找玩家
	for i := range r.players {
		if r.players[i].UserID == userId {
			r.players[i].IsReady = !r.players[i].IsReady
			r.lastActive = time.Now()
			return nil
		}
	}

	return ErrNotInRoom
}

// ChangeSeat 换座位
func (r *Room) ChangeSeat(ctx context.Context, userId int64, targetSeat int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.status != "waiting" {
		return ErrGameStarted
	}

	// 查找玩家
	playerIndex := -1
	for i, p := range r.players {
		if p.UserID == userId {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return ErrNotInRoom
	}

	// 检查是否已准备
	if r.players[playerIndex].IsReady {
		return ErrPlayerReady
	}

	// 检查目标座位
	if targetSeat != -1 {
		for _, p := range r.players {
			if p.SeatIndex == targetSeat {
				return ErrSeatOccupied
			}
		}
	}

	r.players[playerIndex].SeatIndex = targetSeat
	r.lastActive = time.Now()

	return nil
}

// StartGame 开始游戏
func (r *Room) StartGame(ctx context.Context, userId int64, strategy GameTypeStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.status != "waiting" {
		return ErrGameStarted
	}

	// 检查是否是房主
	isHost := false
	for _, p := range r.players {
		if p.UserID == userId && p.IsHost {
			isHost = true
			break
		}
	}
	if !isHost {
		return ErrNotRoomHost
	}

	// 使用策略验证玩家
	if err := strategy.ValidatePlayers(r.GetSnapshot()); err != nil {
		return err
	}

	r.status = "playing"
	r.lastActive = time.Now()

	return nil
}

// LastActiveTime 获取最后活跃时间
func (r *Room) LastActiveTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastActive
}
