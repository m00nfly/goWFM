package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

const passwordResetGenericMessage = "如果该邮箱对应有效账户，重置邮件将在稍后送达"

type resetRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var passwordResetLimiter = resetRateLimiter{attempts: make(map[string][]time.Time)}

func (l *resetRateLimiter) allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-window)
	if len(l.attempts) > 10000 {
		for existingKey, existingItems := range l.attempts {
			fresh := existingItems[:0]
			for _, item := range existingItems {
				if item.After(cutoff) {
					fresh = append(fresh, item)
				}
			}
			if len(fresh) == 0 {
				delete(l.attempts, existingKey)
			} else {
				l.attempts[existingKey] = fresh
			}
		}
		if len(l.attempts) > 10000 {
			if _, exists := l.attempts[key]; !exists {
				return false
			}
		}
	}
	items := l.attempts[key]
	kept := items[:0]
	for _, item := range items {
		if item.After(cutoff) {
			kept = append(kept, item)
		}
	}
	if len(kept) >= limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

func resetIdentifierKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的邮箱地址"})
		return
	}
	if config.GetSecurity().EnableCaptcha {
		if req.CaptchaID == "" || req.CaptchaCode == "" || !services.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误或已过期"})
			return
		}
	}
	started := time.Now()

	// IP 与邮箱标识分别限流。被限流时仍返回统一响应，避免泄露账户是否存在。
	allowed := passwordResetLimiter.allow("ip:"+c.ClientIP(), 5, time.Hour) &&
		passwordResetLimiter.allow("email:"+resetIdentifierKey(req.Email), 3, time.Hour)
	if !allowed {
		respondPasswordResetAccepted(c, started)
		return
	}
	user, err := services.GetUserByEmail(req.Email)
	if err != nil {
		respondPasswordResetAccepted(c, started)
		return
	}
	token, err := services.CreatePasswordResetToken(user.ID, c.ClientIP())
	if err != nil {
		log.Printf("create password reset token for user %d: %v", user.ID, err)
		respondPasswordResetAccepted(c, started)
		return
	}
	username := user.Username
	email := user.Email
	userID := user.ID
	ip := c.ClientIP()
	go func() {
		if err := services.SendResetPasswordEmail(email, username, token, int(services.PasswordResetExpiry/time.Minute)); err != nil {
			services.InvalidatePasswordResetToken(token)
			log.Printf("send password reset email for user %d: %v", userID, err)
			return
		}
		services.CreateLog(userID, models.ActionPasswordResetRequest, "", ip, nil)
	}()
	respondPasswordResetAccepted(c, started)
}

func respondPasswordResetAccepted(c *gin.Context, started time.Time) {
	// 拉齐存在与不存在账户的常见响应时间，降低基于时延的邮箱枚举风险。
	if remaining := 250*time.Millisecond - time.Since(started); remaining > 0 {
		time.Sleep(remaining)
	}
	c.JSON(http.StatusAccepted, gin.H{"message": passwordResetGenericMessage})
}

func PasswordResetStatus(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重置链接无效或已过期"})
		return
	}
	status, err := services.GetPasswordResetStatus(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重置链接无效或已过期"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "totp_required": status.RequiresTOTP, "expires_at": status.ExpiresAt})
}

func ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
		TOTPCode    string `json:"totp_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少需要 6 位"})
		return
	}
	if !passwordResetLimiter.allow("complete-ip:"+c.ClientIP(), 20, 15*time.Minute) ||
		!passwordResetLimiter.allow("complete-token:"+resetIdentifierKey(req.Token), 8, 15*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "验证尝试次数过多，请稍后重新申请重置链接"})
		return
	}
	userID, err := services.CompletePasswordReset(req.Token, req.NewPassword, strings.TrimSpace(req.TOTPCode))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrResetTOTPRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "请输入 TOTP 验证码", "code": "TOTP_REQUIRED"})
		case errors.Is(err, services.ErrResetTOTPInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP 验证码错误", "code": "TOTP_INVALID"})
		case errors.Is(err, services.ErrResetTokenInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "重置链接无效、已过期或已使用"})
		default:
			log.Printf("complete password reset: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码重置失败，请稍后重试"})
		}
		return
	}
	services.CreateLog(userID, models.ActionPasswordResetComplete, "", c.ClientIP(), nil)
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置，请使用新密码登录"})
}
