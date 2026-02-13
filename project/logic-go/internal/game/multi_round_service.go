package game

import (
	"context"
	"fmt"
	"log/slog"

	"sudooom.im.logic/internal/game/mahjong"
	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/mahjong/htmajong"
	"sudooom.im.logic/internal/game/mahjong/thmahjong"
)

// MultiRoundGameService 多轮游戏服务（管理 MultiRoundGame 和 Round 的生命周期）
type MultiRoundGameService struct {
	gameManager *MultiRoundGameManager
	logger      *slog.Logger
}

// NewMultiRoundGameService 创建多轮游戏服务
func NewMultiRoundGameService(gameManager *MultiRoundGameManager) *MultiRoundGameService {
	return &MultiRoundGameService{
		gameManager: gameManager,
		logger:      slog.Default().With("component", "MultiRoundGameService"),
	}
}

// CreateGame 创建游戏
func (s *MultiRoundGameService) CreateGame(ctx context.Context, roomID string, gameType string, config GameConfig) (*MultiRoundGame, error) {
	game, err := s.gameManager.CreateGame(roomID, gameType, config)
	if err != nil {
		s.logger.Error("Failed to create game", "error", err, "roomId", roomID)
		return nil, err
	}

	s.logger.Info("Game created", "roomId", roomID, "gameId", game.ID)
	return game, nil
}

// StartNewRound 开始新的一局
func (s *MultiRoundGameService) StartNewRound(ctx context.Context, game *MultiRoundGame, playerIDs []int64) (*Round, error) {
	// 创建引擎工厂
	engineFactory := func() RoundEngine {
		return s.createEngineForGameType(game.GameType)
	}

	round, err := game.StartNewRound(ctx, playerIDs, engineFactory)
	if err != nil {
		s.logger.Error("Failed to start new round", "error", err, "gameId", game.ID)
		return nil, err
	}

	s.logger.Info("Round started",
		"gameId", game.ID,
		"roundId", round.ID,
		"roundNumber", round.RoundNumber)

	return round, nil
}

// FinishRound 完成当前局
func (s *MultiRoundGameService) FinishRound(ctx context.Context, game *MultiRoundGame) error {
	if err := game.FinishCurrentRound(ctx); err != nil {
		s.logger.Error("Failed to finish round", "error", err, "gameId", game.ID)
		return err
	}

	s.logger.Info("Round finished", "gameId", game.ID)

	// 检查游戏是否结束
	if game.GetStatus() == MultiRoundGameStatusFinished {
		s.logger.Info("Game finished", "gameId", game.ID, "finalScores", game.GetPlayerScores())
		// TODO: 触发游戏结束事件
	}

	return nil
}

// GetGame 获取游戏
func (s *MultiRoundGameService) GetGame(roomID string) (*MultiRoundGame, error) {
	game, ok := s.gameManager.Get(roomID)
	if !ok {
		return nil, ErrGameNotFound
	}
	return game, nil
}

// RemoveGame 移除游戏
func (s *MultiRoundGameService) RemoveGame(roomID string) {
	s.gameManager.Remove(roomID)
	s.logger.Info("Game removed", "roomId", roomID)
}

// createEngineForGameType 根据游戏类型创建引擎
func (s *MultiRoundGameService) createEngineForGameType(gameType string) RoundEngine {
	switch gameType {
	case "HT_MAHJONG":
		// 会同麻将
		coreEngine := htmajong.NewEngine()
		return mahjong.NewSafeMahjongEngine(coreEngine, gameType)
	case "TH_MAHJONG":
		// 太湖麻将
		coreEngine := thmahjong.NewEngine()
		return mahjong.NewSafeMahjongEngine(coreEngine, gameType)
	default:
		// 默认使用会同麻将
		s.logger.Warn("Unknown game type, using default", "gameType", gameType)
		coreEngine := htmajong.NewEngine()
		return mahjong.NewSafeMahjongEngine(coreEngine, "HT_MAHJONG")
	}
}

// HandlePlayerAction 处理玩家操作
func (s *MultiRoundGameService) HandlePlayerAction(ctx context.Context, roomID string, userID int64, action core.Action) error {
	// 获取游戏
	game, err := s.GetGame(roomID)
	if err != nil {
		return err
	}

	// 获取当前 Round
	round := game.GetCurrentRound()
	if round == nil {
		return fmt.Errorf("no current round")
	}

	// 处理动作
	if err := round.HandlePlayerAction(ctx, userID, action); err != nil {
		s.logger.Error("Failed to handle player action",
			"error", err,
			"roomId", roomID,
			"userId", userID,
			"action", action.Type)
		return err
	}

	// 检查 Round 是否结束
	if round.GetStatus() == RoundStatusSettling {
		// Round 结束，进行结算
		if err := s.FinishRound(ctx, game); err != nil {
			return err
		}
	}

	return nil
}
