package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"goWFM/config"
	"goWFM/db"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// ---------- TOTP 密钥管理 ----------

// SetupTOTP 为用户生成 TOTP 密钥并返回 otpauth URI 和二维码 PNG base64。
// 此时尚未启用 TOTP，需要用户用验证码确认后才会激活。
func SetupTOTP(userID int64, username string) (qrBase64 string, err error) {
	// 生成随机 20 字节密钥
	key := make([]byte, 20)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(key)

	issuer := strings.TrimSpace(config.GetBasic().SiteName)
	if issuer == "" {
		issuer = "goWFM"
	}
	label := fmt.Sprintf("%s:%s", url.PathEscape(issuer), url.PathEscape(username))
	otpauthURI := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s", label, secret, url.QueryEscape(issuer))

	// 生成二维码 PNG
	png, err := qrcode.Encode(otpauthURI, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("generate qr code: %w", err)
	}
	qrBase64 = base64Encode(png)

	// 存储密钥到数据库（但 totp_enabled 仍为 false，等待验证）
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.DB.Exec(
		`UPDATE users SET totp_secret = ?, totp_enabled = 0, totp_created_at = ? WHERE id = ?`,
		secret, now, userID,
	)
	if err != nil {
		return "", fmt.Errorf("store totp secret: %w", err)
	}

	return qrBase64, nil
}

// VerifyTOTPSetup 验证 TOTP 设置码，成功后启用 TOTP 并生成恢复码。
func VerifyTOTPSetup(userID int64, code string) ([]string, error) {
	// 获取用户的待验证密钥
	var secret string
	var enabled bool
	err := db.DB.QueryRow(`SELECT COALESCE(totp_secret,''), COALESCE(totp_enabled,0) FROM users WHERE id = ?`, userID).
		Scan(&secret, &enabled)
	if err != nil {
		return nil, fmt.Errorf("get user totp: %w", err)
	}
	if secret == "" {
		return nil, fmt.Errorf("请先开启 TOTP 设置")
	}
	if enabled {
		return nil, fmt.Errorf("TOTP 已启用")
	}

	if !totp.Validate(code, secret) {
		return nil, fmt.Errorf("验证码错误")
	}

	// 验证通过 → 启用 TOTP
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.DB.Exec(`UPDATE users SET totp_enabled = 1, totp_reset_required = 0, totp_created_at = ? WHERE id = ?`, now, userID)
	if err != nil {
		return nil, fmt.Errorf("enable totp: %w", err)
	}

	// 生成恢复码
	codes, err := GenerateRecoveryCodes(userID)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	return codes, nil
}

// DisableTOTP 禁用用户的 TOTP，清除密钥、恢复码和信任设备。
func DisableTOTP(userID int64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE users SET totp_secret = '', totp_enabled = 0, totp_reset_required = 0, totp_created_at = NULL WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM totp_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM totp_trusted_devices WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AdminEnableTOTP 要求用户必须启用 TOTP。未绑定用户仍保持未启用状态，
// 由用户登录后自行扫描并验证密钥，避免生成一个用户无法获知的密钥。
func AdminEnableTOTP(userID int64) error {
	result, err := db.DB.Exec(`UPDATE users SET totp_forced = 1 WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("force totp: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AdminUnforceTOTP 仅解除管理员强制要求，保留用户现有密钥和启用状态。
func AdminUnforceTOTP(userID int64) error {
	result, err := db.DB.Exec(`UPDATE users SET totp_forced = 0 WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("unforce totp: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AdminDisableTOTP 完全关闭并清除现有绑定，仅用于明确的管理员禁用操作。
func AdminDisableTOTP(userID int64) error {
	if err := DisableTOTP(userID); err != nil {
		return err
	}
	_, err := db.DB.Exec(`UPDATE users SET totp_forced = 0, totp_reset_required = 0 WHERE id = ?`, userID)
	return err
}

// AdminResetTOTP 清除现有凭据，并要求用户下次通过账号密码后重新绑定。
func AdminResetTOTP(userID int64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE users SET totp_secret = '', totp_enabled = 0, totp_reset_required = 1, totp_created_at = NULL WHERE id = ? AND totp_enabled = 1`, userID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("该用户未启用 TOTP")
	}
	if _, err = tx.Exec(`DELETE FROM totp_recovery_codes WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM totp_trusted_devices WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- TOTP 验证 ----------

// VerifyTOTP 验证 TOTP 验证码。
func VerifyTOTP(userID int64, code string) error {
	code = strings.TrimSpace(code)
	var secret string
	err := db.DB.QueryRow(`SELECT COALESCE(totp_secret,'') FROM users WHERE id = ? AND totp_enabled = 1`, userID).
		Scan(&secret)
	if err == sql.ErrNoRows || secret == "" {
		return fmt.Errorf("TOTP 未启用")
	}
	if err != nil {
		return fmt.Errorf("get totp secret: %w", err)
	}

	if !totp.Validate(code, secret) {
		return fmt.Errorf("验证码错误")
	}
	return nil
}

// ---------- 恢复码 ----------

// GenerateRecoveryCodes 为用户生成 1 个一次性恢复码，返回明文列表。
func GenerateRecoveryCodes(userID int64) ([]string, error) {
	// 清除旧恢复码
	if _, err := db.DB.Exec(`DELETE FROM totp_recovery_codes WHERE user_id = ?`, userID); err != nil {
		return nil, err
	}

	const recoveryCodeCount = 1
	codes := make([]string, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		// 生成 8 字节随机 → hex → 取前 10 位作为恢复码
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		code := hex.EncodeToString(b)[:10]
		codes[i] = code

		// bcrypt hash 存储
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if _, err := db.DB.Exec(
			`INSERT INTO totp_recovery_codes (user_id, code_hash) VALUES (?, ?)`,
			userID, string(hash),
		); err != nil {
			return nil, err
		}
	}
	return codes, nil
}

// VerifyRecoveryCode 验证并使用恢复码登录。
func VerifyRecoveryCode(userID int64, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("无效的恢复码")
	}

	rows, err := db.DB.Query(
		`SELECT id, code_hash FROM totp_recovery_codes WHERE user_id = ? AND used = 0`, userID,
	)
	if err != nil {
		return fmt.Errorf("query recovery codes: %w", err)
	}
	type recoveryCodeRow struct {
		id   int64
		hash string
	}
	var codes []recoveryCodeRow
	for rows.Next() {
		var id int64
		var codeHash string
		if err := rows.Scan(&id, &codeHash); err != nil {
			continue
		}
		codes = append(codes, recoveryCodeRow{id: id, hash: codeHash})
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close recovery codes: %w", err)
	}

	for _, item := range codes {
		if bcrypt.CompareHashAndPassword([]byte(item.hash), []byte(code)) == nil {
			now := time.Now().UTC().Format(time.RFC3339)
			if _, err := db.DB.Exec(`UPDATE totp_recovery_codes SET used = 1, used_at = ? WHERE id = ?`, now, item.id); err != nil {
				return fmt.Errorf("consume recovery code: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("无效的恢复码")
}

// InvalidateTOTPAfterRecovery 按恢复流程废止旧 OTP 凭据，并要求用户重新处置 OTP。
func InvalidateTOTPAfterRecovery(userID int64) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE users SET totp_secret = '', totp_enabled = 0, totp_reset_required = 1, totp_created_at = NULL WHERE id = ?`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM totp_recovery_codes WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM totp_trusted_devices WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// GetRecoveryCodesRemaining 返回用户剩余未使用的恢复码数量
func GetRecoveryCodesRemaining(userID int64) int {
	var count int
	db.DB.QueryRow(`SELECT COUNT(*) FROM totp_recovery_codes WHERE user_id = ? AND used = 0`, userID).Scan(&count)
	return count
}

// ---------- 信任设备 ----------

// CreateTrustedDevice 为用户创建一个信任设备记录，返回设备 token。
func CreateTrustedDevice(userID int64, deviceInfo string) (string, error) {
	// 清除该用户过期的信任设备
	db.DB.Exec(`DELETE FROM totp_trusted_devices WHERE user_id = ? AND expires_at < ?`,
		userID, time.Now().UTC().Format(time.RFC3339))

	// 生成设备 token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	trustDays := config.GetSecurity().TotpTrustDays
	if trustDays <= 0 {
		trustDays = 30
	}
	expiresAt := time.Now().UTC().Add(time.Duration(trustDays) * 24 * time.Hour).Format(time.RFC3339)

	_, err := db.DB.Exec(
		`INSERT INTO totp_trusted_devices (user_id, token, device_info, expires_at) VALUES (?, ?, ?, ?)`,
		userID, token, deviceInfo, expiresAt,
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

// CheckTrustedDevice 检查设备 token 是否有效且未过期。
func CheckTrustedDevice(userID int64, token string) bool {
	if token == "" {
		return false
	}
	var count int
	db.DB.QueryRow(
		`SELECT COUNT(*) FROM totp_trusted_devices WHERE user_id = ? AND token = ? AND expires_at > ?`,
		userID, token, time.Now().UTC().Format(time.RFC3339),
	).Scan(&count)
	return count > 0
}

// RemoveTrustedDevice 删除指定信任设备。
func RemoveTrustedDevice(userID int64, token string) {
	db.DB.Exec(`DELETE FROM totp_trusted_devices WHERE user_id = ? AND token = ?`, userID, token)
}

// CleanExpiredTrustedDevices 清理所有过期的信任设备。
func CleanExpiredTrustedDevices() {
	db.DB.Exec(`DELETE FROM totp_trusted_devices WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
}

// ---------- 辅助函数 ----------

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
