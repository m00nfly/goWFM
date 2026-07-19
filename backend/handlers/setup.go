package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

type SetupRequest struct {
	SiteName      string `json:"site_name"`
	SiteLink      string `json:"site_link"`
	DataRootPath  string `json:"data_root_path"`
	ServerPort    int    `json:"server_port"`
	AdminUsername string `json:"admin_username"`
	AdminPassword string `json:"admin_password"`
	AdminEmail    string `json:"admin_email"`
	MaxUploadSize int64  `json:"max_upload_size"`
}

func normalizeAdminUsername(value string) (string, bool) {
	username := strings.TrimSpace(value)
	length := utf8.RuneCountInString(username)
	if length == 0 || length > 64 || strings.EqualFold(username, "Guest") {
		return "", false
	}
	for _, r := range username {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", false
		}
	}
	return username, true
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

	adminUsername, validUsername := normalizeAdminUsername(req.AdminUsername)
	if !validUsername {
		c.JSON(http.StatusBadRequest, gin.H{"error": "管理员账号不能为空且不能超过 64 个字符，不能包含空格或使用保留账号 Guest"})
		return
	}
	if req.AdminPassword == "" || len(req.AdminPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin password must be at least 6 characters"})
		return
	}
	if _, err := services.NormalizeEmail(req.AdminEmail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的管理员邮箱"})
		return
	}
	if req.DataRootPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data_root_path is required"})
		return
	}

	// 解析为绝对路径
	absPath, err := filepath.Abs(req.DataRootPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data_root_path"})
		return
	}
	if !strings.HasSuffix(absPath, string(filepath.Separator)) {
		absPath += string(filepath.Separator)
	}

	// 创建数据目录
	if err := os.MkdirAll(absPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create data directory failed"})
		return
	}

	// 更新基础设置
	basicSettings := config.BasicSettings{
		SiteName:      req.SiteName,
		SiteLink:      req.SiteLink,
		DataRootPath:  absPath,
		MaxUploadSize: req.MaxUploadSize,
	}
	if basicSettings.MaxUploadSize == 0 {
		basicSettings.MaxUploadSize = 1073741824 // 默认 1 GB
	}
	if err := services.UpdateBasicSettings(basicSettings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save basic settings failed"})
		return
	}

	// 更新外观设置（端口）
	if req.ServerPort > 0 {
		appSettings := config.GetAppearance()
		appSettings.ServerPort = req.ServerPort
		if err := services.UpdateAppearanceSettings(appSettings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save appearance settings failed"})
			return
		}
	}

	// 创建管理员用户
	admin, err := services.CreateUser(adminUsername, req.AdminPassword, "Administrator", req.AdminEmail, true, models.PermAll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create admin user failed"})
		return
	}

	services.CreateLog(admin.ID, models.ActionLogin, "", c.ClientIP(), nil)
	services.TriggerDashboardStorageScan("setup")

	c.JSON(http.StatusOK, gin.H{"message": "setup completed", "admin_username": adminUsername})
}
