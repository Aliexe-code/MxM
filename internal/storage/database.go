package storage

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/aliexe/blockChain/internal/blockchain"
)

const (
	DefaultDBPath   = "./data/blockchain.db"
	MaxOpenConns    = 25
	MaxIdleConns    = 5
	ConnMaxLifetime = 5 * time.Minute
	ConnMaxIdleTime = time.Minute
)

type DatabaseStorage struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

func NewDatabaseStorage(dbPath string) (*DatabaseStorage, error) {
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	ds := &DatabaseStorage{
		path: dbPath,
	}

	if err := ds.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := ds.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return ds, nil
}

func (ds *DatabaseStorage) connect() error {
	var err error
	ds.db, err = sql.Open("sqlite3", ds.path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	ds.db.SetMaxOpenConns(MaxOpenConns)
	ds.db.SetMaxIdleConns(MaxIdleConns)
	ds.db.SetConnMaxLifetime(ConnMaxLifetime)
	ds.db.SetConnMaxIdleTime(ConnMaxIdleTime)

	// Test connection
	if err := ds.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func (ds *DatabaseStorage) initSchema() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Create blocks table
	_, err := ds.db.Exec(`
		CREATE TABLE IF NOT EXISTS blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			"index" INTEGER UNIQUE NOT NULL,
			timestamp INTEGER NOT NULL,
			data BLOB NOT NULL,
			prev_hash BLOB NOT NULL,
			hash BLOB UNIQUE NOT NULL,
			nonce INTEGER NOT NULL,
			difficulty INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_blocks_index ON blocks("index");
		CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(hash);
		CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp);
	`)
	if err != nil {
		return fmt.Errorf("failed to create blocks table: %w", err)
	}

	// Create mining_rewards table
	_, err = ds.db.Exec(`
		CREATE TABLE IF NOT EXISTS mining_rewards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			miner_id TEXT NOT NULL,
			block_index INTEGER NOT NULL,
			reward REAL NOT NULL,
			timestamp INTEGER NOT NULL,
			difficulty INTEGER NOT NULL,
			FOREIGN KEY (block_index) REFERENCES blocks("index") ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_rewards_miner ON mining_rewards(miner_id);
		CREATE INDEX IF NOT EXISTS idx_rewards_block ON mining_rewards(block_index);
	`)
	if err != nil {
		return fmt.Errorf("failed to create mining_rewards table: %w", err)
	}

	// Create metadata table for version tracking
	_, err = ds.db.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Set schema version
	_, err = ds.db.Exec(`
		INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', '1')
	`)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

func (ds *DatabaseStorage) SaveBlockchain(bc *blockchain.Blockchain) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	tx, err := ds.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Properly handle rollback errors
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
			}
		}
	}()

	// Clear existing data
	if _, err := tx.Exec("DELETE FROM mining_rewards"); err != nil {
		return fmt.Errorf("failed to clear mining rewards: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM blocks"); err != nil {
		return fmt.Errorf("failed to clear blocks: %w", err)
	}

	// Insert blocks
	for i, block := range bc.Blocks {
		_, err := tx.Exec(`
			INSERT INTO blocks ("index", timestamp, data, prev_hash, hash, nonce, difficulty)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, i, block.Timestamp, block.Data, block.PrevHash, block.Hash, block.Nonce, block.Difficulty)
		if err != nil {
			return fmt.Errorf("failed to insert block %d: %w", i, err)
		}
	}

	// Insert mining rewards
	for _, reward := range bc.MiningRewards {
		_, err := tx.Exec(`
			INSERT INTO mining_rewards (miner_id, block_index, reward, timestamp, difficulty)
			VALUES (?, ?, ?, ?, ?)
		`, reward.MinerID, reward.BlockIndex, reward.Reward, reward.Timestamp.Unix(), reward.Difficulty)
		if err != nil {
			return fmt.Errorf("failed to insert mining reward: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (ds *DatabaseStorage) LoadBlockchain() (*blockchain.Blockchain, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Load blocks
	rows, err := ds.db.Query(`
		SELECT "index", timestamp, data, prev_hash, hash, nonce, difficulty
		FROM blocks
		ORDER BY "index" ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	bc := blockchain.NewBlockchain()
	bc.Blocks = []*blockchain.Block{} // Clear genesis block

	for rows.Next() {
		var index int
		var timestamp int64
		var data, prevHash, hash []byte
		var nonce uint32
		var difficulty int

		if err := rows.Scan(&index, &timestamp, &data, &prevHash, &hash, &nonce, &difficulty); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		block := &blockchain.Block{
			Timestamp:  timestamp,
			Data:       data,
			PrevHash:   prevHash,
			Hash:       hash,
			Nonce:      nonce,
			Difficulty: difficulty,
		}

		bc.Blocks = append(bc.Blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blocks: %w", err)
	}

	// Load mining rewards
	rewardRows, err := ds.db.Query(`
		SELECT miner_id, block_index, reward, timestamp, difficulty
		FROM mining_rewards
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query mining rewards: %w", err)
	}
	defer rewardRows.Close()

	bc.MiningRewards = []*blockchain.MiningReward{}
	bc.TotalRewards = 0

	for rewardRows.Next() {
		var minerID string
		var blockIndex int
		var reward float64
		var timestamp int64
		var difficulty int

		if err := rewardRows.Scan(&minerID, &blockIndex, &reward, &timestamp, &difficulty); err != nil {
			return nil, fmt.Errorf("failed to scan mining reward: %w", err)
		}

		miningReward := &blockchain.MiningReward{
			MinerID:    minerID,
			BlockIndex: blockIndex,
			Reward:     reward,
			Timestamp:  time.Unix(timestamp, 0),
			Difficulty: difficulty,
		}

		bc.MiningRewards = append(bc.MiningRewards, miningReward)
		bc.TotalRewards += reward
	}

	if err := rewardRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mining rewards: %w", err)
	}

	// Validate loaded blockchain
	if !bc.IsValid() {
		return nil, fmt.Errorf("loaded blockchain is invalid")
	}

	return bc, nil
}

func (ds *DatabaseStorage) GetBlockByIndex(index int) (*blockchain.Block, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var timestamp int64
	var data, prevHash, hash []byte
	var nonce uint32
	var difficulty int

	err := ds.db.QueryRow(`
		SELECT timestamp, data, prev_hash, hash, nonce, difficulty
		FROM blocks
		WHERE "index" = ?
	`, index).Scan(&timestamp, &data, &prevHash, &hash, &nonce, &difficulty)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("block with index %d not found", index)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query block: %w", err)
	}

	return &blockchain.Block{
		Timestamp:  timestamp,
		Data:       data,
		PrevHash:   prevHash,
		Hash:       hash,
		Nonce:      nonce,
		Difficulty: difficulty,
	}, nil
}

func (ds *DatabaseStorage) GetBlockByHash(hash []byte) (*blockchain.Block, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var index int
	var timestamp int64
	var data, prevHash, blockHash []byte
	var nonce uint32
	var difficulty int

	err := ds.db.QueryRow(`
		SELECT "index", timestamp, data, prev_hash, hash, nonce, difficulty
		FROM blocks
		WHERE hash = ?
	`, hash).Scan(&index, &timestamp, &data, &prevHash, &blockHash, &nonce, &difficulty)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("block with hash %s not found", hex.EncodeToString(hash))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query block: %w", err)
	}

	return &blockchain.Block{
		Timestamp:  timestamp,
		Data:       data,
		PrevHash:   prevHash,
		Hash:       blockHash,
		Nonce:      nonce,
		Difficulty: difficulty,
	}, nil
}

func (ds *DatabaseStorage) GetBlocksByTimeRange(start, end time.Time) ([]*blockchain.Block, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	rows, err := ds.db.Query(`
		SELECT "index", timestamp, data, prev_hash, hash, nonce, difficulty
		FROM blocks
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY "index" ASC
	`, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query blocks by time range: %w", err)
	}
	defer rows.Close()

	var blocks []*blockchain.Block
	for rows.Next() {
		var index int
		var timestamp int64
		var data, prevHash, hash []byte
		var nonce uint32
		var difficulty int

		if err := rows.Scan(&index, &timestamp, &data, &prevHash, &hash, &nonce, &difficulty); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}

		blocks = append(blocks, &blockchain.Block{
			Timestamp:  timestamp,
			Data:       data,
			PrevHash:   prevHash,
			Hash:       hash,
			Nonce:      nonce,
			Difficulty: difficulty,
		})
	}

	return blocks, nil
}

func (ds *DatabaseStorage) GetChainLength() (int, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var count int
	err := ds.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count blocks: %w", err)
	}

	return count, nil
}

func (ds *DatabaseStorage) GetMinerRewards(minerID string) (float64, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var total float64
	err := ds.db.QueryRow(`
		SELECT COALESCE(SUM(reward), 0)
		FROM mining_rewards
		WHERE miner_id = ?
	`, minerID).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf("failed to query miner rewards: %w", err)
	}

	return total, nil
}

func (ds *DatabaseStorage) GetMiningStats() (map[string]interface{}, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Get total blocks
	var totalBlocks int
	if err := ds.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&totalBlocks); err != nil {
		return nil, fmt.Errorf("failed to count blocks: %w", err)
	}

	// Get total rewards
	var totalRewards float64
	if err := ds.db.QueryRow("SELECT COALESCE(SUM(reward), 0) FROM mining_rewards").Scan(&totalRewards); err != nil {
		return nil, fmt.Errorf("failed to sum rewards: %w", err)
	}

	// Get rewards by miner
	rows, err := ds.db.Query(`
		SELECT miner_id, SUM(reward), COUNT(*)
		FROM mining_rewards
		GROUP BY miner_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query miner stats: %w", err)
	}
	defer rows.Close()

	minerRewards := make(map[string]float64)
	minerBlocks := make(map[string]int)

	for rows.Next() {
		var minerID string
		var reward float64
		var blockCount int

		if err := rows.Scan(&minerID, &reward, &blockCount); err != nil {
			return nil, fmt.Errorf("failed to scan miner stats: %w", err)
		}

		minerRewards[minerID] = reward
		minerBlocks[minerID] = blockCount
	}

	return map[string]interface{}{
		"total_blocks":  totalBlocks,
		"total_rewards": totalRewards,
		"miner_rewards": minerRewards,
		"miner_blocks":  minerBlocks,
		"reward_count":  len(minerRewards),
	}, nil
}

func (ds *DatabaseStorage) Exists() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var count int
	err := ds.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&count)
	return err == nil && count > 0
}

func (ds *DatabaseStorage) Delete() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	tx, err := ds.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Properly handle rollback errors
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
			}
		}
	}()

	if _, err := tx.Exec("DELETE FROM mining_rewards"); err != nil {
		return fmt.Errorf("failed to delete mining rewards: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM blocks"); err != nil {
		return fmt.Errorf("failed to delete blocks: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	return nil
}

func (ds *DatabaseStorage) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.db != nil {
		return ds.db.Close()
	}
	return nil
}

func (ds *DatabaseStorage) GetDBPath() string {
	return ds.path
}

func (ds *DatabaseStorage) GetSchemaVersion() (string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var version string
	err := ds.db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&version)
	if err == sql.ErrNoRows {
		return "0", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get schema version: %w", err)
	}

	return version, nil
}
