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

	// 迁移：确保 Guest 系统账户存在（id=0，用于匿名用户操作日志）
	d.Exec(`INSERT OR IGNORE INTO users (id, username, password_hash, display_name, is_admin, permissions) VALUES (0, 'Guest', '', 'Guest', 0, 0)`)

	return nil
}
