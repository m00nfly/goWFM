package services

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goWFM/config"
	"goWFM/db"
)

func setupShareTestDB(t *testing.T) {
	t.Helper()
	if err := db.Init(filepath.Join(t.TempDir(), "share.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	config.InitDefaults()
	t.Cleanup(db.Close)
}

func TestCreateShareUsesUniqueTwelveCharacterIDs(t *testing.T) {
	setupShareTestDB(t)
	seen := make(map[string]struct{})

	for i := 0; i < 200; i++ {
		share, err := CreateShare([]string{"/report.txt"}, 0, 0, "")
		if err != nil {
			t.Fatalf("create share %d: %v", i, err)
		}
		if len(share.Token) != shareTokenLength {
			t.Fatalf("share id length = %d, want %d: %q", len(share.Token), shareTokenLength, share.Token)
		}
		for _, character := range share.Token {
			if !containsRune(shareTokenAlphabet, character) {
				t.Fatalf("share id contains non-alphanumeric character: %q", share.Token)
			}
		}
		if _, exists := seen[share.Token]; exists {
			t.Fatalf("duplicate share id generated: %q", share.Token)
		}
		seen[share.Token] = struct{}{}
	}
}

func TestNewestDownloadLinkInvalidatesPreviousAndIsSingleUse(t *testing.T) {
	setupShareTestDB(t)
	share, err := CreateShare([]string{"/季度报告.xlsx"}, 0, 0, "")
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	files, err := GetShareFiles(share.ID)
	if err != nil || len(files) != 1 {
		t.Fatalf("get share files: %#v, %v", files, err)
	}

	first, err := IssueShareDownloadLink(share.Token, files[0].ID, 5*time.Minute)
	if err != nil {
		t.Fatalf("issue first link: %v", err)
	}
	second, err := IssueShareDownloadLink(share.Token, files[0].ID, 5*time.Minute)
	if err != nil {
		t.Fatalf("issue second link: %v", err)
	}
	if first.Token == second.Token {
		t.Fatal("new link reused the previous random token")
	}
	if _, err := ConsumeShareDownloadLink(first.Token, first.Filename); !errors.Is(err, ErrDownloadLinkInvalid) {
		t.Fatalf("previous link should be invalidated, got %v", err)
	}

	download, err := ConsumeShareDownloadLink(second.Token, second.Filename)
	if err != nil {
		t.Fatalf("consume current link: %v", err)
	}
	if download.File.ID != files[0].ID || download.ShareToken != share.Token {
		t.Fatalf("unexpected download target: %#v", download)
	}
	if _, err := ConsumeShareDownloadLink(second.Token, second.Filename); !errors.Is(err, ErrDownloadLinkInvalid) {
		t.Fatalf("consumed link should not be reusable, got %v", err)
	}

	var rawTokenRows int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM share_download_tokens WHERE token_hash = ?`, second.Token).Scan(&rawTokenRows); err != nil {
		t.Fatalf("query stored token: %v", err)
	}
	if rawTokenRows != 0 {
		t.Fatal("raw download token was stored in the database")
	}
}

func TestDownloadLinkRejectsFileFromAnotherShare(t *testing.T) {
	setupShareTestDB(t)
	firstShare, err := CreateShare([]string{"/first.txt"}, 0, 0, "")
	if err != nil {
		t.Fatalf("create first share: %v", err)
	}
	secondShare, err := CreateShare([]string{"/second.txt"}, 0, 0, "")
	if err != nil {
		t.Fatalf("create second share: %v", err)
	}
	secondFiles, err := GetShareFiles(secondShare.ID)
	if err != nil || len(secondFiles) != 1 {
		t.Fatalf("get second share files: %#v, %v", secondFiles, err)
	}

	if _, err := IssueShareDownloadLink(firstShare.Token, secondFiles[0].ID, 5*time.Minute); !errors.Is(err, ErrShareFileNotFound) {
		t.Fatalf("foreign file should be rejected, got %v", err)
	}
}

func TestDownloadLinkCanOnlyBeConsumedOnceConcurrently(t *testing.T) {
	setupShareTestDB(t)
	share, err := CreateShare([]string{"/archive.zip"}, 0, 0, "")
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	files, err := GetShareFiles(share.ID)
	if err != nil || len(files) != 1 {
		t.Fatalf("get share files: %#v, %v", files, err)
	}
	issued, err := IssueShareDownloadLink(share.Token, files[0].ID, 5*time.Minute)
	if err != nil {
		t.Fatalf("issue link: %v", err)
	}

	var successes atomic.Int32
	var unexpected atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := ConsumeShareDownloadLink(issued.Token, issued.Filename); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrDownloadLinkInvalid) {
				unexpected.Add(1)
			}
		}()
	}
	group.Wait()

	if successes.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("concurrent consumption: successes=%d unexpected_errors=%d", successes.Load(), unexpected.Load())
	}
}

func TestExpiredDownloadLinkCannotBeConsumed(t *testing.T) {
	setupShareTestDB(t)
	share, err := CreateShare([]string{"/manual.pdf"}, 0, 0, "")
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	files, _ := GetShareFiles(share.ID)
	issued, err := IssueShareDownloadLink(share.Token, files[0].ID, time.Millisecond)
	if err != nil {
		t.Fatalf("issue link: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := ConsumeShareDownloadLink(issued.Token, issued.Filename); !errors.Is(err, ErrDownloadLinkInvalid) {
		t.Fatalf("expired link should be invalid, got %v", err)
	}
}

func TestDefaultFileLinkTimeoutIsFiveMinutes(t *testing.T) {
	if timeout := config.DefaultShare().FileLinkTimeoutMinutes; timeout != 5 {
		t.Fatalf("default file link timeout = %d, want 5", timeout)
	}
}

func TestListSharesScopesResultsAndIncludesManagementDetails(t *testing.T) {
	setupShareTestDB(t)
	if _, err := db.DB.Exec(`INSERT INTO users (id, username, password_hash, display_name, avatar_data)
		VALUES (1, 'alice', 'x', 'Alice', 'data:image/png;base64,YQ=='),
		       (2, 'bob', 'x', 'Bob', '')`); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if _, err := CreateShare([]string{"/alice/report.txt"}, 1, 0, "季度报告"); err != nil {
		t.Fatalf("create alice share: %v", err)
	}
	if _, err := CreateShare([]string{"/bob/one.txt", "/bob/two.txt"}, 2, 0, ""); err != nil {
		t.Fatalf("create bob share: %v", err)
	}

	aliceID := int64(1)
	aliceShares, err := ListShares(&aliceID)
	if err != nil {
		t.Fatalf("list alice shares: %v", err)
	}
	if len(aliceShares) != 1 || aliceShares[0].Owner.Username != "alice" {
		t.Fatalf("unexpected scoped shares: %#v", aliceShares)
	}
	if len(aliceShares[0].Files) != 1 || aliceShares[0].Files[0].FilePath != "/alice/report.txt" {
		t.Fatalf("management file details missing: %#v", aliceShares[0].Files)
	}
	if aliceShares[0].Owner.Avatar == "" {
		t.Fatal("owner avatar was not included")
	}

	allShares, err := ListShares(nil)
	if err != nil {
		t.Fatalf("list all shares: %v", err)
	}
	if len(allShares) != 2 {
		t.Fatalf("admin result count = %d, want 2", len(allShares))
	}
	var bobShare *ShareManagementItem
	for index := range allShares {
		if allShares[index].Owner.Username == "bob" {
			bobShare = &allShares[index]
			break
		}
	}
	if bobShare == nil || bobShare.FileCount != 2 || bobShare.FileName != "分享2个文件" {
		t.Fatalf("multi-file summary incorrect: %#v", bobShare)
	}
}

func TestShareMutationsRequireOwnershipUnlessAdministrator(t *testing.T) {
	setupShareTestDB(t)
	if _, err := db.DB.Exec(`INSERT INTO users (id, username, password_hash) VALUES (1, 'alice', 'x'), (2, 'bob', 'x')`); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	share, err := CreateShare([]string{"/alice/private.txt"}, 1, 0, "original")
	if err != nil {
		t.Fatalf("create share: %v", err)
	}

	if err := UpdateShare(share.ID, 2, false, "stolen", nil); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("foreign update should be hidden, got %v", err)
	}
	if err := DeleteShare(share.ID, 2, false); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("foreign delete should be hidden, got %v", err)
	}
	if err := UpdateShare(share.ID, 2, true, "admin edit", nil); err != nil {
		t.Fatalf("admin update: %v", err)
	}
	if err := DeleteShare(share.ID, 2, true); err != nil {
		t.Fatalf("admin delete: %v", err)
	}
}

func containsRune(value string, target rune) bool {
	for _, character := range value {
		if character == target {
			return true
		}
	}
	return false
}
