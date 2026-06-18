package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAppliesMaxFrameSizeDefault(t *testing.T) {
	configPath := writeTestConfig(t, `
server:
  addr: ":8081"
quic: {}
nats: {}
redis: {}
auth: {}
logging: {}
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Server.MaxFrameSize != DefaultMaxFrameSize {
		t.Fatalf("未配置 max_frame_size 时应使用默认值，期望 %d，实际 %d", DefaultMaxFrameSize, cfg.Server.MaxFrameSize)
	}
}

func TestLoadAppliesAccessAllowedOriginsEnv(t *testing.T) {
	t.Setenv("ACCESS_ALLOWED_ORIGINS", "https://a.example, https://b.example ,,")

	configPath := writeTestConfig(t, `
server:
  addr: ":8081"
  allowed_origins:
    - "https://from-file.example"
quic: {}
nats: {}
redis: {}
auth: {}
logging: {}
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	expected := []string{"https://a.example", "https://b.example"}
	if !reflect.DeepEqual(cfg.Server.AllowedOrigins, expected) {
		t.Fatalf("ACCESS_ALLOWED_ORIGINS 应覆盖文件配置，期望 %#v，实际 %#v", expected, cfg.Server.AllowedOrigins)
	}
}

func TestLoadAppliesAccessMaxFrameSizeEnv(t *testing.T) {
	t.Setenv("ACCESS_MAX_FRAME_SIZE", "2048")

	configPath := writeTestConfig(t, `
server:
  addr: ":8081"
  max_frame_size: 1024
quic: {}
nats: {}
redis: {}
auth: {}
logging: {}
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Server.MaxFrameSize != 2048 {
		t.Fatalf("ACCESS_MAX_FRAME_SIZE 应覆盖文件配置，实际 %d", cfg.Server.MaxFrameSize)
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	return path
}
