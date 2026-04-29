package handlers

import (
	"net/http"
	"os"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

func GetSetupStatus(c *gin.Context) {
	hasAdmin, err := services.HasAdminUser()
	if err != nil {
		c.JSON(http.StatusOK, SetupStatusResponse{NeedsSetup: true})
		return
	}
	c.JSON(http.StatusOK, SetupStatusResponse{NeedsSetup: !hasAdmin})
}

type SetupRequest struct {
	OrgName       string `json:"org_name"`
	OrgLink       string `json:"org_link"`
	DataRootPath  string `json:"data_root_path"`
	ServerPort    int    `json:"server_port"`
	SessionSecret string `json:"session_secret"`
	LogLevel      string `json:"log_level"`
	DBPath        string `json:"db_path"`
	AdminPassword string `json:"admin_password"`
	MaxUploadSize int64  `json:"max_upload_size"`
}

func PostSetup(c *gin.Context) {
	hasAdmin, err := services.HasAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check admin user failed"})
		return
	}
	if hasAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "setup already completed"})
		return
	}

	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.AdminPassword == "" || len(req.AdminPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin password must be at least 6 characters"})
		return
	}
	if req.DataRootPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data_root_path is required"})
		return
	}

	newCfg := &config.Config{
		OrgName:       req.OrgName,
		OrgLink:       req.OrgLink,
		DataRootPath:  req.DataRootPath,
		ServerPort:    req.ServerPort,
		SessionSecret: req.SessionSecret,
		LogLevel:      req.LogLevel,
		DBPath:        req.DBPath,
		MaxUploadSize: req.MaxUploadSize,
	}

	if newCfg.ServerPort == 0 {
		newCfg.ServerPort = 8080
	}
	if newCfg.LogLevel == "" {
		newCfg.LogLevel = "info"
	}
	if newCfg.DBPath == "" {
		newCfg.DBPath = "wfm.db"
	}
	if newCfg.SessionSecret == "" {
		newCfg.SessionSecret = config.RandomSecret()
	}
	if newCfg.MaxUploadSize == 0 {
		newCfg.MaxUploadSize = 1073741824 // 默认 1 GB
	}

	if err := os.MkdirAll(req.DataRootPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create data directory failed"})
		return
	}

	if err := config.Save("config.json", newCfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save config failed"})
		return
	}

	*config.C = *newCfg

	admin, err := services.CreateUser("admin", req.AdminPassword, "Administrator", "", true, models.PermAll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create admin user failed"})
		return
	}

	services.CreateLog(admin.ID, models.ActionLogin, "", c.ClientIP(), nil)

	c.JSON(http.StatusOK, gin.H{"message": "setup completed", "admin_username": "admin"})
}

func GetConfigInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"org_name": config.C.OrgName,
		"org_link": config.C.OrgLink,
		"version":  Version,
	})
}
