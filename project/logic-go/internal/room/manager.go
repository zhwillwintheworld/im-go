package room

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"sudooom.im.shared/model"
)

// RoomManager 房间管理器
// 管理所有 Room 实例的生命周期，提供 LRU 淘汰
//
// 使用示例：
//
//	manager := NewRoomManager(5000, 30*time.Minute)
//	room := manager.GetOrCreate(roomId, createParams)
//	manager.Remove(roomId)
type RoomManager struct {
	rooms sync.Map // roomId -> *Room

	// LRU 配置
	maxRooms     int
	evictTimeout time.Duration
	evictTicker  *time.Ticker

	// 用于发送清理通知
	roomService interface{} // 使用 interface{} 避免循环依赖，实际类型为 *RoomService

	logger *slog.Logger
}

// NewRoomManager 创建房间管理器
func NewRoomManager(maxRooms int, evictTimeout time.Duration, evictCheckInterval time.Duration) *RoomManager {
	m := &RoomManager{
		maxRooms:     maxRooms,
		evictTimeout: evictTimeout,
		evictTicker:  time.NewTicker(evictCheckInterval),
		logger:       slog.Default().With("component", "RoomManager"),
	}

	go m.evictLoop()

	return m
}

// SetRoomService 设置 RoomService 引用（用于发送清理通知）
// 由于循环依赖问题，需要在创建 RoomManager 之后调用此方法
func (m *RoomManager) SetRoomService(rs interface{}) {
	m.roomService = rs
}

// GetOrCreate 获取或创建房间
func (m *RoomManager) GetOrCreate(roomId string, creatorID int64, config *model.RoomConfig, gameType string) *Room {
	if val, ok := m.rooms.Load(roomId); ok {
		return val.(*Room)
	}

	room := NewRoom(roomId, creatorID, config, gameType)
	actual, _ := m.rooms.LoadOrStore(roomId, room)
	return actual.(*Room)
}

// Get 获取房间
func (m *RoomManager) Get(roomId string) (*Room, bool) {
	val, ok := m.rooms.Load(roomId)
	if !ok {
		return nil, false
	}
	return val.(*Room), true
}

// Remove 移除房间
func (m *RoomManager) Remove(roomId string) {
	m.rooms.Delete(roomId)
	m.logger.Info("Removed room", "roomId", roomId)
}

// Count 返回当前房间数
func (m *RoomManager) Count() int {
	count := 0
	m.rooms.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// evictLoop 淘汰循环
func (m *RoomManager) evictLoop() {
	for range m.evictTicker.C {
		m.evictInactive()
	}
}

// evictInactive 淘汰不活跃的房间
func (m *RoomManager) evictInactive() {
	now := time.Now()
	toEvict := []string{}

	m.rooms.Range(func(key, value interface{}) bool {
		roomId := key.(string)
		room := value.(*Room)

		if now.Sub(room.LastActiveTime()) > m.evictTimeout {
			toEvict = append(toEvict, roomId)
		}

		return true
	})

	for _, roomId := range toEvict {
		if val, ok := m.rooms.Load(roomId); ok {
			room := val.(*Room)

			// 向房间所有用户发送退出房间消息
			if m.roomService != nil {
				// 使用类型断言
				if rs, ok := m.roomService.(interface {
					BroadcastToRoom(ctx context.Context, roomId string, event string, data interface{}) error
				}); ok {
					ctx := context.Background()
					if err := rs.BroadcastToRoom(ctx, roomId, "room.evicted", map[string]interface{}{
						"roomId": roomId,
						"reason": "房间超时未活跃，已被自动清理",
					}); err != nil {
						m.logger.Warn("Failed to send eviction notification", "roomId", roomId, "error", err)
					}
				}
			}

			m.Remove(roomId)
			m.logger.Info("Evicted inactive room", "roomId", roomId, "lastActive", room.LastActiveTime())
		}
	}
}

// Shutdown 关闭管理器
func (m *RoomManager) Shutdown(ctx context.Context) error {
	m.evictTicker.Stop()

	m.logger.Info("RoomManager shutdown complete")
	return nil
}
