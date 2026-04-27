package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	OrgName       string `json:"org_name"`
	OrgLink       string `json:"org_link"`
	DataRootPath  string `json:"data_root_path"`
	ServerPort    int    `json:"server_port"`
	SessionSecret string `json:"session_secret"`
	LogLevel      string `json:"log_level"`
	DBPath        string `json:"db_path"`
	MaxUploadSize int64  `json:"max_upload_size"`
}

var C *Config

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		c := &Config{
			ServerPort:    8080,
			LogLevel:      "info",
			DBPath:        "wfm.db",
			SessionSecret: RandomSecret(),
			MaxUploadSize: 1073741824,
		}
		C = c
		return c, nil
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if c.DataRootPath != "" {
		absPath, err := filepath.Abs(c.DataRootPath)
		if err != nil {
			return nil, fmt.Errorf("resolve data_root_path: %w", err)
		}
		c.DataRootPath = absPath
		if !strings.HasSuffix(c.DataRootPath, string(filepath.Separator)) {
			c.DataRootPath += string(filepath.Separator)
		}
	}
	if c.ServerPort == 0 {
		c.ServerPort = 8080
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.DBPath == "" {
		c.DBPath = "wfm.db"
	}
	if c.SessionSecret == "" {
		c.SessionSecret = RandomSecret()
	}
	if c.MaxUploadSize == 0 {
		c.MaxUploadSize = 1073741824
	}
	C = &c
	return &c, nil
}

func Save(path string, c *Config) error {
	if c.DataRootPath != "" {
		absPath, err := filepath.Abs(c.DataRootPath)
		if err != nil {
			return fmt.Errorf("resolve data_root_path: %w", err)
		}
		c.DataRootPath = absPath
		if !strings.HasSuffix(c.DataRootPath, string(filepath.Separator)) {
			c.DataRootPath += string(filepath.Separator)
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func RandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
