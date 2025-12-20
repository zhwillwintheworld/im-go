package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sudooom.im.shared/jwt"
	"sudooom.im.shared/snowflake"
	"sudooom.im.web/internal/middleware"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/internal/service"
	"sudooom.im.web/pkg/response"
)

// 测试配置 - 使用环境变量或默认值
var (
	testDBHost     = getEnv("POSTGRES_HOST", "localhost")
	testDBPort     = getEnv("POSTGRES_PORT", "5432")
	testDBUser     = getEnv("POSTGRES_USER", "postgres")
	testDBPassword = getEnv("POSTGRES_PASSWORD", "password")
	testDBName     = getEnv("POSTGRES_DB", "im_db")

	testRedisHost     = getEnv("REDIS_HOST", "localhost")
	testRedisPort     = getEnv("REDIS_PORT", "6379")
	testRedisPassword = getEnv("REDIS_PASSWORD", "")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// testDeps 测试依赖
type testDeps struct {
	db          *pgxpool.Pool
	redisClient *redis.Client
	jwtService  *jwt.Service
	sfNode      *snowflake.Node
	userRepo    *repository.UserRepository
	tokenRepo   *repository.TokenRepository
	authService *service.AuthService
	authHandler *AuthHandler
	router      *gin.Engine
}

// setupIntegrationTest 初始化集成测试环境
func setupIntegrationTest(t *testing.T) *testDeps {
	t.Helper()

	ctx := context.Background()

	// 连接 PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		testDBUser, testDBPassword, testDBHost, testDBPort, testDBName)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("跳过集成测试: 无法连接数据库: %v", err)
	}

	// 测试数据库连接
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("跳过集成测试: 数据库 ping 失败: %v", err)
	}

	// 连接 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", testRedisHost, testRedisPort),
		Password: testRedisPassword,
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		db.Close()
		t.Skipf("跳过集成测试: 无法连接 Redis: %v", err)
	}

	// 初始化组件
	jwtService := jwt.NewService("test-secret-key", 24*time.Hour, 7*24*time.Hour)

	sfNode, err := snowflake.NewNode(1)
	require.NoError(t, err)

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(redisClient)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtService, sfNode)
	authHandler := NewAuthHandler(authService)

	// 创建路由
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/register", authHandler.Register)
	r.POST("/api/v1/auth/login", authHandler.Login)

	// 需要认证的路由
	authenticated := r.Group("")
	authenticated.Use(middleware.TokenAuth(tokenRepo, 24*time.Hour, 4*time.Hour))
	authenticated.POST("/api/v1/auth/logout", authHandler.Logout)

	return &testDeps{
		db:          db,
		redisClient: redisClient,
		jwtService:  jwtService,
		sfNode:      sfNode,
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		authService: authService,
		authHandler: authHandler,
		router:      r,
	}
}

// teardownIntegrationTest 清理测试环境
func (d *testDeps) teardown() {
	if d.db != nil {
		d.db.Close()
	}
	if d.redisClient != nil {
		d.redisClient.Close()
	}
}

// cleanupTestUser 清理测试用户
func (d *testDeps) cleanupTestUser(ctx context.Context, username string) error {
	_, err := d.db.Exec(ctx, "DELETE FROM users WHERE username = $1", username)
	return err
}

// TestIntegration_Login_Success 集成测试: 登录成功
func TestIntegration_Login_Success(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	ctx := context.Background()
	testUsername := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	testPassword := "password123"
	testNickname := "Test User"

	// 清理可能存在的测试用户
	defer deps.cleanupTestUser(ctx, testUsername)

	// Step 1: 先注册用户
	registerBody := map[string]string{
		"username": testUsername,
		"password": testPassword,
		"nickname": testNickname,
	}
	registerJSON, _ := json.Marshal(registerBody)

	registerReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	registerW := httptest.NewRecorder()
	deps.router.ServeHTTP(registerW, registerReq)

	assert.Equal(t, http.StatusOK, registerW.Code)

	var registerResp APIResponse
	err := json.Unmarshal(registerW.Body.Bytes(), &registerResp)
	require.NoError(t, err)
	assert.Equal(t, response.CodeSuccess, registerResp.Code, "注册应该成功")

	// Step 2: 登录
	loginBody := map[string]string{
		"username":  testUsername,
		"password":  testPassword,
		"device_id": "test-device-001",
		"platform":  "pc",
	}
	loginJSON, _ := json.Marshal(loginBody)

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	deps.router.ServeHTTP(loginW, loginReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResp APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeSuccess, loginResp.Code, "登录应该成功")

	// 验证返回的数据
	var loginData service.LoginResponse
	err = json.Unmarshal(loginResp.Data, &loginData)
	require.NoError(t, err)

	assert.NotZero(t, loginData.UserID, "应该返回 user_id")
	assert.NotEmpty(t, loginData.AccessToken, "应该返回 access_token")
	assert.NotEmpty(t, loginData.RefreshToken, "应该返回 refresh_token")
	assert.NotZero(t, loginData.ExpiresAt, "应该返回过期时间")

	// 验证 Token 有效性
	claims, err := deps.jwtService.ValidateAccessToken(loginData.AccessToken)
	require.NoError(t, err)
	assert.NotZero(t, claims.UserID, "Token应该包含用户ID")
	assert.Equal(t, "test-device-001", claims.DeviceID)
	assert.Equal(t, jwt.Platform("pc"), claims.Platform)

	t.Logf("登录成功! UserID: %d, AccessToken: %s...", loginData.UserID, loginData.AccessToken[:20])
}

// TestIntegration_Login_WrongPassword 集成测试: 密码错误
func TestIntegration_Login_WrongPassword(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	ctx := context.Background()
	testUsername := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	testPassword := "password123"
	testNickname := "Test User"

	defer deps.cleanupTestUser(ctx, testUsername)

	// 先注册用户
	registerBody := map[string]string{
		"username": testUsername,
		"password": testPassword,
		"nickname": testNickname,
	}
	registerJSON, _ := json.Marshal(registerBody)

	registerReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	registerW := httptest.NewRecorder()
	deps.router.ServeHTTP(registerW, registerReq)
	require.Equal(t, http.StatusOK, registerW.Code)

	// 使用错误密码登录
	loginBody := map[string]string{
		"username": testUsername,
		"password": "wrongpassword",
	}
	loginJSON, _ := json.Marshal(loginBody)

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	deps.router.ServeHTTP(loginW, loginReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResp APIResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeInvalidCredentials, loginResp.Code, "应该返回密码错误")
	t.Log("密码错误测试通过!")
}

// TestIntegration_Login_UserNotFound 集成测试: 用户不存在
func TestIntegration_Login_UserNotFound(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	// 登录一个不存在的用户
	loginBody := map[string]string{
		"username": "nonexistent_user_12345",
		"password": "password123",
	}
	loginJSON, _ := json.Marshal(loginBody)

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	deps.router.ServeHTTP(loginW, loginReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResp APIResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeInvalidCredentials, loginResp.Code, "应该返回用户不存在(统一为凭证错误)")
	t.Log("用户不存在测试通过!")
}

// TestIntegration_Login_WithExistingUser 集成测试: 使用数据库中已存在的用户登录
// 如果你的数据库中已经有 zhanghua 用户，可以使用此测试
func TestIntegration_Login_WithExistingUser(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	// 使用数据库中已存在的用户 (你需要确保这个用户存在)
	// 如果用户不存在，此测试会失败
	loginBody := map[string]string{
		"username":  "zhanghua",
		"password":  "123456",
		"device_id": "integration-test-device",
		"platform":  "pc",
	}
	loginJSON, _ := json.Marshal(loginBody)

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	deps.router.ServeHTTP(loginW, loginReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResp APIResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	if loginResp.Code == response.CodeSuccess {
		var loginData service.LoginResponse
		err = json.Unmarshal(loginResp.Data, &loginData)
		require.NoError(t, err)

		t.Logf("zhanghua 登录成功! AccessToken: %s...", loginData.AccessToken[:20])
	} else {
		t.Logf("zhanghua 登录失败: code=%d, message=%s (用户可能不存在或密码错误)", loginResp.Code, loginResp.Message)
	}
}

// TestIntegration_Logout_Success 集成测试: 登出成功
// 使用已存在的 zhanghua 账号进行测试
func TestIntegration_Logout_Success(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	// 使用已存在的用户账号
	testUsername := "zhanghua"
	testPassword := "123456"

	// Step 1: 登录获取 Token
	loginBody := map[string]string{
		"username":  testUsername,
		"password":  testPassword,
		"device_id": "test-device-logout-001",
		"platform":  "pc",
	}
	loginJSON, _ := json.Marshal(loginBody)

	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	deps.router.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResp APIResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	if loginResp.Code != response.CodeSuccess {
		t.Skipf("跳过测试: 用户 %s 登录失败 (code=%d, msg=%s)，请确保用户存在", testUsername, loginResp.Code, loginResp.Message)
	}

	var loginData service.LoginResponse
	err = json.Unmarshal(loginResp.Data, &loginData)
	require.NoError(t, err)

	// Step 2: 使用 Token 登出
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", loginData.AccessToken)

	logoutW := httptest.NewRecorder()
	deps.router.ServeHTTP(logoutW, logoutReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, logoutW.Code)

	var logoutResp APIResponse
	err = json.Unmarshal(logoutW.Body.Bytes(), &logoutResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeSuccess, logoutResp.Code, "登出应该成功")
	t.Logf("用户 %s 登出成功!", testUsername)
}

// TestIntegration_Logout_WithoutToken 集成测试: 无Token登出
func TestIntegration_Logout_WithoutToken(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	// 不带 Token 直接登出
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)

	logoutW := httptest.NewRecorder()
	deps.router.ServeHTTP(logoutW, logoutReq)

	// 验证响应 - 应该返回未授权 (401状态码)
	assert.Equal(t, http.StatusUnauthorized, logoutW.Code)

	var logoutResp APIResponse
	err := json.Unmarshal(logoutW.Body.Bytes(), &logoutResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeTokenInvalid, logoutResp.Code, "无Token应该返回Token无效")
	t.Log("无Token登出测试通过!")
}

// TestIntegration_Logout_InvalidToken 集成测试: 无效Token登出
func TestIntegration_Logout_InvalidToken(t *testing.T) {
	deps := setupIntegrationTest(t)
	defer deps.teardown()

	// 使用无效 Token 登出
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer invalid-token-12345")

	logoutW := httptest.NewRecorder()
	deps.router.ServeHTTP(logoutW, logoutReq)

	// 验证响应 - 应该返回Token无效
	assert.Equal(t, http.StatusOK, logoutW.Code)

	var logoutResp APIResponse
	err := json.Unmarshal(logoutW.Body.Bytes(), &logoutResp)
	require.NoError(t, err)

	assert.Equal(t, response.CodeTokenInvalid, logoutResp.Code, "无效Token应该返回Token无效")
	t.Log("无效Token登出测试通过!")
}
