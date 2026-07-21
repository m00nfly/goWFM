package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	if dbPath == "" {
		dbPath = "wfm.db"
	}

	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create db directory: %w", err)
		}
	}

	d, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	d.SetMaxOpenConns(1)

	if err := migrate(d); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	DB = d
	log.Printf("Database initialized: %s", dbPath)
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func migrate(d *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT DEFAULT '',
			email TEXT DEFAULT '',
			avatar_data TEXT DEFAULT '',
			is_admin BOOLEAN DEFAULT 0,
			permissions INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS file_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT UNIQUE NOT NULL,
			is_directory BOOLEAN DEFAULT 0,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS shares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT UNIQUE NOT NULL,
			name TEXT DEFAULT '',
			file_path TEXT DEFAULT '',
			owner_id INTEGER NOT NULL,
			expire_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			access_count INTEGER DEFAULT 0,
			deleted INTEGER DEFAULT 0,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS operation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			action TEXT NOT NULL,
			target_path TEXT,
			details TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS share_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			share_id INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			download_count INTEGER DEFAULT 0,
			FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS gowfm_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT '{}',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS share_download_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT UNIQUE NOT NULL,
			share_id INTEGER NOT NULL,
			share_file_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			used_at DATETIME,
			invalidated_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE,
			FOREIGN KEY (share_file_id) REFERENCES share_files(id) ON DELETE CASCADE
		)`,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nquery: %s", err, m)
		}
	}

	// 兼容已有数据库，添加 deleted 字段
	d.Exec(`ALTER TABLE shares ADD COLUMN deleted INTEGER DEFAULT 0`)

	// 迁移：将已有 shares.file_path 数据同步到 share_files
	d.Exec(`INSERT OR IGNORE INTO share_files (share_id, file_path)
		SELECT id, file_path FROM shares WHERE file_path != '' AND file_path IS NOT NULL
		AND id NOT IN (SELECT share_id FROM share_files)`)

	// 迁移：share_files 增加 download_count 字段
	d.Exec(`ALTER TABLE share_files ADD COLUMN download_count INTEGER DEFAULT 0`)

	// 迁移：shares 增加 name 字段
	d.Exec(`ALTER TABLE shares ADD COLUMN name TEXT DEFAULT ''`)

	if _, err := d.Exec(`CREATE INDEX IF NOT EXISTS idx_share_download_tokens_file
		ON share_download_tokens(share_id, share_file_id, used_at, invalidated_at)`); err != nil {
		return fmt.Errorf("create share download token file index: %w", err)
	}
	if _, err := d.Exec(`CREATE INDEX IF NOT EXISTS idx_share_download_tokens_expiry
		ON share_download_tokens(expires_at)`); err != nil {
		return fmt.Errorf("create share download token expiry index: %w", err)
	}

	// 迁移：确保 Guest 系统账户存在（id=0，用于匿名用户操作日志）
	d.Exec(`INSERT OR IGNORE INTO users (id, username, password_hash, display_name, is_admin, permissions) VALUES (0, 'Guest', '', 'Guest', 0, 0)`)

	// 迁移：TOTP 字段
	d.Exec(`ALTER TABLE users ADD COLUMN avatar_data TEXT DEFAULT ''`)
	d.Exec(`ALTER TABLE users ADD COLUMN totp_secret TEXT DEFAULT ''`)
	d.Exec(`ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN DEFAULT 0`)
	d.Exec(`ALTER TABLE users ADD COLUMN totp_forced BOOLEAN DEFAULT 0`)
	d.Exec(`ALTER TABLE users ADD COLUMN totp_reset_required BOOLEAN DEFAULT 0`)
	d.Exec(`ALTER TABLE users ADD COLUMN totp_created_at DATETIME`)

	// 迁移：TOTP 恢复码表
	d.Exec(`CREATE TABLE IF NOT EXISTS totp_recovery_codes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		code_hash TEXT NOT NULL,
		used BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		used_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`)

	// 迁移：TOTP 信任设备表
	d.Exec(`CREATE TABLE IF NOT EXISTS totp_trusted_devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		device_info TEXT DEFAULT '',
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`)

	// 密码重置令牌仅存储 SHA-256 摘要；原始令牌只会出现在发送给用户的邮件中。
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS password_reset_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		used_at DATETIME,
		request_ip TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("create password reset token table: %w", err)
	}
	if _, err := d.Exec(`CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_reset_tokens(user_id, created_at)`); err != nil {
		return fmt.Errorf("create password reset user index: %w", err)
	}
	if _, err := d.Exec(`CREATE INDEX IF NOT EXISTS idx_password_reset_expiry ON password_reset_tokens(expires_at)`); err != nil {
		return fmt.Errorf("create password reset expiry index: %w", err)
	}

	// 旧数据库可能已含重复邮箱，触发器不会阻止启动，但会保证之后的新增与修改保持唯一。
	if _, err := d.Exec(`CREATE TRIGGER IF NOT EXISTS users_email_unique_insert
		BEFORE INSERT ON users WHEN NEW.id != 0 AND trim(NEW.email) != ''
		BEGIN
			SELECT CASE WHEN EXISTS (SELECT 1 FROM users WHERE id != 0 AND lower(email) = lower(NEW.email))
			THEN RAISE(ABORT, 'email already in use') END;
		END`); err != nil {
		return fmt.Errorf("create user email insert trigger: %w", err)
	}
	if _, err := d.Exec(`CREATE TRIGGER IF NOT EXISTS users_email_unique_update
		BEFORE UPDATE OF email ON users WHEN NEW.id != 0 AND trim(NEW.email) != ''
		BEGIN
			SELECT CASE WHEN EXISTS (SELECT 1 FROM users WHERE id != NEW.id AND id != 0 AND lower(email) = lower(NEW.email))
			THEN RAISE(ABORT, 'email already in use') END;
		END`); err != nil {
		return fmt.Errorf("create user email update trigger: %w", err)
	}

	return nil
}
