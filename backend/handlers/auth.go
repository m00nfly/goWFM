package handlers

import (
	"net/http"
	"time"

	"goWFM/config"
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

	sec := config.GetSecurity()
	ip := c.ClientIP()

	// 1. 检查 IP 封锁
	if services.GlobalBlocker.IsIPBlocked(ip) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "IP 已被临时封锁，请稍后再试"})
		return
	}

	// 2. 检查账号封锁（白名单 IP 可绕过账号封锁）
	if sec.AccountBlockEnabled && !services.GlobalBlocker.IsWhitelisted(ip, sec.WhitelistIPs) {
		if services.GlobalBlocker.IsAccountBlocked(req.Username) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "账号已被临时封锁，请稍后再试"})
			return
		}
	}

	// 3. 验证用户名
	user, err := services.GetUserByUsername(req.Username)
	if err != nil {
		time.Sleep(1 * time.Second)
		services.GlobalBlocker.RecordFailure(ip, req.Username)
		services.CreateLog(0, models.ActionLoginFail, "", ip, map[string]interface{}{"username": req.Username})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 4. 验证密码
	if !services.CheckPassword(user, req.Password) {
		time.Sleep(1 * time.Second)
		services.GlobalBlocker.RecordFailure(ip, req.Username)
		services.CreateLog(0, models.ActionLoginFail, "", ip, map[string]interface{}{"username": req.Username})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 5. 登录成功
	services.GlobalBlocker.ResetOnSuccess(ip, req.Username)

	sessionDuration := time.Duration(sec.SessionTimeout) * time.Minute
	session, err := services.CreateSession(user.ID, sessionDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create session failed"})
		return
	}

	services.CreateLog(user.ID, models.ActionLogin, "", ip, nil)

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
