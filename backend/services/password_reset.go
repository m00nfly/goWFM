package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"goWFM/db"
	"golang.org/x/crypto/bcrypt"
)

const PasswordResetExpiry = 15 * time.Minute

var (
	ErrResetTokenInvalid = errors.New("password reset token is invalid or expired")
	ErrResetTOTPRequired = errors.New("totp verification is required")
	ErrResetTOTPInvalid  = errors.New("totp verification failed")
)

type PasswordResetStatus struct {
	UserID       int64
	RequiresTOTP bool
	ExpiresAt    time.Time
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CreatePasswordResetToken(userID int64, requestIP string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate password reset token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now().UTC()
	tx, err := db.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// 新申请会使该用户之前尚未使用的链接立即失效。
	if _, err = tx.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`, now.Format(time.RFC3339), userID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, request_ip) VALUES (?, ?, ?, ?)`,
		userID, tokenDigest(token), now.Add(PasswordResetExpiry).Format(time.RFC3339), requestIP); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

func InvalidatePasswordResetToken(token string) {
	db.DB.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE token_hash = ? AND used_at IS NULL`, time.Now().UTC().Format(time.RFC3339), tokenDigest(token))
}

func GetPasswordResetStatus(token string) (*PasswordResetStatus, error) {
	if len(token) < 40 {
		return nil, ErrResetTokenInvalid
	}
	var userID int64
	var expiresAt string
	var usedAt sql.NullString
	err := db.DB.QueryRow(`SELECT user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = ?`, tokenDigest(token)).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		return nil, ErrResetTokenInvalid
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || usedAt.Valid || !expires.After(time.Now().UTC()) {
		return nil, ErrResetTokenInvalid
	}
	user, err := GetUserByID(userID)
	if err != nil {
		return nil, ErrResetTokenInvalid
	}
	return &PasswordResetStatus{UserID: userID, RequiresTOTP: user.TotpEnabled, ExpiresAt: expires}, nil
}

func CompletePasswordReset(token, newPassword, totpCode string) (int64, error) {
	status, err := GetPasswordResetStatus(token)
	if err != nil {
		return 0, err
	}
	if status.RequiresTOTP {
		if totpCode == "" {
			return 0, ErrResetTOTPRequired
		}
		if err := VerifyTOTP(status.UserID, totpCode); err != nil {
			return 0, ErrResetTOTPInvalid
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`, now, tokenDigest(token), now)
	if err != nil {
		return 0, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return 0, ErrResetTokenInvalid
	}
	if _, err = tx.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(hash), status.UserID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`, now, status.UserID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, status.UserID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return status.UserID, nil
}

func CleanupExpiredPasswordResetTokens() error {
	_, err := db.DB.Exec(`DELETE FROM password_reset_tokens WHERE expires_at < ? OR used_at IS NOT NULL`, time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339))
	return err
}
