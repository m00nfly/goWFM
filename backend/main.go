package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"goWFM/config"
	"goWFM/db"
	"goWFM/handlers"
	"goWFM/middleware"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/api/setup/status", handlers.GetSetupStatus)
	r.POST("/api/setup", handlers.PostSetup)
	r.GET("/api/config/info", handlers.GetConfigInfo)

	r.POST("/api/auth/login", handlers.Login)
	r.POST("/api/auth/login/totp", handlers.LoginTOTP)
	r.POST("/api/auth/login/totp/setup", handlers.LoginTOTPSetup)
	r.POST("/api/auth/logout", handlers.Logout)
	r.GET("/api/auth/captcha", handlers.GetCaptcha)

	auth := r.Group("/api")
	auth.Use(middleware.AuthRequired())
	{
		auth.GET("/auth/me", handlers.GetMe)

		auth.GET("/users", middleware.AdminRequired(), handlers.ListUsers)
		auth.POST("/users", middleware.AdminRequired(), handlers.CreateUser)
		auth.PUT("/users/:id", middleware.AdminRequired(), handlers.UpdateUser)
		auth.DELETE("/users/:id", middleware.AdminRequired(), handlers.DeleteUser)

		auth.PUT("/users/me", handlers.UpdateMe)
		auth.PUT("/users/me/password", handlers.ChangePassword)

		// TOTP 管理（用户自己）
		auth.GET("/users/me/totp/status", handlers.GetMyTOTPStatus)
		auth.POST("/users/me/totp/setup", handlers.SetupTOTP)
		auth.POST("/users/me/totp/verify", handlers.VerifyTOTPSetup)
		auth.POST("/users/me/totp/disable", handlers.DisableMyTOTP)

		// TOTP 管理（管理员）
		auth.DELETE("/users/:id/totp", middleware.AdminRequired(), handlers.AdminDisableTOTP)
		auth.PUT("/users/:id/totp", middleware.AdminRequired(), handlers.AdminUpdateTOTP)
		auth.POST("/users/:id/totp/reset", middleware.AdminRequired(), handlers.AdminResetTOTP)

		auth.GET("/files", handlers.ListFiles)
		auth.POST("/files/upload", handlers.UploadFile)
		auth.POST("/files/mkdir", handlers.CreateDir)
		auth.DELETE("/files", handlers.DeleteFile)
		auth.GET("/download", handlers.Download)
		auth.PUT("/files/owner", handlers.ChangeOwner)
		auth.PUT("/files/move", handlers.MoveFile)

		auth.POST("/shares", handlers.CreateShareLink)
		auth.GET("/shares/my", handlers.ListMyShares)
		auth.PUT("/shares/:id", handlers.UpdateShareLink)
		auth.DELETE("/shares/:id", handlers.DeleteShareLink)

		auth.GET("/logs", handlers.ListLogs)
		auth.GET("/logs/users", handlers.ListUsersForLog)

		auth.GET("/admin/shares", middleware.AdminRequired(), handlers.ListAllShares)
		auth.GET("/admin/share-users", middleware.AdminRequired(), handlers.ListShareUsers)

		// 配置管理 API
		auth.GET("/admin/config/:category", middleware.AdminRequired(), handlers.GetConfig)
		auth.PUT("/admin/config/:category", middleware.AdminRequired(), handlers.UpdateConfig)
	}

	sharePublic := r.Group("/share")
	sharePublic.Use(middleware.OptionalAuth())
	{
		sharePublic.GET("/:token", handlers.AccessShareEntry)
		sharePublic.GET("/:token/info", handlers.GetShareInfo)
		sharePublic.GET("/:token/:filename", handlers.ShareFileDownload)
	}

	staticFS, err := getFrontendFS()
	if err == nil {
		httpFS := http.FS(staticFS)
		fileServer := http.FileServer(httpFS)
		handlers.SetFrontendFS(fileServer)
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if path == "/favicon.ico" || strings.HasPrefix(path, "/assets/") {
				c.Status(http.StatusOK)
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			c.Request.URL.Path = "/"
			c.Status(http.StatusOK)
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	return r
}

func main() {
	dbPath := flag.String("db", "./gowfm.db", "数据库文件路径")
	flag.Parse()

	// 1. 初始化数据库
	if err := db.Init(*dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. 从数据库加载配置（首次启动自动初始化默认值）
	if err := services.LoadAllConfigs(); err != nil {
		log.Fatalf("Failed to load configs: %v", err)
	}

	// 3. 确保数据目录存在
	basicCfg := config.GetBasic()
	if basicCfg.DataRootPath != "" {
		if err := os.MkdirAll(basicCfg.DataRootPath, 0755); err != nil {
			log.Fatalf("Failed to create data root path: %v", err)
		}
	}

	// 4. 设置路由
	r := setupRouter()
	r.MaxMultipartMemory = basicCfg.MaxUploadSize

	// 5. 后台定时任务
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			// 清理过期会话
			count, err := services.CleanExpiredSessions()
			if err != nil {
				log.Printf("Failed to clean expired sessions: %v", err)
			} else if count > 0 {
				log.Printf("Cleaned %d expired sessions", count)
			}

			// 清理过期日志
			logCfg := config.GetLog()
			if logCfg.RetentionDays > 0 {
				logCount, err := services.CleanOldLogs(logCfg.RetentionDays)
				if err != nil {
					log.Printf("Failed to clean old logs: %v", err)
				} else if logCount > 0 {
					log.Printf("Cleaned %d old log entries", logCount)
				}
			}

			// 清理过期验证码
			services.CleanExpiredCaptchas()

			// 清理过期信任设备
			services.CleanExpiredTrustedDevices()
		}
	}()

	// 6. 启动封锁引擎后台清理
	go services.StartBlockerCleanup()

	// 7. 启动 HTTP/HTTPS 服务器
	appCfg := config.GetAppearance()
	addr := fmt.Sprintf(":%d", appCfg.ServerPort)
	log.Printf("goWFM server starting on %s (HTTPS: %v)", addr, appCfg.EnableHTTPS)

	if appCfg.EnableHTTPS {
		tlsCert, err := services.EnsureTLSCert(appCfg.SSLCert, appCfg.SSLKey)
		if err != nil {
			log.Fatalf("Failed to setup TLS: %v", err)
		}
		server := &http.Server{
			Addr:    addr,
			Handler: r,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
			},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Fatalf("HTTPS server failed: %v", err)
		}
	} else {
		if err := r.Run(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
