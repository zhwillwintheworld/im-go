package htmajong

import (
	"sync/atomic"
	"time"

	"sudooom.im.shared/model"
)

// Table 牌桌对象
type Table struct {
	RoomID               string        // 房间ID
	TableID              string        // 桌号
	OpenUser             *model.User   // 开局的人
	East                 *Seat         // 东
	South                *Seat         // 南
	West                 *Seat         // 西
	North                *Seat         // 北
	Extra                []Mahjong     // 剩下的牌
	CanFireWinner        bool          // 能否能烧庄
	BigBigWinConfig      bool          // 大大胡配置
	CompleteWinnerConfig bool          // 完庄完杠
	FireWinnerConfig     int           // 烧庄配置
	CanPublic            bool          // 是否能够报听
	RandomNumber         int           // 骰子点数
	Viewer               []*model.User // 观众
	CurrentSeat          *Seat         // 当前人
	CreateTime           time.Time     // 创建时间
	LeaseNumber          atomic.Int32  // 等待多家反应的租约编号
	Lease                *LeaseInfo    // 等待多家反应的租约
	Step                 atomic.Int32  // 下了多少手
	Winner               []HuDetail    // 赢家
	Loser                []LoseDetail  // 输家
	SpecificLoser        *Position     // 被打的人（输翻倍）
	SpecificNumber       *int          // 打的点数（输翻倍）
	TaskID               string        // 当前任务id
	TaskType             *TaskType     // 当前任务类型
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
		OpenUser:             openUser,
		East:                 east,
		South:                south,
		West:                 west,
		North:                north,
		Extra:                extra,
		CanFireWinner:        canFireWinner,
		BigBigWinConfig:      bigBigWinConfig,
		CompleteWinnerConfig: completeWinnerConfig,
		FireWinnerConfig:     fireWinnerConfig,
		CanPublic:            canPublic,
		RandomNumber:         randomNumber,
		Viewer:               make([]*model.User, 0),
		CurrentSeat:          nil,
		CreateTime:           time.Now(),
		Winner:               make([]HuDetail, 0),
		Loser:                make([]LoseDetail, 0),
		SpecificLoser:        nil,
		SpecificNumber:       nil,
		TaskID:               "",
		TaskType:             nil,
		Lease:                nil,
	}
}
