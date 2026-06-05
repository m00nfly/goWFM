package models

import "time"

type Share struct {
	ID          int64      `json:"id"`
	Token       string     `json:"token"`
	OwnerID     int64      `json:"owner_id"`
	ExpireAt    *time.Time `json:"expire_at"`
	CreatedAt   time.Time  `json:"created_at"`
	AccessCount int        `json:"access_count"`
	Deleted     bool       `json:"deleted"`
}

type ShareFile struct {
	ID            int64  `json:"id"`
	ShareID       int64  `json:"share_id"`
	FilePath      string `json:"file_path"`
	DownloadCount int    `json:"download_count"`
}
