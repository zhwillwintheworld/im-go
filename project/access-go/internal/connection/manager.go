package connection

import (
	"errors"
	"sync"
)

var ErrConnectionClosed = errors.New("connection closed")

// Manager 管理所有连接
type Manager struct {
	connections map[int64]*Connection            // connID -> Connection
	userConns   map[int64]map[string]*Connection // userID -> platform -> Connection
	mu          sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		connections: make(map[int64]*Connection),
		userConns:   make(map[int64]map[string]*Connection),
	}
}

func (m *Manager) Add(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[conn.ID()] = conn
}

func (m *Manager) Remove(connID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[connID]
	if !ok {
		return
	}

	delete(m.connections, connID)

	// 从用户连接映射中移除（按平台）
	if conn.UserID() > 0 && conn.Platform() != "" {
		if platforms, ok := m.userConns[conn.UserID()]; ok {
			// 只有当前平台的连接是这个 connID 时才删除
			if existingConn, exists := platforms[conn.Platform()]; exists && existingConn.ID() == connID {
				delete(platforms, conn.Platform())
				if len(platforms) == 0 {
					delete(m.userConns, conn.UserID())
				}
			}
		}
	}
}

func (m *Manager) Get(connID int64) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[connID]
}

// BindUser 绑定用户到连接（按平台）
// 如果该用户在同一平台已有连接，会先关闭旧连接
func (m *Manager) BindUser(connID int64, userID int64, platform string) *Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[connID]
	if !ok {
		return nil
	}

	// 初始化用户平台映射
	if _, ok := m.userConns[userID]; !ok {
		m.userConns[userID] = make(map[string]*Connection)
	}

	// 检查是否已有该平台的连接，如果有则需要踢掉旧连接
	var oldConn *Connection
	if existingConn, exists := m.userConns[userID][platform]; exists {
		oldConn = existingConn
	}

	// 绑定新连接
	m.userConns[userID][platform] = conn

	return oldConn // 返回旧连接，调用方负责关闭
}

// GetByUserID 获取用户的所有平台连接
func (m *Manager) GetByUserID(userID int64) []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	platforms, ok := m.userConns[userID]
	if !ok {
		return nil
	}

	conns := make([]*Connection, 0, len(platforms))
	for _, conn := range platforms {
		conns = append(conns, conn)
	}
	return conns
}

// GetByUserIDAndPlatform 获取用户在指定平台的连接
func (m *Manager) GetByUserIDAndPlatform(userID int64, platform string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	platforms, ok := m.userConns[userID]
	if !ok {
		return nil
	}

	return platforms[platform]
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

func (m *Manager) Broadcast(data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, conn := range m.connections {
		if err := conn.Send(data); err != nil {
			// 记录错误但继续给其他连接发送
			// 注意：这里不记录日志，因为 Manager 没有 logger，错误会在 Send 方法中处理
		}
	}
}

// GetAllConnections 返回所有连接（用于心跳检测）
func (m *Manager) GetAllConnections() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns
}
