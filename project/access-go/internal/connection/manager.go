package connection

import (
	"errors"
	"sync"
)

var ErrConnectionClosed = errors.New("connection closed")

// Manager 管理所有连接
type Manager struct {
	connections map[int64]*Connection
	userConns   map[int64]map[int64]*Connection // userID -> connID -> Connection
	mu          sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		connections: make(map[int64]*Connection),
		userConns:   make(map[int64]map[int64]*Connection),
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

	// 从用户连接映射中移除
	if conn.UserID() > 0 {
		if userConns, ok := m.userConns[conn.UserID()]; ok {
			delete(userConns, connID)
			if len(userConns) == 0 {
				delete(m.userConns, conn.UserID())
			}
		}
	}
}

func (m *Manager) Get(connID int64) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[connID]
}

func (m *Manager) BindUser(connID, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[connID]
	if !ok {
		return
	}

	if _, ok := m.userConns[userID]; !ok {
		m.userConns[userID] = make(map[int64]*Connection)
	}
	m.userConns[userID][connID] = conn
}

func (m *Manager) GetByUserID(userID int64) []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userConns, ok := m.userConns[userID]
	if !ok {
		return nil
	}

	conns := make([]*Connection, 0, len(userConns))
	for _, conn := range userConns {
		conns = append(conns, conn)
	}
	return conns
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
		conn.Send(data)
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

