package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"goWFM/config"
	"goWFM/db"
)

// GetConfigValue 从数据库读取指定 key 的配置值
func GetConfigValue(key string) (string, error) {
	var value string
	err := db.DB.QueryRow(`SELECT value FROM gowfm_config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfigValue 写入或更新指定 key 的配置值
func SetConfigValue(key, value string) error {
	_, err := db.DB.Exec(
		`INSERT INTO gowfm_config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

// LoadAllConfigs 启动时从数据库加载所有配置到内存
// 若数据库中无配置记录则初始化默认配置
func LoadAllConfigs() error {
	// 检查是否有配置记录
	var count int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM gowfm_config`).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		log.Println("No config found in database, initializing defaults...")
		if err := InitDefaultConfigs(); err != nil {
			return err
		}
	}

	// 从数据库加载到内存
	if err := config.LoadFromDB(GetConfigValue); err != nil {
		return err
	}
	// 持久化兼容升级后的内置邮件模板，避免每次启动重复执行迁移。
	emailData, err := json.Marshal(config.GetEmail())
	if err != nil {
		return err
	}
	storedEmail, err := GetConfigValue(config.KeyEmail)
	if err != nil {
		return err
	}
	if storedEmail != string(emailData) {
		if err := SetConfigValue(config.KeyEmail, string(emailData)); err != nil {
			return err
		}
	}
	if !config.GetEmail().Active {
		return DisableEmailDependentFeatures()
	}
	return nil
}

// InitDefaultConfigs 初始化所有默认配置到数据库
func InitDefaultConfigs() error {
	defaults := map[string]interface{}{
		config.KeyBasic:      config.DefaultBasic(),
		config.KeySecurity:   config.DefaultSecurity(),
		config.KeyLog:        config.DefaultLog(),
		config.KeyEmail:      config.DefaultEmail(),
		config.KeyAppearance: config.DefaultAppearance(),
		config.KeyShare:      config.DefaultShare(),
		config.KeyScan:       config.DefaultScan(),
	}

	for key, val := range defaults {
		data, err := json.Marshal(val)
		if err != nil {
			return err
		}
		if err := SetConfigValue(key, string(data)); err != nil {
			return err
		}
	}
	return nil
}

// UpdateBasicSettings 更新基础设置
func UpdateBasicSettings(s config.BasicSettings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyBasic, string(data)); err != nil {
		return err
	}
	config.SetBasic(s)
	return nil
}

// UpdateSecuritySettings 更新安全设置
func UpdateSecuritySettings(s config.SecuritySettings) error {
	if s.AllowEmailPasswordReset && !config.GetEmail().Active {
		return errors.New("请先配置并激活 SMTP 服务，再启用邮件重置密码")
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeySecurity, string(data)); err != nil {
		return err
	}
	config.SetSecurity(s)
	return nil
}

// UpdateLogSettings 更新日志设置
func UpdateLogSettings(s config.LogSettings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyLog, string(data)); err != nil {
		return err
	}
	config.SetLog(s)
	return nil
}

// UpdateEmailSettings 更新邮件设置
func UpdateEmailSettings(s config.EmailSettings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyEmail, string(data)); err != nil {
		return err
	}
	config.SetEmail(s)
	return nil
}

// DisableEmailDependentFeatures 在 SMTP 不可用时关闭所有依赖邮件发送的功能。
func DisableEmailDependentFeatures() error {
	security := config.GetSecurity()
	if security.AllowEmailPasswordReset {
		security.AllowEmailPasswordReset = false
		if err := UpdateSecuritySettings(security); err != nil {
			return err
		}
	}
	share := config.GetShare()
	if share.AllowEmailShare {
		share.AllowEmailShare = false
		if err := UpdateShareSettings(share); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAppearanceSettings 更新外观设置
func UpdateAppearanceSettings(s config.AppearanceSettings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyAppearance, string(data)); err != nil {
		return err
	}
	config.SetAppearance(s)
	return nil
}

// UpdateShareSettings 更新分享设置
func UpdateShareSettings(s config.ShareSettings) error {
	if s.AllowEmailShare && !config.GetEmail().Active {
		return errors.New("请先配置并激活 SMTP 服务，再启用邮件发送分享")
	}
	if err := ValidateShareSettings(s); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyShare, string(data)); err != nil {
		return err
	}
	config.SetShare(s)
	return nil
}

func ValidateShareSettings(s config.ShareSettings) error {
	if s.FileLinkTimeoutMinutes < 1 || s.FileLinkTimeoutMinutes > 1440 {
		return errors.New("file link timeout must be between 1 and 1440 minutes")
	}
	return nil
}

func UpdateScanSettings(s config.ScanSettings) error {
	if err := ValidateScanSettings(s); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := SetConfigValue(config.KeyScan, string(data)); err != nil {
		return err
	}
	config.SetScan(s)
	return nil
}

func ValidateScanSettings(s config.ScanSettings) error {
	if s.IntervalHours < 1 || s.IntervalHours > 720 {
		return errors.New("scan interval must be between 1 and 720 hours")
	}
	return nil
}
