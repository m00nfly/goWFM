package services

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"goWFM/db"

	"github.com/pquerna/otp/totp"
)

func TestPasswordResetTokenIsSingleUseAndRevokesSessions(t *testing.T) {
	if err := db.Init(filepath.Join(t.TempDir(), "reset.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	user, err := CreateUser("alice", "old-password", "Alice", "alice@example.com", false, 1)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	session, err := CreateSession(user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	token, err := CreatePasswordResetToken(user.ID, "127.0.0.1")
	if err != nil {
		t.Fatalf("create reset token: %v", err)
	}
	status, err := GetPasswordResetStatus(token)
	if err != nil || status.RequiresTOTP {
		t.Fatalf("unexpected token status: %#v, %v", status, err)
	}
	if _, err := CompletePasswordReset(token, "new-password", ""); err != nil {
		t.Fatalf("complete reset: %v", err)
	}
	updated, err := GetUserByID(user.ID)
	if err != nil || !CheckPassword(updated, "new-password") {
		t.Fatalf("new password was not stored: %v", err)
	}
	if current, err := GetSession(session.Token); err != nil || current != nil {
		t.Fatalf("session was not revoked: %#v, %v", current, err)
	}
	if _, err := CompletePasswordReset(token, "another-password", ""); !errors.Is(err, ErrResetTokenInvalid) {
		t.Fatalf("expected single-use token error, got %v", err)
	}
}

func TestPasswordResetRequiresCurrentTOTP(t *testing.T) {
	if err := db.Init(filepath.Join(t.TempDir(), "reset-totp.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()
	user, err := CreateUser("bob", "old-password", "Bob", "bob@example.com", false, 1)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	const secret = "JBSWY3DPEHPK3PXP"
	if _, err := db.DB.Exec(`UPDATE users SET totp_secret = ?, totp_enabled = 1 WHERE id = ?`, secret, user.ID); err != nil {
		t.Fatalf("enable totp: %v", err)
	}
	token, err := CreatePasswordResetToken(user.ID, "127.0.0.1")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, err := CompletePasswordReset(token, "new-password", ""); !errors.Is(err, ErrResetTOTPRequired) {
		t.Fatalf("expected TOTP requirement, got %v", err)
	}
	if _, err := CompletePasswordReset(token, "new-password", "000000"); !errors.Is(err, ErrResetTOTPInvalid) {
		t.Fatalf("expected invalid TOTP error, got %v", err)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	if _, err := CompletePasswordReset(token, "new-password", code); err != nil {
		t.Fatalf("complete reset with TOTP: %v", err)
	}
}
