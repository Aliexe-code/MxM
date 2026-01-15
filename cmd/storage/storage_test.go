package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aliexe/blockChain/internal/blockchain"
	"github.com/aliexe/blockChain/internal/storage"
)

const storageBinary = "/tmp/storage-test"

func TestStorageCLIHelp(t *testing.T) {
	cmd := exec.Command(storageBinary, "help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run help command: %v", err)
	}

	if len(output) == 0 {
		t.Error("Help command produced no output")
	}
}

func TestStorageCLIInfoEmpty(t *testing.T) {
	tempDir := t.TempDir()
	cmd := exec.Command(storageBinary, "-data-dir", tempDir, "info", "file")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run info command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "No blockchain file found") {
		t.Errorf("Expected 'No blockchain file found' message, got: %s", outputStr)
	}
}

func TestStorageCLIValidateEmpty(t *testing.T) {
	tempDir := t.TempDir()
	cmd := exec.Command(storageBinary, "-data-dir", tempDir, "validate", "file")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run validate command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "No blockchain file found") {
		t.Errorf("Expected 'No blockchain file found' message, got: %s", outputStr)
	}
}

func TestStorageCLIDatabaseInfoEmpty(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	cmd := exec.Command(storageBinary, "-db-path", tempFile, "info", "db")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run info command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "No blockchain data found") {
		t.Errorf("Expected 'No blockchain data found' message, got: %s", outputStr)
	}
}

func TestStorageCLIExportImportWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	backupDir := filepath.Join(tempDir, "backup")

	// Create a test blockchain with file storage first
	// We'll use the storage package directly
	fs, err := storage.NewFileStorage(dataDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")
	bc.AddBlock("Test block 2")

	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test export
	exportPath := filepath.Join(backupDir, "blockchain.json")
	cmd := exec.Command(storageBinary, "-data-dir", dataDir, "export", "file", exportPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run export command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "Successfully exported") {
		t.Errorf("Expected 'Successfully exported' message, got: %s", outputStr)
	}

	// Verify export file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}

	// Test validate on exported file
	cmd = exec.Command(storageBinary, "-data-dir", backupDir, "validate", "file")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run validate command: %v\nOutput: %s", err, string(output))
	}

	outputStr = string(output)
	if !contains(outputStr, "is valid") {
		t.Errorf("Expected 'is valid' message, got: %s", outputStr)
	}

	// Test import to new location
	importDir := filepath.Join(tempDir, "import")
	cmd = exec.Command(storageBinary, "-data-dir", importDir, "import", "file", exportPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run import command: %v\nOutput: %s", err, string(output))
	}

	outputStr = string(output)
	if !contains(outputStr, "Successfully imported") {
		t.Errorf("Expected 'Successfully imported' message, got: %s", outputStr)
	}

	// Verify import
	cmd = exec.Command(storageBinary, "-data-dir", importDir, "info", "file")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run info command: %v\nOutput: %s", err, string(output))
	}

	outputStr = string(output)
	if !contains(outputStr, "Total Blocks: 3") {
		t.Errorf("Expected 'Total Blocks: 3' message, got: %s", outputStr)
	}
}

func TestStorageCLIDatabaseExportImport(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	dbPath := filepath.Join(tempDir, "blockchain.db")

	// Create a test blockchain with file storage
	fs, err := storage.NewFileStorage(dataDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")
	bc.AddBlock("Test block 2")

	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test export to database
	cmd := exec.Command(storageBinary, "-data-dir", dataDir, "export", "db", dbPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run export command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "Successfully exported") {
		t.Errorf("Expected 'Successfully exported' message, got: %s", outputStr)
	}

	// Verify database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Test validate on database
	cmd = exec.Command(storageBinary, "-db-path", dbPath, "validate", "db")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run validate command: %v\nOutput: %s", err, string(output))
	}

	outputStr = string(output)
	if !contains(outputStr, "is valid") {
		t.Errorf("Expected 'is valid' message, got: %s", outputStr)
	}

	// Test info on database
	cmd = exec.Command(storageBinary, "-db-path", dbPath, "info", "db")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run info command: %v\nOutput: %s", err, string(output))
	}

	outputStr = string(output)
	if !contains(outputStr, "Total Blocks: 3") {
		t.Errorf("Expected 'Total Blocks: 3' message, got: %s", outputStr)
	}
}

func TestStorageCLIBackup(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Create a test blockchain
	fs, err := storage.NewFileStorage(dataDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")

	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test backup
	cmd := exec.Command(storageBinary, "-data-dir", dataDir, "backup", "file")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run backup command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "Backup created") {
		t.Errorf("Expected 'Backup created' message, got: %s", outputStr)
	}
}

func TestStorageCLIDatabaseBackup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "blockchain.db")

	// Create a test blockchain in database
	dbStorage, err := storage.NewDatabaseStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer dbStorage.Close()

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")

	if err := dbStorage.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test backup
	cmd := exec.Command(storageBinary, "-db-path", dbPath, "backup", "db")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run backup command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "Database backup created") {
		t.Errorf("Expected 'Database backup created' message, got: %s", outputStr)
	}
}

func TestStorageCLIUnknownCommand(t *testing.T) {
	cmd := exec.Command(storageBinary, "unknown")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unknown command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "Unknown command") {
		t.Errorf("Expected 'Unknown command' message, got: %s", outputStr)
	}
}

func TestStorageCLICleanupConfirmation(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Create a test blockchain
	fs, err := storage.NewFileStorage(dataDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")

	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test cleanup with 'no' confirmation (should cancel)
	cmd := exec.Command("sh", "-c", "echo 'no' | "+storageBinary+" -data-dir "+dataDir+" cleanup file")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run cleanup command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "cancelled") {
		t.Errorf("Expected 'cancelled' message, got: %s", outputStr)
	}

	// Verify blockchain still exists
	if !fs.Exists() {
		t.Error("Blockchain was deleted despite 'no' confirmation")
	}
}

func TestStorageCLICleanupConfirmed(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Create a test blockchain
	fs, err := storage.NewFileStorage(dataDir)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	bc := blockchain.NewBlockchain()
	bc.AddBlock("Test block 1")

	if err := fs.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Test cleanup with 'yes' confirmation
	cmd := exec.Command("sh", "-c", "echo 'yes' | "+storageBinary+" -data-dir "+dataDir+" cleanup file")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run cleanup command: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "cleaned up successfully") {
		t.Errorf("Expected 'cleaned up successfully' message, got: %s", outputStr)
	}

	// Verify blockchain was deleted
	if fs.Exists() {
		t.Error("Blockchain was not deleted despite 'yes' confirmation")
	}
}

func TestStorageCLIInvalidFormat(t *testing.T) {
	cmd := exec.Command(storageBinary, "export", "invalid", "/tmp/test.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run export command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "Unknown format") {
		t.Errorf("Expected 'Unknown format' message, got: %s", outputStr)
	}
}

func TestStorageCLIMissingArguments(t *testing.T) {
	// Test export without arguments
	cmd := exec.Command(storageBinary, "export")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run export command: %v", err)
	}

	outputStr := string(output)
	if !contains(outputStr, "Usage:") {
		t.Errorf("Expected 'Usage:' message, got: %s", outputStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}