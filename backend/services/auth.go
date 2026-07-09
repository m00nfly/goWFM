package services

import (
	"database/sql"
	"errors"
	"time"

	"goWFM/db"
	"goWFM/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func scanUser(row interface{ Scan(...interface{}) error }) (*models.User, error) {
	u := &models.User{}
	var totpCreatedAt sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.DisplayName, &u.Email,
		&u.IsAdmin, &u.Permissions, &u.TotpEnabled, &u.TotpSecret, &totpCreatedAt,
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	result, err := db.DB.Exec(
		`INSERT INTO users (username, password_hash, display_name, email, is_admin, permissions) VALUES (?, ?, ?, ?, ?, ?)`,
		username, string(hash), displayName, email, isAdmin, permissions,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetUserByID(id)
}

const userSelectCols = `id, username, password_hash, display_name, email, is_admin, permissions, COALESCE(totp_enabled,0), COALESCE(totp_secret,''), totp_created_at, created_at, updated_at`

func GetUserByID(id int64) (*models.User, error) {
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE id = ?`, id))
}

func GetUserByUsername(username string) (*models.User, error) {
	return scanUser(db.DB.QueryRow(`SELECT `+userSelectCols+` FROM users WHERE username = ?`, username))
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
	rows, err := db.DB.Query(`SELECT id, username, display_name, email, is_admin, permissions, COALESCE(totp_enabled,0), created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []gin.H
	for rows.Next() {
		var id int64
		var username, displayName, email string
		var isAdmin, totpEnabled bool
		var permissions int
		var createdAt string
		rows.Scan(&id, &username, &displayName, &email, &isAdmin, &permissions, &totpEnabled, &createdAt)
		result = append(result, gin.H{
			"id": id, "username": username, "display_name": displayName,
			"email": email, "is_admin": isAdmin, "permissions": permissions,
			"totp_enabled": totpEnabled, "created_at": createdAt,
		})
	}
	return result, nil
}

func UpdateUserFields(id int64, displayName, email string, isAdmin bool, permissions int) (*models.User, error) {
	_, err := db.DB.Exec(
		`UPDATE users SET display_name = ?, email = ?, is_admin = ?, permissions = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		displayName, email, isAdmin, permissions, id,
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
