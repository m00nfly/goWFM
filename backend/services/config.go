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
	return config.LoadFromDB(GetConfigValue)
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
