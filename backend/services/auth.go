package services

import (
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"goWFM/db"
	"goWFM/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailInUse = errors.New("email is already in use")

func NormalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || value == "" {
		return "", fmt.Errorf("invalid email address")
	}
	return value, nil
}

func scanUser(row interface{ Scan(...interface{}) error }) (*models.User, error) {
	u := &models.User{}
	var totpCreatedAt sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.DisplayName, &u.Email, &u.AvatarData,
		&u.IsAdmin, &u.Permissions, &u.TotpEnabled, &u.TotpForced, &u.TotpResetRequired, &u.TotpSecret, &totpCreatedAt,
		&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if totpCreatedAt.Valid {
		t, _ := time.Parse(time.RFC3339, totpCreatedAt.String)
		u.TotpCreatedAt = &t
	}
	return u, nil
}

func CreateUser(username, password, displayName, email string, isAdmin bool, permissions int) (*models.User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if used, err := IsEmailInUse(normalizedEmail, 0); err != nil {
		return nil, err
	} else if used {
		return nil, ErrEmailInUse
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	result, err := db.DB.Exec(
		`INSERT INTO users (username, password_hash, display_name, email, is_admin, permissions) VALUES (?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(username), string(hash), strings.TrimSpace(displayName), normalizedEmail, isAdmin, permissions,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetUserByID(id)
}

const userSelectCols = `id, username, password_hash, display_name, email, COALESCE(avatar_data,''), is_admin, permissions, COALESCE(totp_enabled,0), COALESCE(totp_forced,0), COALESCE(totp_reset_required,0), COALESCE(totp_secret,''), totp_created_at, created_at, updated_at`

func GetUserByID(id int64) (*models.User, error) {
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE id = ?`, id))
}

func GetUserByUsername(username string) (*models.User, error) {
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE username = ?`, username))
}

// GetUserByLogin 支持使用用户名或 Email 登录。Email 新增/更新时保证唯一。
func GetUserByLogin(identifier string) (*models.User, error) {
	identifier = strings.TrimSpace(identifier)
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE username = ? OR (id != 0 AND lower(email) = lower(?)) LIMIT 1`, identifier, identifier))
}

func GetUserByEmail(email string) (*models.User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	var count int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id != 0 AND lower(email) = lower(?)`, normalized).Scan(&count); err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, sql.ErrNoRows
	}
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE id != 0 AND lower(email) = lower(?)`, normalized))
}

func IsEmailInUse(email string, excludeUserID int64) (bool, error) {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id != 0 AND id != ? AND lower(email) = lower(?)`, excludeUserID, email).Scan(&count)
	return count > 0, err
}

func CheckPassword(user *models.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func UpdateUserPassword(userID int64, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(hash), userID)
	return err
}

func HasAdminUser() (bool, error) {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&count)
	return count > 0, err
}

func CreateSession(userID int64, duration time.Duration) (*models.Session, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(duration)
	_, err := db.DB.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	return &models.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}, nil
}

func GetSession(token string) (*models.Session, error) {
	s := &models.Session{}
	var expiresAtStr string
	err := db.DB.QueryRow(
		`SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = ?`,
		token,
	).Scan(&s.Token, &s.UserID, &expiresAtStr, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	if s.ExpiresAt.Before(time.Now()) {
		DeleteSession(token)
		return nil, nil
	}
	return s, nil
}

func DeleteSession(token string) error {
	_, err := db.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func DeleteAllUserSessions(userID int64) error {
	_, err := db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func CleanupExpiredSessions() error {
	_, err := db.DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().Format(time.RFC3339))
	return err
}

func CleanExpiredSessions() (int64, error) {
	result, err := db.DB.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func ListAllUsers() ([]gin.H, error) {
	rows, err := db.DB.Query(`SELECT id, username, display_name, email, COALESCE(avatar_data,''), is_admin, permissions, COALESCE(totp_enabled,0), COALESCE(totp_forced,0), COALESCE(totp_reset_required,0), created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []gin.H
	for rows.Next() {
		var id int64
		var username, displayName, email, avatarData string
		var isAdmin, totpEnabled, totpForced, totpResetRequired bool
		var permissions int
		var createdAt string
		rows.Scan(&id, &username, &displayName, &email, &avatarData, &isAdmin, &permissions, &totpEnabled, &totpForced, &totpResetRequired, &createdAt)
		result = append(result, gin.H{
			"id": id, "username": username, "display_name": displayName,
			"email": email, "avatar": avatarData, "is_admin": isAdmin, "permissions": permissions,
			"totp_enabled": totpEnabled, "totp_forced": totpForced,
			"totp_reset_required": totpResetRequired, "created_at": createdAt,
		})
	}
	return result, nil
}

func UpdateUserAvatar(id int64, avatarData string) error {
	_, err := db.DB.Exec(
		`UPDATE users SET avatar_data = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		avatarData, id,
	)
	return err
}

func UpdateUserFields(id int64, displayName, email string, isAdmin bool, permissions int) (*models.User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if used, err := IsEmailInUse(normalizedEmail, id); err != nil {
		return nil, err
	} else if used {
		return nil, ErrEmailInUse
	}
	_, err = db.DB.Exec(
		`UPDATE users SET display_name = ?, email = ?, is_admin = ?, permissions = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		strings.TrimSpace(displayName), normalizedEmail, isAdmin, permissions, id,
	)
	if err != nil {
		return nil, err
	}
	return GetUserByID(id)
}

func DeleteUserByID(id int64) error {
	_, err := db.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func AdminCount() (int, error) {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&count)
	return count, err
}
