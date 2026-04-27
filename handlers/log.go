package handlers

import (
	"net/http"
	"strconv"

	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func ListLogs(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	if !user.HasPermission(models.PermManageLogs) {
		c.JSON(http.StatusForbidden, gin.H{"error": "log access denied"})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	filterUserID := c.Query("user_id")
	action := c.Query("action")
	targetPath := c.Query("target_path")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 200 { pageSize = 50 }

	logs, total, err := services.QueryLogs(startTime, endTime, filterUserID, action, targetPath, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query logs failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}