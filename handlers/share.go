package handlers

import (
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
		FilePath   string `json:"file_path" binding:"required"`
		ExpireDays int    `json:"expire_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = services.SafePath(req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := services.DownloadFile(req.FilePath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not found or is directory"})
		return
	}

	share, err := services.CreateShare(req.FilePath, user.ID, req.ExpireDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create share failed"})
		return
	}

	services.CreateLog(user.ID, models.ActionShareCreate, req.FilePath, c.ClientIP(), map[string]interface{}{"token": share.Token})

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

func GetShareInfo(c *gin.Context) {
	token := c.Param("token")

	share, err := services.ValidateShareAccess(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fullPath, err := services.DownloadFile(share.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file stat failed"})
		return
	}

	filename := filepath.Base(fullPath)
	c.JSON(http.StatusOK, gin.H{
		"file_name": filename,
		"file_size": info.Size(),
	})
}

func AccessShare(c *gin.Context) {
	token := c.Param("token")

	share, err := services.ValidateShareAccess(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fullPath, err := services.DownloadFile(share.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.IncrementShareAccess(token)
	services.CreateLog(share.OwnerID, models.ActionShareAccess, share.FilePath, c.ClientIP(), map[string]interface{}{"token": token})

	filename := fullPath[len(config.C.DataRootPath):]
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.File(fullPath)
}
