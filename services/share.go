package services

import (
	"fmt"
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
	err := db.DB.QueryRow(
		`SELECT id, token, file_path, owner_id, expire_at, created_at, access_count FROM shares WHERE token = ?`,
		token,
	).Scan(&s.ID, &s.Token, &s.FilePath, &s.OwnerID, &expireAtStr, &s.CreatedAt, &s.AccessCount)
	if err != nil {
		return nil, err
	}
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
		`SELECT id, token, file_path, expire_at, created_at, access_count FROM shares WHERE owner_id = ? ORDER BY created_at DESC`,
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
		var expireAtStr sqlNullString
		var createdAt string
		var accessCount int
		rows.Scan(&id, &token, &filePath, &expireAtStr, &createdAt, &accessCount)

		entry := map[string]interface{}{
			"id":           id,
			"token":        token,
			"file_path":    filePath,
			"created_at":   createdAt,
			"access_count": accessCount,
		}
		if expireAtStr.Valid {
			entry["expire_at"] = expireAtStr.String
		} else {
			entry["expire_at"] = nil
		}
		result = append(result, entry)
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
	if s.ExpireAt != nil && s.ExpireAt.Before(time.Now()) {
		return nil, fmt.Errorf("share link expired")
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
	s.Valid = true
	switch v := value.(type) {
	case string:
		s.String = v
	case []byte:
		s.String = string(v)
	}
	return nil
}
