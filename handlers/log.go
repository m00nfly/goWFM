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
	action := c.Query("action")
	targetPath := c.Query("target_path")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	// 支持按用户名过滤：将 username 转换为 user_id
	filterUserID := c.Query("user_id")
	if username := c.Query("username"); username != "" {
		if u, err := services.GetUserByUsername(username); err == nil && u != nil {
			filterUserID = strconv.FormatInt(u.ID, 10)
		}
	}

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

// ListUsersForLog 返回用户列表（id + username），供日志过滤器使用。
// 需要 PermManageLogs 权限，无需 admin。
func ListUsersForLog(c *gin.Context) {
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

	users, err := services.ListAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list users failed"})
		return
	}

	// 仅返回日志过滤所需的最小属性
	type userItem struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	result := make([]userItem, 0, len(users))
	for _, u := range users {
		result = append(result, userItem{
			ID:       u["id"].(int64),
			Username: u["username"].(string),
		})
	}
	c.JSON(http.StatusOK, result)
}
