package main

import (
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

var Version string = "dev"

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
	r.POST("/api/auth/logout", handlers.Logout)

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

		auth.GET("/files", handlers.ListFiles)
		auth.POST("/files/upload", handlers.UploadFile)
		auth.POST("/files/mkdir", handlers.CreateDir)
		auth.DELETE("/files", handlers.DeleteFile)
		auth.GET("/download", handlers.Download)
		auth.PUT("/files/owner", handlers.ChangeOwner)
		auth.PUT("/files/move", handlers.MoveFile)

		auth.POST("/shares", handlers.CreateShareLink)
		auth.GET("/shares/my", handlers.ListMyShares)
		auth.DELETE("/shares/:id", handlers.DeleteShareLink)

		auth.GET("/logs", handlers.ListLogs)
		auth.GET("/logs/users", handlers.ListUsersForLog)

		auth.GET("/admin/shares", middleware.AdminRequired(), handlers.ListAllShares)
		auth.GET("/admin/share-users", middleware.AdminRequired(), handlers.ListShareUsers)
	}

	r.GET("/share/:token", handlers.AccessShareEntry)
	r.GET("/share/:token/info", handlers.GetShareInfo)
	r.GET("/share/:token/:filename", handlers.ShareFileDownload)

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
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if config.C.DataRootPath != "" {
		if err := os.MkdirAll(config.C.DataRootPath, 0755); err != nil {
			log.Fatalf("Failed to create data root path: %v", err)
		}
	}

	if err := db.Init(config.C.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	handlers.Version = Version

	r := setupRouter()
	r.MaxMultipartMemory = cfg.MaxUploadSize

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			count, err := services.CleanExpiredSessions()
			if err != nil {
				log.Printf("Failed to clean expired sessions: %v", err)
				continue
			}
			if count > 0 {
				log.Printf("Cleaned %d expired sessions", count)
			}
		}
	}()

	addr := fmt.Sprintf(":%d", config.C.ServerPort)
	log.Printf("goWFM server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
