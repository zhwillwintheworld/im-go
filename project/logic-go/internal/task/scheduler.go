package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Scheduler 任务调度器
type Scheduler struct {
	wheel      *TimeWheel  // 时间轮
	workerPool *WorkerPool // 工作协程池
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     *slog.Logger
	running    bool
	runningMu  sync.RWMutex
}

// NewScheduler 创建任务调度器
func NewScheduler(workerCount int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		wheel:      NewTimeWheel(),
		workerPool: NewWorkerPool(workerCount),
		ctx:        ctx,
		cancel:     cancel,
		logger:     slog.Default(),
		running:    false,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("调度器已经在运行中")
	}
	s.running = true
	s.runningMu.Unlock()

	s.logger.Info("启动任务调度器")

	// 启动工作协程池
	s.workerPool.Start()

	// 启动时钟协程
	s.wg.Add(1)
	go s.tickLoop()

	s.logger.Info("任务调度器已启动")

	return nil
}

// tickLoop 时钟循环协程
func (s *Scheduler) tickLoop() {
	defer s.wg.Done()

	ticker := s.wheel.GetTicker()

	s.logger.Info("时钟协程启动")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("时钟协程退出")
			return

		case <-ticker.C:
			s.onTick()
		}
	}
}

// onTick 时钟触发处理
func (s *Scheduler) onTick() {
	// 推进时间轮,获取当前槽位的所有任务
	tasks := s.wheel.Tick()

	if len(tasks) == 0 {
		return
	}

	currentSlot := s.wheel.GetCurrentSlot()
	s.logger.Debug("时钟触发",
		"currentSlot", currentSlot,
		"taskCount", len(tasks))

	// 批量提交任务到工作池
	s.workerPool.SubmitBatch(tasks)
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = false
	s.runningMu.Unlock()

	s.logger.Info("停止任务调度器")

	// 发送取消信号
	s.cancel()

	// 等待时钟协程退出
	s.wg.Wait()

	// 停止时间轮
	s.wheel.Stop()

	// 停止工作协程池
	s.workerPool.Stop()

	s.logger.Info("任务调度器已停止")
}

// AddTask 添加任务
func (s *Scheduler) AddTask(task *Task) error {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()

	if !s.running {
		return fmt.Errorf("调度器未运行")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	if task.ID == "" {
		return fmt.Errorf("任务ID不能为空")
	}

	s.logger.Debug("添加任务",
		"taskID", task.ID,
		"target", task.Target,
		"delay", task.Delay)

	return s.wheel.AddTask(task)
}

// RemoveTask 删除任务
func (s *Scheduler) RemoveTask(taskID string, delay int) error {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()

	if !s.running {
		return fmt.Errorf("调度器未运行")
	}

	if taskID == "" {
		return fmt.Errorf("任务ID不能为空")
	}

	s.logger.Debug("删除任务",
		"taskID", taskID,
		"delay", delay)

	removed := s.wheel.RemoveTask(taskID, delay)
	if !removed {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	return nil
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()

	return s.running
}

// GetStats 获取调度器统计信息
func (s *Scheduler) GetStats() map[string]any {
	return map[string]any{
		"running":        s.IsRunning(),
		"currentSlot":    s.wheel.GetCurrentSlot(),
		"totalTaskCount": s.wheel.GetTotalTaskCount(),
		"workerCount":    s.workerPool.workerCount,
	}
}
