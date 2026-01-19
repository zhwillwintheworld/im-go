package task

import "sync"

// Slot 时间轮槽位
type Slot struct {
	mu    sync.Mutex       // 槽内互斥锁
	tasks map[string]*Task // 任务映射 (key: taskID)
}

// NewSlot 创建新槽位
func NewSlot() *Slot {
	return &Slot{
		tasks: make(map[string]*Task),
	}
}

// AddTask 添加任务到槽位
func (s *Slot) AddTask(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
}

// RemoveTask 从槽位删除任务
func (s *Slot) RemoveTask(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[taskID]; exists {
		delete(s.tasks, taskID)
		return true
	}
	return false
}

// GetAndClear 获取所有任务并清空槽位
func (s *Slot) GetAndClear() []*Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.tasks) == 0 {
		return nil
	}

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	// 清空槽位
	s.tasks = make(map[string]*Task)

	return tasks
}

// Count 获取槽位任务数量
func (s *Slot) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.tasks)
}
