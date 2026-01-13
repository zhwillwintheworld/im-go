package htmajong

import "sync"

// ClaimStatus 认领状态枚举
type ClaimStatus int

const (
	// PENG 碰
	PENG ClaimStatus = iota
	// HU 胡
	HU
	// GANG_STATUS 杠
	GANG_STATUS
	// PRIVATE_GANG 暗杠
	PRIVATE_GANG
	// PUBLIC_GANG 明杠
	PUBLIC_GANG
	// PUBLIC 报听
	PUBLIC
	// NONE 不要
	NONE
)

func (c ClaimStatus) String() string {
	switch c {
	case PENG:
		return "PENG"
	case HU:
		return "HU"
	case GANG_STATUS:
		return "GANG"
	case PRIVATE_GANG:
		return "PRIVATE_GANG"
	case PUBLIC_GANG:
		return "PUBLIC_GANG"
	case PUBLIC:
		return "PUBLIC"
	case NONE:
		return "NONE"
	default:
		return "UNKNOWN"
	}
}

// ClaimDetail 认领详情
type ClaimDetail struct {
	ReceiveUser *Seat       // 接收用户
	Status      ClaimStatus // 状态
	Reply       int32       // 0:未回复 1:同意 2:拒绝
}

// ClaimResult 认领结果
type ClaimResult struct {
	Status  ClaimStatus // 状态
	Trigger []*Seat     // 触发者列表
}

// ClaimInfo 认领信息
type ClaimInfo struct {
	mu sync.RWMutex // 保护所有字段

	supplierType SupplierType   // 提供方式
	isSelf       bool           // 是否发给自己
	claimNumber  int32          // 认领编号（统一使用锁保护）
	happenedUser *Seat          // 谁触发的
	happened     Mahjong        // 什么牌触发的
	first        []*ClaimDetail // 胡
	second       []*ClaimDetail // 碰 杠 不要
	anyReply     bool           // 是否有人回复
	isPublic     bool           // 是否是报听认领
	result       *ClaimResult   // 结果
}

// GenerateClaim 生成认领
func GenerateClaim(
	supplierType SupplierType,
	isSelf bool,
	claimNumber int32,
	happenedUser *Seat,
	happened Mahjong,
	isPublic bool,
) *ClaimInfo {
	return &ClaimInfo{
		supplierType: supplierType,
		isSelf:       isSelf,
		claimNumber:  claimNumber,
		happenedUser: happenedUser,
		happened:     happened,
		first:        make([]*ClaimDetail, 0),
		second:       make([]*ClaimDetail, 0),
		anyReply:     false,
		isPublic:     isPublic,
		result:       nil,
	}
}

// ========== 访问器方法 ==========

// GetSupplierType 获取提供方式
func (c *ClaimInfo) GetSupplierType() SupplierType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.supplierType
}

// IsSelf 是否发给自己
func (c *ClaimInfo) IsSelf() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isSelf
}

// GetClaimNumber 获取认领编号
func (c *ClaimInfo) GetClaimNumber() int32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.claimNumber
}

// GetHappenedUser 获取触发用户
func (c *ClaimInfo) GetHappenedUser() *Seat {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.happenedUser
}

// GetHappened 获取触发的牌
func (c *ClaimInfo) GetHappened() Mahjong {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.happened
}

// GetFirst 获取第一优先级列表（胡）
func (c *ClaimInfo) GetFirst() []*ClaimDetail {
	c.mu.RLock()
	defer c.mu.RUnlock()
	details := make([]*ClaimDetail, len(c.first))
	copy(details, c.first)
	return details
}

// AddFirst 添加第一优先级详情（胡）
func (c *ClaimInfo) AddFirst(detail *ClaimDetail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.first = append(c.first, detail)
}

// GetSecond 获取第二优先级列表（碰 杠 不要）
func (c *ClaimInfo) GetSecond() []*ClaimDetail {
	c.mu.RLock()
	defer c.mu.RUnlock()
	details := make([]*ClaimDetail, len(c.second))
	copy(details, c.second)
	return details
}

// AddSecond 添加第二优先级详情（碰 杠 不要）
func (c *ClaimInfo) AddSecond(detail *ClaimDetail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.second = append(c.second, detail)
}

// IsAnyReply 是否有人回复
func (c *ClaimInfo) IsAnyReply() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.anyReply
}

// SetAnyReply 设置是否有人回复
func (c *ClaimInfo) SetAnyReply(replied bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.anyReply = replied
}

// IsPublic 是否是报听认领
func (c *ClaimInfo) IsPublic() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isPublic
}

// GetResult 获取认领结果
func (c *ClaimInfo) GetResult() *ClaimResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.result
}

// SetResult 设置认领结果
func (c *ClaimInfo) SetResult(result *ClaimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.result = result
}

// ========== 向后兼容属性（供外部代码迁移使用） ==========

// SupplierType 获取提供方式（deprecated: 使用 GetSupplierType() 代替）
func (c *ClaimInfo) SupplierType() SupplierType {
	return c.GetSupplierType()
}

// HappenedUser 获取触发用户（deprecated: 使用 GetHappenedUser() 代替）
func (c *ClaimInfo) HappenedUser() *Seat {
	return c.GetHappenedUser()
}

// Happened 获取触发的牌（deprecated: 使用 GetHappened() 代替）
func (c *ClaimInfo) Happened() Mahjong {
	return c.GetHappened()
}
