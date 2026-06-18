package config

import (
	"time"

	sharedConfig "sudooom.im.shared/config"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Batch    BatchConfig    `mapstructure:"batch"`
	Room     RoomConfig     `mapstructure:"room"`
	Fanout   FanoutConfig   `mapstructure:"fanout"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log_level"`
}

type NATSConfig struct {
	URL             string        `mapstructure:"url"`
	MaxReconnects   int           `mapstructure:"max_reconnects"`
	ReconnectWait   time.Duration `mapstructure:"reconnect_wait"`
	WorkerCount     int           `mapstructure:"worker_count"`      // Worker Pool 并发数
	BufferSize      int           `mapstructure:"buffer_size"`       // 消息缓冲区大小
	RoomWorkerCount int           `mapstructure:"room_worker_count"` // room/game Worker Pool 并发数
	RoomBufferSize  int           `mapstructure:"room_buffer_size"`  // room/game 消息缓冲区大小
	RoomShardCount  int           `mapstructure:"room_shard_count"`  // room/game shard 总数
	RoomShardIndex  int           `mapstructure:"room_shard_index"`  // 当前 Logic 节点负责的 shard
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type BatchConfig struct {
	Size          int           `mapstructure:"size"`           // 批量大小阈值
	FlushInterval time.Duration `mapstructure:"flush_interval"` // 强制刷新间隔
}

type RoomConfig struct {
	MaxRooms           int           `mapstructure:"max_rooms"`            // 最大房间数
	EvictCheckInterval time.Duration `mapstructure:"evict_check_interval"` // 房间清理检查间隔
	EvictTimeout       time.Duration `mapstructure:"evict_timeout"`        // 房间无响应超时时间
}

type FanoutConfig struct {
	WorkerCount              int           `mapstructure:"worker_count"`               // 大群 fan-out worker 数
	BufferSize               int           `mapstructure:"buffer_size"`                // 大群 fan-out 队列大小
	LargeGroupThreshold      int           `mapstructure:"large_group_threshold"`      // 超过该成员数走异步 fan-out
	LocationQueryConcurrency int           `mapstructure:"location_query_concurrency"` // 位置查询并发上限
	LocationBatchSize        int           `mapstructure:"location_batch_size"`        // 单批位置查询用户数
	DispatchConcurrency      int           `mapstructure:"dispatch_concurrency"`       // 下行分发并发上限
	GroupMemberCacheTTL      time.Duration `mapstructure:"group_member_cache_ttl"`     // 群成员短 TTL 缓存
}

// Load 从指定路径加载配置
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 从环境变量覆盖配置
	cfg.applyEnv()

	return &cfg, nil
}

// applyEnv 从环境变量覆盖配置
func (c *Config) applyEnv() {
	// App
	c.App.LogLevel = sharedConfig.GetEnv("LOG_LEVEL", c.App.LogLevel)

	// NATS
	c.NATS.URL = sharedConfig.GetEnv("NATS_URL", c.NATS.URL)
	c.NATS.MaxReconnects = sharedConfig.GetEnvInt("NATS_MAX_RECONNECTS", c.NATS.MaxReconnects)
	c.NATS.ReconnectWait = sharedConfig.GetEnvDuration("NATS_RECONNECT_WAIT", c.NATS.ReconnectWait)
	c.NATS.WorkerCount = sharedConfig.GetEnvInt("NATS_WORKER_COUNT", c.NATS.WorkerCount)
	c.NATS.BufferSize = sharedConfig.GetEnvInt("NATS_BUFFER_SIZE", c.NATS.BufferSize)
	c.NATS.RoomWorkerCount = sharedConfig.GetEnvInt("NATS_ROOM_WORKER_COUNT", c.NATS.RoomWorkerCount)
	c.NATS.RoomBufferSize = sharedConfig.GetEnvInt("NATS_ROOM_BUFFER_SIZE", c.NATS.RoomBufferSize)
	c.NATS.RoomShardCount = sharedConfig.GetEnvInt("ROOM_SHARD_COUNT", c.NATS.RoomShardCount)
	c.NATS.RoomShardIndex = sharedConfig.GetEnvInt("ROOM_SHARD_INDEX", c.NATS.RoomShardIndex)

	// Database
	c.Database.Host = sharedConfig.GetEnv("POSTGRES_HOST", c.Database.Host)
	c.Database.Port = sharedConfig.GetEnvInt("POSTGRES_PORT", c.Database.Port)
	c.Database.User = sharedConfig.GetEnv("POSTGRES_USER", c.Database.User)
	c.Database.Password = sharedConfig.GetEnv("POSTGRES_PASSWORD", c.Database.Password)
	c.Database.Name = sharedConfig.GetEnv("POSTGRES_DB", c.Database.Name)
	c.Database.MaxOpenConns = sharedConfig.GetEnvInt("POSTGRES_MAX_OPEN_CONNS", c.Database.MaxOpenConns)
	c.Database.MaxIdleConns = sharedConfig.GetEnvInt("POSTGRES_MAX_IDLE_CONNS", c.Database.MaxIdleConns)

	// Redis
	c.Redis.Host = sharedConfig.GetEnv("REDIS_HOST", c.Redis.Host)
	c.Redis.Port = sharedConfig.GetEnvInt("REDIS_PORT", c.Redis.Port)
	c.Redis.Password = sharedConfig.GetEnv("REDIS_PASSWORD", c.Redis.Password)
	c.Redis.DB = sharedConfig.GetEnvInt("REDIS_DB", c.Redis.DB)
	c.Redis.PoolSize = sharedConfig.GetEnvInt("REDIS_POOL_SIZE", c.Redis.PoolSize)

	// Batch
	c.Batch.Size = sharedConfig.GetEnvInt("BATCH_SIZE", c.Batch.Size)
	c.Batch.FlushInterval = sharedConfig.GetEnvDuration("BATCH_FLUSH_INTERVAL", c.Batch.FlushInterval)

	// Room
	c.Room.MaxRooms = sharedConfig.GetEnvInt("ROOM_MAX_ROOMS", c.Room.MaxRooms)
	c.Room.EvictCheckInterval = sharedConfig.GetEnvDuration("ROOM_EVICT_CHECK_INTERVAL", c.Room.EvictCheckInterval)
	c.Room.EvictTimeout = sharedConfig.GetEnvDuration("ROOM_EVICT_TIMEOUT", c.Room.EvictTimeout)

	// Fanout
	c.Fanout.WorkerCount = sharedConfig.GetEnvInt("FANOUT_WORKER_COUNT", c.Fanout.WorkerCount)
	c.Fanout.BufferSize = sharedConfig.GetEnvInt("FANOUT_BUFFER_SIZE", c.Fanout.BufferSize)
	c.Fanout.LargeGroupThreshold = sharedConfig.GetEnvInt("FANOUT_LARGE_GROUP_THRESHOLD", c.Fanout.LargeGroupThreshold)
	c.Fanout.LocationQueryConcurrency = sharedConfig.GetEnvInt("FANOUT_LOCATION_QUERY_CONCURRENCY", c.Fanout.LocationQueryConcurrency)
	c.Fanout.LocationBatchSize = sharedConfig.GetEnvInt("FANOUT_LOCATION_BATCH_SIZE", c.Fanout.LocationBatchSize)
	c.Fanout.DispatchConcurrency = sharedConfig.GetEnvInt("FANOUT_DISPATCH_CONCURRENCY", c.Fanout.DispatchConcurrency)
	c.Fanout.GroupMemberCacheTTL = sharedConfig.GetEnvDuration("GROUP_MEMBER_CACHE_TTL", c.Fanout.GroupMemberCacheTTL)
}
