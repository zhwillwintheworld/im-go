package htmajong

import "sync/atomic"

// LeaseStatus 租约状态枚举
type LeaseStatus int

const (
	// PENG 碰
	PENG LeaseStatus = iota
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

func (l LeaseStatus) String() string {
	switch l {
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

// LeaseDetail 租约详情
type LeaseDetail struct {
	ReceiveUser *Seat       // 接收用户
	Status      LeaseStatus // 状态
	Reply       int32       // 0:未回复 1:同意 2:拒绝
}

// LeaseResult 租约结果
type LeaseResult struct {
	Status  LeaseStatus // 状态
	Trigger []*Seat     // 触发者列表
}

// LeaseInfo 租约信息
type LeaseInfo struct {
	SupplierType SupplierType   // 提供方式
	IsSelf       bool           // 是否发给自己
	LeaseNumber  *atomic.Int32  // 租约编号
	HappenedUser *Seat          // 谁触发的
	Happened     Mahjong        // 什么牌触发的
	First        []*LeaseDetail // 胡
	Second       []*LeaseDetail // 碰 杠 不要
	AnyReply     bool           // 是否有人回复
	IsPublic     bool           // 是否是报听租约
	Result       *LeaseResult   // 结果
}

// GenerateLease 生成租约
func GenerateLease(
	supplierType SupplierType,
	isSelf bool,
	leaseNumber *atomic.Int32,
	happenedUser *Seat,
	happened Mahjong,
	isPublic bool,
) *LeaseInfo {
	return &LeaseInfo{
		SupplierType: supplierType,
		IsSelf:       isSelf,
		LeaseNumber:  leaseNumber,
		HappenedUser: happenedUser,
		Happened:     happened,
		First:        make([]*LeaseDetail, 0),
		Second:       make([]*LeaseDetail, 0),
		AnyReply:     false,
		IsPublic:     isPublic,
		Result:       nil,
	}
}
