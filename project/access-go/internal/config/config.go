package config

import (
	"os"
	"time"

	sharedConfig "sudooom.im.shared/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	QUIC    QUICConfig    `yaml:"quic"`
	NATS    NATSConfig    `yaml:"nats"`
	Redis   RedisConfig   `yaml:"redis"`
	Auth    AuthConfig    `yaml:"auth"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Addr                   string        `yaml:"addr"`
	NodeID                 string        `yaml:"node_id"`
	MaxConnections         int           `yaml:"max_connections"`
	HeartbeatTimeout       time.Duration `yaml:"heartbeat_timeout"`        // 心跳超时时间，默认 90s
	HeartbeatCheckInterval time.Duration `yaml:"heartbeat_check_interval"` // 检测间隔，默认 30s
	WorkerPoolSize         int           `yaml:"worker_pool_size"`         // Worker Pool 大小，默认 1000
	WorkerQueueSize        int           `yaml:"worker_queue_size"`        // Worker 任务队列大小，默认 10000
}

type QUICConfig struct {
	MaxIdleTimeout        time.Duration `yaml:"max_idle_timeout"`
	KeepAlivePeriod       time.Duration `yaml:"keep_alive_period"`
	MaxIncomingStreams    int64         `yaml:"max_incoming_streams"`
	MaxIncomingUniStreams int64         `yaml:"max_incoming_uni_streams"`
	Allow0RTT             bool          `yaml:"allow_0rtt"`
	CertFile              string        `yaml:"cert_file"`
	KeyFile               string        `yaml:"key_file"`
}

type NATSConfig struct {
	URL           string        `yaml:"url"`
	MaxReconnects int           `yaml:"max_reconnects"`
	ReconnectWait time.Duration `yaml:"reconnect_wait"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type AuthConfig struct {
	TokenSecret string        `yaml:"token_secret"`
	TokenExpire time.Duration `yaml:"token_expire"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 从环境变量覆盖配置
	cfg.applyEnv()

	return &cfg, nil
}

// applyEnv 从环境变量覆盖配置
func (c *Config) applyEnv() {
	// NATS
	c.NATS.URL = sharedConfig.GetEnv("NATS_URL", c.NATS.URL)
	c.NATS.MaxReconnects = sharedConfig.GetEnvInt("NATS_MAX_RECONNECTS", c.NATS.MaxReconnects)
	c.NATS.ReconnectWait = sharedConfig.GetEnvDuration("NATS_RECONNECT_WAIT", c.NATS.ReconnectWait)

	// Redis
	c.Redis.Host = sharedConfig.GetEnv("REDIS_HOST", c.Redis.Host)
	c.Redis.Port = sharedConfig.GetEnvInt("REDIS_PORT", c.Redis.Port)
	c.Redis.Password = sharedConfig.GetEnv("REDIS_PASSWORD", c.Redis.Password)
	c.Redis.DB = sharedConfig.GetEnvInt("REDIS_DB", c.Redis.DB)
	c.Redis.PoolSize = sharedConfig.GetEnvInt("REDIS_POOL_SIZE", c.Redis.PoolSize)

	// 组装 Redis addr
	if c.Redis.Host != "" && c.Redis.Port > 0 {
		c.Redis.Addr = c.Redis.Host + ":" + sharedConfig.GetEnv("REDIS_PORT", "6379")
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = sharedConfig.GetEnv("REDIS_HOST", "localhost") + ":" + sharedConfig.GetEnv("REDIS_PORT", "6379")
	}

	// Auth/JWT
	c.Auth.TokenSecret = sharedConfig.GetEnv("JWT_SECRET", c.Auth.TokenSecret)

	// TLS
	c.QUIC.CertFile = sharedConfig.GetEnv("TLS_CERT_FILE", c.QUIC.CertFile)
	c.QUIC.KeyFile = sharedConfig.GetEnv("TLS_KEY_FILE", c.QUIC.KeyFile)

	// Logging
	c.Logging.Level = sharedConfig.GetEnv("LOG_LEVEL", c.Logging.Level)
	c.Logging.Format = sharedConfig.GetEnv("LOG_FORMAT", c.Logging.Format)
}
