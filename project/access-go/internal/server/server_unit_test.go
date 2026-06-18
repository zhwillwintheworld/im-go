package server

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"sudooom.im.access/internal/config"
)

func TestHasConnectionCapacity(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		current int
		want    bool
	}{
		{name: "unlimited when max is zero", max: 0, current: 100, want: true},
		{name: "below limit", max: 2, current: 1, want: true},
		{name: "at limit", max: 2, current: 2, want: false},
		{name: "over limit", max: 2, current: 3, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasConnectionCapacity(tt.max, tt.current)
			if got != tt.want {
				t.Fatalf("hasConnectionCapacity(%d, %d) = %v，期望 %v", tt.max, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheckOriginAllowsEmptyOrigin(t *testing.T) {
	srv := newOriginTestServer([]string{"https://app.example"})
	req := httptest.NewRequest("GET", "/webtransport", nil)

	if !srv.checkOrigin(req) {
		t.Fatal("无 Origin 的非浏览器请求应允许，避免影响本地探测和集成测试")
	}
}

func TestCheckOriginAllowsConfiguredOrigin(t *testing.T) {
	srv := newOriginTestServer([]string{"https://app.example"})
	req := httptest.NewRequest("GET", "/webtransport", nil)
	req.Header.Set("Origin", "https://app.example")

	if !srv.checkOrigin(req) {
		t.Fatal("配置白名单内的 Origin 应允许")
	}
}

func TestCheckOriginRejectsUnknownOrigin(t *testing.T) {
	srv := newOriginTestServer([]string{"https://app.example"})
	req := httptest.NewRequest("GET", "/webtransport", nil)
	req.Header.Set("Origin", "https://evil.example")

	if srv.checkOrigin(req) {
		t.Fatal("未配置的 Origin 应拒绝")
	}
}

func TestCheckOriginAllowsAllWhenWildcardConfigured(t *testing.T) {
	srv := newOriginTestServer([]string{"*"})
	req := httptest.NewRequest("GET", "/webtransport", nil)
	req.Header.Set("Origin", "https://any.example")

	if !srv.checkOrigin(req) {
		t.Fatal("allowed_origins 配置 * 时应允许所有 Origin")
	}
}

func newOriginTestServer(allowedOrigins []string) *Server {
	return &Server{
		cfg: &config.Config{
			Server: config.ServerConfig{
				AllowedOrigins: allowedOrigins,
			},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}
