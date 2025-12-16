package config

import (
	"os"
	"time"

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
	Addr           string `yaml:"addr"`
	NodeID         string `yaml:"node_id"`
	MaxConnections int    `yaml:"max_connections"`
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

	return &cfg, nil
}
