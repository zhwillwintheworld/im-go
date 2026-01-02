package server

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
	goredis "github.com/redis/go-redis/v9"
	"sudooom.im.access/internal/config"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/redis"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
)

const (
	frameHeaderSize = 5
	frameTypeAuth   = byte(1)
)

// TestWebTransportAuth 测试 WebTransport 认证流程
func TestWebTransportAuth(t *testing.T) {
	// 1. 启动测试服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建测试配置
	cfg := createTestConfig()

	// 创建测试依赖
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 创建 Redis 客户端（需要 Redis 服务运行）
	redisClient := redis.NewClient(cfg.Redis, cfg.Server.NodeID)

	// 创建 NATS 客户端（需要 NATS 服务运行）
	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		t.Fatalf("创建 NATS 客户端失败: %v", err)
	}
	defer natsClient.Close()

	// 创建并启动服务器
	server := New(cfg, natsClient, redisClient, logger)

	// 在 goroutine 中启动服务器
	serverErr := make(chan error, 1)
	go func() {
		err := server.Start(ctx)
		if err != nil {
			serverErr <- err
		}
	}()

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 2. 创建测试用户的 token
	testUserID := int64(12345)
	testDeviceID := "test-device-001"
	testPlatform := "web"
	testToken := "test-token-12345"

	// 在 Redis 中设置测试用户信息（模拟 web-go 登录时设置的数据）
	err = setTestUserToken(ctx, redisClient, testUserID, testPlatform, testToken, testDeviceID)
	if err != nil {
		t.Fatalf("设置测试用户 token 失败: %v", err)
	}

	// 3. 创建 WebTransport 客户端并连接
	url := "https://" + cfg.Server.Addr + "/webtransport"
	dialer := createWebTransportDialer(t)

	// 4. 建立 WebTransport 连接
	resp, session, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("建立 WebTransport 连接失败: %v", err)
	}
	defer session.CloseWithError(0, "test completed")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("WebTransport 握手失败，状态码: %d", resp.StatusCode)
	}

	t.Logf("WebTransport 连接建立成功")

	// 5. 创建认证请求
	authReq := buildAuthRequest(testToken, testDeviceID, im_protocol.PlatformWEB)

	// 6. 打开双向流并发送认证请求
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("打开双向流失败: %v", err)
	}
	defer stream.Close()

	// 发送认证帧
	err = sendAuthFrame(stream, authReq)
	if err != nil {
		t.Fatalf("发送认证帧失败: %v", err)
	}

	t.Logf("认证请求已发送")

	// 7. 读取认证响应
	response, err := readResponse(stream)
	if err != nil {
		t.Fatalf("读取认证响应失败: %v", err)
	}

	// 8. 验证响应
	if response.Code() != im_protocol.ErrorCodeSUCCESS {
		t.Fatalf("认证失败，错误码: %s, 消息: %s",
			response.Code().String(),
			string(response.Msg()))
	}

	t.Logf("认证成功！错误码: %s", response.Code().String())

	// 9. 验证用户位置已在 Redis 中注册
	time.Sleep(100 * time.Millisecond) // 等待异步操作完成
	location, err := redisClient.GetUserLocation(ctx, testUserID, testPlatform)
	if err != nil {
		t.Fatalf("获取用户位置失败: %v", err)
	}
	if location == "" {
		t.Fatalf("用户位置未在 Redis 中注册")
	}

	t.Logf("用户位置已注册: %s", location)

	// 清理：移除测试数据
	cleanupTestData(ctx, t, redisClient, testUserID, testPlatform, testToken)
}

// TestWebTransportAuthFail 测试认证失败场景
func TestWebTransportAuthFail(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("跳过集成测试，设置 INTEGRATION_TEST=1 来运行")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := createTestConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	redisClient := redis.NewClient(cfg.Redis, cfg.Server.NodeID)
	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		t.Fatalf("创建 NATS 客户端失败: %v", err)
	}
	defer natsClient.Close()

	server := New(cfg, natsClient, redisClient, logger)
	go func() {
		server.Start(ctx)
	}()
	time.Sleep(2 * time.Second)

	url := "https://" + cfg.Server.Addr + "/webtransport"
	dialer := createWebTransportDialer(t)
	_, session, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("建立 WebTransport 连接失败: %v", err)
	}
	defer session.CloseWithError(0, "test completed")

	// 使用无效的 token
	invalidToken := "invalid-token-xyz"
	authReq := buildAuthRequest(invalidToken, "device-001", im_protocol.PlatformWEB)

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("打开双向流失败: %v", err)
	}
	defer stream.Close()

	err = sendAuthFrame(stream, authReq)
	if err != nil {
		t.Fatalf("发送认证帧失败: %v", err)
	}

	// 读取响应
	response, err := readResponse(stream)
	if err != nil {
		t.Fatalf("读取认证响应失败: %v", err)
	}

	// 验证认证失败
	if response.Code() == im_protocol.ErrorCodeSUCCESS {
		t.Fatalf("期望认证失败，但返回成功")
	}

	t.Logf("认证正确失败，错误码: %s, 消息: %s",
		response.Code().String(),
		string(response.Msg()))
}

// createTestConfig 创建测试配置
func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Addr:                   "localhost:18081", // 使用不同端口避免冲突
			NodeID:                 "test-access-1",
			MaxConnections:         1000,
			HeartbeatTimeout:       90 * time.Second,
			HeartbeatCheckInterval: 30 * time.Second,
		},
		QUIC: config.QUICConfig{
			MaxIdleTimeout:        90 * time.Second,
			KeepAlivePeriod:       30 * time.Second,
			MaxIncomingStreams:    100,
			MaxIncomingUniStreams: 50,
			Allow0RTT:             true,
			CertFile:              "../../localhost+2.pem",
			KeyFile:               "../../localhost+2-key.pem",
		},
		NATS: config.NATSConfig{
			URL:           "nats://localhost:4222",
			MaxReconnects: -1,
			ReconnectWait: 2 * time.Second,
		},
		Redis: config.RedisConfig{
			Addr:     "localhost:6379",
			Password: "xhxxygwl",
			DB:       0,
			PoolSize: 10,
		},
	}
}

// createWebTransportDialer 创建 WebTransport 拨号器
func createWebTransportDialer(t *testing.T) *webtransport.Dialer {
	// 使用简单配置的拨号器
	return &webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 测试环境跳过证书验证
			NextProtos:         []string{"h3"},
		},
		QUICConfig: &quic.Config{
			MaxIdleTimeout:  30 * time.Second,
			EnableDatagrams: true, // 启用数据报支持，WebTransport 需要此选项
		},
	}
}

// setTestUserToken 设置测试用户 token（模拟 web-go 登录时设置的数据）
func setTestUserToken(ctx context.Context, redisClient *redis.Client, userID int64, platform, token, deviceID string) error {
	// Mock Redis token info
	mockUserInfo := &redis.UserTokenInfo{
		UserID:   userID,
		DeviceID: deviceID,
		Platform: platform,
	}
	tokenInfoJSON, err := json.Marshal(mockUserInfo)
	if err != nil {
		return err
	}

	// 2. 使用反射获取底层 redis client（因为它是私有字段）
	// 注意：这是测试代码的临时方案，生产代码应该在 redis.Client 中添加公开方法

	// 通过反射获取 redis.Client 的私有 client 字段
	clientValue := reflect.ValueOf(redisClient).Elem()
	clientField := clientValue.FieldByName("client")

	// 使用 reflect.NewAt 创建可访问的指针
	clientFieldPtr := reflect.NewAt(clientField.Type(), unsafe.Pointer(clientField.UnsafeAddr())).Elem()
	underlyingClient := clientFieldPtr.Interface().(*goredis.Client)

	// 3. 设置 token:info:{token} -> UserTokenInfo JSON (24小时过期)
	tokenKey := "token:info:" + token
	err = underlyingClient.Set(ctx, tokenKey, string(tokenInfoJSON), 24*time.Hour).Err()
	if err != nil {
		return err
	}

	// 4. 设置 user:token:{userId}:{platform} -> token (24小时过期)
	userTokenKey := fmt.Sprintf("user:token:%d:%s", userID, strings.ToLower(platform))
	err = underlyingClient.Set(ctx, userTokenKey, token, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}

// cleanupTestData 清理测试数据
func cleanupTestData(ctx context.Context, t *testing.T, redisClient *redis.Client, userID int64, platform, token string) {
	// 移除用户位置
	err := redisClient.UnregisterUserLocation(ctx, userID, platform)
	if err != nil {
		t.Logf("清理用户位置失败: %v", err)
	}

	// 清理 token 相关数据
	// 通过反射获取底层 client
	clientValue := reflect.ValueOf(redisClient).Elem()
	clientField := clientValue.FieldByName("client")
	clientFieldPtr := reflect.NewAt(clientField.Type(), unsafe.Pointer(clientField.UnsafeAddr())).Elem()
	underlyingClient := clientFieldPtr.Interface().(*goredis.Client)

	// 删除 token:info:{token}
	tokenKey := "token:info:" + token
	err = underlyingClient.Del(ctx, tokenKey).Err()
	if err != nil {
		t.Logf("清理 token info 失败: %v", err)
	}

	// 删除 user:token:{userId}:{platform}
	userTokenKey := fmt.Sprintf("user:token:%d:%s", userID, strings.ToLower(platform))
	err = underlyingClient.Del(ctx, userTokenKey).Err()
	if err != nil {
		t.Logf("清理 user token 失败: %v", err)
	}
}

// buildAuthRequest 构建认证请求
func buildAuthRequest(token, deviceID string, platform im_protocol.Platform) []byte {
	builder := flatbuffers.NewBuilder(256)

	tokenOffset := builder.CreateString(token)
	deviceIDOffset := builder.CreateString(deviceID)
	appVersionOffset := builder.CreateString("1.0.0")

	im_protocol.AuthRequestStart(builder)
	im_protocol.AuthRequestAddToken(builder, tokenOffset)
	im_protocol.AuthRequestAddDeviceId(builder, deviceIDOffset)
	im_protocol.AuthRequestAddPlatform(builder, platform)
	im_protocol.AuthRequestAddAppVersion(builder, appVersionOffset)
	authReqOffset := im_protocol.AuthRequestEnd(builder)

	builder.Finish(authReqOffset)
	return builder.FinishedBytes()
}

// sendAuthFrame 发送认证帧
func sendAuthFrame(stream *webtransport.Stream, authReq []byte) error {
	// 构建帧头：4 bytes length + 1 byte frame type
	frame := make([]byte, frameHeaderSize+len(authReq))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(authReq)))
	frame[4] = frameTypeAuth
	copy(frame[frameHeaderSize:], authReq)

	_, err := stream.Write(frame)
	return err
}

// readResponse 读取响应
func readResponse(stream *webtransport.Stream) (*im_protocol.ClientResponse, error) {
	// 读取帧头
	header := make([]byte, frameHeaderSize)
	if _, err := io.ReadFull(stream, header); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header[:4])
	// frameType := header[4] // 响应帧类型

	// 读取消息体
	body := make([]byte, length)
	if _, err := io.ReadFull(stream, body); err != nil {
		return nil, err
	}

	// 解析 ClientResponse
	response := im_protocol.GetRootAsClientResponse(body, 0)
	return response, nil
}
