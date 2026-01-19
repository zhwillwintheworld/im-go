package task

import (
	"context"
	"log/slog"
	"sync"
)

// WorkerPool 工作协程池
type WorkerPool struct {
	workerCount int        // 工作协程数量
	taskChan    chan *Task // 任务通道
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      *slog.Logger
}

// NewWorkerPool 创建工作协程池
func NewWorkerPool(workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 10 // 默认10个工作协程
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount: workerCount,
		taskChan:    make(chan *Task, workerCount*2), // buffered channel
		ctx:         ctx,
		cancel:      cancel,
		logger:      slog.Default(),
	}
}

// Start 启动工作协程池
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	wp.logger.Info("工作协程池已启动", "workerCount", wp.workerCount)
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("工作协程启动", "workerID", id)

	for {
		select {
		case <-wp.ctx.Done():
			wp.logger.Debug("工作协程退出", "workerID", id)
			return

		case task := <-wp.taskChan:
			if task == nil {
				continue
			}

			wp.executeTask(id, task)
		}
	}
}

// executeTask 执行任务
func (wp *WorkerPool) executeTask(workerID int, task *Task) {
	defer func() {
		if r := recover(); r != nil {
			wp.logger.Error("任务执行 panic",
				"workerID", workerID,
				"taskID", task.ID,
				"target", task.Target,
				"panic", r)
		}
	}()

	wp.logger.Debug("执行任务",
		"workerID", workerID,
		"taskID", task.ID,
		"target", task.Target,
		"version", task.Version)

	if err := task.Execute(wp.ctx); err != nil {
		wp.logger.Error("任务执行失败",
			"workerID", workerID,
			"taskID", task.ID,
			"target", task.Target,
			"error", err)
	} else {
		wp.logger.Debug("任务执行成功",
			"workerID", workerID,
			"taskID", task.ID,
			"target", task.Target)
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task *Task) {
	select {
	case wp.taskChan <- task:
		// 任务已提交
	case <-wp.ctx.Done():
		// 工作池已关闭
		wp.logger.Warn("工作池已关闭,任务提交失败", "taskID", task.ID)
	default:
		// 通道已满,记录警告
		wp.logger.Warn("任务通道已满,任务可能延迟执行", "taskID", task.ID)
		// 阻塞等待
		select {
		case wp.taskChan <- task:
		case <-wp.ctx.Done():
		}
	}
}

// SubmitBatch 批量提交任务
func (wp *WorkerPool) SubmitBatch(tasks []*Task) {
	for _, task := range tasks {
		wp.Submit(task)
	}
}

// Stop 停止工作协程池
func (wp *WorkerPool) Stop() {
	wp.logger.Info("停止工作协程池")

	// 发送取消信号
	wp.cancel()

	// 等待所有工作协程退出
	wp.wg.Wait()

	// 关闭任务通道
	close(wp.taskChan)

	wp.logger.Info("工作协程池已停止")
}
