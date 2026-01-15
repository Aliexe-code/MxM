package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
)

const (
	DefaultDataDir   = "./data"
	ChainFileName    = "blockchain.json"
	BackupDirName    = "backups"
	MaxBackupFiles   = 5
	ChecksumFileName = "checksum.sha256"
)

type FileStorage struct {
	dataDir   string
	chainFile string
	backupDir string
	mu        sync.RWMutex
}

func NewFileStorage(dataDir string) (*FileStorage, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir
	}
	fs := &FileStorage{
		dataDir:   dataDir,
		chainFile: filepath.Join(dataDir, ChainFileName),
		backupDir: filepath.Join(dataDir, BackupDirName),
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create data directory: %w", err)
	}
	if err := os.MkdirAll(fs.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create backup directory: %w", err)
	}
	return fs, nil
}

func (fs *FileStorage) SaveBlockchain(bc *blockchain.Blockchain) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.createBackup(); err != nil {
		return fmt.Errorf("Failed to create backup: %w", err)
	}

	jsonData, err := bc.ToJSON()
	if err != nil {
		return fmt.Errorf("Failed to serialize blockchain: %w", err)
	}

	// Calculate checksum
	checksum := sha256.Sum256(jsonData)

	// Write checksum first
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)
	if err := os.WriteFile(checksumFile, []byte(hex.EncodeToString(checksum[:])), 0600); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	// Sync checksum to disk
	if f, err := os.Open(checksumFile); err == nil {
		f.Sync()
		f.Close()
	}

	// Write to temporary file first
	tempFile := fs.chainFile + ".tmp"
	lockFile := fs.chainFile + ".lock"

	// Create exclusive lock file to prevent concurrent writes
	lock, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		lock.Close()
		os.Remove(lockFile)
	}()

	if err := os.WriteFile(tempFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Sync to disk before rename
	if f, err := os.Open(tempFile); err == nil {
		f.Sync()
		f.Close()
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempFile, fs.chainFile); err != nil {
		os.Remove(tempFile) // Cleanup on failure
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func (fs *FileStorage) LoadBlockchain() (*blockchain.Blockchain, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if _, err := os.Stat(fs.chainFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("Blockchain file not found: %s", fs.chainFile)
	}

	jsonData, err := os.ReadFile(fs.chainFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read blockchain file: %w", err)
	}
	if err := fs.verifyChecksum(jsonData); err != nil {
		if recovered, recoverErr := fs.recoverFromBackup(); recoverErr == nil {
			return recovered, nil
		}
		return nil, fmt.Errorf("checksum verification & backup recovery both failed: %w", err)
	}
	bc := blockchain.NewBlockchain()
	if err := bc.FromJSON(jsonData); err != nil {
		return nil, fmt.Errorf("Failed to load blockchain from file: %w", err)
	}
	return bc, nil
}

func (fs *FileStorage) verifyChecksum(data []byte) error {
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)

	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		return nil
	}
	storedChecksum, err := os.ReadFile(checksumFile)
	if err != nil {
		return fmt.Errorf("Failed to read checksum file: %w", err)
	}
	calculatedChecksum := sha256.Sum256(data)
	calculatedChecksumHex := hex.EncodeToString(calculatedChecksum[:])
	if string(storedChecksum) != calculatedChecksumHex {
		return fmt.Errorf("Checksum mismatch: expected %s, got %s", string(storedChecksum), calculatedChecksumHex)
	}
	return nil
}

func (fs *FileStorage) createBackup() error {
	if _, err := os.Stat(fs.chainFile); os.IsNotExist(err) {
		return nil
	}
	timestamp := time.Now().Format("20060102-150405.000000")
	backupFile := filepath.Join(fs.backupDir, fmt.Sprintf("blockchain-%s.json", timestamp))
	if err := copyFile(fs.chainFile, backupFile); err != nil {
		return fmt.Errorf("Failed to copy file to backup: %w", err)
	}
	// Set secure permissions on backup
	if err := os.Chmod(backupFile, 0600); err != nil {
		return fmt.Errorf("Failed to set permissions on backup: %w", err)
	}
	if err := fs.cleanOldBackups(); err != nil {
		return fmt.Errorf("Failed to clean old backups: %w", err)
	}
	return nil
}

func (fs *FileStorage) recoverFromBackup() (*blockchain.Blockchain, error) {
	backups, err := filepath.Glob(filepath.Join(fs.backupDir, "blockchain-*.json"))
	if err != nil {
		return nil, fmt.Errorf("Failed to list backups: %w", err)
	}
	if len(backups) == 0 {
		return nil, fmt.Errorf("No backups available")
	}
	var latestBackup string
	var latestModTime time.Time
	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestModTime) {
			latestModTime = info.ModTime()
			latestBackup = backup
		}
	}
	if latestBackup == "" {
		return nil, fmt.Errorf("No valid backup found")
	}
	jsonData, err := os.ReadFile(latestBackup)
	if err != nil {
		return nil, fmt.Errorf("Failed to read backup file: %w", err)
	}
	bc := blockchain.NewBlockchain()
	if err := bc.FromJSON(jsonData); err != nil {
		return nil, fmt.Errorf("Failed to load blockchain from backup: %w", err)
	}
	if err := copyFile(latestBackup, fs.chainFile); err != nil {
		return nil, fmt.Errorf("Failed to restore from backup: %w", err)
	}
	// Set secure permissions on restored file
	if err := os.Chmod(fs.chainFile, 0600); err != nil {
		return nil, fmt.Errorf("Failed to set permissions on restored file: %w", err)
	}
	// Recalculate and save checksum for restored blockchain
	checksum := sha256.Sum256(jsonData)
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)
	if err := os.WriteFile(checksumFile, []byte(hex.EncodeToString(checksum[:])), 0600); err != nil {
		return nil, fmt.Errorf("Failed to write checksum after restore: %w", err)
	}
	return bc, nil
}

func (fs *FileStorage) cleanOldBackups() error {
	backups, err := filepath.Glob(filepath.Join(fs.backupDir, "blockchain-*.json"))
	if err != nil {
		return fmt.Errorf("Failed to list backups: %w", err)
	}
	if len(backups) <= MaxBackupFiles {
		return nil
	}
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo
	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			path:    backup,
			modTime: info.ModTime(),
		})
	}
	for i := range len(files) {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	filesToDelete := len(files) - MaxBackupFiles
	for i := range filesToDelete {
		if err := os.Remove(files[i].path); err != nil {
			return fmt.Errorf("Failed to remove backup file: %w", err)
		}
	}
	return nil
}
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()
	_, err = io.Copy(dest, source)
	return err
}

func (fs *FileStorage) GetDataDir() string {
	return fs.dataDir
}

func (fs *FileStorage) GetChainFile() string {
	return fs.chainFile
}

func (fs *FileStorage) Exists() bool {
	_, err := os.Stat(fs.chainFile)
	return !os.IsNotExist(err)
}

func (fs *FileStorage) Delete() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if err := os.Remove(fs.chainFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to remove chain file: %w", err)
	}
	checksumFile := filepath.Join(fs.dataDir, ChecksumFileName)
	if err := os.Remove(checksumFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to remove checksum file: %w", err)
	}
	backups, err := filepath.Glob(filepath.Join(fs.backupDir, "blockchain-*.json"))
	if err != nil {
		return fmt.Errorf("Failed to list backups: %w", err)
	}
	for _, backup := range backups {
		if err := os.Remove(backup); err != nil {
			return fmt.Errorf("Failed to remove backup file: %w", err)
		}
	}
	return nil
}

func (fs *FileStorage) GetBackupList() ([]string, error) {
	backups, err := filepath.Glob(filepath.Join(fs.backupDir, "blockchain-*.json"))

	if err != nil {
		return nil, fmt.Errorf("Failed to list backups: %w", err)
	}
	var backupNames []string
	for _, backup := range backups {
		backupNames = append(backupNames, filepath.Base(backup))
	}
	return backupNames, nil
}
