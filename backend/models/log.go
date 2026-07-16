package models

import "time"

type OperationLog struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Action     string    `json:"action"`
	TargetPath string    `json:"target_path"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

const (
	ActionLogin                 = "LOGIN"
	ActionLoginFail             = "LOGIN_FAIL"
	ActionLoginTOTP             = "LOGIN_TOTP"
	ActionLoginTOTPFail         = "LOGIN_TOTP_FAIL"
	ActionBlockIP               = "BLOCK_IP"
	ActionBlockAccount          = "BLOCK_ACCOUNT"
	ActionConfigChange          = "CONFIG_CHANGE"
	ActionCreateDir             = "CREATE_DIR"
	ActionUpload                = "UPLOAD"
	ActionDownload              = "DOWNLOAD"
	ActionDeleteFile            = "DELETE_FILE"
	ActionDeleteDir             = "DELETE_DIR"
	ActionShareCreate           = "SHARE_CREATE"
	ActionShareAccess           = "SHARE_ACCESS"
	ActionShareDelete           = "SHARE_DELETE"
	ActionChangeOwner           = "CHANGE_OWNER"
	ActionUserCreate            = "USER_CREATE"
	ActionUserUpdate            = "USER_UPDATE"
	ActionUserDelete            = "USER_DELETE"
	ActionMove                  = "MOVE"
	ActionTOTPEnable            = "TOTP_ENABLE"
	ActionTOTPDisable           = "TOTP_DISABLE"
	ActionTOTPRecovery          = "TOTP_RECOVERY"
	ActionPasswordResetRequest  = "PASSWORD_RESET_REQUEST"
	ActionPasswordResetComplete = "PASSWORD_RESET_COMPLETE"
)

// AllLogTypes 返回所有日志类型列表
func AllLogTypes() []string {
	return []string{
		ActionLogin, ActionLoginFail, ActionLoginTOTP, ActionLoginTOTPFail,
		ActionBlockIP, ActionBlockAccount, ActionConfigChange,
		ActionCreateDir, ActionUpload, ActionDownload, ActionDeleteFile, ActionDeleteDir,
		ActionShareCreate, ActionShareAccess, ActionShareDelete,
		ActionChangeOwner, ActionUserCreate, ActionUserUpdate, ActionUserDelete, ActionMove,
		ActionTOTPEnable, ActionTOTPDisable, ActionTOTPRecovery,
		ActionPasswordResetRequest, ActionPasswordResetComplete,
	}
}
