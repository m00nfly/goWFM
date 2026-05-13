package services

import (
	"errors"
	"fmt"
	"path"
	"time"

	"goWFM/db"
	"goWFM/models"

	"github.com/google/uuid"
)

func CreateShare(relativePath string, ownerID int64, expireDays int) (*models.Share, error) {
	token := uuid.New().String()
	var expireAt *time.Time
	if expireDays > 0 {
		t := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
		expireAt = &t
	}

	var expireAtStr interface{}
	if expireAt != nil {
		expireAtStr = expireAt.Format(time.RFC3339)
	}

	result, err := db.DB.Exec(
		`INSERT INTO shares (token, file_path, owner_id, expire_at) VALUES (?, ?, ?, ?)`,
		token, relativePath, ownerID, expireAtStr,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()

	return &models.Share{
		ID:        id,
		Token:     token,
		FilePath:  relativePath,
		OwnerID:   ownerID,
		ExpireAt:  expireAt,
		CreatedAt: time.Now(),
	}, nil
}

func GetShareByToken(token string) (*models.Share, error) {
	s := &models.Share{}
	var expireAtStr sqlNullString
	var deleted int
	err := db.DB.QueryRow(
		`SELECT id, token, file_path, owner_id, deleted, expire_at, created_at, access_count FROM shares WHERE token = ?`,
		token,
	).Scan(&s.ID, &s.Token, &s.FilePath, &s.OwnerID, &deleted, &expireAtStr, &s.CreatedAt, &s.AccessCount)
	if err != nil {
		return nil, err
	}
	s.Deleted = deleted == 1
	if expireAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, expireAtStr.String)
		s.ExpireAt = &t
	}
	return s, nil
}

func IncrementShareAccess(token string) error {
	_, err := db.DB.Exec(`UPDATE shares SET access_count = access_count + 1 WHERE token = ?`, token)
	return err
}

func ListMyShares(ownerID int64) ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(
		`SELECT id, token, file_path, deleted, expire_at, created_at, access_count FROM shares WHERE owner_id = ? ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var token, filePath string
		var deleted int
		var expireAtStr sqlNullString
		var createdAt string
		var accessCount int
		rows.Scan(&id, &token, &filePath, &deleted, &expireAtStr, &createdAt, &accessCount)

		// Format created_at
		formattedCreatedAt := createdAt
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			formattedCreatedAt = t.Format("2006-01-02 15:04:05")
		}

		// Compute status and format expire_at
		var status string
		var formattedExpireAt interface{}
		var expireAt *time.Time
		if expireAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, expireAtStr.String); err == nil {
				formattedExpireAt = t.Format("2006-01-02 15:04:05")
				expireAt = &t
			} else {
				formattedExpireAt = expireAtStr.String
			}
		}
		if deleted == 1 {
			status = "deleted"
		} else if expireAt != nil && expireAt.Before(time.Now()) {
			status = "expired"
		} else {
			status = "valid"
		}

		entry := map[string]interface{}{
			"id":           id,
			"token":        token,
			"file_path":    filePath,
			"file_name":    path.Base(filePath),
			"status":       status,
			"created_at":   formattedCreatedAt,
			"expire_at":    formattedExpireAt,
			"access_count": accessCount,
		}
		result = append(result, entry)
	}
	return result, nil
}

func ListAllShares() ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(
		`SELECT id, token, file_path, owner_id, deleted, expire_at, created_at, access_count FROM shares ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, ownerID int64
		var token, filePath string
		var deleted int
		var expireAtStr sqlNullString
		var createdAt string
		var accessCount int
		rows.Scan(&id, &token, &filePath, &ownerID, &deleted, &expireAtStr, &createdAt, &accessCount)

		formattedCreatedAt := createdAt
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			formattedCreatedAt = t.Format("2006-01-02 15:04:05")
		}

		var status string
		var formattedExpireAt interface{}
		var expireAt *time.Time
		if expireAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, expireAtStr.String); err == nil {
				formattedExpireAt = t.Format("2006-01-02 15:04:05")
				expireAt = &t
			} else {
				formattedExpireAt = expireAtStr.String
			}
		}
		if deleted == 1 {
			status = "deleted"
		} else if expireAt != nil && expireAt.Before(time.Now()) {
			status = "expired"
		} else {
			status = "valid"
		}

		entry := map[string]interface{}{
			"id":           id,
			"token":        token,
			"file_name":    path.Base(filePath),
			"file_path":    filePath,
			"owner_id":     ownerID,
			"status":       status,
			"expire_at":    formattedExpireAt,
			"created_at":   formattedCreatedAt,
			"access_count": accessCount,
		}
		result = append(result, entry)
	}
	return result, nil
}

func ListShareUsers() ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(`SELECT id, username FROM users WHERE is_admin = 1 OR (permissions & 8) != 0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var username string
		rows.Scan(&id, &username)
		result = append(result, map[string]interface{}{
			"id":       id,
			"username": username,
		})
	}
	return result, nil
}

func DeleteShare(id int64) error {
	_, err := db.DB.Exec(`DELETE FROM shares WHERE id = ?`, id)
	return err
}

func ValidateShareAccess(token string) (*models.Share, error) {
	s, err := GetShareByToken(token)
	if err != nil {
		return nil, fmt.Errorf("share not found")
	}
	if s.Deleted {
		return nil, errors.New("source file has been deleted")
	}
	if s.ExpireAt != nil && s.ExpireAt.Before(time.Now()) {
		return nil, errors.New("share link expired")
	}
	return s, nil
}

type sqlNullString struct {
	String string
	Valid  bool
}

func (s *sqlNullString) Scan(value interface{}) error {
	if value == nil {
		s.Valid = false
		return nil
	}
	switch v := value.(type) {
	case string:
		s.String = v
		s.Valid = true
	case []byte:
		s.String = string(v)
		s.Valid = true
	case time.Time:
		if v.IsZero() {
			s.Valid = false
			return nil
		}
		s.String = v.Format(time.RFC3339)
		s.Valid = true
	default:
		s.Valid = false
	}
	return nil
}
