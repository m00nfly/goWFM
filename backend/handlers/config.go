package handlers

import (
	"encoding/json"
	"net/http"

	"goWFM/config"
	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

// categoryToKey 将 URL 中的 category 参数映射到配置 key
func categoryToKey(category string) string {
	switch category {
	case "basic":
		return config.KeyBasic
	case "security":
		return config.KeySecurity
	case "log":
		return config.KeyLog
	case "email":
		return config.KeyEmail
	case "appearance":
		return config.KeyAppearance
	case "share":
		return config.KeyShare
	default:
		return ""
	}
}

// needsRestart 判断更新某个分类是否需要重启
func needsRestart(category string, oldData, newData string) bool {
	switch category {
	case "appearance":
		var oldApp, newApp config.AppearanceSettings
		json.Unmarshal([]byte(oldData), &oldApp)
		json.Unmarshal([]byte(newData), &newApp)
		return oldApp.ServerPort != newApp.ServerPort ||
			oldApp.EnableHTTPS != newApp.EnableHTTPS ||
			oldApp.SSLCert != newApp.SSLCert ||
			oldApp.SSLKey != newApp.SSLKey
	case "basic":
		var oldBasic, newBasic config.BasicSettings
		json.Unmarshal([]byte(oldData), &oldBasic)
		json.Unmarshal([]byte(newData), &newBasic)
		return oldBasic.DataRootPath != newBasic.DataRootPath
	default:
		return false
	}
}

// GetConfig 获取指定分类的配置
func GetConfig(c *gin.Context) {
	category := c.Param("category")
	key := categoryToKey(category)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}

	var data interface{}
	switch category {
	case "basic":
		data = config.GetBasic()
	case "security":
		cfg := config.GetSecurity()
		// 不返回 session_secret
		data = gin.H{
			"session_timeout":            cfg.SessionTimeout,
			"enable_captcha":             cfg.EnableCaptcha,
			"captcha_code_length":        cfg.CaptchaCodeLength,
			"ip_block_enabled":           cfg.IPBlockEnabled,
			"ip_block_max_failures":      cfg.IPBlockMaxFailures,
			"ip_block_window":            cfg.IPBlockWindow,
			"ip_block_duration":          cfg.IPBlockDuration,
			"account_block_enabled":      cfg.AccountBlockEnabled,
			"account_block_max_failures": cfg.AccountBlockMaxFails,
			"account_block_window":       cfg.AccountBlockWindow,
			"account_block_duration":     cfg.AccountBlockDuration,
			"whitelist_ips":              cfg.WhitelistIPs,
			"totp_trust_days":            cfg.TotpTrustDays,
		}
	case "log":
		data = config.GetLog()
	case "email":
		cfg := config.GetEmail()
		senderName := services.EffectiveSenderName(cfg.SenderName)
		// 密码不返回明文，只告知是否已配置
		data = gin.H{
			"smtp_host":       cfg.SMTPHost,
			"smtp_port":       cfg.SMTPPort,
			"smtp_username":   cfg.SMTPUsername,
			"has_password":    cfg.SMTPPassword != "",
			"enable_tls":      cfg.EnableTLS,
			"skip_tls_verify": cfg.SkipTLSVerify,
			"sender_name":     senderName,
			"sender_email":    cfg.SenderEmail,
			"templates":       cfg.Templates,
		}
	case "appearance":
		cfg := config.GetAppearance()
		// 不返回 SSL 证书和私钥完整内容
		data = gin.H{
			"login_bg_url":  cfg.LoginBgURL,
			"default_theme": cfg.DefaultTheme,
			"theme_color":   cfg.ThemeColor,
			"custom_logo":   cfg.CustomLogo,
			"server_port":   cfg.ServerPort,
			"enable_https":  cfg.EnableHTTPS,
			"has_ssl_cert":  cfg.SSLCert != "",
			"has_ssl_key":   cfg.SSLKey != "",
		}
	case "share":
		data = config.GetShare()
	}

	c.JSON(http.StatusOK, data)
}

// UpdateConfig 更新指定分类的配置
func UpdateConfig(c *gin.Context) {
	category := c.Param("category")
	key := categoryToKey(category)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}

	// 获取旧值用于判断是否需要重启
	oldValue, _ := services.GetConfigValue(key)

	var err error
	var newData []byte

	switch category {
	case "basic":
		var s config.BasicSettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateBasicSettings(s)
	case "appearance":
		var s config.AppearanceSettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		// 如果前端没有提交 ssl_cert/ssl_key 则保留原值
		current := config.GetAppearance()
		if s.SSLCert == "" {
			s.SSLCert = current.SSLCert
		}
		if s.SSLKey == "" {
			s.SSLKey = current.SSLKey
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateAppearanceSettings(s)
	case "security":
		var s config.SecuritySettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		// 保留现有 session_secret
		current := config.GetSecurity()
		if s.SessionSecret == "" {
			s.SessionSecret = current.SessionSecret
		}
		if s.WhitelistIPs == nil {
			s.WhitelistIPs = []string{}
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateSecuritySettings(s)
	case "log":
		var s config.LogSettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if s.EnabledLogTypes == nil {
			s.EnabledLogTypes = []string{}
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateLogSettings(s)
	case "email":
		var s config.EmailSettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		current := config.GetEmail()
		// 如果密码为空字符串则保留原密码
		if s.SMTPPassword == "" {
			s.SMTPPassword = current.SMTPPassword
		}
		if s.Templates == nil {
			s.Templates = current.Templates
		}
		if s.SenderEmail == "" {
			s.SenderEmail = s.SenderAddress
		}
		if err := services.ValidateEmailSettings(s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateEmailSettings(s)
	case "share":
		var s config.ShareSettings
		if err := c.ShouldBindJSON(&s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		newData, _ = json.Marshal(s)
		err = services.UpdateShareSettings(s)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save config failed"})
		return
	}

	// 记录配置变更日志
	userID := c.GetInt64("userID")
	services.CreateLog(userID, models.ActionConfigChange, category, c.ClientIP(), nil)

	restart := needsRestart(category, oldValue, string(newData))
	c.JSON(http.StatusOK, gin.H{
		"message":          "配置已更新",
		"restart_required": restart,
	})
}

// TestEmailSettings 使用已保存的 SMTP 配置向指定邮箱发送测试邮件。
func TestEmailSettings(c *gin.Context) {
	recipient := config.GetEmail().SenderEmail
	if err := services.SendTestEmail(recipient); err != nil {
		result := gin.H{"error": err.Error()}
		if code, message, ok := services.SMTPErrorDetails(err); ok {
			result["smtp_code"] = code
			result["smtp_message"] = message
		}
		c.JSON(http.StatusBadGateway, result)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送", "recipient": recipient})
}

// GetConfigInfo 返回公开配置信息（无需登录）
func GetConfigInfo(c *gin.Context) {
	basicCfg := config.GetBasic()
	appearanceCfg := config.GetAppearance()
	securityCfg := config.GetSecurity()

	c.JSON(http.StatusOK, gin.H{
		"site_name":       basicCfg.SiteName,
		"site_link":       basicCfg.SiteLink,
		"version":         config.Version,
		"login_bg_url":    appearanceCfg.LoginBgURL,
		"default_theme":   appearanceCfg.DefaultTheme,
		"theme_color":     appearanceCfg.ThemeColor,
		"custom_logo":     appearanceCfg.CustomLogo,
		"enable_captcha":  securityCfg.EnableCaptcha,
		"totp_trust_days": securityCfg.TotpTrustDays,
	})
}
