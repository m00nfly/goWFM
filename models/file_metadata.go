package models

import "time"

type FileMetadata struct {
	ID          int64     `json:"id"`
	FilePath    string    `json:"file_path"`
	IsDirectory bool      `json:"is_directory"`
	OwnerID     int64     `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}