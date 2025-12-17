package config

import (
	"time"

	sharedConfig "sudooom.im.shared/config"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type JWTConfig struct {
	SecretKey     string        `mapstructure:"secret_key"`
	AccessExpire  time.Duration `mapstructure:"access_expire"`
	RefreshExpire time.Duration `mapstructure:"refresh_expire"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
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

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
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
	c.App.Port = sharedConfig.GetEnvInt("WEB_PORT", c.App.Port)

	// JWT
	c.JWT.SecretKey = sharedConfig.GetEnv("JWT_SECRET", c.JWT.SecretKey)
	c.JWT.AccessExpire = sharedConfig.GetEnvDuration("JWT_ACCESS_EXPIRE", c.JWT.AccessExpire)
	c.JWT.RefreshExpire = sharedConfig.GetEnvDuration("JWT_REFRESH_EXPIRE", c.JWT.RefreshExpire)

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
}
