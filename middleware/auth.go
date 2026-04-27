package middleware

import (
	"net/http"

	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("wfm_session")
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
		c.Next()
	}
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie("wfm_session")
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