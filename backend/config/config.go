package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
)

// 配置分类 key 常量
const (
	KeyBasic      = "basic_settings"
	KeySecurity   = "security_settings"
	KeyLog        = "log_settings"
	KeyEmail      = "email_settings"
	KeyAppearance = "appearance_settings"
	KeyShare      = "share_settings"
)

// AllKeys 返回所有配置分类 key
func AllKeys() []string {
	return []string{KeyBasic, KeySecurity, KeyLog, KeyEmail, KeyAppearance, KeyShare}
}

// 全局配置实例（读写锁保护）
var (
	mu         sync.RWMutex
	basic      BasicSettings
	security   SecuritySettings
	logCfg     LogSettings
	email      EmailSettings
	appearance AppearanceSettings
	share      ShareSettings
)

// --- 对外只读访问（返回副本） ---

func GetBasic() BasicSettings           { mu.RLock(); defer mu.RUnlock(); return basic }
func GetSecurity() SecuritySettings     { mu.RLock(); defer mu.RUnlock(); return security }
func GetLog() LogSettings               { mu.RLock(); defer mu.RUnlock(); return logCfg }
func GetEmail() EmailSettings           { mu.RLock(); defer mu.RUnlock(); return email }
func GetAppearance() AppearanceSettings { mu.RLock(); defer mu.RUnlock(); return appearance }
func GetShare() ShareSettings           { mu.RLock(); defer mu.RUnlock(); return share }

// --- 更新（由 service 层调用） ---

func SetBasic(s BasicSettings)           { mu.Lock(); defer mu.Unlock(); basic = s }
func SetSecurity(s SecuritySettings)     { mu.Lock(); defer mu.Unlock(); security = s }
func SetLog(s LogSettings)               { mu.Lock(); defer mu.Unlock(); logCfg = s }
func SetEmail(s EmailSettings)           { mu.Lock(); defer mu.Unlock(); email = s }
func SetAppearance(s AppearanceSettings) { mu.Lock(); defer mu.Unlock(); appearance = s }
func SetShare(s ShareSettings)           { mu.Lock(); defer mu.Unlock(); share = s }

// InitDefaults 将所有配置设为默认值（内存中）
func InitDefaults() {
	mu.Lock()
	defer mu.Unlock()
	basic = DefaultBasic()
	security = DefaultSecurity()
	logCfg = DefaultLog()
	email = DefaultEmail()
	appearance = DefaultAppearance()
	share = DefaultShare()
}

// LoadFromDB 从数据库加载所有配置到内存
// getter 是从数据库获取指定 key 值的回调函数
func LoadFromDB(getter func(key string) (string, error)) error {
	mu.Lock()
	defer mu.Unlock()

	// 先设置默认值
	basic = DefaultBasic()
	security = DefaultSecurity()
	logCfg = DefaultLog()
	email = DefaultEmail()
	appearance = DefaultAppearance()
	share = DefaultShare()

	// 从数据库覆盖
	if val, err := getter(KeyBasic); err == nil && val != "" {
		json.Unmarshal([]byte(val), &basic)
	}
	if val, err := getter(KeySecurity); err == nil && val != "" {
		json.Unmarshal([]byte(val), &security)
	}
	if val, err := getter(KeyLog); err == nil && val != "" {
		json.Unmarshal([]byte(val), &logCfg)
	}
	if val, err := getter(KeyEmail); err == nil && val != "" {
		json.Unmarshal([]byte(val), &email)
	}
	if val, err := getter(KeyAppearance); err == nil && val != "" {
		json.Unmarshal([]byte(val), &appearance)
	}
	if val, err := getter(KeyShare); err == nil && val != "" {
		json.Unmarshal([]byte(val), &share)
	}

	return nil
}

// RandomSecret 生成随机 32 字节十六进制密钥
func RandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
