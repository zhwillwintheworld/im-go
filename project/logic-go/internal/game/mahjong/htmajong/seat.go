package htmajong

import (
	"sync"

	"sudooom.im.shared/model"
)

// ListeningState 报听状态
type ListeningState struct {
	IsListening      bool      // 是否已报听
	ListeningTiles   []Mahjong // 可以胡的牌
	ListeningAtRound int32     // 报听时的回合数
}

// Seat 座位对象（玩家游戏状态管理）
type Seat struct {
	mu sync.RWMutex // 读写锁，保护座位状态

	// 玩家信息
	user     *model.User
	position Position

	// 牌组管理
	hand        *Hand        // 手牌管理器
	publicTiles *PublicTiles // 公开牌管理器（碰、杠）
	discardPile *DiscardPile // 出牌堆

	// 游戏状态
	points         int             // 分数
	step           int32           // 下了多少手（统一使用锁保护）
	listeningState *ListeningState // 报听状态
}

// NewSeat 创建新座位
func NewSeat(user *model.User, position Position) *Seat {
	return &Seat{
		user:           user,
		position:       position,
		hand:           NewHand(),
		publicTiles:    NewPublicTiles(),
		discardPile:    NewDiscardPile(),
		points:         0,
		listeningState: &ListeningState{},
	}
}

// ========== 玩家信息获取 ==========

// GetUser 获取用户信息
func (s *Seat) GetUser() *model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user
}

// GetUserID 获取用户ID
func (s *Seat) GetUserID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user.UserID
}

// GetPosition 获取位置
func (s *Seat) GetPosition() Position {
	return s.position
}

// ========== 手牌操作 ==========

// DrawTile 摸牌
func (s *Seat) DrawTile(tile Mahjong) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hand.Size() >= StandardHandSize+1 {
		return ErrHandFull
	}
	return s.hand.Add(tile)
}

// DiscardTile 出牌
func (s *Seat) DiscardTile(tile Mahjong) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hand.Contains(tile) {
		return ErrTileNotInHand.WithContext("tile", tile.Number)
	}

	if err := s.hand.Remove(tile); err != nil {
		return err
	}

	s.discardPile.Add(tile)
	s.step++
	return nil
}

// DiscardTileByNumber 根据数字出牌
func (s *Seat) DiscardTileByNumber(number int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hand.ContainsNumber(number) {
		return ErrTileNotInHand.WithContext("number", number)
	}

	if err := s.hand.RemoveByNumber(number); err != nil {
		return err
	}

	tile, _ := GenerateByNumber(number)
	s.discardPile.Add(tile)
	s.step++
	return nil
}

// GetHandTiles 获取手牌（返回副本）
func (s *Seat) GetHandTiles() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hand.GetTiles()
}

// GetHandSize 获取手牌数量
func (s *Seat) GetHandSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hand.Size()
}

// ========== 公开牌操作 ==========

// Pong 碰牌
func (s *Seat) Pong(tiles []Mahjong) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(tiles) != 3 {
		return ErrInvalidMove.WithContext("operation", "pong")
	}

	// 从手牌移除2张（第3张是别人出的）
	for i := 0; i < 2; i++ {
		if err := s.hand.Remove(tiles[i]); err != nil {
			return err
		}
	}

	return s.publicTiles.AddPong(tiles)
}

// Kong 杠牌
func (s *Seat) Kong(tiles []Mahjong, kongType TileGroupType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(tiles) != 4 {
		return ErrInvalidMove.WithContext("operation", "kong")
	}

	// 根据杠牌类型决定移除多少张
	removeCount := 3 // 默认移除3张（明杠）
	if kongType == GroupTypeConcealedKong {
		removeCount = 4 // 暗杠移除4张
	}

	for i := 0; i < removeCount; i++ {
		if err := s.hand.Remove(tiles[i]); err != nil {
			return err
		}
	}

	return s.publicTiles.AddKong(tiles, kongType)
}

// GetPublicTiles 获取所有公开牌
func (s *Seat) GetPublicTiles() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.publicTiles.GetAllTiles()
}

// HasPong 判断是否有碰牌组
func (s *Seat) HasPong() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.publicTiles.HasPong()
}

// ========== 出牌堆操作 ==========

// GetDiscardedTiles 获取所有出牌
func (s *Seat) GetDiscardedTiles() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.discardPile.GetAll()
}

// GetLastDiscardedTile 获取最后出的牌
func (s *Seat) GetLastDiscardedTile() (Mahjong, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.discardPile.GetLast()
}

// ========== 报听相关 ==========

// DeclareListening 报听
func (s *Seat) DeclareListening(tiles []Mahjong) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listeningState.IsListening {
		return ErrInvalidMove.WithContext("reason", "already listening")
	}

	s.listeningState.IsListening = true
	s.listeningState.ListeningTiles = tiles
	s.listeningState.ListeningAtRound = s.step
	return nil
}

// IsListening 是否已报听
func (s *Seat) IsListening() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listeningState.IsListening
}

// GetListeningTiles 获取报听后可以胡的牌
func (s *Seat) GetListeningTiles() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tiles := make([]Mahjong, len(s.listeningState.ListeningTiles))
	copy(tiles, s.listeningState.ListeningTiles)
	return tiles
}

// CanDeclareListening 判断是否可以报听
func (s *Seat) CanDeclareListening() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 必须是第一手
	if s.step != 1 {
		return false
	}

	// 不能有碰牌组
	if s.publicTiles.HasPong() {
		return false
	}

	// 手牌必须是13张
	if s.hand.Size() != StandardHandSize {
		return false
	}

	return true
}

// ========== 状态查询 ==========

// GetPoints 获取分数
func (s *Seat) GetPoints() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.points
}

// AddPoints 增加分数
func (s *Seat) AddPoints(points int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.points += points
}

// GetStep 获取回合数
func (s *Seat) GetStep() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.step
}

// IsFirstRound 判断是否是第一回合
func (s *Seat) IsFirstRound() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.step == 1
}

// ========== 内部访问（供算法使用） ==========

// GetHandRef 获取手牌管理器引用（仅供内部算法使用）
func (s *Seat) GetHandRef() *Hand {
	return s.hand
}

// GetPublicTilesRef 获取公开牌管理器引用（仅供内部算法使用）
func (s *Seat) GetPublicTilesRef() *PublicTiles {
	return s.publicTiles
}

// ========== 工厂方法（向后兼容） ==========

// GenerateSeat 生成座位（向后兼容旧代码）
func GenerateSeat(user *model.User, position Position) *Seat {
	return NewSeat(user, position)
}

// ========== 辅助函数（保留用于算法） ==========

// FindNextSeat 查找下一个座位
func FindNextSeat(table *Table, seat *Seat) *Seat {
	switch seat.position {
	case EAST:
		return table.GetNorth()
	case SOUTH:
		return table.GetEast()
	case WEST:
		return table.GetSouth()
	case NORTH:
		return table.GetWest()
	default:
		return nil
	}
}

// FindSeat 根据位置查找座位
func FindSeat(table *Table, position Position) *Seat {
	switch position {
	case EAST:
		return table.GetEast()
	case SOUTH:
		return table.GetSouth()
	case WEST:
		return table.GetWest()
	case NORTH:
		return table.GetNorth()
	default:
		return nil
	}
}

// FindSeatByUserID 根据用户ID查找座位
func FindSeatByUserID(table *Table, userID int64) *Seat {
	if east := table.GetEast(); east != nil && east.GetUserID() == userID {
		return east
	}
	if south := table.GetSouth(); south != nil && south.GetUserID() == userID {
		return south
	}
	if west := table.GetWest(); west != nil && west.GetUserID() == userID {
		return west
	}
	if north := table.GetNorth(); north != nil && north.GetUserID() == userID {
		return north
	}
	return nil
}

// ========== 向后兼容（用于algorithm.go） ==========
// 以下属性和方法用于保持与旧算法代码的兼容性
// TODO: 重构算法后移除这些兼容代码

// ExtraList 获取手牌（向后兼容）
func (s *Seat) ExtraList() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hand.GetTilesRef()
}

// PublicList 获取公开牌（向后兼容）
func (s *Seat) PublicList() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.publicTiles.GetAllTiles()
}

// IsPublic 是否报听（向后兼容）
func (s *Seat) IsPublic() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listeningState.IsListening
}

// Step 获取步数（向后兼容）
func (s *Seat) Step() int32 {
	return s.GetStep()
}

// PublicWinMahjong 获取报听可胡的牌（向后兼容）
func (s *Seat) PublicWinMahjong() []Mahjong {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tiles := make([]Mahjong, len(s.listeningState.ListeningTiles))
	copy(tiles, s.listeningState.ListeningTiles)
	return tiles
}

// SetPublicWinMahjong 设置报听可胡的牌（向后兼容）
func (s *Seat) SetPublicWinMahjong(tiles []Mahjong) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeningState.ListeningTiles = tiles
}

// User 获取用户（向后兼容）
func (s *Seat) User() *model.User {
	return s.GetUser()
}

// Position 获取位置（向后兼容）
func (s *Seat) Position() Position {
	return s.GetPosition()
}
