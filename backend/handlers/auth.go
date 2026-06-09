package handlers

import (
	"net/http"
	"time"

	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := services.GetUserByUsername(req.Username)
	if err != nil {
		time.Sleep(1 * time.Second)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if !services.CheckPassword(user, req.Password) {
		time.Sleep(1 * time.Second)
		services.CreateLog(0, models.ActionLoginFail, "", c.ClientIP(), map[string]interface{}{"username": req.Username})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	sessionDuration := 7 * 24 * time.Hour
	session, err := services.CreateSession(user.ID, sessionDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create session failed"})
		return
	}

	services.CreateLog(user.ID, models.ActionLogin, "", c.ClientIP(), nil)

	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("gowfm_session", session.Token, int(sessionDuration.Seconds()), "/", "", secure, true)
	c.JSON(http.StatusOK, gin.H{"message": "login successful"})
}

func Logout(c *gin.Context) {
	token, _ := c.Cookie("gowfm_session")
	if token != "" {
		services.DeleteSession(token)
	}
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("gowfm_session", "", -1, "/", "", secure, true)
	c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

func GetMe(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get user failed"})
		return
	}

	result := gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"email":        user.Email,
		"is_admin":     user.IsAdmin,
		"permissions":  user.Permissions,
	}

	// Include share stats if user has share permission
	if user.IsAdmin || (user.Permissions&8) != 0 {
		ownerID := user.ID
		if user.IsAdmin {
			ownerID = 0 // admin sees all shares
		}
		expired, valid, err := services.GetShareStats(ownerID)
		if err == nil {
			result["share_stats"] = gin.H{
				"expired": expired,
				"valid":   valid,
			}
		}
	}

	c.JSON(http.StatusOK, result)
}
