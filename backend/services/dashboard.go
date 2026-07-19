package services

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"goWFM/config"
	"goWFM/db"
)

type DashboardStorageCategory struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Bytes int64  `json:"bytes"`
	Count int64  `json:"count"`
}

type DashboardScanStatus struct {
	Ready           bool       `json:"ready"`
	Scanning        bool       `json:"scanning"`
	LastReason      string     `json:"last_reason"`
	LastError       string     `json:"last_error"`
	LastStartedAt   *time.Time `json:"last_started_at"`
	LastCompletedAt *time.Time `json:"last_completed_at"`
	NextScanAt      *time.Time `json:"next_scan_at"`
}

type DashboardStorage struct {
	RootPath           string                     `json:"root_path"`
	SharedUsedBytes    int64                      `json:"shared_used_bytes"`
	DiskAvailableBytes uint64                     `json:"disk_available_bytes"`
	DiskTotalBytes     uint64                     `json:"disk_total_bytes"`
	FileCount          int64                      `json:"file_count"`
	DirectoryCount     int64                      `json:"directory_count"`
	ScanComplete       bool                       `json:"scan_complete"`
	ScannedAt          time.Time                  `json:"scanned_at"`
	Categories         []DashboardStorageCategory `json:"categories"`
}

type DashboardSummary struct {
	Users struct {
		Total       int `json:"total"`
		Admins      int `json:"admins"`
		TOTPEnabled int `json:"totp_enabled"`
	} `json:"users"`
	Shares struct {
		Valid        int   `json:"valid"`
		ExpiringSoon int   `json:"expiring_soon"`
		Expired      int   `json:"expired"`
		AccessCount  int64 `json:"access_count"`
	} `json:"shares"`
	Today map[string]int `json:"today"`
}

type DashboardActivityDay struct {
	Date        string `json:"date"`
	Upload      int    `json:"upload"`
	Download    int    `json:"download"`
	ShareCreate int    `json:"share_create"`
	LoginFail   int    `json:"login_fail"`
}

type DashboardRecentLog struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	Action     string `json:"action"`
	TargetPath string `json:"target_path"`
	IPAddress  string `json:"ip_address"`
	CreatedAt  string `json:"created_at"`
}

type DashboardRecentShare struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	AccessCount int    `json:"access_count"`
	ExpireAt    string `json:"expire_at"`
}

type DashboardSecurityCheck struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	OK          bool   `json:"ok"`
	Optional    bool   `json:"optional,omitempty"`
	Status      string `json:"status,omitempty"`
}

type DashboardData struct {
	GeneratedAt     time.Time                `json:"generated_at"`
	Storage         DashboardStorage         `json:"storage"`
	Summary         DashboardSummary         `json:"summary"`
	Activity        []DashboardActivityDay   `json:"activity"`
	RecentLogs      []DashboardRecentLog     `json:"recent_logs"`
	RecentShares    []DashboardRecentShare   `json:"recent_shares"`
	Security        []DashboardSecurityCheck `json:"security"`
	ScanStatus      DashboardScanStatus      `json:"scan_status"`
	ActivityDays    int                      `json:"activity_days"`
	ActivityMaxDays int                      `json:"activity_max_days"`
}

var dashboardStorageState struct {
	mu              sync.RWMutex
	scanGate        sync.RWMutex
	value           DashboardStorage
	ready           bool
	scanning        bool
	lastReason      string
	lastError       string
	lastStartedAt   *time.Time
	lastCompletedAt *time.Time
	paths           map[string]DashboardStoragePathState
}

func GetDashboardData() (*DashboardData, error) {
	storage := GetDashboardStorageSnapshot()
	result := &DashboardData{GeneratedAt: time.Now(), Storage: storage}
	var err error
	if err := loadDashboardSummary(&result.Summary); err != nil {
		return nil, err
	}
	result.ActivityDays, result.ActivityMaxDays = dashboardActivityRange(7)
	if result.Activity, err = loadDashboardActivity(result.ActivityDays); err != nil {
		return nil, err
	}
	if result.RecentLogs, err = loadDashboardRecentLogs(8); err != nil {
		return nil, err
	}
	if result.RecentShares, err = loadDashboardRecentShares(5); err != nil {
		return nil, err
	}
	result.Security = dashboardSecurityChecks(result.Summary)
	result.ScanStatus = GetDashboardScanStatus()
	return result, nil
}

func GetDashboardActivity(days int) ([]DashboardActivityDay, int, int, error) {
	days, maxDays := dashboardActivityRange(days)
	activity, err := loadDashboardActivity(days)
	return activity, days, maxDays, err
}

func dashboardActivityRange(requested int) (int, int) {
	maxDays := config.GetLog().RetentionDays
	if maxDays < 1 {
		maxDays = 1
	}
	if requested < 1 {
		requested = 7
	}
	if requested > maxDays {
		requested = maxDays
	}
	return requested, maxDays
}

// InitializeDashboardStorage performs the required full scan during process startup.
func InitializeDashboardStorage() error {
	root := config.GetBasic().DataRootPath
	if root == "" {
		return nil
	}
	return runDashboardStorageScan("startup")
}

// TriggerDashboardStorageScan starts an asynchronous full scan when none is running.
func TriggerDashboardStorageScan(reason string) bool {
	dashboardStorageState.mu.Lock()
	if dashboardStorageState.scanning {
		dashboardStorageState.mu.Unlock()
		return false
	}
	dashboardStorageState.scanning = true
	now := time.Now()
	dashboardStorageState.lastStartedAt = &now
	dashboardStorageState.lastReason = reason
	dashboardStorageState.lastError = ""
	dashboardStorageState.mu.Unlock()
	go func() { _ = performDashboardStorageScan(reason) }()
	return true
}

func runDashboardStorageScan(reason string) error {
	dashboardStorageState.mu.Lock()
	if dashboardStorageState.scanning {
		dashboardStorageState.mu.Unlock()
		return fmt.Errorf("storage scan already running")
	}
	dashboardStorageState.scanning = true
	now := time.Now()
	dashboardStorageState.lastStartedAt = &now
	dashboardStorageState.lastReason = reason
	dashboardStorageState.lastError = ""
	dashboardStorageState.mu.Unlock()
	return performDashboardStorageScan(reason)
}

func performDashboardStorageScan(reason string) error {
	root := config.GetBasic().DataRootPath
	dashboardStorageState.scanGate.Lock()
	defer dashboardStorageState.scanGate.Unlock()
	value, paths, err := collectDashboardStorageDetailed(root)

	dashboardStorageState.mu.Lock()
	defer dashboardStorageState.mu.Unlock()
	dashboardStorageState.scanning = false
	dashboardStorageState.lastReason = reason
	if err != nil {
		dashboardStorageState.lastError = err.Error()
		return err
	}
	now := time.Now()
	dashboardStorageState.value = value
	dashboardStorageState.paths = paths
	dashboardStorageState.ready = true
	dashboardStorageState.lastError = ""
	dashboardStorageState.lastCompletedAt = &now
	return nil
}

func GetDashboardStorageSnapshot() DashboardStorage {
	dashboardStorageState.mu.RLock()
	value := dashboardStorageState.value
	categories := make([]DashboardStorageCategory, len(value.Categories))
	copy(categories, value.Categories)
	value.Categories = categories
	dashboardStorageState.mu.RUnlock()
	root := config.GetBasic().DataRootPath
	if value.RootPath == "" {
		value.RootPath = root
		value.Categories = make([]DashboardStorageCategory, 0)
	}
	if root != "" {
		if available, total, err := diskSpace(root); err == nil {
			value.DiskAvailableBytes = available
			value.DiskTotalBytes = total
		}
	}
	return value
}

func GetDashboardScanStatus() DashboardScanStatus {
	dashboardStorageState.mu.RLock()
	status := DashboardScanStatus{
		Ready: dashboardStorageState.ready, Scanning: dashboardStorageState.scanning,
		LastReason: dashboardStorageState.lastReason, LastError: dashboardStorageState.lastError,
		LastStartedAt: dashboardStorageState.lastStartedAt, LastCompletedAt: dashboardStorageState.lastCompletedAt,
	}
	dashboardStorageState.mu.RUnlock()
	cfg := config.GetScan()
	if cfg.AutoScanEnabled && status.LastCompletedAt != nil {
		next := status.LastCompletedAt.Add(time.Duration(cfg.IntervalHours) * time.Hour)
		status.NextScanAt = &next
	}
	return status
}

// RunDashboardScanScheduler periodically checks whether an enabled automatic scan is due.
func RunDashboardScanScheduler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cfg := config.GetScan()
		if !cfg.AutoScanEnabled {
			continue
		}
		status := GetDashboardScanStatus()
		if status.Scanning {
			continue
		}
		if status.LastCompletedAt == nil || time.Now().After(status.LastCompletedAt.Add(time.Duration(cfg.IntervalHours)*time.Hour)) {
			TriggerDashboardStorageScan("scheduled")
		}
	}
}

func collectDashboardStorage(root string) (DashboardStorage, error) {
	storage, _, err := collectDashboardStorageDetailed(root)
	return storage, err
}

func collectDashboardStorageDetailed(root string) (DashboardStorage, map[string]DashboardStoragePathState, error) {
	if _, err := os.Stat(root); err != nil {
		return DashboardStorage{}, nil, fmt.Errorf("read data root: %w", err)
	}
	available, total, err := diskSpace(root)
	if err != nil {
		return DashboardStorage{}, nil, fmt.Errorf("read disk space: %w", err)
	}
	result := DashboardStorage{
		RootPath: root, DiskAvailableBytes: available, DiskTotalBytes: total,
		ScanComplete: true, ScannedAt: time.Now(), Categories: make([]DashboardStorageCategory, 0),
	}
	categories := map[string]*DashboardStorageCategory{
		"document": {Key: "document", Label: "文档"},
		"image":    {Key: "image", Label: "图片"},
		"video":    {Key: "video", Label: "视频"},
		"audio":    {Key: "audio", Label: "音频"},
		"archive":  {Key: "archive", Label: "压缩包"},
		"other":    {Key: "other", Label: "其它"},
	}
	seenFiles := make(map[string]struct{})
	paths := make(map[string]DashboardStoragePathState)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.ScanComplete = false
			if path == root {
				return walkErr
			}
			return nil
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			result.DirectoryCount++
			paths[storagePathKey(root, path)] = DashboardStoragePathState{RelativePath: storagePathKey(root, path), IsDirectory: true}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			result.ScanComplete = false
			return nil
		}
		result.FileCount++
		category := categories[fileCategory(entry.Name())]
		category.Count++
		size, identity := fileDiskUsage(path, info)
		state := DashboardStoragePathState{RelativePath: storagePathKey(root, path), Bytes: size, Category: category.Key}
		if identity != "" {
			if _, exists := seenFiles[identity]; exists {
				state.Bytes = 0
				paths[state.RelativePath] = state
				return nil
			}
			seenFiles[identity] = struct{}{}
		}
		result.SharedUsedBytes += size
		category.Bytes += size
		paths[state.RelativePath] = state
		return nil
	})
	if err != nil {
		return DashboardStorage{}, nil, fmt.Errorf("scan data root: %w", err)
	}
	for _, category := range categories {
		if category.Count > 0 {
			result.Categories = append(result.Categories, *category)
		}
	}
	sort.Slice(result.Categories, func(i, j int) bool { return result.Categories[i].Bytes > result.Categories[j].Bytes })
	return result, paths, nil
}

func storagePathKey(root, fullPath string) string {
	relative, err := filepath.Rel(root, fullPath)
	if err != nil || relative == "." {
		return string(filepath.Separator)
	}
	return filepath.Clean(string(filepath.Separator) + relative)
}

type DashboardStoragePathState struct {
	RelativePath string
	IsDirectory  bool
	Bytes        int64
	Category     string
}

func CaptureDashboardStoragePath(relativePath string) (DashboardStoragePathState, error) {
	fullPath, err := SafePath(relativePath)
	if err != nil {
		return DashboardStoragePathState{}, err
	}
	info, err := os.Lstat(fullPath)
	if err != nil {
		return DashboardStoragePathState{}, err
	}
	state := DashboardStoragePathState{RelativePath: filepath.Clean(relativePath), IsDirectory: info.IsDir()}
	if !state.IsDirectory {
		state.Bytes, _ = fileDiskUsage(fullPath, info)
		state.Category = fileCategory(fullPath)
	}
	return state, nil
}

func RecordDashboardStorageCreated(relativePath string) {
	state, err := CaptureDashboardStoragePath(relativePath)
	if err != nil {
		return
	}
	applyDashboardStorageDelta(state, 1)
}

func RecordDashboardStorageDeleted(state DashboardStoragePathState) {
	applyDashboardStorageDelta(state, -1)
}

func RecordDashboardStorageMoved(previous DashboardStoragePathState, newRelativePath string) {
	dashboardStorageState.scanGate.RLock()
	defer dashboardStorageState.scanGate.RUnlock()
	dashboardStorageState.mu.Lock()
	defer dashboardStorageState.mu.Unlock()
	if !dashboardStorageState.ready {
		return
	}
	oldKey := filepath.Clean(previous.RelativePath)
	newKey := filepath.Clean(newRelativePath)
	stored, exists := dashboardStorageState.paths[oldKey]
	if !exists {
		return
	}
	if stored.IsDirectory {
		moved := make(map[string]DashboardStoragePathState)
		prefix := oldKey + string(filepath.Separator)
		for key, state := range dashboardStorageState.paths {
			if key != oldKey && !strings.HasPrefix(key, prefix) {
				continue
			}
			suffix := strings.TrimPrefix(key, oldKey)
			delete(dashboardStorageState.paths, key)
			state.RelativePath = newKey + suffix
			moved[state.RelativePath] = state
		}
		for key, state := range moved {
			dashboardStorageState.paths[key] = state
		}
		return
	}
	newCategory := fileCategory(newRelativePath)
	if stored.Category != newCategory {
		adjustStorageCategory(&dashboardStorageState.value, stored.Category, -stored.Bytes, -1)
		adjustStorageCategory(&dashboardStorageState.value, newCategory, stored.Bytes, 1)
		stored.Category = newCategory
	}
	delete(dashboardStorageState.paths, oldKey)
	stored.RelativePath = newKey
	dashboardStorageState.paths[newKey] = stored
	sort.Slice(dashboardStorageState.value.Categories, func(i, j int) bool {
		return dashboardStorageState.value.Categories[i].Bytes > dashboardStorageState.value.Categories[j].Bytes
	})
}

func applyDashboardStorageDelta(state DashboardStoragePathState, direction int64) {
	dashboardStorageState.scanGate.RLock()
	defer dashboardStorageState.scanGate.RUnlock()
	dashboardStorageState.mu.Lock()
	defer dashboardStorageState.mu.Unlock()
	if !dashboardStorageState.ready {
		return
	}
	if dashboardStorageState.paths == nil {
		dashboardStorageState.paths = make(map[string]DashboardStoragePathState)
	}
	key := filepath.Clean(state.RelativePath)
	if direction > 0 {
		if _, exists := dashboardStorageState.paths[key]; exists {
			return
		}
		state.RelativePath = key
		dashboardStorageState.paths[key] = state
	} else {
		stored, exists := dashboardStorageState.paths[key]
		if !exists {
			return
		}
		state = stored
		delete(dashboardStorageState.paths, key)
	}
	if state.IsDirectory {
		dashboardStorageState.value.DirectoryCount += direction
		if dashboardStorageState.value.DirectoryCount < 0 {
			dashboardStorageState.value.DirectoryCount = 0
		}
		return
	}
	dashboardStorageState.value.FileCount += direction
	dashboardStorageState.value.SharedUsedBytes += direction * state.Bytes
	if dashboardStorageState.value.FileCount < 0 {
		dashboardStorageState.value.FileCount = 0
	}
	if dashboardStorageState.value.SharedUsedBytes < 0 {
		dashboardStorageState.value.SharedUsedBytes = 0
	}
	adjustStorageCategory(&dashboardStorageState.value, state.Category, direction*state.Bytes, direction)
	sort.Slice(dashboardStorageState.value.Categories, func(i, j int) bool {
		return dashboardStorageState.value.Categories[i].Bytes > dashboardStorageState.value.Categories[j].Bytes
	})
}

func adjustStorageCategory(storage *DashboardStorage, key string, byteDelta, countDelta int64) {
	for index := range storage.Categories {
		if storage.Categories[index].Key != key {
			continue
		}
		storage.Categories[index].Bytes += byteDelta
		storage.Categories[index].Count += countDelta
		if storage.Categories[index].Bytes < 0 {
			storage.Categories[index].Bytes = 0
		}
		if storage.Categories[index].Count <= 0 {
			storage.Categories = append(storage.Categories[:index], storage.Categories[index+1:]...)
		}
		return
	}
	if countDelta <= 0 {
		return
	}
	labels := map[string]string{"document": "文档", "image": "图片", "video": "视频", "audio": "音频", "archive": "压缩包", "other": "其它"}
	storage.Categories = append(storage.Categories, DashboardStorageCategory{Key: key, Label: labels[key], Bytes: byteDelta, Count: countDelta})
}

func fileCategory(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".doc", ".docx", ".pdf", ".ppt", ".pptx", ".txt", ".xls", ".xlsx", ".csv", ".xml", ".md":
		return "document"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".heic":
		return "image"
	case ".mp4", ".mov", ".avi", ".mkv", ".flv", ".mpeg", ".webm":
		return "video"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a":
		return "audio"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".iso":
		return "archive"
	default:
		return "other"
	}
}

func loadDashboardSummary(summary *DashboardSummary) error {
	if err := db.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(is_admin),0), COALESCE(SUM(totp_enabled),0) FROM users WHERE id != 0`).
		Scan(&summary.Users.Total, &summary.Users.Admins, &summary.Users.TOTPEnabled); err != nil {
		return err
	}
	if err := db.DB.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN deleted = 0 AND (expire_at IS NULL OR datetime(expire_at) >= datetime('now')) THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN deleted = 0 AND expire_at IS NOT NULL AND datetime(expire_at) >= datetime('now') AND datetime(expire_at) <= datetime('now','+3 days') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN deleted != 0 OR (expire_at IS NOT NULL AND datetime(expire_at) < datetime('now')) THEN 1 ELSE 0 END),0),
		COALESCE(SUM(access_count),0) FROM shares`).
		Scan(&summary.Shares.Valid, &summary.Shares.ExpiringSoon, &summary.Shares.Expired, &summary.Shares.AccessCount); err != nil {
		return err
	}
	summary.Today = map[string]int{"upload": 0, "download": 0, "share_access": 0, "login_fail": 0}
	rows, err := db.DB.Query(`SELECT action, COUNT(*) FROM operation_logs
		WHERE date(created_at,'localtime') = date('now','localtime') GROUP BY action`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return err
		}
		switch action {
		case "UPLOAD":
			summary.Today["upload"] = count
		case "DOWNLOAD":
			summary.Today["download"] = count
		case "SHARE_ACCESS":
			summary.Today["share_access"] = count
		case "LOGIN_FAIL", "LOGIN_TOTP_FAIL":
			summary.Today["login_fail"] += count
		}
	}
	return rows.Err()
}

func loadDashboardActivity(days int) ([]DashboardActivityDay, error) {
	start := time.Now().AddDate(0, 0, -(days - 1))
	result := make([]DashboardActivityDay, days)
	index := make(map[string]int, days)
	for i := range result {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		result[i].Date = date
		index[date] = i
	}
	rows, err := db.DB.Query(`SELECT date(created_at,'localtime'), action, COUNT(*) FROM operation_logs
		WHERE date(created_at,'localtime') >= ? GROUP BY date(created_at,'localtime'), action`, start.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var date, action string
		var count int
		if err := rows.Scan(&date, &action, &count); err != nil {
			return nil, err
		}
		i, ok := index[date]
		if !ok {
			continue
		}
		switch action {
		case "UPLOAD":
			result[i].Upload += count
		case "DOWNLOAD":
			result[i].Download += count
		case "SHARE_CREATE":
			result[i].ShareCreate += count
		case "LOGIN_FAIL", "LOGIN_TOTP_FAIL":
			result[i].LoginFail += count
		}
	}
	return result, rows.Err()
}

func loadDashboardRecentLogs(limit int) ([]DashboardRecentLog, error) {
	rows, err := db.DB.Query(`SELECT l.id, COALESCE(u.username,'Guest'), l.action, COALESCE(l.target_path,''),
		COALESCE(l.ip_address,''), l.created_at FROM operation_logs l LEFT JOIN users u ON u.id=l.user_id
		ORDER BY l.created_at DESC, l.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DashboardRecentLog, 0, limit)
	for rows.Next() {
		var item DashboardRecentLog
		if err := rows.Scan(&item.ID, &item.Username, &item.Action, &item.TargetPath, &item.IPAddress, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func loadDashboardRecentShares(limit int) ([]DashboardRecentShare, error) {
	rows, err := db.DB.Query(`SELECT s.id, COALESCE(NULLIF(s.name,''),
		CASE WHEN COUNT(sf.id)>1 THEN '分享' || COUNT(sf.id) || '个文件' ELSE COALESCE(MAX(sf.file_path),'未命名分享') END),
		COALESCE(NULLIF(u.display_name,''),u.username), s.deleted, s.expire_at, s.access_count
		FROM shares s JOIN users u ON u.id=s.owner_id LEFT JOIN share_files sf ON sf.share_id=s.id
		GROUP BY s.id ORDER BY s.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DashboardRecentShare, 0, limit)
	for rows.Next() {
		var item DashboardRecentShare
		var deleted bool
		var expire sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.Owner, &deleted, &expire, &item.AccessCount); err != nil {
			return nil, err
		}
		item.Name = filepath.Base(item.Name)
		if expire.Valid {
			item.ExpireAt = expire.String
		}
		item.Status = "valid"
		if deleted {
			item.Status = "deleted"
		} else if expire.Valid {
			if t, parseErr := time.Parse(time.RFC3339, expire.String); parseErr == nil && t.Before(time.Now()) {
				item.Status = "expired"
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func dashboardSecurityChecks(summary DashboardSummary) []DashboardSecurityCheck {
	security := config.GetSecurity()
	appearance := config.GetAppearance()
	email := config.GetEmail()
	scan := config.GetScan()
	scanDescription := "未启用；仅在存在绕过 goWFM 的外部文件操作时需要"
	scanStatus := "未启用"
	if scan.AutoScanEnabled {
		scanDescription = fmt.Sprintf("已启用，每 %d 小时完整扫描一次", scan.IntervalHours)
		scanStatus = "已启用"
	}
	return []DashboardSecurityCheck{
		{Key: "https", Label: "HTTPS 传输", Description: "为登录与文件传输提供加密保护", OK: appearance.EnableHTTPS},
		{Key: "totp", Label: "管理员双重认证", Description: fmt.Sprintf("%d 位用户已启用 TOTP", summary.Users.TOTPEnabled), OK: summary.Users.TOTPEnabled >= summary.Users.Admins && summary.Users.Admins > 0},
		{Key: "ip_block", Label: "IP 登录防护", Description: "连续登录失败时临时封锁来源 IP", OK: security.IPBlockEnabled},
		{Key: "account_block", Label: "账户登录防护", Description: "连续登录失败时临时封锁账户", OK: security.AccountBlockEnabled},
		{Key: "captcha", Label: "登录验证码", Description: "降低自动化登录尝试风险", OK: security.EnableCaptcha},
		{Key: "email", Label: "邮件服务", Description: "用于密码重置和安全通知", OK: strings.TrimSpace(email.SMTPHost) != "" && strings.TrimSpace(email.SenderEmail) != ""},
		{Key: "storage_scan", Label: "后台磁盘扫描", Description: scanDescription, OK: scan.AutoScanEnabled, Optional: true, Status: scanStatus},
	}
}
