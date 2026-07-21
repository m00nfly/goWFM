package services

import (
	"os"
	"path/filepath"
	"testing"

	"goWFM/config"
)

func TestCollectDashboardStorageOnlyCountsConfiguredRoot(t *testing.T) {
	partitionRoot := t.TempDir()
	sharedRoot := filepath.Join(partitionRoot, "shared")
	outsideRoot := filepath.Join(partitionRoot, "outside")
	if err := os.MkdirAll(filepath.Join(sharedRoot, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideRoot, 0755); err != nil {
		t.Fatal(err)
	}
	sharedContent := []byte("shared content")
	outsideContent := make([]byte, 128*1024)
	if err := os.WriteFile(filepath.Join(sharedRoot, "docs", "report.pdf"), sharedContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideRoot, "not-shared.bin"), outsideContent, 0644); err != nil {
		t.Fatal(err)
	}

	storage, err := collectDashboardStorage(sharedRoot)
	if err != nil {
		t.Fatalf("collect storage: %v", err)
	}
	if storage.FileCount != 1 || storage.DirectoryCount != 1 {
		t.Fatalf("unexpected counts: files=%d directories=%d", storage.FileCount, storage.DirectoryCount)
	}
	if storage.SharedUsedBytes < int64(len(sharedContent)) {
		t.Fatalf("shared used bytes = %d, want at least %d", storage.SharedUsedBytes, len(sharedContent))
	}
	if storage.SharedUsedBytes >= int64(len(outsideContent)) {
		t.Fatalf("outside file appears to be counted: shared used bytes = %d", storage.SharedUsedBytes)
	}
	if storage.DiskAvailableBytes == 0 || storage.DiskTotalBytes == 0 {
		t.Fatalf("disk metrics missing: available=%d total=%d", storage.DiskAvailableBytes, storage.DiskTotalBytes)
	}
	if len(storage.Categories) != 1 || storage.Categories[0].Key != "document" {
		t.Fatalf("unexpected categories: %#v", storage.Categories)
	}
}

func TestFileCategory(t *testing.T) {
	tests := map[string]string{
		"report.PDF": "document",
		"photo.webp": "image",
		"movie.mkv":  "video",
		"voice.flac": "audio",
		"backup.7z":  "archive",
		"binary":     "other",
	}
	for name, want := range tests {
		if got := fileCategory(name); got != want {
			t.Errorf("fileCategory(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCollectDashboardStorageReturnsEmptyCategoriesForEmptyRoot(t *testing.T) {
	storage, err := collectDashboardStorage(t.TempDir())
	if err != nil {
		t.Fatalf("collect empty storage: %v", err)
	}
	if storage.Categories == nil || len(storage.Categories) != 0 {
		t.Fatalf("empty categories must be a non-nil empty slice: %#v", storage.Categories)
	}
}

func TestCollectDashboardStorageDoesNotDoubleCountHardLinks(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "original.bin")
	if err := os.WriteFile(original, make([]byte, 32*1024), 0644); err != nil {
		t.Fatal(err)
	}
	before, err := collectDashboardStorage(root)
	if err != nil {
		t.Fatalf("collect before link: %v", err)
	}
	if err := os.Link(original, filepath.Join(root, "linked.bin")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	after, err := collectDashboardStorage(root)
	if err != nil {
		t.Fatalf("collect after link: %v", err)
	}
	if after.FileCount != 2 {
		t.Fatalf("file entries = %d, want 2", after.FileCount)
	}
	if after.SharedUsedBytes != before.SharedUsedBytes {
		t.Fatalf("hard link doubled disk usage: before=%d after=%d", before.SharedUsedBytes, after.SharedUsedBytes)
	}
}

func TestDashboardStorageSnapshotUpdatesForGoWFMFileOperations(t *testing.T) {
	config.InitDefaults()
	root := t.TempDir()
	basic := config.GetBasic()
	basic.DataRootPath = root
	config.SetBasic(basic)

	dashboardStorageState.mu.Lock()
	dashboardStorageState.value = DashboardStorage{}
	dashboardStorageState.ready = false
	dashboardStorageState.scanning = false
	dashboardStorageState.lastError = ""
	dashboardStorageState.lastStartedAt = nil
	dashboardStorageState.lastCompletedAt = nil
	dashboardStorageState.mu.Unlock()

	if err := runDashboardStorageScan("test"); err != nil {
		t.Fatalf("initial scan: %v", err)
	}
	initial := GetDashboardStorageSnapshot()
	if initial.FileCount != 0 || initial.Categories == nil || !GetDashboardScanStatus().Ready {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}

	filePath := filepath.Join(root, "report.txt")
	if err := os.WriteFile(filePath, make([]byte, 4096), 0644); err != nil {
		t.Fatal(err)
	}
	RecordDashboardStorageCreated("/report.txt")
	created := GetDashboardStorageSnapshot()
	if created.FileCount != 1 || created.SharedUsedBytes == 0 || len(created.Categories) != 1 || created.Categories[0].Key != "document" {
		t.Fatalf("create was not reflected: %#v", created)
	}

	previous, err := CaptureDashboardStoragePath("/report.txt")
	if err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(root, "report.png")
	if err := os.Rename(filePath, imagePath); err != nil {
		t.Fatal(err)
	}
	RecordDashboardStorageMoved(previous, "/report.png")
	moved := GetDashboardStorageSnapshot()
	if len(moved.Categories) != 1 || moved.Categories[0].Key != "image" {
		t.Fatalf("move category was not reflected: %#v", moved.Categories)
	}

	deletedState, err := CaptureDashboardStoragePath("/report.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	RecordDashboardStorageDeleted(deletedState)
	deleted := GetDashboardStorageSnapshot()
	if deleted.FileCount != 0 || deleted.SharedUsedBytes != 0 || len(deleted.Categories) != 0 {
		t.Fatalf("delete was not reflected: %#v", deleted)
	}
}

func TestDefaultScanSettingsAreDisabledHourly(t *testing.T) {
	settings := config.DefaultScan()
	if settings.AutoScanEnabled || settings.IntervalHours != 1 {
		t.Fatalf("unexpected scan defaults: %#v", settings)
	}
	if err := ValidateScanSettings(config.ScanSettings{IntervalHours: 0}); err == nil {
		t.Fatal("zero-hour scan interval should be rejected")
	}
}

func TestDashboardActivityRangeUsesLogRetentionLimit(t *testing.T) {
	config.InitDefaults()
	logSettings := config.GetLog()
	logSettings.RetentionDays = 5
	config.SetLog(logSettings)

	if days, maxDays := dashboardActivityRange(30); days != 5 || maxDays != 5 {
		t.Fatalf("range = %d/%d, want 5/5", days, maxDays)
	}
	if days, maxDays := dashboardActivityRange(3); days != 3 || maxDays != 5 {
		t.Fatalf("range = %d/%d, want 3/5", days, maxDays)
	}
}
