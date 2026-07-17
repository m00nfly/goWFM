package config

// DefaultBasic 返回基础设置默认值
func DefaultBasic() BasicSettings {
	return BasicSettings{
		SiteName:      "",
		SiteLink:      "",
		DataRootPath:  "",
		MaxUploadSize: 1073741824, // 1 GB
	}
}

// DefaultSecurity 返回安全设置默认值
func DefaultSecurity() SecuritySettings {
	return SecuritySettings{
		SessionSecret:        RandomSecret(),
		SessionTimeout:       720, // 12小时 = 12*60 分钟
		EnableCaptcha:        false,
		CaptchaCodeLength:    6,
		IPBlockEnabled:       false,
		IPBlockMaxFailures:   5,
		IPBlockWindow:        300,  // 5分钟
		IPBlockDuration:      1800, // 30分钟
		AccountBlockEnabled:  false,
		AccountBlockMaxFails: 5,
		AccountBlockWindow:   300,
		AccountBlockDuration: 1800,
		WhitelistIPs:         []string{},
		TotpTrustDays:        30, // 信任设备默认 30 天
	}
}

// DefaultLog 返回日志设置默认值
func DefaultLog() LogSettings {
	return LogSettings{
		RetentionDays: 30,
		EnabledLogTypes: []string{
			"LOGIN", "LOGIN_FAIL", "BLOCK_IP", "BLOCK_ACCOUNT",
			"CREATE_DIR", "UPLOAD", "DOWNLOAD", "DELETE_FILE", "DELETE_DIR",
			"SHARE_CREATE", "SHARE_ACCESS", "SHARE_DELETE",
			"CHANGE_OWNER", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "MOVE",
			"CONFIG_CHANGE",
			"PASSWORD_RESET_REQUEST", "PASSWORD_RESET_COMPLETE",
		},
	}
}

// DefaultEmail 返回邮件设置默认值
func DefaultEmail() EmailSettings {
	return EmailSettings{
		SMTPHost:      "",
		SMTPPort:      587,
		SMTPUsername:  "",
		SMTPPassword:  "",
		EnableTLS:     true,
		SkipTLSVerify: false,
		SenderName:    "",
		SenderEmail:   "",
		Templates: map[string]EmailTemplate{
			"reset_password": DefaultResetPasswordTemplate(),
		},
	}
}

// DefaultResetPasswordTemplate 返回密码重置邮件的默认模板。
func DefaultResetPasswordTemplate() EmailTemplate {
	return EmailTemplate{
		Subject: "重置您的 {{.SiteName}} 密码",
		HTML: `<!doctype html>
<html lang="zh-CN">
<body style="margin:0;background:#f4f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033">
  <div style="max-width:560px;margin:0 auto;padding:40px 20px">
    <div style="background:#ffffff;border-radius:16px;padding:32px;box-shadow:0 12px 36px rgba(28,45,72,.10)">
      <h1 style="font-size:22px;margin:0 0 18px">重置密码</h1>
      <p>您好，{{.Username}}：</p>
      <p>我们收到了您的密码重置请求。请在 {{.ExpiresMinutes}} 分钟内点击下方按钮完成重置。</p>
      <p style="margin:28px 0"><a href="{{.ResetURL}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;padding:12px 22px;border-radius:8px">重置密码</a></p>
      <p style="font-size:13px;color:#667085">若您未发起此请求，请忽略本邮件。此链接仅可使用一次。</p>
    </div>
  </div>
</body>
</html>`,
	}
}

// DefaultAppearance 返回外观设置默认值
func DefaultAppearance() AppearanceSettings {
	return AppearanceSettings{
		LoginBgURL:   "",
		DefaultTheme: "light",
		ThemeColor:   "#3B82F6",
		CustomLogo:   "",
		ServerPort:   8080,
		EnableHTTPS:  false,
		SSLCert:      "",
		SSLKey:       "",
	}
}

// DefaultShare 返回分享设置默认值
func DefaultShare() ShareSettings {
	return ShareSettings{
		DefaultExpireDays:      7,
		MaxSharesPerUser:       0, // 0 表示无限制
		AllowAnonymousDownload: true,
		FileLinkTimeoutMinutes: 5,
	}
}
