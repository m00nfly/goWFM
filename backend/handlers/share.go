package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

// frontendFileServer holds the http.Handler for serving embedded SPA files.
// Must be initialized via SetFrontendFS from main.go.
var frontendFileServer http.Handler

// SetFrontendFS sets the frontend file server for serving SPA index.html.
func SetFrontendFS(h http.Handler) {
	frontendFileServer = h
}

// serveIndexHTML serves the embedded SPA index.html to the client.
func serveIndexHTML(c *gin.Context) {
	if frontendFileServer == nil {
		c.String(http.StatusInternalServerError, "frontend not available")
		return
	}
	c.Request.URL.Path = "/"
	c.Status(http.StatusOK)
	frontendFileServer.ServeHTTP(c.Writer, c.Request)
}

func CreateShareLink(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermShare) {
		c.JSON(http.StatusForbidden, gin.H{"error": "share permission denied"})
		return
	}

	var req struct {
		FilePaths  []string `json:"file_paths" binding:"required"`
		ExpireDays int      `json:"expire_days"`
		Name       string   `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.FilePaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_paths must contain at least one file"})
		return
	}

	// Validate each file path
	for _, fp := range req.FilePaths {
		if _, err := services.SafePath(fp); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid path %q: %s", fp, err.Error())})
			return
		}
		if _, err := services.DownloadFile(fp); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file not found or is directory: %s", fp)})
			return
		}
	}

	share, err := services.CreateShare(req.FilePaths, user.ID, req.ExpireDays, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create share failed"})
		return
	}

	pathsJSON, _ := json.Marshal(req.FilePaths)
	services.CreateLog(user.ID, models.ActionShareCreate, string(pathsJSON), c.ClientIP(), map[string]interface{}{"token": share.Token})

	link := fmt.Sprintf("%s/share/%s", config.GetBasic().SiteLink, share.Token)
	c.JSON(http.StatusOK, gin.H{
		"id":         share.ID,
		"token":      share.Token,
		"name":       share.Name,
		"link":       link,
		"expire_at":  share.ExpireAt,
		"created_at": share.CreatedAt,
	})
}

func ListShares(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}
	if !user.HasPermission(models.PermShare) {
		c.JSON(http.StatusForbidden, gin.H{"error": "share permission denied"})
		return
	}

	var ownerID *int64
	if !user.IsAdmin {
		ownerID = &userID
	}
	shares, err := services.ListShares(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list shares failed"})
		return
	}
	c.JSON(http.StatusOK, shares)
}

func DeleteShareLink(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}
	if err := services.DeleteShare(id, userID, user.IsAdmin); err != nil {
		if errors.Is(err, services.ErrShareNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete share failed"})
		return
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionShareDelete, "", c.ClientIP(), map[string]interface{}{"share_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "share deleted"})
}

func UpdateShareLink(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}

	var req struct {
		Name       string `json:"name"`
		ExpireDays *int   `json:"expire_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}
	if err := services.UpdateShare(id, userID, user.IsAdmin, req.Name, req.ExpireDays); err != nil {
		if errors.Is(err, services.ErrShareNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update share failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share updated"})
}

// EmailShareLink 使用已保存的 SMTP 参数主动发送一条分享通知。
// 此入口仅在 SMTP 邮件服务已经测试并激活时开放。
func EmailShareLink(c *gin.Context) {
	if !config.GetShare().AllowEmailShare {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统未启用邮件发送分享功能"})
		return
	}
	if !config.GetEmail().Active {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SMTP 服务未激活，无法发送分享邮件"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的收件人 Email 地址"})
		return
	}
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}
	if !user.HasPermission(models.PermShare) {
		c.JSON(http.StatusForbidden, gin.H{"error": "share permission denied"})
		return
	}
	share, err := services.GetShareForActor(id, userID, user.IsAdmin)
	if err != nil {
		if errors.Is(err, services.ErrShareNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get share failed"})
		return
	}
	sharer := share.Owner.DisplayName
	if strings.TrimSpace(sharer) == "" {
		sharer = share.Owner.Username
	}
	if err := services.SendShareNotificationEmail(req.Email, sharer, share.FileName, share.FileCount, share.Token); err != nil {
		result := gin.H{"error": err.Error()}
		if code, message, ok := services.SMTPErrorDetails(err); ok {
			result["smtp_code"] = code
			result["smtp_message"] = message
		}
		c.JSON(http.StatusBadGateway, result)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分享邮件已发送", "recipient": req.Email})
}

func GetShareInfo(c *gin.Context) {
	token := c.Param("token")

	share, err := services.ValidateShareAccess(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shareFiles, err := services.GetShareFiles(share.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get share files failed"})
		return
	}

	// 查询分享者信息
	ownerName := ""
	ownerAvatar := ""
	if owner, ownerErr := services.GetUserByID(share.OwnerID); ownerErr == nil && owner != nil {
		ownerName = owner.DisplayName
		if ownerName == "" {
			ownerName = owner.Username
		}
		ownerAvatar = owner.AvatarData
	}

	type fileInfo struct {
		ID            int64  `json:"id"`
		FileName      string `json:"file_name"`
		FileSize      int64  `json:"file_size"`
		DownloadCount int    `json:"download_count"`
	}

	var totalSize int64
	files := make([]fileInfo, 0, len(shareFiles))
	for _, f := range shareFiles {
		fullPath, err := services.DownloadFile(f.FilePath)
		if err != nil {
			continue // skip unavailable files
		}
		info, statErr := os.Stat(fullPath)
		if statErr != nil {
			continue // skip files that can't be stat'd
		}
		totalSize += info.Size()
		files = append(files, fileInfo{
			ID:            f.ID,
			FileName:      filepath.Base(fullPath),
			FileSize:      info.Size(),
			DownloadCount: f.DownloadCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"name":         share.Name,
		"owner_name":   ownerName,
		"owner_avatar": ownerAvatar,
		"expire_at":    share.ExpireAt,
		"created_at":   share.CreatedAt,
		"file_count":   len(files),
		"total_size":   totalSize,
		"files":        files,
	})
}

// AccessShareEntry serves the SPA index.html for all share access.
// The frontend handles token validation via /share/:token/info API.
// Only direct browser visits to this route count as share link access.
func AccessShareEntry(c *gin.Context) {
	token := c.Param("token")

	// Validate share before counting access; silently ignore invalid tokens
	// (the frontend will show an appropriate error via /share/:token/info)
	if share, err := services.ValidateShareAccess(token); err == nil {
		_ = share
		services.IncrementShareAccess(token)

		// Log share access: use logged-in userID or 0 for Guest
		userID := c.GetInt64("userID")
		services.CreateLog(userID, models.ActionShareAccess, token, c.ClientIP(), map[string]interface{}{
			"user_agent": c.Request.UserAgent(),
		})
	}

	serveIndexHTML(c)
}

// CreateShareDownloadLink validates the selected share file and issues a
// short-lived, single-use URL. Issuing a new URL revokes older URLs for it.
func CreateShareDownloadLink(c *gin.Context) {
	if !config.GetShare().AllowAnonymousDownload && c.GetInt64("userID") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请登录后获取下载链接"})
		return
	}

	fileID, err := strconv.ParseInt(c.Param("fileID"), 10, 64)
	if err != nil || fileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}
	timeoutMinutes := config.GetShare().FileLinkTimeoutMinutes
	if timeoutMinutes <= 0 {
		timeoutMinutes = config.DefaultShare().FileLinkTimeoutMinutes
	}
	issued, err := services.IssueShareDownloadLink(
		c.Param("token"), fileID, time.Duration(timeoutMinutes)*time.Minute,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrShareFileNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	linkPath := fmt.Sprintf("/share/download/%s/%s", issued.Token, url.PathEscape(issued.Filename))
	siteLink := strings.TrimRight(config.GetBasic().SiteLink, "/")
	if siteLink != "" {
		linkPath = siteLink + linkPath
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"url":        linkPath,
		"expires_at": issued.ExpiresAt,
	})
}

// TemporaryShareFileDownload consumes a one-time token before opening the
// file. The trailing filename keeps curl and wget filename detection working.
func TemporaryShareFileDownload(c *gin.Context) {
	if !config.GetShare().AllowAnonymousDownload && c.GetInt64("userID") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请登录后下载文件"})
		return
	}

	download, err := services.ConsumeShareDownloadLink(c.Param("downloadToken"), c.Param("filename"))
	if err != nil {
		status := http.StatusInternalServerError
		message := "download failed"
		if errors.Is(err, services.ErrDownloadLinkInvalid) {
			status = http.StatusGone
			message = "下载链接无效、已使用或已过期"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	// The token is already committed as used before any filesystem access.
	fullPath, err := services.DownloadFile(download.File.FilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	filename := filepath.Base(download.File.FilePath)
	c.Header("Content-Disposition", BuildAttachmentDisposition(filename))
	c.Header("Cache-Control", "private, no-store, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	c.File(fullPath)

	services.IncrementFileDownload(download.File.ID)
	userID := c.GetInt64("userID")
	services.CreateLog(userID, models.ActionDownload, download.File.FilePath, c.ClientIP(), map[string]interface{}{
		"token":      download.ShareToken,
		"user_agent": c.Request.UserAgent(),
	})
}
