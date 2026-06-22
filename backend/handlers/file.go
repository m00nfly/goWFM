package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func ListFiles(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermBrowse) {
		c.JSON(http.StatusForbidden, gin.H{"error": "browse permission denied"})
		return
	}

	relativePath := c.Query("path")
	if relativePath == "" {
		relativePath = "/"
	}

	entries, err := services.ListDirectory(relativePath, user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []services.FileEntry{}
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    relativePath,
		"entries": entries,
	})
}

func UploadFile(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermUpload) {
		c.JSON(http.StatusForbidden, gin.H{"error": "upload permission denied"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}

	if file.Size > config.GetBasic().MaxUploadSize {
		maxMB := config.GetBasic().MaxUploadSize / 1024 / 1024
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file too large, max size: %d MB", maxMB)})
		return
	}

	targetDir := c.PostForm("path")
	if targetDir == "" {
		targetDir = "/"
	}

	fullDir, err := services.SafePath(targetDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create target directory failed"})
		return
	}

	filename := services.SanitizeFilename(file.Filename)
	fullPath := filepath.Join(fullDir, filename)
	fullPath = services.ResolveDuplicatePath(fullPath)

	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
		return
	}

	relativePath := services.RelativePath(fullPath)
	services.CreateFileMetadata(relativePath, false, user.ID)
	services.CreateLog(user.ID, models.ActionUpload, relativePath, c.ClientIP(), map[string]interface{}{"size": file.Size})

	c.JSON(http.StatusOK, gin.H{"message": "upload successful", "path": relativePath})
}

func CreateDir(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermUpload) {
		c.JSON(http.StatusForbidden, gin.H{"error": "upload permission denied"})
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	relativePath := filepath.Join(req.Path, services.SanitizeFilename(req.Name))
	if err := services.CreateDirectory(relativePath, user.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.CreateLog(user.ID, models.ActionCreateDir, relativePath, c.ClientIP(), nil)
	c.JSON(http.StatusOK, gin.H{"message": "directory created", "path": relativePath})
}

func DeleteFile(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.DeleteFileOrDir(req.Path, user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := models.ActionDeleteFile
	dataRoot := config.GetBasic().DataRootPath
	if info, e := os.Stat(dataRoot + req.Path); e == nil && info.IsDir() {
		action = models.ActionDeleteDir
	}
	services.CreateLog(user.ID, action, req.Path, c.ClientIP(), nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func Download(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermDownload) {
		c.JSON(http.StatusForbidden, gin.H{"error": "download permission denied"})
		return
	}

	relativePath := c.Query("path")
	fullPath, err := services.DownloadFile(relativePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filename := filepath.Base(fullPath)
	c.Header("Content-Disposition", BuildAttachmentDisposition(filename))
	c.File(fullPath)

	services.CreateLog(user.ID, models.ActionDownload, relativePath, c.ClientIP(), nil)
}

func ChangeOwner(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req struct {
		Path    string `json:"path" binding:"required"`
		OwnerID int64  `json:"owner_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateFileMetadataOwner(req.Path, req.OwnerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "change owner failed"})
		return
	}

	services.CreateLog(user.ID, models.ActionChangeOwner, req.Path, c.ClientIP(), map[string]interface{}{"new_owner_id": req.OwnerID})
	c.JSON(http.StatusOK, gin.H{"message": "owner changed"})
}

func MoveFile(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	var req struct {
		Source      string `json:"source" binding:"required"`
		Destination string `json:"destination" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	srcFullPath, err := services.SafePath(req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dstFullPath, err := services.SafePath(req.Destination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dstFullPath = services.ResolveDuplicatePath(dstFullPath)

	if err := os.Rename(srcFullPath, dstFullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "move failed: " + err.Error()})
		return
	}

	services.UpdateFileMetadataPath(req.Source, services.RelativePath(dstFullPath))
	services.CreateLog(user.ID, models.ActionMove, req.Source, c.ClientIP(), map[string]interface{}{"destination": services.RelativePath(dstFullPath)})
	c.JSON(http.StatusOK, gin.H{"message": "moved", "new_path": services.RelativePath(dstFullPath)})
}
