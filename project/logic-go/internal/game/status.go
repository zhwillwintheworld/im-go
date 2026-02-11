package game

import "fmt"

// GameStatus 游戏状态
type GameStatus int8

const (
	GameStatusWaiting  GameStatus = iota // 等待（引擎已创建未初始化）
	GameStatusPlaying                    // 游戏进行中
	GameStatusFinished                   // 游戏已结束
)

// String 返回游戏状态字符串
func (s GameStatus) String() string {
	switch s {
	case GameStatusWaiting:
		return "waiting"
	case GameStatusPlaying:
		return "playing"
	case GameStatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// validTransitions 合法状态转换表
var validTransitions = map[GameStatus][]GameStatus{
	GameStatusWaiting:  {GameStatusPlaying, GameStatusFinished},
	GameStatusPlaying:  {GameStatusFinished},
	GameStatusFinished: {},
}

// CanTransitionTo 检查是否可以转换到目标状态
func (s GameStatus) CanTransitionTo(target GameStatus) bool {
	allowed := validTransitions[s]
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

// ValidateTransition 验证状态转换，不合法时返回错误
func (s GameStatus) ValidateTransition(target GameStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition from %s to %s",
			ErrInvalidGameState, s, target)
	}
	return nil
}
