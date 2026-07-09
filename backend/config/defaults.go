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
		SenderAddress: "",
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
	}
}
