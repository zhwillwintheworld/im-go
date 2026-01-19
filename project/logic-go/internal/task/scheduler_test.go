package task

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewTask 测试创建任务
func TestNewTask(t *testing.T) {
	fn := func(ctx context.Context, target string, metadata map[string]any) error {
		return nil
	}

	task := NewTask("task-1", "user-123", 5, fn)

	if task.ID != "task-1" {
		t.Errorf("期望 ID = task-1, 实际 = %s", task.ID)
	}

	if task.Target != "user-123" {
		t.Errorf("期望 Target = user-123, 实际 = %s", task.Target)
	}

	if task.Delay != 5 {
		t.Errorf("期望 Delay = 5, 实际 = %d", task.Delay)
	}

	if task.Version != 1 {
		t.Errorf("期望 Version = 1, 实际 = %d", task.Version)
	}
}

// TestSlotAddAndRemove 测试槽位添加和删除
func TestSlotAddAndRemove(t *testing.T) {
	slot := NewSlot()

	task1 := NewTask("task-1", "user-1", 5, nil)
	task2 := NewTask("task-2", "user-2", 5, nil)

	// 添加任务
	slot.AddTask(task1)
	slot.AddTask(task2)

	if slot.Count() != 2 {
		t.Errorf("期望任务数 = 2, 实际 = %d", slot.Count())
	}

	// 删除任务
	removed := slot.RemoveTask("task-1")
	if !removed {
		t.Error("期望删除成功")
	}

	if slot.Count() != 1 {
		t.Errorf("期望任务数 = 1, 实际 = %d", slot.Count())
	}

	// 删除不存在的任务
	removed = slot.RemoveTask("task-not-exist")
	if removed {
		t.Error("期望删除失败")
	}
}

// TestSlotGetAndClear 测试获取并清空
func TestSlotGetAndClear(t *testing.T) {
	slot := NewSlot()

	task1 := NewTask("task-1", "user-1", 5, nil)
	task2 := NewTask("task-2", "user-2", 5, nil)

	slot.AddTask(task1)
	slot.AddTask(task2)

	// 获取并清空
	tasks := slot.GetAndClear()

	if len(tasks) != 2 {
		t.Errorf("期望获取2个任务, 实际 = %d", len(tasks))
	}

	if slot.Count() != 0 {
		t.Errorf("期望槽位已清空, 实际任务数 = %d", slot.Count())
	}

	// 再次获取应该为空
	tasks = slot.GetAndClear()
	if tasks != nil {
		t.Errorf("期望 nil, 实际 = %v", tasks)
	}
}

// TestTimeWheelAddTask 测试时间轮添加任务
func TestTimeWheelAddTask(t *testing.T) {
	wheel := NewTimeWheel()

	task := NewTask("task-1", "user-1", 5, nil)
	err := wheel.AddTask(task)

	if err != nil {
		t.Errorf("添加任务失败: %v", err)
	}

	// 检查总任务数
	if wheel.GetTotalTaskCount() != 1 {
		t.Errorf("期望总任务数 = 1, 实际 = %d", wheel.GetTotalTaskCount())
	}
}

// TestTimeWheelTick 测试时间轮推进
func TestTimeWheelTick(t *testing.T) {
	wheel := NewTimeWheel()

	// 添加延迟1秒的任务
	task := NewTask("task-1", "user-1", 1, nil)
	wheel.AddTask(task)

	// 推进1次
	tasks := wheel.Tick()

	// 第一次推进应该获取到任务
	if len(tasks) != 1 {
		t.Errorf("期望获取1个任务, 实际 = %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("期望任务ID = task-1, 实际 = %s", tasks[0].ID)
	}
}

// TestSchedulerStartStop 测试调度器启动和停止
func TestSchedulerStartStop(t *testing.T) {
	scheduler := NewScheduler(5)

	// 启动
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("启动调度器失败: %v", err)
	}

	if !scheduler.IsRunning() {
		t.Error("期望调度器运行中")
	}

	// 重复启动应该失败
	err = scheduler.Start()
	if err == nil {
		t.Error("期望重复启动失败")
	}

	// 停止
	scheduler.Stop()

	if scheduler.IsRunning() {
		t.Error("期望调度器已停止")
	}
}

// TestSchedulerAddRemoveTask 测试添加和删除任务
func TestSchedulerAddRemoveTask(t *testing.T) {
	scheduler := NewScheduler(5)
	scheduler.Start()
	defer scheduler.Stop()

	fn := func(ctx context.Context, target string, metadata map[string]any) error {
		return nil
	}

	task := NewTask("task-1", "user-1", 5, fn)

	// 添加任务
	err := scheduler.AddTask(task)
	if err != nil {
		t.Errorf("添加任务失败: %v", err)
	}

	// 删除任务
	err = scheduler.RemoveTask("task-1", 5)
	if err != nil {
		t.Errorf("删除任务失败: %v", err)
	}

	// 删除不存在的任务
	err = scheduler.RemoveTask("task-not-exist", 5)
	if err == nil {
		t.Error("期望删除失败")
	}
}

// TestSchedulerTaskExecution 测试任务执行
func TestSchedulerTaskExecution(t *testing.T) {
	scheduler := NewScheduler(5)
	scheduler.Start()
	defer scheduler.Stop()

	var executed atomic.Int32
	var mu sync.Mutex
	var results []string

	fn := func(ctx context.Context, target string, metadata map[string]any) error {
		mu.Lock()
		results = append(results, target)
		mu.Unlock()
		executed.Add(1)
		return nil
	}

	// 添加多个任务,延迟1秒
	for i := 1; i <= 5; i++ {
		task := NewTask("task-"+string(rune('0'+i)), "user-"+string(rune('0'+i)), 1, fn)
		scheduler.AddTask(task)
	}

	// 等待任务执行 (2秒足够)
	time.Sleep(2 * time.Second)

	if executed.Load() != 5 {
		t.Errorf("期望执行5个任务, 实际 = %d", executed.Load())
	}

	if len(results) != 5 {
		t.Errorf("期望5个结果, 实际 = %d", len(results))
	}
}

// TestSchedulerConcurrent 测试并发安全
func TestSchedulerConcurrent(t *testing.T) {
	scheduler := NewScheduler(10)
	scheduler.Start()
	defer scheduler.Stop()

	var executed atomic.Int32

	fn := func(ctx context.Context, target string, metadata map[string]any) error {
		executed.Add(1)
		return nil
	}

	// 并发添加任务
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 使用唯一ID
			task := NewTask("task-"+string(rune('0'+id)), "user", 1, fn)
			scheduler.AddTask(task)
		}(i)
	}

	wg.Wait()

	// 等待任务执行
	time.Sleep(2 * time.Second)

	// 检查任务是否都执行了
	if executed.Load() != 100 {
		t.Errorf("期望执行100个任务, 实际 = %d", executed.Load())
	}
}

// TestWorkerPoolPanicRecover 测试 panic 恢复
func TestWorkerPoolPanicRecover(t *testing.T) {
	scheduler := NewScheduler(5)
	scheduler.Start()
	defer scheduler.Stop()

	var executed atomic.Int32

	panicFn := func(ctx context.Context, target string, metadata map[string]any) error {
		executed.Add(1)
		panic("测试 panic")
	}

	normalFn := func(ctx context.Context, target string, metadata map[string]any) error {
		executed.Add(1)
		return nil
	}

	// 添加会 panic 的任务
	task1 := NewTask("task-panic", "user-1", 1, panicFn)
	scheduler.AddTask(task1)

	// 添加正常任务
	task2 := NewTask("task-normal", "user-2", 1, normalFn)
	scheduler.AddTask(task2)

	// 等待执行
	time.Sleep(2 * time.Second)

	// 两个任务都应该被执行 (panic 被恢复)
	if executed.Load() != 2 {
		t.Errorf("期望执行2个任务, 实际 = %d", executed.Load())
	}
}

// BenchmarkSchedulerAddTask 性能测试: 添加任务
func BenchmarkSchedulerAddTask(b *testing.B) {
	scheduler := NewScheduler(10)
	scheduler.Start()
	defer scheduler.Stop()

	fn := func(ctx context.Context, target string, metadata map[string]any) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := NewTask("task", "user", 1, fn)
		scheduler.AddTask(task)
	}
}

// BenchmarkTimeWheelTick 性能测试: 时间轮推进
func BenchmarkTimeWheelTick(b *testing.B) {
	wheel := NewTimeWheel()

	// 添加一些任务
	for i := 0; i < 100; i++ {
		task := NewTask("task", "user", 1, nil)
		wheel.AddTask(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wheel.Tick()
	}
}
