package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/config"
	"github.com/webdav-gateway/internal/middleware"
	"github.com/webdav-gateway/internal/share"
	"github.com/webdav-gateway/internal/storage"
	"github.com/webdav-gateway/internal/webdav"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Info("Connected to PostgreSQL")

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Test Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	logger.Info("Connected to Redis")

	// Initialize services
	storageService, err := storage.NewService(cfg)
	if err != nil {
		logger.Fatalf("Failed to create storage service: %v", err)
	}
	logger.Info("Storage service initialized")

	authService := auth.NewService(db, cfg)
	shareService := share.NewService(db, cfg)
	
	// Initialize property service
	propertyService, err := webdav.NewPropertyService(cfg.App.DataPath + "/properties.db")
	if err != nil {
		logger.Fatalf("Failed to create property service: %v", err)
	}
	logger.Info("Property service initialized")
	
	webdavHandler := webdav.NewHandler(storageService, authService, propertyService)

	// Setup Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()

	// Global middleware
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.LoggerMiddleware(logger))
	
	if cfg.App.EnableCORS {
		router.Use(middleware.CORSMiddleware())
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	// Auth routes
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/register", handleRegister(authService))
		authGroup.POST("/login", handleLogin(authService, storageService))
		authGroup.GET("/me", middleware.AuthMiddleware(authService), handleGetMe(authService))
	}

	// Share routes
	shareGroup := router.Group("/api/shares")
	shareGroup.Use(middleware.AuthMiddleware(authService))
	{
		shareGroup.POST("", handleCreateShare(shareService))
		shareGroup.GET("", handleListShares(shareService))
		shareGroup.DELETE("/:id", handleDeleteShare(shareService))
	}

	// Public share access
	router.GET("/share/:token", handleGetShare(shareService, storageService, authService))
	router.POST("/share/:token/access", handleAccessShare(shareService))

	// WebDAV routes
	webdavGroup := router.Group("/webdav")
	webdavGroup.Use(middleware.AuthMiddleware(authService))
	webdavGroup.Use(middleware.StorageQuotaMiddleware(authService))
	{
		webdavGroup.Handle("OPTIONS", "/*path", webdavHandler.HandleOptions)
		webdavGroup.Handle("PROPFIND", "/*path", webdavHandler.HandlePropfind)
		webdavGroup.Handle("PROPPATCH", "/*path", webdavHandler.HandleProppatch)
		webdavGroup.Handle("GET", "/*path", webdavHandler.HandleGet)
		webdavGroup.Handle("HEAD", "/*path", webdavHandler.HandleHead)
		webdavGroup.Handle("PUT", "/*path", webdavHandler.HandlePut)
		webdavGroup.Handle("DELETE", "/*path", webdavHandler.HandleDelete)
		webdavGroup.Handle("MKCOL", "/*path", webdavHandler.HandleMkcol)
		webdavGroup.Handle("MOVE", "/*path", webdavHandler.HandleMove)
		webdavGroup.Handle("COPY", "/*path", webdavHandler.HandleCopy)
		webdavGroup.Handle("LOCK", "/*path", webdavHandler.HandleLock)
		webdavGroup.Handle("UNLOCK", "/*path", webdavHandler.HandleUnlock)
	}

	// Setup HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    15 * time.Minute,
		WriteTimeout:   15 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	// Graceful shutdown
	go func() {
		logger.Infof("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// ========================================
// HTTP Handlers
// ========================================

// handleRegister 处理用户注册
func handleRegister(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username    string `json:"username" binding:"required,min=3,max=50"`
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=6"`
			DisplayName string `json:"display_name" binding:"max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := authService.Register(req.Username, req.Email, req.Password, req.DisplayName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "注册成功",
			"user":    user,
		})
	}
}

// handleLogin 处理用户登录
func handleLogin(authService *auth.Service, storageService *storage.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, user, err := authService.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}

		// 获取用户存储使用情况
		storageUsed, err := storageService.GetUserStorageUsed(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取存储信息失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":            user.ID,
				"username":      user.Username,
				"email":         user.Email,
				"display_name":  user.DisplayName,
				"storage_quota": user.StorageQuota,
				"storage_used":  storageUsed,
			},
		})
	}
}

// handleGetMe 获取当前用户信息
func handleGetMe(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		user, err := authService.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	}
}

// handleCreateShare 处理创建分享
func handleCreateShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		var req struct {
			FilePath     string `json:"file_path" binding:"required"`
			ShareName    string `json:"share_name" binding:"required,max=100"`
			Password     string `json:"password"`
			ExpiresIn    int    `json:"expires_in"` // 小时
			MaxDownloads int    `json:"max_downloads"`
			Permissions  string `json:"permissions" binding:"oneof=read write"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		share, err := shareService.CreateShare(userID, req.FilePath, req.ShareName, req.Password, req.ExpiresIn, req.MaxDownloads, req.Permissions)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"share_url":  fmt.Sprintf("/share/%s", share.ShareToken),
			"share_token": share.ShareToken,
			"expires_at":  share.ExpiresAt,
		})
	}
}

// handleListShares 处理列出分享
func handleListShares(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		shares, err := shareService.ListShares(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分享列表失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"shares": shares,
		})
	}
}

// handleDeleteShare 处理删除分享
func handleDeleteShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		shareID := c.Param("id")
		if shareID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
			return
		}

		err := shareService.DeleteShare(userID, shareID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "分享已删除",
		})
	}
}

// handleGetShare 处理获取分享
func handleGetShare(shareService *share.Service, storageService *storage.Service, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分享令牌不能为空"})
			return
		}

		share, err := shareService.GetShareByToken(token)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已过期"})
			return
		}

		// 获取文件信息
		fileInfo, err := storageService.GetFileInfo(share.UserID, share.FilePath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"share":   share,
			"file":    fileInfo,
			"requires_password": share.Password != "",
		})
	}
}

// handleAccessShare 处理访问分享
func handleAccessShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分享令牌不能为空"})
			return
		}

		var req struct {
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证分享访问权限
		err := shareService.ValidateShareAccess(token, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "访问被拒绝"})
			return
		}

		// 获取分享信息
		share, err := shareService.GetShareByToken(token)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
			return
		}

		// 记录访问
		err = shareService.RecordAccess(token)
		if err != nil {
			// 记录访问失败不影响主要功能
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "访问允许",
			"share":   share,
		})
	}
}