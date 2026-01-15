package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
	"github.com/aliexe/blockChain/internal/storage"
)

type StorageCLI struct {
	dataDir string
	dbPath  string
}

const (
	DefaultDataDir = "./data"
	DefaultDBPath  = "./data/blockchain.db"
)

func main() {
	cli := &StorageCLI{
		dataDir: DefaultDataDir,
		dbPath:  DefaultDBPath,
	}

	flag.StringVar(&cli.dataDir, "data-dir", DefaultDataDir, "Data directory for file storage")
	flag.StringVar(&cli.dbPath, "db-path", DefaultDBPath, "Path to SQLite database")

	flag.Parse()

	if len(flag.Args()) == 0 {
		cli.showHelp()
		return
	}

	command := strings.ToLower(flag.Args()[0])

	switch command {
	case "export":
		cli.handleExport()
	case "import":
		cli.handleImport()
	case "validate":
		cli.handleValidate()
	case "info":
		cli.handleInfo()
	case "backup":
		cli.handleBackup()
	case "cleanup":
		cli.handleCleanup()
	case "help", "--help", "-h":
		cli.showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		cli.showHelp()
	}
}

func (cli *StorageCLI) showHelp() {
	fmt.Println("💾 CryptoChain Storage CLI")
	fmt.Println("==========================")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  storage [COMMAND] [OPTIONS]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  export [format] [path]  Export blockchain data")
	fmt.Println("  import [format] [path]  Import blockchain data")
	fmt.Println("  validate [format]       Validate blockchain data integrity")
	fmt.Println("  info [format]           Show blockchain information")
	fmt.Println("  backup [format]         Create backup of blockchain")
	fmt.Println("  cleanup [format]        Clean up storage data")
	fmt.Println("  help                    Show this help message")
	fmt.Println()
	fmt.Println("FORMATS:")
	fmt.Println("  file                    File-based storage (JSON)")
	fmt.Println("  db                      Database storage (SQLite)")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -data-dir string        Data directory for file storage (default \"./data\")")
	fmt.Println("  -db-path string         Path to SQLite database (default \"./data/blockchain.db\")")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  storage export file ./backup/blockchain.json")
	fmt.Println("  storage import file ./backup/blockchain.json")
	fmt.Println("  storage validate file")
	fmt.Println("  storage info db")
	fmt.Println("  storage backup file")
	fmt.Println("  storage cleanup db")
}

func (cli *StorageCLI) handleExport() {
	if len(flag.Args()) < 3 {
		fmt.Println("❌ Usage: storage export [format] [output-path]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])
	outputPath := flag.Args()[2]

	switch format {
	case "file":
		cli.exportToFile(outputPath)
	case "db":
		cli.exportToDatabase(outputPath)
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) handleImport() {
	if len(flag.Args()) < 3 {
		fmt.Println("❌ Usage: storage import [format] [input-path]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])
	inputPath := flag.Args()[2]

	switch format {
	case "file":
		cli.importFromFile(inputPath)
	case "db":
		cli.importFromDatabase(inputPath)
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) handleValidate() {
	if len(flag.Args()) < 2 {
		fmt.Println("❌ Usage: storage validate [format]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])

	switch format {
	case "file":
		cli.validateFileStorage()
	case "db":
		cli.validateDatabaseStorage()
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) handleInfo() {
	if len(flag.Args()) < 2 {
		fmt.Println("❌ Usage: storage info [format]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])

	switch format {
	case "file":
		cli.showFileInfo()
	case "db":
		cli.showDatabaseInfo()
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) handleBackup() {
	if len(flag.Args()) < 2 {
		fmt.Println("❌ Usage: storage backup [format]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])

	switch format {
	case "file":
		cli.backupFileStorage()
	case "db":
		cli.backupDatabaseStorage()
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) handleCleanup() {
	if len(flag.Args()) < 2 {
		fmt.Println("❌ Usage: storage cleanup [format]")
		fmt.Println("   format: file | db")
		return
	}

	format := strings.ToLower(flag.Args()[1])

	switch format {
	case "file":
		cli.cleanupFileStorage()
	case "db":
		cli.cleanupDatabaseStorage()
	default:
		fmt.Printf("❌ Unknown format: %s (use 'file' or 'db')\n", format)
	}
}

func (cli *StorageCLI) exportToFile(outputPath string) {
	fmt.Printf("📤 Exporting blockchain from file storage to %s\n", outputPath)

	// Load from file storage
	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	bc, err := fs.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Export to new file
	exportFS, err := storage.NewFileStorage(outputDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize export storage: %v", err)
	}

	if err := exportFS.SaveBlockchain(bc); err != nil {
		log.Fatalf("❌ Failed to save blockchain: %v", err)
	}

	fmt.Printf("✅ Successfully exported blockchain with %d blocks\n", bc.GetChainLength())
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) exportToDatabase(outputPath string) {
	fmt.Printf("📤 Exporting blockchain from file storage to database: %s\n", outputPath)

	// Load from file storage
	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	bc, err := fs.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Export to database
	dbStorage, err := storage.NewDatabaseStorage(outputPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	if err := dbStorage.SaveBlockchain(bc); err != nil {
		log.Fatalf("❌ Failed to save blockchain to database: %v", err)
	}

	fmt.Printf("✅ Successfully exported blockchain with %d blocks to database\n", bc.GetChainLength())
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) importFromFile(inputPath string) {
	fmt.Printf("📥 Importing blockchain from %s to file storage\n", inputPath)

	// Load from input file
	inputDir := filepath.Dir(inputPath)
	inputFS, err := storage.NewFileStorage(inputDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize input storage: %v", err)
	}

	bc, err := inputFS.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	// Validate blockchain
	if !bc.IsValid() {
		log.Fatalf("❌ Imported blockchain is invalid")
	}

	// Save to file storage
	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if err := fs.SaveBlockchain(bc); err != nil {
		log.Fatalf("❌ Failed to save blockchain: %v", err)
	}

	fmt.Printf("✅ Successfully imported blockchain with %d blocks\n", bc.GetChainLength())
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) importFromDatabase(inputPath string) {
	fmt.Printf("📥 Importing blockchain from database: %s to file storage\n", inputPath)

	// Load from database
	dbStorage, err := storage.NewDatabaseStorage(inputPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	bc, err := dbStorage.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	// Validate blockchain
	if !bc.IsValid() {
		log.Fatalf("❌ Imported blockchain is invalid")
	}

	// Save to file storage
	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if err := fs.SaveBlockchain(bc); err != nil {
		log.Fatalf("❌ Failed to save blockchain: %v", err)
	}

	fmt.Printf("✅ Successfully imported blockchain with %d blocks\n", bc.GetChainLength())
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) validateFileStorage() {
	fmt.Println("🔍 Validating file storage...")

	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if !fs.Exists() {
		fmt.Println("⚠️  No blockchain file found")
		return
	}

	startTime := time.Now()
	bc, err := fs.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}
	duration := time.Since(startTime)

	if !bc.IsValid() {
		log.Fatalf("❌ Blockchain validation failed")
	}

	fmt.Printf("✅ File storage is valid\n")
	fmt.Printf("⏱️  Validation time: %v\n", duration.Round(time.Millisecond))
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) validateDatabaseStorage() {
	fmt.Println("🔍 Validating database storage...")

	dbStorage, err := storage.NewDatabaseStorage(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	if !dbStorage.Exists() {
		fmt.Println("⚠️  No blockchain data found in database")
		return
	}

	startTime := time.Now()
	bc, err := dbStorage.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}
	duration := time.Since(startTime)

	if !bc.IsValid() {
		log.Fatalf("❌ Blockchain validation failed")
	}

	fmt.Printf("✅ Database storage is valid\n")
	fmt.Printf("⏱️  Validation time: %v\n", duration.Round(time.Millisecond))
	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) showFileInfo() {
	fmt.Println("📊 File Storage Information")
	fmt.Println("==========================")

	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if !fs.Exists() {
		fmt.Println("⚠️  No blockchain file found")
		return
	}

	// Get file info
	fileInfo, err := os.Stat(fs.GetChainFile())
	if err != nil {
		log.Fatalf("❌ Failed to get file info: %v", err)
	}

	fmt.Printf("📁 Data Directory: %s\n", fs.GetDataDir())
	fmt.Printf("📄 Chain File: %s\n", fs.GetChainFile())
	fmt.Printf("📦 File Size: %.2f KB\n", float64(fileInfo.Size())/1024)
	fmt.Printf("🕒 Last Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))

	// Load blockchain for more info
	bc, err := fs.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	cli.showBlockchainInfo(bc)

	// Show backup info
	backups, err := fs.GetBackupList()
	if err != nil {
		log.Fatalf("❌ Failed to get backup list: %v", err)
	}

	if len(backups) > 0 {
		fmt.Printf("\n💾 Backups: %d\n", len(backups))
		for i, backup := range backups {
			if i >= 3 {
				fmt.Printf("   ... and %d more\n", len(backups)-3)
				break
			}
			backupPath := filepath.Join(fs.GetDataDir(), "backups", backup)
			backupInfo, err := os.Stat(backupPath)
			if err == nil {
				fmt.Printf("   - %s (%.2f KB)\n", backup, float64(backupInfo.Size())/1024)
			}
		}
	}
}

func (cli *StorageCLI) showDatabaseInfo() {
	fmt.Println("📊 Database Storage Information")
	fmt.Println("==============================")

	dbStorage, err := storage.NewDatabaseStorage(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	if !dbStorage.Exists() {
		fmt.Println("⚠️  No blockchain data found in database")
		return
	}

	// Get database file info
	fileInfo, err := os.Stat(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to get database file info: %v", err)
	}

	schemaVersion, err := dbStorage.GetSchemaVersion()
	if err != nil {
		log.Fatalf("❌ Failed to get schema version: %v", err)
	}

	fmt.Printf("📁 Database Path: %s\n", dbStorage.GetDBPath())
	fmt.Printf("📦 File Size: %.2f KB\n", float64(fileInfo.Size())/1024)
	fmt.Printf("🕒 Last Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))
	fmt.Printf("🔢 Schema Version: %s\n", schemaVersion)

	// Load blockchain for more info
	bc, err := dbStorage.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	cli.showBlockchainInfo(bc)
}

func (cli *StorageCLI) backupFileStorage() {
	fmt.Println("💾 Creating file storage backup...")

	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if !fs.Exists() {
		fmt.Println("⚠️  No blockchain file found to backup")
		return
	}

	bc, err := fs.LoadBlockchain()
	if err != nil {
		log.Fatalf("❌ Failed to load blockchain: %v", err)
	}

	// Save again to trigger backup creation
	if err := fs.SaveBlockchain(bc); err != nil {
		log.Fatalf("❌ Failed to create backup: %v", err)
	}

	backups, err := fs.GetBackupList()
	if err != nil {
		log.Fatalf("❌ Failed to get backup list: %v", err)
	}

	if len(backups) > 0 {
		fmt.Printf("✅ Backup created successfully\n")
		fmt.Printf("📦 Total backups: %d\n", len(backups))
	} else {
		fmt.Println("⚠️  No backup was created")
	}
}

func (cli *StorageCLI) backupDatabaseStorage() {
	fmt.Println("💾 Creating database storage backup...")

	dbStorage, err := storage.NewDatabaseStorage(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	if !dbStorage.Exists() {
		fmt.Println("⚠️  No blockchain data found to backup")
		return
	}

	// For SQLite, we can copy the database file
	backupDir := filepath.Dir(cli.dbPath)
	backupPath := filepath.Join(backupDir, "backups")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Fatalf("❌ Failed to create backup directory: %v", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(backupPath, fmt.Sprintf("blockchain-%s.db", timestamp))

	// Copy database file
	input, err := os.ReadFile(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to read database file: %v", err)
	}

	if err := os.WriteFile(backupFile, input, 0644); err != nil {
		log.Fatalf("❌ Failed to write backup file: %v", err)
	}

	fmt.Printf("✅ Database backup created: %s\n", backupFile)
}

func (cli *StorageCLI) cleanupFileStorage() {
	fmt.Println("🧹 Cleaning up file storage...")

	fs, err := storage.NewFileStorage(cli.dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize file storage: %v", err)
	}

	if !fs.Exists() {
		fmt.Println("⚠️  No blockchain file found")
		return
	}

	fmt.Println("⚠️  This will delete all blockchain data including backups!")
	fmt.Print("Are you sure? (yes/no): ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("❌ Cleanup cancelled")
		return
	}

	if err := fs.Delete(); err != nil {
		log.Fatalf("❌ Failed to cleanup storage: %v", err)
	}

	fmt.Println("✅ File storage cleaned up successfully")
}

func (cli *StorageCLI) cleanupDatabaseStorage() {
	fmt.Println("🧹 Cleaning up database storage...")

	dbStorage, err := storage.NewDatabaseStorage(cli.dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database storage: %v", err)
	}
	defer dbStorage.Close()

	if !dbStorage.Exists() {
		fmt.Println("⚠️  No blockchain data found")
		return
	}

	fmt.Println("⚠️  This will delete all blockchain data from the database!")
	fmt.Print("Are you sure? (yes/no): ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("❌ Cleanup cancelled")
		return
	}

	if err := dbStorage.Delete(); err != nil {
		log.Fatalf("❌ Failed to cleanup storage: %v", err)
	}

	fmt.Println("✅ Database storage cleaned up successfully")
}

func (cli *StorageCLI) showBlockchainInfo(bc *blockchain.Blockchain) {
	fmt.Printf("\n📦 Blockchain Information\n")
	fmt.Printf("=======================\n")
	fmt.Printf("📊 Total Blocks: %d\n", bc.GetChainLength())
	fmt.Printf("🕒 Genesis Block: %s\n", time.Unix(bc.Blocks[0].Timestamp, 0).Format(time.RFC3339))
	fmt.Printf("🕒 Latest Block: %s\n", time.Unix(bc.Blocks[len(bc.Blocks)-1].Timestamp, 0).Format(time.RFC3339))

	// Show mining rewards
	stats := bc.GetMiningStats()
	totalRewards := stats["total_rewards"].(float64)
	if totalRewards > 0 {
		fmt.Printf("💰 Total Rewards: %.2f\n", totalRewards)
		fmt.Printf("🏆 Miners: %d\n", stats["reward_count"])

		if minerRewards, ok := stats["miner_rewards"].(map[string]float64); ok {
			fmt.Println("\n👥 Miner Rewards:")
			for miner, reward := range minerRewards {
				fmt.Printf("   💎 %s: %.2f\n", miner, reward)
			}
		}
	}
}
