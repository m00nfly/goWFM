package models

import "time"

type User struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	Password          string     `json:"-"`
	DisplayName       string     `json:"display_name"`
	Email             string     `json:"email"`
	AvatarData        string     `json:"avatar"`
	IsAdmin           bool       `json:"is_admin"`
	Permissions       int        `json:"permissions"`
	TotpEnabled       bool       `json:"totp_enabled"`
	TotpForced        bool       `json:"totp_forced"`
	TotpResetRequired bool       `json:"totp_reset_required"`
	TotpSecret        string     `json:"-"`
	TotpCreatedAt     *time.Time `json:"totp_created_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

const (
	PermBrowse       = 1 << 0 // 1
	PermDownload     = 1 << 1 // 2
	PermGlobalUpload = 1 << 2 // 4
	PermShare        = 1 << 3 // 8
	PermManageLogs   = 1 << 4 // 16
	PermAll          = PermBrowse | PermDownload | PermGlobalUpload | PermShare | PermManageLogs

	// PermUpload 保留旧名称兼容性；该权限现在表示可向任意目录上传。
	PermUpload = PermGlobalUpload
)

func (u *User) HasPermission(bit int) bool {
	if u.IsAdmin {
		return true
	}
	return (u.Permissions & bit) != 0
}
