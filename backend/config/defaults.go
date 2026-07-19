package config

import "strings"

const DefaultEmailFooterHTML = `
    <p style="margin:18px 0 0;font-size:12px;color:#98a2b3;text-align:center">{{.SiteName}} | Powered by <a href="https://gowfm.dev" style="color:#667085;text-decoration:none" target="_blank" rel="noopener noreferrer">{{.PoweredBy}}</a></p>`

const legacyInlineEmailFooterHTML = `
      <p style="margin:28px 0 0;padding-top:18px;border-top:1px solid #e4e7ec;font-size:12px;color:#98a2b3;text-align:center">{{.SiteName}} | <a href="https://gowfm.dev" style="color:#667085;text-decoration:none" target="_blank" rel="noopener noreferrer">{{.PoweredBy}}</a></p>`

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
		SessionSecret:           RandomSecret(),
		SessionTimeout:          720, // 12小时 = 12*60 分钟
		EnableCaptcha:           false,
		CaptchaCodeLength:       6,
		IPBlockEnabled:          false,
		IPBlockMaxFailures:      5,
		IPBlockWindow:           300,  // 5分钟
		IPBlockDuration:         1800, // 30分钟
		AccountBlockEnabled:     false,
		AccountBlockMaxFails:    5,
		AccountBlockWindow:      300,
		AccountBlockDuration:    1800,
		WhitelistIPs:            []string{},
		TotpTrustDays:           30, // 信任设备默认 30 天
		AllowEmailPasswordReset: false,
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
		Active:        false,
		SMTPHost:      "",
		SMTPPort:      587,
		SMTPUsername:  "",
		SMTPPassword:  "",
		EnableTLS:     true,
		SkipTLSVerify: false,
		SenderName:    "",
		SenderEmail:   "",
		Templates: map[string]EmailTemplate{
			"reset_password":     DefaultResetPasswordTemplate(),
			"share_notification": DefaultShareNotificationTemplate(),
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
` + DefaultEmailFooterHTML + `
  </div>
</body>
</html>`,
	}
}

// DefaultShareNotificationTemplate 返回分享链接邮件的默认模板。
func DefaultShareNotificationTemplate() EmailTemplate {
	return EmailTemplate{
		Subject: "{{.Sharer}} 向您分享了「{{.ShareName}}」",
		HTML: `<!doctype html>
<html lang="zh-CN">
<body style="margin:0;background:#f4f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033">
  <div style="max-width:560px;margin:0 auto;padding:40px 20px">
    <div style="background:#ffffff;border-radius:16px;padding:32px;box-shadow:0 12px 36px rgba(28,45,72,.10)">
      <h1 style="font-size:22px;margin:0 0 18px">文件分享通知</h1>
      <p><strong>{{.Sharer}}</strong> 向您发送了一个文件分享。</p>
      <p>分享名称：{{.ShareName}}</p>
      <p>文件数量：{{.FileCount}}</p>
      <p style="margin:28px 0"><a href="{{.ShareURL}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;padding:12px 22px;border-radius:8px">访问分享</a></p>
      <p style="font-size:13px;color:#667085;word-break:break-all">分享访问链接：{{.ShareURL}}</p>
    </div>
` + DefaultEmailFooterHTML + `
  </div>
</body>
</html>`,
	}
}

// UpgradeBuiltinEmailTemplates 将未修改的旧版内置模板升级到当前默认版本，
// 已由管理员自定义的模板保持原样。
func UpgradeBuiltinEmailTemplates(templates map[string]EmailTemplate) {
	defaults := map[string]EmailTemplate{
		"reset_password":     DefaultResetPasswordTemplate(),
		"share_notification": DefaultShareNotificationTemplate(),
	}
	for key, current := range templates {
		latest, ok := defaults[key]
		if !ok {
			continue
		}
		withoutFooter := latest
		withoutFooter.HTML = strings.Replace(withoutFooter.HTML, DefaultEmailFooterHTML, "", 1)
		withoutFooter.HTML = strings.Replace(withoutFooter.HTML, "\n\n  </div>", "\n  </div>", 1)
		inlineFooter := withoutFooter
		inlineFooter.HTML = strings.Replace(
			inlineFooter.HTML,
			"    </div>\n  </div>",
			legacyInlineEmailFooterHTML+"\n    </div>\n  </div>",
			1,
		)
		if current == withoutFooter || current == inlineFooter {
			templates[key] = latest
		}
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
		AllowEmailShare:        false,
	}
}

// DefaultScan 返回磁盘扫描默认设置。默认不启用后台定时扫描。
func DefaultScan() ScanSettings {
	return ScanSettings{AutoScanEnabled: false, IntervalHours: 1}
}
