package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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

	share, err := services.CreateShare(req.FilePaths, user.ID, req.ExpireDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create share failed"})
		return
	}

	pathsJSON, _ := json.Marshal(req.FilePaths)
	services.CreateLog(user.ID, models.ActionShareCreate, string(pathsJSON), c.ClientIP(), map[string]interface{}{"token": share.Token})

	link := fmt.Sprintf("%s/share/%s", config.C.OrgLink, share.Token)
	c.JSON(http.StatusOK, gin.H{
		"id":         share.ID,
		"token":      share.Token,
		"link":       link,
		"expire_at":  share.ExpireAt,
		"created_at": share.CreatedAt,
	})
}

func ListMyShares(c *gin.Context) {
	userID := c.GetInt64("userID")
	shares, err := services.ListMyShares(userID)
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

	if err := services.DeleteShare(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete share failed"})
		return
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionShareDelete, "", c.ClientIP(), map[string]interface{}{"share_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "share deleted"})
}

func ListAllShares(c *gin.Context) {
	shares, err := services.ListAllShares()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shares)
}

func ListShareUsers(c *gin.Context) {
	users, err := services.ListShareUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func GetShareInfo(c *gin.Context) {
	token := c.Param("token")

	share, err := services.ValidateShareAccess(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 每次访问分享页面，递增访问计数
	services.IncrementShareAccess(token)

	shareFiles, err := services.GetShareFiles(share.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get share files failed"})
		return
	}

	type fileInfo struct {
		FileName      string `json:"file_name"`
		FileSize      int64  `json:"file_size"`
		FilePath      string `json:"file_path"`
		DownloadCount int    `json:"download_count"`
	}

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
		files = append(files, fileInfo{
			FileName:      filepath.Base(fullPath),
			FileSize:      info.Size(),
			FilePath:      f.FilePath,
			DownloadCount: f.DownloadCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
	})
}

// AccessShareEntry serves the SPA index.html for all share access.
// The frontend handles token validation via /share/:token/info API.
func AccessShareEntry(c *gin.Context) {
	serveIndexHTML(c)
}

// ShareFileDownload handles file download via /share/:token/:filename.
// The filename in URL allows wget/curl to save with the correct name.
func ShareFileDownload(c *gin.Context) {
	token := c.Param("token")
	filename := c.Param("filename")

	// 1. Validate token
	share, err := services.ValidateShareAccess(token)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 2. Get share files and find matching filename
	shareFiles, err := services.GetShareFiles(share.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get share files failed"})
		return
	}

	var matchedFile *models.ShareFile
	for i, f := range shareFiles {
		if filepath.Base(f.FilePath) == filename {
			matchedFile = &shareFiles[i]
			break
		}
	}
	if matchedFile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	// 3. Get full file path
	fullPath, err := services.DownloadFile(matchedFile.FilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// 4. Set Content-Disposition for browser download
	c.Header("Content-Disposition", BuildAttachmentDisposition(filename))

	// 5. Serve file
	c.File(fullPath)

	// 6. Increment access count + file download count + audit log
	services.IncrementShareAccess(token)
	services.IncrementFileDownload(matchedFile.ID)
	services.CreateLog(0, models.ActionShareAccess, matchedFile.FilePath, c.ClientIP(), map[string]interface{}{
		"token": token,
		"ip":    c.ClientIP(),
	})
}
