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
	ActionLogin        = "LOGIN"
	ActionLoginFail    = "LOGIN_FAIL"
	ActionCreateDir    = "CREATE_DIR"
	ActionUpload       = "UPLOAD"
	ActionDownload     = "DOWNLOAD"
	ActionDeleteFile   = "DELETE_FILE"
	ActionDeleteDir    = "DELETE_DIR"
	ActionShareCreate  = "SHARE_CREATE"
	ActionShareAccess  = "SHARE_ACCESS"
	ActionShareDelete  = "SHARE_DELETE"
	ActionChangeOwner  = "CHANGE_OWNER"
	ActionUserCreate   = "USER_CREATE"
	ActionUserUpdate   = "USER_UPDATE"
	ActionUserDelete   = "USER_DELETE"
	ActionMove         = "MOVE"
)