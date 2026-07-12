package middleware

import (
	"net/http"

	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("gowfm_session")
		if token == "" || err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			c.Abort()
			return
		}

		session, err := services.GetSession(token)
		if session == nil || err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired or invalid"})
			c.Abort()
			return
		}

		c.Set("userID", session.UserID)
		c.Set("sessionToken", token)

		// 被管理员要求启用 TOTP 的用户，在完成绑定前只能访问绑定所需端点。
		user, userErr := services.GetUserByID(session.UserID)
		if userErr != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}
		if user.TotpResetRequired || (user.TotpForced && !user.TotpEnabled) {
			path := c.Request.URL.Path
			allowed := path == "/api/auth/me" || path == "/api/users/me/totp/status" ||
				path == "/api/users/me/totp/setup" || path == "/api/users/me/totp/verify" ||
				(path == "/api/users/me/totp/disable" && !user.TotpForced)
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "请先完成 TOTP 绑定", "code": "TOTP_SETUP_REQUIRED"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie("gowfm_session")
		if token != "" {
			session, err := services.GetSession(token)
			if session != nil && err == nil {
				c.Set("userID", session.UserID)
				c.Set("sessionToken", token)
			}
		}
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("userID")
		user, err := services.GetUserByID(userID)
		if err != nil || !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
