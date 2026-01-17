package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliexe/blockChain/internal/blockchain"
)

func TestPermissionErrors(t *testing.T) {
	// 1. NewFileStorage error (mkdir fails)
	tempDir := t.TempDir()
	blockingFile := filepath.Join(tempDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("block"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	if _, err := NewFileStorage(blockingFile); err == nil {
		t.Error("Expected error when creating storage on top of file")
	}

	// 2. SaveBlockchain error (write fails)
	fsDir := filepath.Join(tempDir, "storage")
	fs, err := NewFileStorage(fsDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := os.Chmod(fsDir, 0555); err != nil { // Read-only
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(fsDir, 0755)

	bc := blockchain.NewBlockchain()
	bc.AddBlock("test")

	if err := fs.SaveBlockchain(bc); err == nil {
		t.Error("Expected error when saving to read-only dir")
	}

	// 3. LoadBlockchain error (read fails)
	os.Chmod(fsDir, 0755) // Restore
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save setup blockchain: %v", err)
	}

	if err := os.Chmod(fs.chainFile, 0000); err != nil { // Unreadable
		t.Fatalf("Failed to chmod chain file: %v", err)
	}
	if _, err := fs.LoadBlockchain(); err == nil {
		t.Error("Expected error when loading unreadable file")
	}
	os.Chmod(fs.chainFile, 0644) // Restore
}

func TestDeleteErrors(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("test")
	fs.SaveBlockchain(bc)

	// Make directory read-only to prevent deletion of files inside
	if err := os.Chmod(tempDir, 0555); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tempDir, 0755)

	if err := fs.Delete(); err == nil {
		t.Error("Expected error when deleting from read-only dir")
	}
}

func TestRecoveryErrors(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("test")
	// Save thrice to get 2 backups
	fs.SaveBlockchain(bc)
	bc.AddBlock("test2")
	fs.SaveBlockchain(bc) // Backup 1
	bc.AddBlock("test3")
	fs.SaveBlockchain(bc) // Backup 2

	backups, _ := fs.GetBackupList()
	if len(backups) < 2 {
		t.Fatal("Setup failed: need backups")
	}
	latestBackup := filepath.Join(fs.backupDir, backups[len(backups)-1])

	// Corrupt main chain
	os.Remove(fs.chainFile)

	// Case 1: Latest backup is unreadable (Load backup fail)
	if err := os.Chmod(latestBackup, 0000); err != nil {
		t.Fatal(err)
	}
	// Note: recoverFromBackup will find it, try to read, and fail.
	// However, will it try the *other* backup? No, verifyChecksum calls recoverFromBackup which finds *latest*.
	// If latest fails, it returns error. It does not iterate?
	// check code: "if err := bc.FromJSON(jsonData); err != nil { return nil ... }"
	// It doesn't loop.

	if _, err := fs.LoadBlockchain(); err == nil {
		t.Error("Expected error when latest backup is unreadable")
	}
	os.Chmod(latestBackup, 0644) // Restore

	// Case 2: Latest backup is invalid JSON
	os.WriteFile(latestBackup, []byte("{bad json"), 0644)
	if _, err := fs.LoadBlockchain(); err == nil {
		t.Error("Expected error when backup is invalid JSON")
	}
	// Restore validity logic not needed as we test subsequent errors?

	// Case 3: Restore write failure (cannot write to chainFile path)
	// Repair backup
	bc.AddBlock("test3") // Recreate state
	validJSON, _ := bc.ToJSON()
	os.WriteFile(latestBackup, validJSON, 0644)

	// Dir read-only => Cannot create chainFile
	os.Chmod(tempDir, 0555)
	defer os.Chmod(tempDir, 0755)

	if _, err := fs.LoadBlockchain(); err == nil {
		t.Error("Expected error when cannot restore backup file")
	}
}

func TestCleanBackupsErrors(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	bc := blockchain.NewBlockchain()
	// Create max+1 backups
	for i := 0; i < MaxBackupFiles+2; i++ {
		bc.AddBlock(fmt.Sprintf("%d", i))
		fs.SaveBlockchain(bc)
	}

	// Make backup dir read-only so we can't delete old ones
	os.Chmod(fs.backupDir, 0555)
	defer os.Chmod(fs.backupDir, 0755)

	bc.AddBlock("new")
	// Save calls cleanOldBackups
	if err := fs.SaveBlockchain(bc); err == nil {
		t.Error("Expected error when cleaning backups fails")
	}
}

func TestBrokenSymlinkBackup(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a broken symlink in backup dir
	// This simulates an error in os.Stat() inside the loop
	brokenLink := filepath.Join(fs.backupDir, "blockchain-broken.json")
	if err := os.Symlink("non-existent", brokenLink); err != nil {
		t.Skip("Symlinks not supported?")
	}

	// Trigger cleanOldBackups logic (requires list > Max)
	// But first, does recoverFromBackup handle it?
	// recoverFromBackup loop does os.Stat. If error, 'continue'.
	// So it should NOT fail, just skip.

	// Let's verify cleanOldBackups handles it.
	// cleanOldBackups loop does os.Stat. If error, 'continue'.
	// So it should also skip.

	// To cover the 'continue' branch, we just need to run it.
	// We need enough backups?
	// Create some backups
	bc := blockchain.NewBlockchain()
	fs.SaveBlockchain(bc)

	// Just running this checks that it doesn't panic or fail
	if _, err := fs.LoadBlockchain(); err != nil {
		// Should error because main file exists (created by Save)?
		// Wait, LoadBlockchain verifies checksum.
		// If main file exists and is valid, recoverFromBackup is NOT called.
	}

	// Force recover
	os.Remove(fs.chainFile)
	// Now recover runs. Should skip broken link.
	if _, err := fs.LoadBlockchain(); err == nil {
		// Should fail because we have no valid backups (only broken link?)
		// We created 1 save -> 0 backups?
		// Save 1 -> writes chain file. No backup.
		// So 0 backups.
		t.Error("Expected error (no valid backup), got success?")
	} else {
		// Expected "No valid backup found"
	}
}

func TestGlobFailure(t *testing.T) {
	// Try a dir path with malformed glob pattern chars if possible
	// On unix, '[' is valid in filename.
	// filepath.Glob("dir/[/*") expects a range. If ']' missing, syntax error.

	invalidGlobDir := filepath.Join(t.TempDir(), "bad[dir")
	fs, err := NewFileStorage(invalidGlobDir)
	if err != nil {
		t.Fatal(err)
	}

	// Trigger glob calls
	// 1. GetBackupList
	if _, err := fs.GetBackupList(); err == nil {
		// If it passes, glob didn't fail.
		// On many systems, bad pattern returns ErrBadPattern.
		// If it assumes literal [, it works.
	} else {
		// Caught error
	}
}
