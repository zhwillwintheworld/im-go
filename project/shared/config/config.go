package config

import (
	"os"
	"strconv"
	"time"
)

// GetEnv 获取环境变量，如果不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取环境变量并转换为 int，如果不存在或转换失败则返回默认值
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvInt64 获取环境变量并转换为 int64
func GetEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvBool 获取环境变量并转换为 bool
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetEnvDuration 获取环境变量并转换为 time.Duration
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

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

// DatabaseConfig 数据库公共配置
type DatabaseConfig struct {
	Host            string        `yaml:"host" mapstructure:"host"`
	Port            int           `yaml:"port" mapstructure:"port"`
	User            string        `yaml:"user" mapstructure:"user"`
	Password        string        `yaml:"password" mapstructure:"password"`
	Name            string        `yaml:"name" mapstructure:"name"`
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
}

// ApplyEnv 从环境变量覆盖配置
func (c *DatabaseConfig) ApplyEnv() {
	c.Host = GetEnv("POSTGRES_HOST", c.Host)
	c.Port = GetEnvInt("POSTGRES_PORT", c.Port)
	c.User = GetEnv("POSTGRES_USER", c.User)
	c.Password = GetEnv("POSTGRES_PASSWORD", c.Password)
	c.Name = GetEnv("POSTGRES_DB", c.Name)
	c.MaxOpenConns = GetEnvInt("POSTGRES_MAX_OPEN_CONNS", c.MaxOpenConns)
	c.MaxIdleConns = GetEnvInt("POSTGRES_MAX_IDLE_CONNS", c.MaxIdleConns)
}

// ApplyEnv 从环境变量覆盖 Redis 配置
func (c *RedisConfig) ApplyEnv() {
	c.Host = GetEnv("REDIS_HOST", c.Host)
	c.Port = GetEnvInt("REDIS_PORT", c.Port)
	c.Password = GetEnv("REDIS_PASSWORD", c.Password)
	c.DB = GetEnvInt("REDIS_DB", c.DB)
	c.PoolSize = GetEnvInt("REDIS_POOL_SIZE", c.PoolSize)

	// 如果有 host:port，组装 addr
	if c.Host != "" && c.Port > 0 && c.Addr == "" {
		c.Addr = c.Host + ":" + strconv.Itoa(c.Port)
	}
}

// ApplyEnv 从环境变量覆盖 NATS 配置
func (c *NATSConfig) ApplyEnv() {
	c.URL = GetEnv("NATS_URL", c.URL)
	c.MaxReconnects = GetEnvInt("NATS_MAX_RECONNECTS", c.MaxReconnects)
	c.ReconnectWait = GetEnvDuration("NATS_RECONNECT_WAIT", c.ReconnectWait)
}

// JWTConfig JWT 公共配置
type JWTConfig struct {
	SecretKey     string        `yaml:"secret_key" mapstructure:"secret_key"`
	AccessExpire  time.Duration `yaml:"access_expire" mapstructure:"access_expire"`
	RefreshExpire time.Duration `yaml:"refresh_expire" mapstructure:"refresh_expire"`
}

// ApplyEnv 从环境变量覆盖 JWT 配置
func (c *JWTConfig) ApplyEnv() {
	c.SecretKey = GetEnv("JWT_SECRET", c.SecretKey)
	c.AccessExpire = GetEnvDuration("JWT_ACCESS_EXPIRE", c.AccessExpire)
	c.RefreshExpire = GetEnvDuration("JWT_REFRESH_EXPIRE", c.RefreshExpire)
}
