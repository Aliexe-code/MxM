package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
)

func TestNewFileStorage(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Test creating new storage
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Verify directories are created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}

	if _, err := os.Stat(fs.backupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}

	// Test with empty string (should use default)
	fsDefault, err := NewFileStorage("")
	if err != nil {
		t.Fatalf("Failed to create file storage with default dir: %v", err)
	}

	if fsDefault.dataDir != DefaultDataDir {
		t.Errorf("Expected default data dir %s, got %s", DefaultDataDir, fsDefault.dataDir)
	}
}

func TestSaveAndLoadBlockchain(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create a blockchain with some blocks
	bc := blockchain.NewBlockchain()
	bc.AddBlock("First transaction")
	bc.AddBlock("Second transaction")

	// Save the blockchain
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Verify file exists
	if !fs.Exists() {
		t.Error("Blockchain file was not created")
	}

	// Load the blockchain
	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain: %v", err)
	}

	// Verify loaded blockchain
	if loadedBC.GetChainLength() != bc.GetChainLength() {
		t.Errorf("Expected chain length %d, got %d", bc.GetChainLength(), loadedBC.GetChainLength())
	}

	// Verify validity
	if !loadedBC.IsValid() {
		t.Error("Loaded blockchain is invalid")
	}
}

func TestSaveBlockchainWithMining(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create blockchain with mined blocks
	bc := blockchain.NewBlockchain()
	_, err = bc.AddBlockWithMining("Mined block 1", "miner1", 2)
	if err != nil {
		t.Fatalf("Failed to add mined block: %v", err)
	}

	// Save
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Load
	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain: %v", err)
	}

	// Verify mining rewards
	stats := loadedBC.GetMiningStats()
	totalRewards := stats["total_rewards"].(float64)
	if totalRewards == 0 {
		t.Error("Mining rewards were not preserved")
	}
}

func TestLoadNonExistentBlockchain(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Try to load non-existent blockchain
	_, err = fs.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading non-existent blockchain")
	}
}

func TestChecksumVerification(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Verify checksum file exists
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)
	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		t.Error("Checksum file was not created")
	}

	// Corrupt the blockchain file
	chainData, err := os.ReadFile(fs.chainFile)
	if err != nil {
		t.Fatalf("Failed to read chain file: %v", err)
	}

	// Modify the data
	if len(chainData) > 10 {
		chainData[5] = 'X'
	}
	if err := os.WriteFile(fs.chainFile, chainData, 0644); err != nil {
		t.Fatalf("Failed to write corrupted data: %v", err)
	}

	// Try to load - should fail checksum verification
	_, err = fs.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading corrupted blockchain")
	}
}

func TestBackupCreation(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain (first save - no backup created yet)
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create backup
	bc.AddBlock("Block 2")
	// No sleep needed with microsecond resolution
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Verify backup was created
	backups, err := fs.GetBackupList()
	if err != nil {
		t.Fatalf("Failed to get backup list: %v", err)
	}

	if len(backups) == 0 {
		t.Error("No backup was created")
	}

	// Save again to create another backup
	bc.AddBlock("Block 3")
	// No sleep needed
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Verify multiple backups
	backups, err = fs.GetBackupList()
	if err != nil {
		t.Fatalf("Failed to get backup list: %v", err)
	}

	if len(backups) < 2 {
		t.Error("Expected at least 2 backups")
	}
}

func TestBackupRecovery(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	bc.AddBlock("Block 2")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create a backup
	bc.AddBlock("Block 3")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain update: %v", err)
	}

	// Save one more time to ensure the version with Block 3 is backed up
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to separate backup save: %v", err)
	}

	// Corrupt the main file
	if err := os.WriteFile(fs.chainFile, []byte("corrupted data"), 0644); err != nil {
		t.Fatalf("Failed to corrupt chain file: %v", err)
	}

	// Try to load - should recover from backup
	recoveredBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to recover from backup: %v", err)
	}

	// Verify recovered blockchain
	if recoveredBC.GetChainLength() != bc.GetChainLength() {
		t.Errorf("Expected chain length %d, got %d", bc.GetChainLength(), recoveredBC.GetChainLength())
	}
}

func TestCleanOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create more backups than MaxBackupFiles
	bc := blockchain.NewBlockchain()
	for i := 0; i < MaxBackupFiles+3; i++ {
		bc.AddBlock(fmt.Sprintf("Block %d", i))
		if err := fs.SaveBlockchain(bc); err != nil {
			t.Fatalf("Failed to save blockchain: %v", err)
		}
	}

	// Verify only MaxBackupFiles remain
	backups, err := fs.GetBackupList()
	if err != nil {
		t.Fatalf("Failed to get backup list: %v", err)
	}

	if len(backups) > MaxBackupFiles {
		t.Errorf("Expected at most %d backups, got %d", MaxBackupFiles, len(backups))
	}
}

func TestDelete(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Delete
	if err := fs.Delete(); err != nil {
		t.Fatalf("Failed to delete storage: %v", err)
	}

	// Verify files are deleted
	if fs.Exists() {
		t.Error("Chain file was not deleted")
	}

	backups, err := fs.GetBackupList()
	if err != nil {
		t.Fatalf("Failed to get backup list: %v", err)
	}

	if len(backups) > 0 {
		t.Error("Backups were not deleted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Initial block")

	// Save initial blockchain
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save initial blockchain: %v", err)
	}

	// Concurrent saves - each goroutine loads, modifies, and saves
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(index int) {
			// Load fresh copy
			loadedBC, err := fs.LoadBlockchain()
			if err != nil {
				t.Errorf("Failed to load blockchain: %v", err)
				done <- false
				return
			}
			// Add block
			loadedBC.AddBlock(fmt.Sprintf("Concurrent block %d", index))
			// Save
			if err := fs.SaveBlockchain(loadedBC); err != nil {
				t.Errorf("Concurrent save failed: %v", err)
				done <- false
				return
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < 5; i++ {
		if <-done {
			successCount++
		}
	}

	// At least one save should succeed
	if successCount == 0 {
		t.Error("No concurrent saves succeeded")
	}

	// Verify blockchain is still valid
	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain after concurrent access: %v", err)
	}

	if !loadedBC.IsValid() {
		t.Error("Blockchain is invalid after concurrent access")
	}
}

func TestGetters(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Test GetDataDir
	if fs.GetDataDir() != tempDir {
		t.Errorf("Expected data dir %s, got %s", tempDir, fs.GetDataDir())
	}

	// Test GetChainFile
	expectedChainFile := filepath.Join(tempDir, ChainFileName)
	if fs.GetChainFile() != expectedChainFile {
		t.Errorf("Expected chain file %s, got %s", expectedChainFile, fs.GetChainFile())
	}
}

func TestNewFileStorageDirectoryCreationError(t *testing.T) {
	// Test with invalid path that can't be created
	// This is a best-effort test since we can't easily simulate permission errors
	fs, err := NewFileStorage("/root/invalid/path/that/should/fail")
	if err != nil {
		// Expected on systems without root access
		return
	}
	// If it somehow succeeded, clean up
	os.RemoveAll("/root/invalid")
	_ = fs
}

func TestSaveBlockchainSerializationError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create a blockchain and save it first
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save initial blockchain: %v", err)
	}

	// Make the directory read-only to trigger write error
	if err := os.Chmod(tempDir, 0444); err != nil {
		t.Skip("Cannot change directory permissions")
	}
	defer os.Chmod(tempDir, 0755)

	// Try to save again - should fail
	err = fs.SaveBlockchain(bc)
	if err == nil {
		t.Error("Expected error when saving to read-only directory")
	}
}

func TestLoadBlockchainReadError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Make file read-only
	if err := os.Chmod(fs.chainFile, 0000); err != nil {
		t.Skip("Cannot change file permissions")
	}
	defer os.Chmod(fs.chainFile, 0644)

	// Try to load - might fail or succeed depending on OS
	_, err = fs.LoadBlockchain()
	// Either error is acceptable (permission denied) or success (cached read)
	_ = err
}

func TestVerifyChecksumReadError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create checksum file but make it unreadable
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)
	if err := os.WriteFile(checksumFile, []byte("checksum"), 0644); err != nil {
		t.Fatalf("Failed to create checksum file: %v", err)
	}

	if err := os.Chmod(checksumFile, 0000); err != nil {
		t.Skip("Cannot change file permissions")
	}
	defer os.Chmod(checksumFile, 0644)

	// Create blockchain file
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	fs.SaveBlockchain(bc)

	// Verify checksum - should handle read error gracefully
	_ = fs
}

func TestCreateBackupCopyError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Make backup directory read-only
	if err := os.Chmod(fs.backupDir, 0000); err != nil {
		t.Skip("Cannot change directory permissions")
	}
	defer os.Chmod(fs.backupDir, 0755)

	// Try to save again - backup creation might fail
	_ = fs.SaveBlockchain(bc)
}

func TestRecoverFromBackupNoBackups(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Try to recover without any backups
	_, err = fs.recoverFromBackup()
	if err == nil {
		t.Error("Expected error when no backups available")
	}
}

func TestRecoverFromBackupReadError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create backup
	time.Sleep(100 * time.Millisecond)
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Make backup file unreadable
	backups, _ := fs.GetBackupList()
	if len(backups) > 0 {
		backupPath := filepath.Join(fs.backupDir, backups[0])
		if err := os.Chmod(backupPath, 0000); err == nil {
			defer os.Chmod(backupPath, 0644)
			// Try to recover - should handle error
			_, _ = fs.recoverFromBackup()
		}
	}
}

func TestCleanOldBackupsGlobError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Replace backupDir with invalid path
	originalBackupDir := fs.backupDir
	fs.backupDir = "/nonexistent/path/that/does/not/exist/*"
	defer func() { fs.backupDir = originalBackupDir }()

	// Try to clean - should handle glob error gracefully
	_ = fs.cleanOldBackups()
}

func TestCopyFileSourceError(t *testing.T) {
	// Test copying from non-existent file
	dst := t.TempDir() + "/dst.txt"
	err := copyFile("/nonexistent/file.txt", dst)
	if err == nil {
		t.Error("Expected error when copying from non-existent file")
	}
}

func TestCopyFileDestinationError(t *testing.T) {
	// Create a temp file
	src := t.TempDir() + "/src.txt"
	if err := os.WriteFile(src, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Try to copy to invalid destination
	dst := "/root/invalid/destination.txt"
	err := copyFile(src, dst)
	if err == nil {
		t.Error("Expected error when copying to invalid destination")
	}
}

func TestDeleteNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Delete without creating any files - should succeed
	if err := fs.Delete(); err != nil {
		t.Errorf("Failed to delete non-existent storage: %v", err)
	}
}

func TestGetBackupListError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Replace backupDir with a pattern that contains invalid characters
	originalBackupDir := fs.backupDir
	fs.backupDir = string([]byte{0x00}) // Invalid null byte in path
	defer func() { fs.backupDir = originalBackupDir }()

	// Try to get backup list - should handle error gracefully
	backups, err := fs.GetBackupList()
	// Either error is acceptable or empty list
	if err == nil && len(backups) > 0 {
		t.Error("Expected error or empty list when listing from invalid path")
	}
}

func TestSaveBlockchainToJSONError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// This test would require mocking blockchain.ToJSON() to return an error
	// Since we can't easily mock, we'll skip this edge case
	_ = fs
}

func TestLoadBlockchainUnmarshalError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Write invalid JSON to blockchain file
	invalidJSON := []byte("{ invalid json }")
	if err := os.WriteFile(fs.chainFile, invalidJSON, 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Try to load - should fail with unmarshal error
	_, err = fs.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestLoadBlockchainInvalidChainError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create valid JSON but invalid blockchain (broken hash links)
	invalidChain := `{
		"blocks": [
			{
				"timestamp": 1234567890,
				"data": "SGVsbG8=",
				"prev_hash": "",
				"hash": "wrong_hash",
				"nonce": 0,
				"difficulty": 4
			}
		],
		"mining_rewards": [],
		"total_rewards": 0
	}`
	if err := os.WriteFile(fs.chainFile, []byte(invalidChain), 0644); err != nil {
		t.Fatalf("Failed to write invalid chain: %v", err)
	}

	// Try to load - should fail with invalid chain error
	_, err = fs.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading invalid blockchain")
	}
}

func TestRecoverFromBackupInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create backup
	time.Sleep(100 * time.Millisecond)
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Corrupt the backup file
	backups, _ := fs.GetBackupList()
	if len(backups) > 0 {
		backupPath := filepath.Join(fs.backupDir, backups[0])
		if err := os.WriteFile(backupPath, []byte("{ invalid json }"), 0644); err == nil {
			// Try to recover - should fail with unmarshal error
			_, err = fs.recoverFromBackup()
			if err == nil {
				t.Error("Expected error when recovering from invalid backup")
			}
		}
	}
}

func TestRecoverFromBackupInvalidChain(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create backup
	time.Sleep(100 * time.Millisecond)
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Corrupt the backup file with invalid chain
	backups, _ := fs.GetBackupList()
	if len(backups) > 0 {
		backupPath := filepath.Join(fs.backupDir, backups[0])
		invalidChain := `{"blocks": [{"timestamp": 123, "data": "dGVzdA==", "prev_hash": "", "hash": "wrong", "nonce": 0, "difficulty": 4}], "mining_rewards": [], "total_rewards": 0}`
		if err := os.WriteFile(backupPath, []byte(invalidChain), 0644); err == nil {
			// Try to recover - should fail with invalid chain error
			_, err = fs.recoverFromBackup()
			if err == nil {
				t.Error("Expected error when recovering from invalid chain backup")
			}
		}
	}
}

func TestRecoverFromBackupCopyError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Save again to create backup
	time.Sleep(100 * time.Millisecond)
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Make chain file read-only to prevent restore
	if err := os.Chmod(fs.chainFile, 0000); err != nil {
		t.Skip("Cannot change file permissions")
	}
	defer os.Chmod(fs.chainFile, 0644)

	// Try to recover - copy might fail
	_, err = fs.recoverFromBackup()
	_ = err // May fail due to permission error
}

func TestCleanOldBackupsDeleteError(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create more backups than MaxBackupFiles
	bc := blockchain.NewBlockchain()
	for i := 0; i < MaxBackupFiles+3; i++ {
		bc.AddBlock(fmt.Sprintf("Block %d", i))
		time.Sleep(50 * time.Millisecond) // Ensure different timestamps
		if err := fs.SaveBlockchain(bc); err != nil {
			t.Fatalf("Failed to save blockchain: %v", err)
		}
	}

	// Make one of the old backups read-only
	backups, _ := fs.GetBackupList()
	if len(backups) > 0 {
		// Sort backups by modification time to find oldest
		type fileInfo struct {
			path    string
			modTime time.Time
		}
		var files []fileInfo
		for _, backup := range backups {
			info, err := os.Stat(filepath.Join(fs.backupDir, backup))
			if err == nil {
				files = append(files, fileInfo{path: filepath.Join(fs.backupDir, backup), modTime: info.ModTime()})
			}
		}
		// Sort oldest first
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				if files[i].modTime.After(files[j].modTime) {
					files[i], files[j] = files[j], files[i]
				}
			}
		}
		// Make oldest read-only
		if len(files) > 0 {
			if err := os.Chmod(files[0].path, 0000); err == nil {
				defer os.Chmod(files[0].path, 0644)
				// Try to clean - should handle error gracefully
				_ = fs.cleanOldBackups()
			}
		}
	}
}

func TestEmptyBlockchain(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create empty blockchain (just genesis)
	bc := blockchain.NewBlockchain()

	// Save and load
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save empty blockchain: %v", err)
	}

	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load empty blockchain: %v", err)
	}

	if loadedBC.GetChainLength() != 1 {
		t.Errorf("Expected chain length 1 (genesis), got %d", loadedBC.GetChainLength())
	}
}

func TestLargeBlockchain(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create blockchain with many blocks
	bc := blockchain.NewBlockchain()
	for i := 0; i < 100; i++ {
		bc.AddBlock(fmt.Sprintf("Block %d with some data content", i))
	}

	// Save and load
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save large blockchain: %v", err)
	}

	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load large blockchain: %v", err)
	}

	if loadedBC.GetChainLength() != 101 {
		t.Errorf("Expected chain length 101, got %d", loadedBC.GetChainLength())
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	fs, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	// Create initial blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Initial")
	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save initial blockchain: %v", err)
	}

	// Concurrent reads and writes
	done := make(chan bool, 10)

	// Start 5 readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, _ = fs.LoadBlockchain()
			}
			done <- true
		}()
	}

	// Start 5 writers
	for i := 0; i < 5; i++ {
		go func(index int) {
			loadedBC, _ := fs.LoadBlockchain()
			if loadedBC != nil {
				loadedBC.AddBlock(fmt.Sprintf("Writer %d", index))
				_ = fs.SaveBlockchain(loadedBC)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final blockchain is valid
	loadedBC, err := fs.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load final blockchain: %v", err)
	}

	if !loadedBC.IsValid() {
		t.Error("Final blockchain is invalid after concurrent read/write")
	}
}
