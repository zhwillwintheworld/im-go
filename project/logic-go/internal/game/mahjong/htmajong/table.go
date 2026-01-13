package htmajong

import (
	"sync"
	"time"

	"sudooom.im.shared/model"
)

// Table 牌桌对象
type Table struct {
	mu sync.RWMutex // 保护所有可变状态

	// 基本信息（创建后不变，可以不加锁直接访问）
	RoomID     string    // 房间ID
	TableID    string    // 桌号
	CreateTime time.Time // 创建时间

	// 配置信息（创建后不变）
	CanFireWinner        bool // 能否能烧庄
	BigBigWinConfig      bool // 大大胡配置
	CompleteWinnerConfig bool // 完庄完杠
	FireWinnerConfig     int  // 烧庄配置
	CanPublic            bool // 是否能够报听

	// 可变状态（需要锁保护）
	openUser       *model.User   // 开局的人
	east           *Seat         // 东
	south          *Seat         // 南
	west           *Seat         // 西
	north          *Seat         // 北
	extra          []Mahjong     // 剩下的牌
	randomNumber   int           // 骰子点数
	viewer         []*model.User // 观众
	currentSeat    *Seat         // 当前人
	claimNumber    int32         // 等待多家反应的认领编号（统一使用锁保护）
	claim          *ClaimInfo    // 等待多家反应的认领
	step           int32         // 下了多少手（统一使用锁保护）
	winner         []HuDetail    // 赢家
	loser          []LoseDetail  // 输家
	specificLoser  *Position     // 被打的人（输翻倍）
	specificNumber *int          // 打的点数（输翻倍）
	taskID         string        // 当前任务id
	taskType       *TaskType     // 当前任务类型
}

// NewTable 创建新桌子
func NewTable(
	roomID string,
	tableID string,
	openUser *model.User,
	east *Seat,
	south *Seat,
	west *Seat,
	north *Seat,
	extra []Mahjong,
	canFireWinner bool,
	bigBigWinConfig bool,
	completeWinnerConfig bool,
	fireWinnerConfig int,
	canPublic bool,
	randomNumber int,
) *Table {
	return &Table{
		RoomID:               roomID,
		TableID:              tableID,
		CreateTime:           time.Now(),
		CanFireWinner:        canFireWinner,
		BigBigWinConfig:      bigBigWinConfig,
		CompleteWinnerConfig: completeWinnerConfig,
		FireWinnerConfig:     fireWinnerConfig,
		CanPublic:            canPublic,
		openUser:             openUser,
		east:                 east,
		south:                south,
		west:                 west,
		north:                north,
		extra:                extra,
		randomNumber:         randomNumber,
		viewer:               make([]*model.User, 0),
		currentSeat:          nil,
		claimNumber:          0,
		claim:                nil,
		step:                 0,
		winner:               make([]HuDetail, 0),
		loser:                make([]LoseDetail, 0),
		specificLoser:        nil,
		specificNumber:       nil,
		taskID:               "",
		taskType:             nil,
	}
}

// ========== 访问器方法 ==========

// GetOpenUser 获取开局用户
func (t *Table) GetOpenUser() *model.User {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.openUser
}

// GetEast 获取东座位
func (t *Table) GetEast() *Seat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.east
}

// GetSouth 获取南座位
func (t *Table) GetSouth() *Seat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.south
}

// GetWest 获取西座位
func (t *Table) GetWest() *Seat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.west
}

// GetNorth 获取北座位
func (t *Table) GetNorth() *Seat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.north
}

// GetExtra 获取剩余牌（返回副本）
func (t *Table) GetExtra() []Mahjong {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tiles := make([]Mahjong, len(t.extra))
	copy(tiles, t.extra)
	return tiles
}

// GetExtraCount 获取剩余牌数量
func (t *Table) GetExtraCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.extra)
}

// DrawExtra 从牌堆摸一张牌
func (t *Table) DrawExtra() (Mahjong, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.extra) == 0 {
		return Mahjong{}, ErrDeckEmpty
	}
	tile := t.extra[0]
	t.extra = t.extra[1:]
	return tile, nil
}

// GetRandomNumber 获取骰子点数
func (t *Table) GetRandomNumber() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.randomNumber
}

// GetViewer 获取观众列表（返回副本）
func (t *Table) GetViewer() []*model.User {
	t.mu.RLock()
	defer t.mu.RUnlock()
	viewers := make([]*model.User, len(t.viewer))
	copy(viewers, t.viewer)
	return viewers
}

// AddViewer 添加观众
func (t *Table) AddViewer(user *model.User) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.viewer = append(t.viewer, user)
}

// GetCurrentSeat 获取当前座位
func (t *Table) GetCurrentSeat() *Seat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentSeat
}

// SetCurrentSeat 设置当前座位
func (t *Table) SetCurrentSeat(seat *Seat) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentSeat = seat
}

// GetClaimNumber 获取认领编号
func (t *Table) GetClaimNumber() int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.claimNumber
}

// IncrementClaimNumber 递增认领编号并返回新值
func (t *Table) IncrementClaimNumber() int32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.claimNumber++
	return t.claimNumber
}

// GetClaim 获取当前认领
func (t *Table) GetClaim() *ClaimInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.claim
}

// SetClaim 设置当前认领
func (t *Table) SetClaim(claim *ClaimInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.claim = claim
}

// GetStep 获取已下手数
func (t *Table) GetStep() int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.step
}

// IncrementStep 递增已下手数
func (t *Table) IncrementStep() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.step++
}

// GetWinner 获取赢家列表（返回副本）
func (t *Table) GetWinner() []HuDetail {
	t.mu.RLock()
	defer t.mu.RUnlock()
	winners := make([]HuDetail, len(t.winner))
	copy(winners, t.winner)
	return winners
}

// AddWinner 添加赢家
func (t *Table) AddWinner(detail HuDetail) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.winner = append(t.winner, detail)
}

// GetLoser 获取输家列表（返回副本）
func (t *Table) GetLoser() []LoseDetail {
	t.mu.RLock()
	defer t.mu.RUnlock()
	losers := make([]LoseDetail, len(t.loser))
	copy(losers, t.loser)
	return losers
}

// AddLoser 添加输家
func (t *Table) AddLoser(detail LoseDetail) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.loser = append(t.loser, detail)
}

// GetSpecificLoser 获取被打的人
func (t *Table) GetSpecificLoser() *Position {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.specificLoser
}

// SetSpecificLoser 设置被打的人
func (t *Table) SetSpecificLoser(pos *Position) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.specificLoser = pos
}

// GetSpecificNumber 获取打的点数
func (t *Table) GetSpecificNumber() *int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.specificNumber
}

// SetSpecificNumber 设置打的点数
func (t *Table) SetSpecificNumber(num *int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.specificNumber = num
}

// GetTaskID 获取当前任务ID
func (t *Table) GetTaskID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.taskID
}

// SetTaskID 设置当前任务ID
func (t *Table) SetTaskID(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskID = id
}

// GetTaskType 获取当前任务类型
func (t *Table) GetTaskType() *TaskType {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.taskType
}

// SetTaskType 设置当前任务类型
func (t *Table) SetTaskType(taskType *TaskType) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskType = taskType
}

// ========== 向后兼容属性（供外部代码迁移使用，标记为 deprecated） ==========

// East 获取东座位（deprecated: 使用 GetEast() 代替）
func (t *Table) East() *Seat {
	return t.GetEast()
}

// South 获取南座位（deprecated: 使用 GetSouth() 代替）
func (t *Table) South() *Seat {
	return t.GetSouth()
}

// West 获取西座位（deprecated: 使用 GetWest() 代替）
func (t *Table) West() *Seat {
	return t.GetWest()
}

// North 获取北座位（deprecated: 使用 GetNorth() 代替）
func (t *Table) North() *Seat {
	return t.GetNorth()
}

// Extra 获取剩余牌（deprecated: 使用 GetExtra() 代替）
func (t *Table) Extra() []Mahjong {
	return t.GetExtra()
}

// CurrentSeat 获取当前座位（deprecated: 使用 GetCurrentSeat() 代替）
func (t *Table) CurrentSeat() *Seat {
	return t.GetCurrentSeat()
}

// Claim 获取当前认领（deprecated: 使用 GetClaim() 代替）
func (t *Table) Claim() *ClaimInfo {
	return t.GetClaim()
}
