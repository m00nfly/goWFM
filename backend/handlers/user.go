package handlers

import (
	"net/http"
	"strconv"

	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func ListUsers(c *gin.Context) {
	rows, err := services.ListAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list users failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Permissions int    `json:"permissions"`
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.CreateUser(req.Username, req.Password, req.DisplayName, req.Email, false, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed: " + err.Error()})
		return
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionUserCreate, "", c.ClientIP(), map[string]interface{}{"username": req.Username})
	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"display_name": user.DisplayName,
		"email":       user.Email,
		"is_admin":    user.IsAdmin,
		"permissions": user.Permissions,
	})
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Permissions int    `json:"permissions"`
	IsAdmin     *bool  `json:"is_admin"`
}

func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if id == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system Guest account"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.IsAdmin && req.IsAdmin != nil && !*req.IsAdmin {
		adminCount, _ := services.AdminCount()
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the last admin"})
			return
		}
	}

	isAdmin := user.IsAdmin
	if req.IsAdmin != nil {
		isAdmin = *req.IsAdmin
	}

	_, err = services.UpdateUserFields(id, req.DisplayName, req.Email, isAdmin, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update user failed"})
		return
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionUserUpdate, "", c.ClientIP(), map[string]interface{}{"target_user_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if id == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system Guest account"})
		return
	}

	user, err := services.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.IsAdmin {
		adminCount, _ := services.AdminCount()
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last admin"})
			return
		}
	}

	if err := services.DeleteUserByID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete user failed"})
		return
	}

	services.DeleteAllUserSessions(id)
	services.CreateLog(c.GetInt64("userID"), models.ActionUserDelete, "", c.ClientIP(), map[string]interface{}{"target_user_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

type UpdateMeRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func UpdateMe(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	_, err = services.UpdateUserFields(userID, req.DisplayName, req.Email, user.IsAdmin, user.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	if !services.CheckPassword(user, req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
		return
	}

	if err := services.UpdateUserPassword(userID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update password failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}