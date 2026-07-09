package models

import "time"

// TotpRecoveryCode 恢复码
type TotpRecoveryCode struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	CodeHash  string    `json:"-"` // bcrypt hash of recovery code
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
	UsedAt    *time.Time `json:"used_at"`
}

// TotpTrustedDevice 信任设备
type TotpTrustedDevice struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	DeviceInfo string   `json:"device_info"` // browser/OS info
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
