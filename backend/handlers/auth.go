package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 1. 验证码校验
	sec := config.GetSecurity()
	if sec.EnableCaptcha {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
			return
		}
		if !services.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "验证码错误或已过期"})
			return
		}
	}

	ip := c.ClientIP()

	// 2. 检查 IP 封锁
	if services.GlobalBlocker.IsIPBlocked(ip) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "IP 已被临时封锁，请稍后再试"})
		return
	}

	// 3. 检查账号封锁（白名单 IP 可绕过账号封锁）
	if sec.AccountBlockEnabled && !services.GlobalBlocker.IsWhitelisted(ip, sec.WhitelistIPs) {
		if services.GlobalBlocker.IsAccountBlocked(req.Username) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "账号已被临时封锁，请稍后再试"})
			return
		}
	}

	// 4. 验证用户名
	user, err := services.GetUserByUsername(req.Username)
	if err != nil {
		time.Sleep(1 * time.Second)
		services.GlobalBlocker.RecordFailure(ip, req.Username)
		services.CreateLog(0, models.ActionLoginFail, "", ip, map[string]interface{}{"username": req.Username})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 5. 验证密码
	if !services.CheckPassword(user, req.Password) {
		time.Sleep(1 * time.Second)
		services.GlobalBlocker.RecordFailure(ip, req.Username)
		services.CreateLog(0, models.ActionLoginFail, "", ip, map[string]interface{}{"username": req.Username})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 6. 检查是否需要 TOTP 验证
	services.GlobalBlocker.ResetOnSuccess(ip, req.Username)

	if user.TotpEnabled {
		// 检查信任设备
		trustedToken, _ := c.Cookie("gowfm_trusted")
		if services.CheckTrustedDevice(user.ID, trustedToken) {
			// 信任设备 → 直接登录
			doLoginSession(c, user.ID, sec.SessionTimeout, ip)
			return
		}

		// 需要 TOTP 验证 → 生成临时登录 token
		loginToken := generateToken()
		secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		c.SetCookie("gowfm_login_token", loginToken, 300, "/", "", secure, true)

		// 存储 login_token → userID 映射（用内存 map）
		storeLoginToken(loginToken, user.ID)

		c.JSON(http.StatusOK, gin.H{
			"totp_required": true,
			"login_token":   loginToken,
		})
		return
	}

	// 无 TOTP → 直接登录
	doLoginSession(c, user.ID, sec.SessionTimeout, ip)
}

func doLoginSession(c *gin.Context, userID int64, sessionTimeout int, ip string) {
	sessionDuration := time.Duration(sessionTimeout) * time.Minute
	session, err := services.CreateSession(userID, sessionDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create session failed"})
		return
	}

	services.CreateLog(userID, models.ActionLogin, "", ip, nil)

	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("gowfm_session", session.Token, int(sessionDuration.Seconds()), "/", "", secure, true)
	c.JSON(http.StatusOK, gin.H{"message": "login successful"})
}

// ---------- TOTP 登录验证 ----------

type LoginTOTPRequest struct {
	LoginToken string `json:"login_token" binding:"required"`
	Code       string `json:"code" binding:"required"`
	TrustDevice bool  `json:"trust_device"`
}

func LoginTOTP(c *gin.Context) {
	var req LoginTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID, ok := consumeLoginToken(req.LoginToken)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "登录凭证已过期，请重新登录"})
		return
	}

	ip := c.ClientIP()

	// 先尝试 TOTP 验证码
	err := services.VerifyTOTP(userID, req.Code)
	if err != nil {
		// 再尝试恢复码
		if recErr := services.VerifyRecoveryCode(userID, req.Code); recErr != nil {
			services.CreateLog(userID, models.ActionLoginTOTPFail, "", ip, nil)
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
			return
		}
		// 恢复码登录成功
		services.CreateLog(userID, models.ActionTOTPRecovery, "", ip, nil)
	} else {
		services.CreateLog(userID, models.ActionLoginTOTP, "", ip, nil)
	}

	// 信任设备
	if req.TrustDevice {
		deviceToken, err := services.CreateTrustedDevice(userID, c.GetHeader("User-Agent"))
		if err == nil {
			trustDays := config.GetSecurity().TotpTrustDays
			if trustDays <= 0 {
				trustDays = 30
			}
			secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
			c.SetCookie("gowfm_trusted", deviceToken, trustDays*86400, "/", "", secure, true)
		}
	}

	sec := config.GetSecurity()
	doLoginSession(c, userID, sec.SessionTimeout, ip)
}

// ---------- 临时登录 token（内存存储） ----------

var loginTokens = map[string]loginTokenEntry{}

type loginTokenEntry struct {
	UserID    int64
	CreatedAt time.Time
}

func storeLoginToken(token string, userID int64) {
	loginTokens[token] = loginTokenEntry{UserID: userID, CreatedAt: time.Now()}
	// 清理过期 token（5 分钟）
	for k, v := range loginTokens {
		if time.Since(v.CreatedAt) > 5*time.Minute {
			delete(loginTokens, k)
		}
	}
}

func consumeLoginToken(token string) (int64, bool) {
	entry, ok := loginTokens[token]
	if !ok {
		return 0, false
	}
	if time.Since(entry.CreatedAt) > 5*time.Minute {
		delete(loginTokens, token)
		return 0, false
	}
	delete(loginTokens, token)
	return entry.UserID, true
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func GetCaptcha(c *gin.Context) {
	sec := config.GetSecurity()
	if !sec.EnableCaptcha {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}

	captchaID, _, pngBase64 := services.GenerateCaptcha()

	c.JSON(http.StatusOK, gin.H{
		"enabled":       true,
		"captcha_id":    captchaID,
		"captcha_image": "data:image/png;base64," + pngBase64,
	})
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
		"totp_enabled": user.TotpEnabled,
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
