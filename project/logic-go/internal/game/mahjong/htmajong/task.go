package htmajong

// TaskType 任务类型枚举
type TaskType int

const (
	// OUT_TASK 出牌任务
	OUT_TASK TaskType = iota
	// LEASE_TASK 租约任务
	LEASE_TASK
)

func (t TaskType) String() string {
	switch t {
	case OUT_TASK:
		return "OUT"
	case LEASE_TASK:
		return "LEASE"
	default:
		return "UNKNOWN"
	}
}
