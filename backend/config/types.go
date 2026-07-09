package config

// BasicSettings 基础设置
type BasicSettings struct {
	SiteName      string `json:"site_name"`
	SiteLink      string `json:"site_link"`
	DataRootPath  string `json:"data_root_path"`
	MaxUploadSize int64  `json:"max_upload_size"` // 字节
}

// SecuritySettings 安全设置
type SecuritySettings struct {
	SessionSecret        string   `json:"session_secret"`
	SessionTimeout       int      `json:"session_timeout"` // 分钟
	EnableCaptcha        bool     `json:"enable_captcha"`
	CaptchaCodeLength    int      `json:"captcha_code_length"`
	IPBlockEnabled       bool     `json:"ip_block_enabled"`
	IPBlockMaxFailures   int      `json:"ip_block_max_failures"`
	IPBlockWindow        int      `json:"ip_block_window"`   // 秒
	IPBlockDuration      int      `json:"ip_block_duration"` // 秒
	AccountBlockEnabled  bool     `json:"account_block_enabled"`
	AccountBlockMaxFails int      `json:"account_block_max_failures"`
	AccountBlockWindow   int      `json:"account_block_window"`   // 秒
	AccountBlockDuration int      `json:"account_block_duration"` // 秒
	WhitelistIPs         []string `json:"whitelist_ips"`
	TotpTrustDays        int      `json:"totp_trust_days"` // TOTP 信任设备天数，0 表示不信任设备
}

// LogSettings 日志设置
type LogSettings struct {
	RetentionDays   int      `json:"retention_days"`
	EnabledLogTypes []string `json:"enabled_log_types"`
}

// EmailSettings 邮件设置
type EmailSettings struct {
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      int    `json:"smtp_port"`
	SMTPUsername  string `json:"smtp_username"`
	SMTPPassword  string `json:"smtp_password"`
	EnableTLS     bool   `json:"enable_tls"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
	SenderAddress string `json:"sender_address"`
}

// AppearanceSettings 外观设置（含原 Web 设置字段）
type AppearanceSettings struct {
	LoginBgURL   string `json:"login_bg_url"`
	DefaultTheme string `json:"default_theme"` // "light" 或 "dark"
	ThemeColor   string `json:"theme_color"`   // 主题色，如 "#3B82F6"
	CustomLogo   string `json:"custom_logo"`   // base64 格式图片
	ServerPort   int    `json:"server_port"`
	EnableHTTPS  bool   `json:"enable_https"`
	SSLCert      string `json:"ssl_cert"` // PEM 格式证书内容
	SSLKey       string `json:"ssl_key"`  // PEM 格式私钥内容
}

// ShareSettings 分享设置
type ShareSettings struct {
	DefaultExpireDays      int  `json:"default_expire_days"`
	MaxSharesPerUser       int  `json:"max_shares_per_user"` // 0 表示无限制
	AllowAnonymousDownload bool `json:"allow_anonymous_download"`
}
