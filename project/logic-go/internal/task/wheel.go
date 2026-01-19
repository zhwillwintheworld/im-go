package task

import (
	"sync"
	"time"
)

const (
	// SlotCount 时间轮槽位数量 (60秒)
	SlotCount = 60
)

// TimeWheel 时间轮
type TimeWheel struct {
	slots       [SlotCount]*Slot // 60个槽位
	currentSlot int              // 当前槽位索引
	slotMu      sync.RWMutex     // 当前槽位索引锁
	ticker      *time.Ticker     // 1秒定时器
}

// NewTimeWheel 创建时间轮
func NewTimeWheel() *TimeWheel {
	tw := &TimeWheel{
		currentSlot: 0,
		ticker:      time.NewTicker(time.Second),
	}

	// 初始化所有槽位
	for i := 0; i < SlotCount; i++ {
		tw.slots[i] = NewSlot()
	}

	return tw
}

// AddTask 添加任务到时间轮
func (tw *TimeWheel) AddTask(task *Task) error {
	if task.Delay < 1 || task.Delay > SlotCount {
		task.Delay = 1 // 默认1秒
	}

	// 计算目标槽位
	tw.slotMu.RLock()
	targetSlot := (tw.currentSlot + task.Delay) % SlotCount
	tw.slotMu.RUnlock()

	// 添加到槽位
	tw.slots[targetSlot].AddTask(task)

	return nil
}

// RemoveTask 从时间轮删除任务
func (tw *TimeWheel) RemoveTask(taskID string, delay int) bool {
	if delay < 1 || delay > SlotCount {
		delay = 1
	}

	// 计算目标槽位
	tw.slotMu.RLock()
	targetSlot := (tw.currentSlot + delay) % SlotCount
	tw.slotMu.RUnlock()

	// 从槽位删除
	return tw.slots[targetSlot].RemoveTask(taskID)
}

// Tick 推进时间轮 (由调度器调用)
func (tw *TimeWheel) Tick() []*Task {
	// 推进到下一个槽位
	tw.slotMu.Lock()
	tw.currentSlot = (tw.currentSlot + 1) % SlotCount
	currentSlot := tw.currentSlot
	tw.slotMu.Unlock()

	// 获取当前槽位的所有任务并清空
	return tw.slots[currentSlot].GetAndClear()
}

// GetCurrentSlot 获取当前槽位索引
func (tw *TimeWheel) GetCurrentSlot() int {
	tw.slotMu.RLock()
	defer tw.slotMu.RUnlock()

	return tw.currentSlot
}

// Stop 停止时间轮
func (tw *TimeWheel) Stop() {
	tw.ticker.Stop()
}

// GetTicker 获取定时器
func (tw *TimeWheel) GetTicker() *time.Ticker {
	return tw.ticker
}

// GetSlotTaskCount 获取指定槽位的任务数量
func (tw *TimeWheel) GetSlotTaskCount(slot int) int {
	if slot < 0 || slot >= SlotCount {
		return 0
	}
	return tw.slots[slot].Count()
}

// GetTotalTaskCount 获取所有槽位的任务总数
func (tw *TimeWheel) GetTotalTaskCount() int {
	total := 0
	for i := 0; i < SlotCount; i++ {
		total += tw.slots[i].Count()
	}
	return total
}
