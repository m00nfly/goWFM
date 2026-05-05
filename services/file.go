package services

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goWFM/config"
	"goWFM/db"
	"goWFM/models"
)

// SafePath 获取安全的文件路径，防止路径遍历攻击
func SafePath(relativePath string) (string, error) {
	if config.C.DataRootPath == "" {
		return "", fmt.Errorf("data_root_path not configured, please complete setup first")
	}
	if relativePath == "" || relativePath == "/" {
		return config.C.DataRootPath, nil
	}
	cleaned := filepath.Clean(relativePath)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected")
	}
	fullPath := filepath.Join(config.C.DataRootPath, cleaned)
	if !strings.HasPrefix(fullPath+string(filepath.Separator), config.C.DataRootPath) && fullPath != config.C.DataRootPath {
		return "", fmt.Errorf("path traversal detected")
	}
	return fullPath, nil
}

// RelativePath 转换绝对路径为安全的相对路径，返回从DataRoot目录开始的相对路径
func RelativePath(fullPath string) string {
	rel := strings.TrimPrefix(fullPath, config.C.DataRootPath)
	if rel == "" {
		return "/"
	}
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	return rel
}

type FileEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDirectory bool      `json:"is_directory"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	OwnerName   string    `json:"owner_name"`
	OwnerID     int64     `json:"owner_id"`
	CreatedAt   string    `json:"created_at"`
	CanDelete   bool      `json:"can_delete"`
	CanDownload bool      `json:"can_download"`
	CanShare    bool      `json:"can_share"`
	CanChange   bool      `json:"can_change"`
}

func ListDirectory(relativePath string, user *models.User) ([]FileEntry, error) {
	fullPath, err := SafePath(relativePath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	result := make([]FileEntry, 0)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		relPath := RelativePath(filepath.Join(fullPath, e.Name()))
		entry := FileEntry{
			Name:        e.Name(),
			Path:        relPath,
			IsDirectory: e.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			CanDownload: !e.IsDir() && user.HasPermission(models.PermDownload),
			CanShare:    !e.IsDir() && user.HasPermission(models.PermShare),
			CanDelete:   user.IsAdmin,
		}

		meta, _ := GetFileMetadata(relPath)
		if meta != nil {
			entry.OwnerID = meta.OwnerID
			owner, _ := GetUserByID(meta.OwnerID)
			if owner != nil {
				entry.OwnerName = owner.DisplayName
				if entry.OwnerName == "" {
					entry.OwnerName = owner.Username
				}
			}
			entry.CreatedAt = meta.CreatedAt.Format(time.RFC3339)
			if !user.IsAdmin && entry.OwnerID == user.ID {
				entry.CanDelete = true
			}
		} else {
			owner, _ := GetUserByID(1)
			entry.OwnerName = owner.DisplayName
			entry.OwnerID = owner.ID
			entry.CreatedAt = info.ModTime().Format(time.RFC3339)
		}

		if (user.HasPermission(models.PermUpload) && entry.OwnerID == user.ID) || user.IsAdmin {
			entry.CanChange = true
		}

		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDirectory != result[j].IsDirectory {
			return result[i].IsDirectory
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func GetFileMetadata(relativePath string) (*models.FileMetadata, error) {
	m := &models.FileMetadata{}
	err := db.DB.QueryRow(
		`SELECT id, file_path, is_directory, owner_id, created_at, updated_at FROM file_metadata WHERE file_path = ?`,
		relativePath,
	).Scan(&m.ID, &m.FilePath, &m.IsDirectory, &m.OwnerID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func CreateFileMetadata(relativePath string, isDirectory bool, ownerID int64) error {
	_, err := db.DB.Exec(
		`INSERT INTO file_metadata (file_path, is_directory, owner_id) VALUES (?, ?, ?)`,
		relativePath, isDirectory, ownerID,
	)
	return err
}

func DeleteFileMetadata(relativePath string) error {
	_, err := db.DB.Exec(`DELETE FROM file_metadata WHERE file_path = ?`, relativePath)
	return err
}

func UpdateFileMetadataOwner(relativePath string, ownerID int64) error {
	_, err := db.DB.Exec(
		`UPDATE file_metadata SET owner_id = ?, updated_at = CURRENT_TIMESTAMP WHERE file_path = ?`,
		ownerID, relativePath,
	)
	return err
}

func UpdateFileMetadataPath(oldPath, newPath string) error {
	_, err := db.DB.Exec(
		`UPDATE file_metadata SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE file_path = ?`,
		newPath, oldPath,
	)
	if err != nil {
		return err
	}
	// 同步更新分享记录中的文件路径
	_, err = db.DB.Exec(`UPDATE shares SET file_path = ? WHERE file_path = ?`, newPath, oldPath)
	return err
}

func IsDirEmpty(fullPath string) (bool, error) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func DownloadFile(relativePath string) (string, error) {
	fullPath, err := SafePath(relativePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("file not found")
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot download directory")
	}

	return fullPath, nil
}

func CreateDirectory(relativePath string, ownerID int64) error {
	fullPath, err := SafePath(relativePath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("directory already exists")
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return err
	}

	return CreateFileMetadata(relativePath, true, ownerID)
}

func DeleteFileOrDir(relativePath string, user *models.User) error {
	fullPath, err := SafePath(relativePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("path not found")
	}

	if info.IsDir() {
		empty, err := IsDirEmpty(fullPath)
		if err != nil {
			return err
		}
		if !empty {
			return fmt.Errorf("directory is not empty")
		}
	}

	if !user.IsAdmin {
		meta, _ := GetFileMetadata(relativePath)
		if meta == nil || meta.OwnerID != user.ID {
			return fmt.Errorf("permission denied: you can only delete your own files")
		}
	}

	if err := os.Remove(fullPath); err != nil {
		return err
	}

	// 标记相关分享记录为已删除
	db.DB.Exec(`UPDATE shares SET deleted = 1 WHERE file_path = ?`, relativePath)

	DeleteFileMetadata(relativePath)
	return nil
}

func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	invalid := []string{"..", "/", "\\", "<", ">", "|", "?", "*", ":"}
	for _, ch := range invalid {
		name = strings.ReplaceAll(name, ch, "")
	}
	return name
}

func ResolveDuplicatePath(fullPath string) string {
	if _, err := os.Stat(fullPath); err != nil {
		return fullPath
	}

	ext := filepath.Ext(fullPath)
	base := fullPath[:len(fullPath)-len(ext)]
	counter := 1
	for {
		newPath := fmt.Sprintf("%s (%d)%s", base, counter, ext)
		if _, err := os.Stat(newPath); err != nil {
			return newPath
		}
		counter++
	}
}

func WalkDir(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}
