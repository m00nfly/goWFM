package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"goWFM/db"
	"goWFM/models"
)

const (
	shareTokenAlphabet  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	shareTokenLength    = 12
	shareTokenTimeChars = 7
	shareTokenAttempts  = 10
)

var (
	ErrShareFileNotFound   = errors.New("file does not belong to this share")
	ErrDownloadLinkInvalid = errors.New("download link is invalid, used, or expired")
)

type IssuedShareDownloadLink struct {
	Token     string
	Filename  string
	ExpiresAt time.Time
}

// GetShareStats returns expired and valid share counts.
// If ownerID > 0, counts only that user's shares; otherwise counts all shares.
func GetShareStats(ownerID int64) (expired int, valid int, err error) {
	var query string
	var args []interface{}
	if ownerID > 0 {
		query = `SELECT
			COUNT(CASE WHEN deleted = 0 AND expire_at IS NOT NULL AND expire_at < datetime('now') THEN 1 END),
			COUNT(CASE WHEN deleted = 0 AND (expire_at IS NULL OR expire_at >= datetime('now')) THEN 1 END)
			FROM shares WHERE owner_id = ?`
		args = append(args, ownerID)
	} else {
		query = `SELECT
			COUNT(CASE WHEN deleted = 0 AND expire_at IS NOT NULL AND expire_at < datetime('now') THEN 1 END),
			COUNT(CASE WHEN deleted = 0 AND (expire_at IS NULL OR expire_at >= datetime('now')) THEN 1 END)
			FROM shares`
	}
	err = db.DB.QueryRow(query, args...).Scan(&expired, &valid)
	return
}

func CreateShare(filePaths []string, ownerID int64, expireDays int, name string) (*models.Share, error) {
	if len(filePaths) == 0 {
		return nil, errors.New("at least one file path is required")
	}

	var expireAt *time.Time
	if expireDays > 0 {
		t := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
		expireAt = &t
	}

	var expireAtStr interface{}
	if expireAt != nil {
		expireAtStr = expireAt.Format(time.RFC3339)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		token string
		id    int64
	)
	for attempt := 0; attempt < shareTokenAttempts; attempt++ {
		token, err = generateShareToken(time.Now())
		if err != nil {
			return nil, err
		}
		result, insertErr := tx.Exec(
			`INSERT INTO shares (token, name, file_path, owner_id, expire_at) VALUES (?, ?, '', ?, ?)`,
			token, name, ownerID, expireAtStr,
		)
		if insertErr == nil {
			id, err = result.LastInsertId()
			if err != nil {
				return nil, err
			}
			break
		}
		if !strings.Contains(insertErr.Error(), "UNIQUE constraint failed: shares.token") {
			return nil, insertErr
		}
	}
	if id == 0 {
		return nil, errors.New("failed to generate a unique share id")
	}

	for _, fp := range filePaths {
		_, err := tx.Exec(
			`INSERT INTO share_files (share_id, file_path) VALUES (?, ?)`, id, fp)
		if err != nil {
			return nil, fmt.Errorf("insert share_files: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Share{
		ID:        id,
		Token:     token,
		Name:      name,
		OwnerID:   ownerID,
		ExpireAt:  expireAt,
		CreatedAt: time.Now(),
	}, nil
}

// generateShareToken combines a millisecond time factor with cryptographic
// randomness. The database UNIQUE constraint and retry loop in CreateShare
// provide the final collision guarantee.
func generateShareToken(now time.Time) (string, error) {
	token := make([]byte, shareTokenLength)
	timestamp := uint64(now.UnixMilli())
	for i := shareTokenTimeChars - 1; i >= 0; i-- {
		token[i] = shareTokenAlphabet[timestamp%uint64(len(shareTokenAlphabet))]
		timestamp /= uint64(len(shareTokenAlphabet))
	}

	randomPart, err := randomAlphaNumeric(shareTokenLength - shareTokenTimeChars)
	if err != nil {
		return "", err
	}
	copy(token[shareTokenTimeChars:], randomPart)
	return string(token), nil
}

func randomAlphaNumeric(length int) ([]byte, error) {
	result := make([]byte, 0, length)
	buffer := make([]byte, length*2)
	for len(result) < length {
		if _, err := rand.Read(buffer); err != nil {
			return nil, err
		}
		for _, value := range buffer {
			// 248 is the largest multiple of 62 below 256, avoiding modulo bias.
			if value >= 248 {
				continue
			}
			result = append(result, shareTokenAlphabet[int(value)%len(shareTokenAlphabet)])
			if len(result) == length {
				break
			}
		}
	}
	return result, nil
}

func generateDownloadToken() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashDownloadToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func parseStoredTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid stored time %q", value)
}

// IssueShareDownloadLink validates the share and selected file, revokes every
// previous link for that file, and creates the only currently valid link.
func IssueShareDownloadLink(shareToken string, fileID int64, ttl time.Duration) (*IssuedShareDownloadLink, error) {
	if ttl <= 0 {
		return nil, errors.New("download link timeout must be positive")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		shareID  int64
		deleted  int
		expireAt sqlNullString
	)
	err = tx.QueryRow(`SELECT id, deleted, expire_at FROM shares WHERE token = ?`, shareToken).
		Scan(&shareID, &deleted, &expireAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("share not found")
	}
	if err != nil {
		return nil, err
	}
	if deleted == 1 {
		return nil, errors.New("source file has been deleted")
	}
	if expireAt.Valid {
		parsed, parseErr := parseStoredTime(expireAt.String)
		if parseErr != nil {
			return nil, parseErr
		}
		if parsed.Before(time.Now()) {
			return nil, errors.New("share link expired")
		}
	}

	var filePath string
	err = tx.QueryRow(`SELECT file_path FROM share_files WHERE id = ? AND share_id = ?`, fileID, shareID).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrShareFileNotFound
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	formattedNow := now.Format(time.RFC3339Nano)
	if _, err := tx.Exec(`UPDATE share_download_tokens
		SET invalidated_at = ?
		WHERE share_id = ? AND share_file_id = ? AND used_at IS NULL AND invalidated_at IS NULL`,
		formattedNow, shareID, fileID); err != nil {
		return nil, err
	}

	expiresAt := now.Add(ttl)
	var rawToken string
	for attempt := 0; attempt < shareTokenAttempts; attempt++ {
		rawToken, err = generateDownloadToken()
		if err != nil {
			return nil, err
		}
		_, insertErr := tx.Exec(`INSERT INTO share_download_tokens
			(token_hash, share_id, share_file_id, expires_at) VALUES (?, ?, ?, ?)`,
			hashDownloadToken(rawToken), shareID, fileID, expiresAt.Format(time.RFC3339Nano))
		if insertErr == nil {
			break
		}
		if !strings.Contains(insertErr.Error(), "UNIQUE constraint failed: share_download_tokens.token_hash") {
			return nil, insertErr
		}
		rawToken = ""
	}
	if rawToken == "" {
		return nil, errors.New("failed to generate a unique download token")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &IssuedShareDownloadLink{
		Token:     rawToken,
		Filename:  filepath.Base(filePath),
		ExpiresAt: expiresAt,
	}, nil
}

// ConsumeShareDownloadLink atomically marks a temporary link as used before
// returning the file metadata. Only the transaction winner may start a download.
func ConsumeShareDownloadLink(rawToken, requestedFilename string) (*models.ShareDownload, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		file            models.ShareFile
		shareToken      string
		shareDeleted    int
		linkExpiresAt   string
		linkUsedAt      sqlNullString
		linkInvalidated sqlNullString
		shareExpiresAt  sqlNullString
	)
	err = tx.QueryRow(`SELECT sf.id, sf.share_id, sf.file_path, sf.download_count,
		s.token, s.deleted, s.expire_at, d.expires_at, d.used_at, d.invalidated_at
		FROM share_download_tokens d
		JOIN shares s ON s.id = d.share_id
		JOIN share_files sf ON sf.id = d.share_file_id AND sf.share_id = d.share_id
		WHERE d.token_hash = ?`, hashDownloadToken(rawToken)).Scan(
		&file.ID, &file.ShareID, &file.FilePath, &file.DownloadCount,
		&shareToken, &shareDeleted, &shareExpiresAt, &linkExpiresAt, &linkUsedAt, &linkInvalidated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDownloadLinkInvalid
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt, err := parseStoredTime(linkExpiresAt)
	if err != nil || linkUsedAt.Valid || linkInvalidated.Valid || !expiresAt.After(now) {
		return nil, ErrDownloadLinkInvalid
	}
	if shareDeleted == 1 {
		return nil, ErrDownloadLinkInvalid
	}
	if shareExpiresAt.Valid {
		parsed, parseErr := parseStoredTime(shareExpiresAt.String)
		if parseErr != nil || parsed.Before(now) {
			return nil, ErrDownloadLinkInvalid
		}
	}
	if filepath.Base(file.FilePath) != requestedFilename {
		return nil, ErrDownloadLinkInvalid
	}

	result, err := tx.Exec(`UPDATE share_download_tokens SET used_at = ?
		WHERE token_hash = ? AND used_at IS NULL AND invalidated_at IS NULL`,
		now.UTC().Format(time.RFC3339Nano), hashDownloadToken(rawToken))
	if err != nil {
		return nil, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if updated != 1 {
		return nil, ErrDownloadLinkInvalid
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.ShareDownload{ShareToken: shareToken, File: file}, nil
}

func GetShareByToken(token string) (*models.Share, error) {
	s := &models.Share{}
	var expireAtStr sqlNullString
	var deleted int
	var name sqlNullString
	err := db.DB.QueryRow(
		`SELECT id, token, COALESCE(name,''), owner_id, deleted, expire_at, created_at, access_count FROM shares WHERE token = ?`,
		token,
	).Scan(&s.ID, &s.Token, &name, &s.OwnerID, &deleted, &expireAtStr, &s.CreatedAt, &s.AccessCount)
	if err != nil {
		return nil, err
	}
	s.Deleted = deleted == 1
	if name.Valid {
		s.Name = name.String
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

func GetShareFiles(shareID int64) ([]models.ShareFile, error) {
	rows, err := db.DB.Query(
		`SELECT id, share_id, file_path, download_count FROM share_files WHERE share_id = ?`, shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.ShareFile
	for rows.Next() {
		var f models.ShareFile
		rows.Scan(&f.ID, &f.ShareID, &f.FilePath, &f.DownloadCount)
		files = append(files, f)
	}
	return files, nil
}

func IncrementFileDownload(fileID int64) error {
	_, err := db.DB.Exec(`UPDATE share_files SET download_count = download_count + 1 WHERE id = ?`, fileID)
	return err
}

func ListMyShares(ownerID int64) ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(
		`SELECT s.id, s.token, COALESCE(s.name,''), s.deleted, s.expire_at, s.created_at, s.access_count,
			(SELECT COUNT(*) FROM share_files WHERE share_id = s.id) as file_count,
			COALESCE((SELECT file_path FROM share_files WHERE share_id = s.id LIMIT 1), '') as first_file_path
		FROM shares s WHERE s.owner_id = ? ORDER BY s.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var token string
		var shareName string
		var deleted int
		var expireAtStr sqlNullString
		var createdAt string
		var accessCount int
		var fileCount int
		var firstFilePath string
		rows.Scan(&id, &token, &shareName, &deleted, &expireAtStr, &createdAt, &accessCount, &fileCount, &firstFilePath)

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

		// 文件名展示逻辑：优先使用 name，为空时退回原有逻辑
		var fileName string
		if shareName != "" {
			fileName = shareName
		} else if fileCount > 1 {
			fileName = fmt.Sprintf("分享%d个文件", fileCount)
		} else {
			fileName = path.Base(firstFilePath)
		}

		entry := map[string]interface{}{
			"id":           id,
			"token":        token,
			"name":         shareName,
			"file_path":    firstFilePath,
			"file_name":    fileName,
			"file_count":   fileCount,
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
		`SELECT s.id, s.token, COALESCE(s.name,''), s.owner_id, s.deleted, s.expire_at, s.created_at, s.access_count,
			(SELECT COUNT(*) FROM share_files WHERE share_id = s.id) as file_count,
			COALESCE((SELECT file_path FROM share_files WHERE share_id = s.id LIMIT 1), '') as first_file_path
		FROM shares s ORDER BY s.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, ownerID int64
		var token string
		var shareName string
		var deleted int
		var expireAtStr sqlNullString
		var createdAt string
		var accessCount int
		var fileCount int
		var firstFilePath string
		rows.Scan(&id, &token, &shareName, &ownerID, &deleted, &expireAtStr, &createdAt, &accessCount, &fileCount, &firstFilePath)

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

		// 文件名展示逻辑：优先使用 name，为空时退回原有逻辑
		var fileName string
		if shareName != "" {
			fileName = shareName
		} else if fileCount > 1 {
			fileName = fmt.Sprintf("分享%d个文件", fileCount)
		} else {
			fileName = path.Base(firstFilePath)
		}

		entry := map[string]interface{}{
			"id":           id,
			"token":        token,
			"name":         shareName,
			"file_name":    fileName,
			"file_path":    firstFilePath,
			"file_count":   fileCount,
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

func UpdateShare(id int64, name string, expireDays *int) error {
	if expireDays != nil {
		var expireAtStr interface{}
		if *expireDays > 0 {
			t := time.Now().Add(time.Duration(*expireDays) * 24 * time.Hour)
			expireAtStr = t.Format(time.RFC3339)
		}
		_, err := db.DB.Exec(`UPDATE shares SET name = ?, expire_at = ? WHERE id = ?`, name, expireAtStr, id)
		return err
	}
	_, err := db.DB.Exec(`UPDATE shares SET name = ? WHERE id = ?`, name, id)
	return err
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
