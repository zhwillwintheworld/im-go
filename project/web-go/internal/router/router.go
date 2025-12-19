package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"sudooom.im.web/internal/config"
	"sudooom.im.web/internal/handler"
	"sudooom.im.web/internal/middleware"
	"sudooom.im.web/internal/repository"
)

// SetupRouter 设置路由
func SetupRouter(
	cfg *config.Config,
	tokenRepo *repository.TokenRepository,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	friendHandler *handler.FriendHandler,
) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(cfg.App.Mode)

	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(
		cfg.CORS.AllowedOrigins,
		cfg.CORS.AllowedMethods,
		cfg.CORS.AllowCredentials,
	))

	// Swagger 文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证接口（无需登录）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// 需要认证的接口
		authenticated := v1.Group("")
		authenticated.Use(middleware.TokenAuth(tokenRepo, cfg.JWT.AccessExpire, cfg.JWT.AutoRenewThreshold))
		{
			// 登出
			authenticated.POST("/auth/logout", authHandler.Logout)

			// 用户接口
			user := authenticated.Group("/user")
			{
				user.GET("/profile", userHandler.GetProfile)
				user.PUT("/profile", userHandler.UpdateProfile)
				user.GET("/search", userHandler.Search)
				user.GET("/:id", userHandler.GetUserByID)
			}

			// 好友接口
			friends := authenticated.Group("/friends")
			{
				friends.GET("", friendHandler.GetFriendList)
				friends.POST("/request", friendHandler.SendRequest)
				friends.GET("/requests", friendHandler.GetPendingRequests)
				friends.POST("/accept/:id", friendHandler.AcceptRequest)
				friends.POST("/reject/:id", friendHandler.RejectRequest)
				friends.DELETE("/:id", friendHandler.DeleteFriend)
			}
		}
	}

	return r
}
