package config

import "time"

// NATSConfig NATS 配置
type NATSConfig struct {
	URL           string        `yaml:"url" mapstructure:"url"`
	MaxReconnects int           `yaml:"max_reconnects" mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `yaml:"reconnect_wait" mapstructure:"reconnect_wait"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
	PoolSize int    `yaml:"pool_size" mapstructure:"pool_size"`
}

// GetAddr 获取 Redis 地址
func (c *RedisConfig) GetAddr() string {
	if c.Addr != "" {
		return c.Addr
	}
	if c.Host != "" && c.Port > 0 {
		return c.Host + ":" + string(rune(c.Port))
	}
	return "localhost:6379"
}
