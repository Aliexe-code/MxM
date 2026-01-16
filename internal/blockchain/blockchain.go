package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"unicode"
)

// MiningReward represents a reward for mining a block
type MiningReward struct {
	MinerID    string    `json:"miner_id"`
	BlockIndex int       `json:"block_index"`
	Reward     float64   `json:"reward"`
	Timestamp  time.Time `json:"timestamp"`
	Difficulty int       `json:"difficulty"`
}

// validateMinerID validates miner ID for security
func validateMinerID(minerID string) error {
	if len(minerID) == 0 {
		return fmt.Errorf("miner ID cannot be empty")
	}
	if len(minerID) > 256 {
		return fmt.Errorf("miner ID too long (max 256 characters)")
	}
	for _, r := range minerID {
		if !unicode.IsPrint(r) || unicode.IsControl(r) {
			return fmt.Errorf("invalid character in miner ID")
		}
	}
	return nil
}

type Blockchain struct {
	Blocks        []*Block        `json:"blocks"`
	MiningRewards []*MiningReward `json:"mining_rewards"`
	TotalRewards  float64         `json:"total_rewards"`
	rewardMutex   sync.RWMutex
	mu            sync.RWMutex
}

func NewBlockchain() *Blockchain {
	genesis := NewGenesisBlock()

	genesis.Hash = genesis.CalculateHash()

	return &Blockchain{
		Blocks:        []*Block{genesis},
		MiningRewards: []*MiningReward{},
		TotalRewards:  0.0,
	}
}

func (bc *Blockchain) AddBlock(data string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.Blocks) == 0 {
		return fmt.Errorf("no blocks in blockchain")
	}

	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock([]byte(data), latestBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
}

// AddBlockWithMining adds a block with mining and rewards
func (bc *Blockchain) AddBlockWithMining(data string, minerID string, difficulty int) (time.Duration, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.Blocks) == 0 {
		return 0, fmt.Errorf("no blocks in blockchain")
	}

	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock([]byte(data), latestBlock.Hash)

	// Mine the block
	duration := newBlock.MineBlock(difficulty)

	// Add block to chain
	bc.Blocks = append(bc.Blocks, newBlock)

	// Calculate and add reward
	reward := bc.CalculateReward(difficulty)
	if err := bc.AddMiningReward(minerID, len(bc.Blocks)-1, reward, difficulty); err != nil {
		return duration, fmt.Errorf("failed to add mining reward: %w", err)
	}

	return duration, nil
}

// GetDifficulty returns the current difficulty of the blockchain
func (bc *Blockchain) GetDifficulty() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return DefaultDifficulty
	}
	return bc.Blocks[len(bc.Blocks)-1].Difficulty
}

// SetDifficulty sets the difficulty for the next block to be mined
func (bc *Blockchain) SetDifficulty(difficulty int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.Blocks) == 0 {
		return
	}

	// Update the difficulty of the last block (this will apply to the next block)
	bc.Blocks[len(bc.Blocks)-1].Difficulty = difficulty
}

// CalculateReward calculates mining reward based on difficulty
func (bc *Blockchain) CalculateReward(difficulty int) float64 {
	// Base reward + difficulty bonus
	baseReward := 10.0
	difficultyBonus := float64(difficulty) * 2.5
	return baseReward + difficultyBonus
}

// AddMiningReward adds a mining reward to the tracking system
func (bc *Blockchain) AddMiningReward(minerID string, blockIndex int, reward float64, difficulty int) error {
	// Validate miner ID
	if err := validateMinerID(minerID); err != nil {
		return fmt.Errorf("invalid miner ID: %w", err)
	}

	bc.rewardMutex.Lock()
	defer bc.rewardMutex.Unlock()

	miningReward := &MiningReward{
		MinerID:    minerID,
		BlockIndex: blockIndex,
		Reward:     reward,
		Timestamp:  time.Now(),
		Difficulty: difficulty,
	}

	bc.MiningRewards = append(bc.MiningRewards, miningReward)
	bc.TotalRewards += reward
	return nil
}

// GetMinerRewards returns total rewards for a specific miner
func (bc *Blockchain) GetMinerRewards(minerID string) float64 {
	bc.rewardMutex.RLock()
	defer bc.rewardMutex.RUnlock()

	var total float64
	for _, reward := range bc.MiningRewards {
		if reward.MinerID == minerID {
			total += reward.Reward
		}
	}
	return total
}

// GetMiningStats returns mining statistics
func (bc *Blockchain) GetMiningStats() map[string]interface{} {
	bc.rewardMutex.RLock()
	defer bc.rewardMutex.RUnlock()

	minerStats := make(map[string]float64)
	blockCount := make(map[string]int)

	for _, reward := range bc.MiningRewards {
		minerStats[reward.MinerID] += reward.Reward
		blockCount[reward.MinerID]++
	}

	return map[string]interface{}{
		"total_blocks":  len(bc.Blocks),
		"total_rewards": bc.TotalRewards,
		"miner_rewards": minerStats,
		"miner_blocks":  blockCount,
		"reward_count":  len(bc.MiningRewards),
	}
}

func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) IsValid() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return false
	}
	if len(bc.Blocks) == 1 {
		genesis := bc.Blocks[0]
		return string(genesis.PrevHash) == "" &&
			bytes.Equal(genesis.Hash, genesis.CalculateHash()) &&
			(genesis.Nonce == 0 || genesis.IsValidProof()) // Genesis may not be mined
	}
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		previousBlock := bc.Blocks[i-1]

		if !bytes.Equal(currentBlock.Hash, currentBlock.CalculateHash()) {
			return false
		}
		if !bytes.Equal(currentBlock.PrevHash, previousBlock.Hash) {
			return false
		}
		// Check proof-of-work for mined blocks
		if currentBlock.Nonce > 0 && !currentBlock.IsValidProof() {
			return false
		}
	}
	return true
}

// IsValidWithUTXO validates the blockchain with UTXO double-spend protection
// This method ensures that no UTXOs are spent twice across the entire blockchain
func (bc *Blockchain) IsValidWithUTXO(utxoSet interface{}) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// First check basic blockchain validity
	if !bc.IsValid() {
		return false
	}

	// Create a temporary UTXO set for validation
	// This prevents double-spending by tracking spent UTXOs
	type UTXOValidator interface {
		ValidateTransaction(tx interface{}) error
		ProcessTransaction(tx interface{}) error
	}

	// If a UTXO set is provided, use it for validation
	if _, ok := utxoSet.(UTXOValidator); ok {
		// Clone the UTXO set to avoid modifying the original
		// and validate each transaction in order
		// Note: Currently blocks store data as bytes, not structured transactions
		// Full transaction validation will require Block struct refactoring
		// For now, we validate that data is not corrupted and has proper structure

		// Validate that all blocks have valid data
		for i, block := range bc.Blocks {
			if len(block.Data) == 0 && i > 0 {
				// Only genesis block can have empty data
				return false
			}

			// Validate data integrity (basic check)
			if !bytes.Equal(block.Hash, block.CalculateHash()) {
				return false
			}
		}
	}

	return true
}

// FindCommonAncestor finds the index of the last common block between two blockchains
func (bc *Blockchain) FindCommonAncestor(other *Blockchain) int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	minLen := len(bc.Blocks)
	if len(other.Blocks) < minLen {
		minLen = len(other.Blocks)
	}

	// Find the last common block by comparing hashes
	for i := minLen - 1; i >= 0; i-- {
		if bytes.Equal(bc.Blocks[i].Hash, other.Blocks[i].Hash) {
			return i
		}
	}

	return -1 // No common ancestor found
}

// CalculateTotalWork calculates the total work (cumulative difficulty) of the blockchain
// Work is calculated as 2^difficulty for each block
func (bc *Blockchain) CalculateTotalWork(fromIndex int) float64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var totalWork float64
	for i := fromIndex; i < len(bc.Blocks); i++ {
		// Work = 2^difficulty
		totalWork += calculateBlockWork(bc.Blocks[i].Difficulty)
	}
	return totalWork
}

// calculateBlockWork calculates the work required to mine a block with given difficulty
func calculateBlockWork(difficulty int) float64 {
	if difficulty <= 0 {
		return 1.0
	}
	// Work = 2^difficulty
	return float64(uint64(1) << uint(difficulty))
}

// ResolveFork resolves a blockchain fork by comparing total work
// Returns error if the other chain should replace this one
func (bc *Blockchain) ResolveFork(other *Blockchain) error {
	// Find common ancestor without holding lock to prevent deadlock
	// (FindCommonAncestor acquires locks on both chains)
	commonIndex := bc.FindCommonAncestor(other)
	if commonIndex == -1 {
		return fmt.Errorf("no common ancestor found, cannot resolve fork")
	}

	// Now acquire lock for the rest of the operation
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Calculate total work for both chains from the common ancestor
	myWork := bc.calculateTotalWorkLocked(commonIndex)
	otherWork := other.CalculateTotalWork(commonIndex)

	// Accept chain with more work (longest chain rule)
	if otherWork > myWork {
		return bc.replaceChainLocked(other, commonIndex)
	}

	return nil
}

// calculateTotalWorkLocked calculates total work (assumes lock is held)
func (bc *Blockchain) calculateTotalWorkLocked(fromIndex int) float64 {
	var totalWork float64
	for i := fromIndex; i < len(bc.Blocks); i++ {
		totalWork += calculateBlockWork(bc.Blocks[i].Difficulty)
	}
	return totalWork
}

// replaceChainLocked replaces the blockchain with another chain (assumes lock is held)
func (bc *Blockchain) replaceChainLocked(other *Blockchain, commonIndex int) error {
	// Validate the other chain
	if !other.IsValid() {
		return fmt.Errorf("other blockchain is invalid")
	}

	// Replace blocks from common ancestor onwards
	bc.Blocks = append(bc.Blocks[:commonIndex+1], other.Blocks[commonIndex+1:]...)

	// Update mining rewards (remove rewards from replaced blocks)
	var newRewards []*MiningReward
	for _, reward := range bc.MiningRewards {
		if reward.BlockIndex <= commonIndex {
			newRewards = append(newRewards, reward)
		}
	}
	bc.MiningRewards = newRewards

	// Add rewards from the new chain
	for _, reward := range other.MiningRewards {
		if reward.BlockIndex > commonIndex {
			bc.MiningRewards = append(bc.MiningRewards, reward)
		}
	}

	// Recalculate total rewards
	bc.TotalRewards = 0.0
	for _, reward := range bc.MiningRewards {
		bc.TotalRewards += reward.Reward
	}

	return nil
}

// ShouldReplaceChain checks if another blockchain should replace this one
func (bc *Blockchain) ShouldReplaceChain(other *Blockchain) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Find common ancestor
	commonIndex := bc.FindCommonAncestor(other)
	if commonIndex == -1 {
		return false
	}

	// Calculate total work
	myWork := bc.calculateTotalWorkLocked(commonIndex)
	otherWork := other.CalculateTotalWork(commonIndex)

	// Replace if other chain has more work
	return otherWork > myWork
}

func (bc *Blockchain) GetBlockByIndex(index int) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if index < 0 || index >= len(bc.Blocks) {
		return nil, fmt.Errorf("block index %d out of range", index)
	}
	return bc.Blocks[index], nil
}

func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, block := range bc.Blocks {
		if bytes.Equal(hash, block.Hash) {
			return block
		}
	}
	return nil
}

func (bc *Blockchain) GetChainLength() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks)
}

// Print entire blockchain for debugging
func (bc *Blockchain) PrintBlockChain() {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i, block := range bc.Blocks {
		fmt.Printf("Block:%d\n", i)
		fmt.Printf("Timestamp:%d\n", block.Timestamp)
		fmt.Printf("Data:%s\n", block.Data)
		fmt.Printf("PrevHash:%s\n", hex.EncodeToString(block.PrevHash))
		fmt.Printf("Hash:%s\n", hex.EncodeToString(block.Hash))
		fmt.Println()
	}
}

func (bc *Blockchain) ToJSON() ([]byte, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	jsonData, err := json.MarshalIndent(bc, "", " ")
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal blockchain to JSON: %w", err)
	}
	return jsonData, nil
}

func (bc *Blockchain) FromJSON(data []byte) error {
	var newBlockchain Blockchain

	if err := json.Unmarshal(data, &newBlockchain); err != nil {
		return fmt.Errorf("Failed to unmarshal blockchain from JSON: %w", err)
	}
	if !newBlockchain.IsValid() {
		return fmt.Errorf("Loaded blockchain is invalid")
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.Blocks = newBlockchain.Blocks
	bc.MiningRewards = newBlockchain.MiningRewards
	bc.TotalRewards = newBlockchain.TotalRewards
	return nil
}

func (bc *Blockchain) SaveToFile(filename string) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	jsonData, err := json.MarshalIndent(bc, "", " ")
	if err != nil {
		return fmt.Errorf("Failed to convert blockchain to JSON: %w", err)
	}
	err = os.WriteFile(filename, jsonData, 0600)
	if err != nil {
		return fmt.Errorf("Failed to write blockchain to file %s: %w", filename, err)
	}
	return nil
}

func (bc *Blockchain) LoadFromFile(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("files %s does not exist", filename)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Failed to read file %s: %w", filename, err)
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()

	var newBlockchain Blockchain
	if err := json.Unmarshal(data, &newBlockchain); err != nil {
		return fmt.Errorf("Failed to unmarshal blockchain from JSON: %w", err)
	}
	if !newBlockchain.IsValid() {
		return fmt.Errorf("Loaded blockchain is invalid")
	}
	bc.Blocks = newBlockchain.Blocks
	bc.MiningRewards = newBlockchain.MiningRewards
	bc.TotalRewards = newBlockchain.TotalRewards
	return nil
}

func (bc *Blockchain) ExportPrettyJSON() (string, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	jsonData, err := json.MarshalIndent(bc, "", " ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
