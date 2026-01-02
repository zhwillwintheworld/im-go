package workerpool

import (
	"context"
	"log/slog"
	"sync"
)

// Task 定义任务函数类型
type Task func()

// Pool Worker Pool 实现
type Pool struct {
	workers   int
	taskQueue chan Task
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *slog.Logger
}

// New 创建一个新的 Worker Pool
// workers: worker 数量
// queueSize: 任务队列大小
func New(workers int, queueSize int, logger *slog.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
	}

	// 启动 workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	pool.logger.Info("Worker pool started",
		"workers", workers,
		"queue_size", queueSize)

	return pool
}

// worker 工作协程
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Worker pool shutting down
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				// Task queue closed
				return
			}

			// 执行任务，捕获 panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						p.logger.Error("Task panic recovered",
							"worker_id", id,
							"panic", r)
					}
				}()
				task()
			}()
		}
	}
}

// Submit 提交任务到 Worker Pool
// 如果队列满了，会阻塞直到有空位或 context 被取消
func (p *Pool) Submit(task Task) bool {
	select {
	case <-p.ctx.Done():
		return false
	case p.taskQueue <- task:
		return true
	}
}

// TrySubmit 尝试提交任务，如果队列满了立即返回 false
func (p *Pool) TrySubmit(task Task) bool {
	select {
	case <-p.ctx.Done():
		return false
	case p.taskQueue <- task:
		return true
	default:
		// 队列满了
		return false
	}
}

// Shutdown 优雅关闭 Worker Pool
// 等待所有任务完成
func (p *Pool) Shutdown() {
	p.cancel()
	close(p.taskQueue)
	p.wg.Wait()
	p.logger.Info("Worker pool shutdown completed")
}
