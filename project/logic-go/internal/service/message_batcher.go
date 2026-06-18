package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.shared/proto"
	"sudooom.im.shared/snowflake"
)

// ErrMessageBatchQueueFull 表示消息批量写入队列已满，调用方应丢弃或降级处理。
var ErrMessageBatchQueueFull = errors.New("message batch queue full")

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
	OnPersisted func(serverMsgId int64)
	idemKey     string
}

type idempotentMessage struct {
	serverMsgId int64
	persisted   bool
	callbacks   []func(serverMsgId int64)
}

// MessageBatcher 消息批量写入器
type MessageBatcher struct {
	db               *pgxpool.Pool
	sf               *snowflake.Node
	config           MessageBatcherConfig
	msgChan          chan *MessageToSave
	logger           *slog.Logger
	wg               sync.WaitGroup
	stopChan         chan struct{}
	idemMu           sync.Mutex
	idempotentMsgIDs map[string]*idempotentMessage
	failedFlushCount atomic.Int64
	queueFullCount   atomic.Int64
	lastFlushNanos   atomic.Int64
	lastFlushSize    atomic.Int64
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
		db:               db,
		sf:               sf,
		config:           config,
		msgChan:          make(chan *MessageToSave, config.BatchSize*10),
		logger:           slog.Default(),
		stopChan:         make(chan struct{}),
		idempotentMsgIDs: make(map[string]*idempotentMessage),
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
	serverMsgId, _, err := b.SaveMessageWithCallback(msg, nil)
	return serverMsgId, err
}

// SaveMessageWithCallback 异步保存消息，入队成功后立即返回 serverMsgId。
// duplicate=true 表示命中了 fromUserId + clientMsgId 幂等映射，调用方应只回 ACK，不能重复路由。
func (b *MessageBatcher) SaveMessageWithCallback(msg *proto.UserMessage, onPersisted func(serverMsgId int64)) (int64, bool, error) {
	idemKey := messageIdempotencyKey(msg)
	if idemKey != "" {
		if serverMsgId, ok := b.addPersistedCallbackIfDuplicate(idemKey, onPersisted); ok {
			return serverMsgId, true, nil
		}
	}

	serverMsgId := b.sf.Generate().Int64()
	if idemKey != "" {
		b.setIdempotentMsgID(idemKey, serverMsgId, onPersisted)
	}

	msgToSave := &MessageToSave{
		ServerMsgId: serverMsgId,
		Msg:         msg,
		ResultChan:  make(chan error, 1),
		OnPersisted: onPersisted,
		idemKey:     idemKey,
	}

	select {
	case b.msgChan <- msgToSave:
		// 入队成功，立即返回（不等待数据库写入）
		return serverMsgId, false, nil
	default:
		// 队列满时立即返回，遵守消息处理非阻塞约束
		if idemKey != "" {
			b.deleteIdempotentMsgID(idemKey, serverMsgId)
		}
		b.queueFullCount.Add(1)
		b.logger.Warn("Message batch queue full, message dropped", "serverMsgId", serverMsgId)
		return 0, false, ErrMessageBatchQueueFull
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
		INSERT INTO messages (id, object_code, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, m := range batch {
		pgBatch.Queue(query,
			m.ServerMsgId,
			strconv.FormatInt(m.ServerMsgId, 10),
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
			b.failedFlushCount.Add(1)
			b.deleteIdempotentMsgID(batch[i].idemKey, batch[i].ServerMsgId)
			b.logger.Error("Failed to save message in batch",
				"serverMsgId", batch[i].ServerMsgId,
				"error", err,
			)
		} else if batch[i].idemKey != "" {
			b.notifyPersisted(batch[i].idemKey, batch[i].ServerMsgId)
		} else if batch[i].OnPersisted != nil {
			batch[i].OnPersisted(batch[i].ServerMsgId)
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
	b.lastFlushNanos.Store(elapsed.Nanoseconds())
	b.lastFlushSize.Store(int64(len(batch)))
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

// GetQueueCapacity 获取当前队列容量（用于监控）。
func (b *MessageBatcher) GetQueueCapacity() int {
	return cap(b.msgChan)
}

// GetFailedFlushCount 获取批写失败消息数，用于后续异步指标采集。
func (b *MessageBatcher) GetFailedFlushCount() int64 {
	return b.failedFlushCount.Load()
}

// MessageBatcherStats 是批写器的轻量观测快照。
type MessageBatcherStats struct {
	QueueSize          int
	QueueCapacity      int
	FailedFlushCount   int64
	QueueFullCount     int64
	LastFlushDuration  time.Duration
	LastFlushBatchSize int64
}

// Stats 返回批写器当前状态快照，不阻塞消息主链路。
func (b *MessageBatcher) Stats() MessageBatcherStats {
	return MessageBatcherStats{
		QueueSize:          b.GetQueueSize(),
		QueueCapacity:      b.GetQueueCapacity(),
		FailedFlushCount:   b.GetFailedFlushCount(),
		QueueFullCount:     b.queueFullCount.Load(),
		LastFlushDuration:  time.Duration(b.lastFlushNanos.Load()),
		LastFlushBatchSize: b.lastFlushSize.Load(),
	}
}

func messageIdempotencyKey(msg *proto.UserMessage) string {
	if msg == nil || msg.ClientMsgId == "" || msg.FromUserId == 0 {
		return ""
	}
	return strconv.FormatInt(msg.FromUserId, 10) + ":" + msg.ClientMsgId
}

func (b *MessageBatcher) getIdempotentMsgID(key string) (int64, bool) {
	b.idemMu.Lock()
	defer b.idemMu.Unlock()
	entry, ok := b.idempotentMsgIDs[key]
	if !ok {
		return 0, false
	}
	return entry.serverMsgId, true
}

func (b *MessageBatcher) addPersistedCallbackIfDuplicate(key string, callback func(serverMsgId int64)) (int64, bool) {
	b.idemMu.Lock()
	entry, ok := b.idempotentMsgIDs[key]
	if !ok {
		b.idemMu.Unlock()
		return 0, false
	}
	serverMsgId := entry.serverMsgId
	persisted := entry.persisted
	if callback != nil && !persisted {
		entry.callbacks = append(entry.callbacks, callback)
	}
	b.idemMu.Unlock()

	if callback != nil && persisted {
		go callback(serverMsgId)
	}
	return serverMsgId, true
}

func (b *MessageBatcher) setIdempotentMsgID(key string, serverMsgId int64, callback func(serverMsgId int64)) {
	b.idemMu.Lock()
	defer b.idemMu.Unlock()
	entry := &idempotentMessage{serverMsgId: serverMsgId}
	if callback != nil {
		entry.callbacks = append(entry.callbacks, callback)
	}
	b.idempotentMsgIDs[key] = entry
}

func (b *MessageBatcher) deleteIdempotentMsgID(key string, serverMsgId int64) {
	if key == "" {
		return
	}
	b.idemMu.Lock()
	defer b.idemMu.Unlock()
	if existing, ok := b.idempotentMsgIDs[key]; ok && existing.serverMsgId == serverMsgId {
		delete(b.idempotentMsgIDs, key)
	}
}

func (b *MessageBatcher) notifyPersisted(key string, serverMsgId int64) {
	var callbacks []func(serverMsgId int64)

	b.idemMu.Lock()
	if entry, ok := b.idempotentMsgIDs[key]; ok && entry.serverMsgId == serverMsgId {
		entry.persisted = true
		callbacks = append(callbacks, entry.callbacks...)
		entry.callbacks = nil
	}
	b.idemMu.Unlock()

	for _, callback := range callbacks {
		callback(serverMsgId)
	}
}
