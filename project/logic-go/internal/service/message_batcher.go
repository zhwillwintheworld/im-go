package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.shared/proto"
	"sudooom.im.shared/snowflake"
)

// MessageBatcherConfig 批量写入配置
type MessageBatcherConfig struct {
	BatchSize     int           // 批量大小阈值
	FlushInterval time.Duration // 强制刷新间隔
}

// MessageToSave 待保存的消息
type MessageToSave struct {
	ServerMsgId int64
	Msg         *proto.UserMessage
	ResultChan  chan error // 用于通知保存结果
}

// MessageBatcher 消息批量写入器
type MessageBatcher struct {
	db       *pgxpool.Pool
	sf       *snowflake.Node
	config   MessageBatcherConfig
	msgChan  chan *MessageToSave
	logger   *slog.Logger
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// NewMessageBatcher 创建消息批量写入器
func NewMessageBatcher(db *pgxpool.Pool, sf *snowflake.Node, config MessageBatcherConfig) *MessageBatcher {
	// 设置默认值
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 10 * time.Second
	}

	return &MessageBatcher{
		db:       db,
		sf:       sf,
		config:   config,
		msgChan:  make(chan *MessageToSave, config.BatchSize*10),
		logger:   slog.Default(),
		stopChan: make(chan struct{}),
	}
}

// Start 启动批量写入器
func (b *MessageBatcher) Start(ctx context.Context) {
	b.wg.Add(1)
	go b.worker(ctx)
	b.logger.Info("MessageBatcher started",
		"batchSize", b.config.BatchSize,
		"flushInterval", b.config.FlushInterval,
	)
}

// Stop 停止批量写入器
func (b *MessageBatcher) Stop() {
	close(b.stopChan)
	b.wg.Wait()
	b.logger.Info("MessageBatcher stopped")
}

// SaveMessage 异步保存消息（立即返回 serverMsgId）
func (b *MessageBatcher) SaveMessage(msg *proto.UserMessage) (int64, error) {
	serverMsgId := b.sf.Generate().Int64()

	msgToSave := &MessageToSave{
		ServerMsgId: serverMsgId,
		Msg:         msg,
		ResultChan:  make(chan error, 1),
	}

	select {
	case b.msgChan <- msgToSave:
		// 入队成功，立即返回（不等待数据库写入）
		return serverMsgId, nil
	default:
		// 队列满，记录警告，同步等待
		b.logger.Warn("Message batch queue full, waiting...")
		b.msgChan <- msgToSave
		return serverMsgId, nil
	}
}

// SaveMessageSync 同步保存消息（等待写入完成）
func (b *MessageBatcher) SaveMessageSync(msg *proto.UserMessage) (int64, error) {
	serverMsgId := b.sf.Generate().Int64()

	msgToSave := &MessageToSave{
		ServerMsgId: serverMsgId,
		Msg:         msg,
		ResultChan:  make(chan error, 1),
	}

	b.msgChan <- msgToSave

	// 等待写入结果
	err := <-msgToSave.ResultChan
	return serverMsgId, err
}

// worker 后台工作协程
func (b *MessageBatcher) worker(ctx context.Context) {
	defer b.wg.Done()

	batch := make([]*MessageToSave, 0, b.config.BatchSize)
	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，刷入剩余消息
			if len(batch) > 0 {
				b.flush(ctx, batch)
			}
			return
		case <-b.stopChan:
			// 停止信号，刷入剩余消息
			if len(batch) > 0 {
				b.flush(context.Background(), batch)
			}
			return
		case msg := <-b.msgChan:
			batch = append(batch, msg)
			// 达到批量大小阈值，立即刷入
			if len(batch) >= b.config.BatchSize {
				b.flush(ctx, batch)
				batch = make([]*MessageToSave, 0, b.config.BatchSize)
			}
		case <-ticker.C:
			// 定时刷入（即使未满也写入）
			if len(batch) > 0 {
				b.flush(ctx, batch)
				batch = make([]*MessageToSave, 0, b.config.BatchSize)
			}
		}
	}
}

// flush 批量写入数据库
func (b *MessageBatcher) flush(ctx context.Context, batch []*MessageToSave) {
	if len(batch) == 0 {
		return
	}

	startTime := time.Now()

	// 使用 pgx.Batch 批量插入
	pgBatch := &pgx.Batch{}
	query := `
		INSERT INTO messages (id, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	for _, m := range batch {
		pgBatch.Queue(query,
			m.ServerMsgId,
			m.Msg.ClientMsgId,
			m.Msg.FromUserId,
			m.Msg.ToUserId,
			m.Msg.ToGroupId,
			m.Msg.MsgType,
			m.Msg.Content,
			0, // status: 未读
		)
	}

	// 执行批量操作
	br := b.db.SendBatch(ctx, pgBatch)
	defer func(br pgx.BatchResults) {
		err := br.Close()
		if err != nil {
			logger.Error("Failed to close batch results", "error", err)
		}
	}(br)

	// 收集结果
	var batchErr error
	for i := 0; i < len(batch); i++ {
		_, err := br.Exec()
		if err != nil {
			batchErr = err
			b.logger.Error("Failed to save message in batch",
				"serverMsgId", batch[i].ServerMsgId,
				"error", err,
			)
		}
		// 通知等待的调用者
		if batch[i].ResultChan != nil {
			select {
			case batch[i].ResultChan <- err:
			default:
			}
		}
	}

	elapsed := time.Since(startTime)
	if batchErr != nil {
		b.logger.Error("Batch flush completed with errors",
			"count", len(batch),
			"elapsed", elapsed,
		)
	} else {
		b.logger.Debug("Batch flush completed",
			"count", len(batch),
			"elapsed", elapsed,
			"avgPerMsg", elapsed/time.Duration(len(batch)),
		)
	}
}

// GetQueueSize 获取当前队列大小（用于监控）
func (b *MessageBatcher) GetQueueSize() int {
	return len(b.msgChan)
}
