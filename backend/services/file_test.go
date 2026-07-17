package services

import (
	"os"
	"path/filepath"
	"testing"

	"goWFM/config"
	"goWFM/db"
	"goWFM/models"
)

func setupFileTestDB(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	if err := db.Init(filepath.Join(tempDir, "file.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	config.InitDefaults()
	basic := config.GetBasic()
	basic.DataRootPath = filepath.Join(tempDir, "data")
	config.SetBasic(basic)
	if err := os.MkdirAll(basic.DataRootPath, 0755); err != nil {
		t.Fatalf("create data root: %v", err)
	}
	if _, err := db.DB.Exec(
		`INSERT INTO users (id, username, password_hash, display_name) VALUES (1, 'admin', '', 'Admin'), (2, 'user', '', 'User')`,
	); err != nil {
		t.Fatalf("create users: %v", err)
	}
	t.Cleanup(db.Close)
	return basic.DataRootPath
}

func TestCanUploadToDirectoryAllowsOwnerWithoutGlobalPermission(t *testing.T) {
	dataRoot := setupFileTestDB(t)
	directoryPath := filepath.Join(dataRoot, "owned")
	if err := os.Mkdir(directoryPath, 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := CreateFileMetadata("/owned", true, 2); err != nil {
		t.Fatalf("create metadata: %v", err)
	}

	allowed, err := CanUploadToDirectory("/owned", &models.User{ID: 2, Permissions: models.PermBrowse})
	if err != nil {
		t.Fatalf("check upload permission: %v", err)
	}
	if !allowed {
		t.Fatal("directory owner without global upload permission should be allowed")
	}
}

func TestCanUploadToDirectoryRequiresOwnershipOrGlobalPermission(t *testing.T) {
	dataRoot := setupFileTestDB(t)
	directoryPath := filepath.Join(dataRoot, "someone-elses")
	if err := os.Mkdir(directoryPath, 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := CreateFileMetadata("/someone-elses", true, 1); err != nil {
		t.Fatalf("create metadata: %v", err)
	}

	regularUser := &models.User{ID: 2, Permissions: models.PermBrowse}
	allowed, err := CanUploadToDirectory("/someone-elses", regularUser)
	if err != nil {
		t.Fatalf("check regular user permission: %v", err)
	}
	if allowed {
		t.Fatal("non-owner without global upload permission should be denied")
	}

	regularUser.Permissions |= models.PermGlobalUpload
	allowed, err = CanUploadToDirectory("/someone-elses", regularUser)
	if err != nil {
		t.Fatalf("check global upload permission: %v", err)
	}
	if !allowed {
		t.Fatal("global upload permission should allow uploading to another user's directory")
	}
}

func TestUpdateFileMetadataOwnerCreatesMetadataForExistingFile(t *testing.T) {
	dataRoot := setupFileTestDB(t)
	filePath := filepath.Join(dataRoot, "existing.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	if err := UpdateFileMetadataOwner("/existing.txt", 2); err != nil {
		t.Fatalf("change owner: %v", err)
	}

	metadata, err := GetFileMetadata("/existing.txt")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if metadata.OwnerID != 2 {
		t.Fatalf("owner id = %d, want 2", metadata.OwnerID)
	}
	if metadata.IsDirectory {
		t.Fatal("file was stored as a directory")
	}
}

func TestUpdateFileMetadataOwnerUpdatesExistingDirectoryMetadata(t *testing.T) {
	dataRoot := setupFileTestDB(t)
	directoryPath := filepath.Join(dataRoot, "reports")
	if err := os.Mkdir(directoryPath, 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := CreateFileMetadata("/reports", true, 1); err != nil {
		t.Fatalf("create metadata: %v", err)
	}

	if err := UpdateFileMetadataOwner("/reports", 2); err != nil {
		t.Fatalf("change owner: %v", err)
	}

	metadata, err := GetFileMetadata("/reports")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if metadata.OwnerID != 2 {
		t.Fatalf("owner id = %d, want 2", metadata.OwnerID)
	}
	if !metadata.IsDirectory {
		t.Fatal("directory was stored as a file")
	}
}
