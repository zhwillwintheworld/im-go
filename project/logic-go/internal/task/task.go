package task

import (
	"context"
	"time"
)

// TaskFunc 任务执行函数类型
type TaskFunc func(ctx context.Context, target string, metadata map[string]any) error

// Task 任务定义
type Task struct {
	ID        string         `json:"id"`        // 任务唯一ID
	Version   int64          `json:"version"`   // 版本号 (用于乐观锁控制)
	Target    string         `json:"target"`    // 操作对象标识
	Delay     int            `json:"delay"`     // 延迟秒数 (1-60)
	Fn        TaskFunc       `json:"-"`         // 执行函数
	Metadata  map[string]any `json:"metadata"`  // 元数据
	CreatedAt time.Time      `json:"createdAt"` // 创建时间
}

// NewTask 创建新任务
func NewTask(id, target string, delay int, fn TaskFunc) *Task {
	return &Task{
		ID:        id,
		Version:   1,
		Target:    target,
		Delay:     delay,
		Fn:        fn,
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
	}
}

// WithMetadata 添加元数据
func (t *Task) WithMetadata(key string, value any) *Task {
	t.Metadata[key] = value
	return t
}

// Execute 执行任务
func (t *Task) Execute(ctx context.Context) error {
	if t.Fn == nil {
		return nil
	}
	return t.Fn(ctx, t.Target, t.Metadata)
}
