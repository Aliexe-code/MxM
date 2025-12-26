package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// MiningReward represents a reward for mining a block
type MiningReward struct {
	MinerID    string    `json:"miner_id"`
	BlockIndex int       `json:"block_index"`
	Reward     float64   `json:"reward"`
	Timestamp  time.Time `json:"timestamp"`
	Difficulty int       `json:"difficulty"`
}

type Blockchain struct {
	Blocks         []*Block       `json:"blocks"`
	MiningRewards  []*MiningReward `json:"mining_rewards"`
	TotalRewards   float64         `json:"total_rewards"`
	rewardMutex    sync.RWMutex
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
	lastestBlock := bc.GetLatestBlock()

	newBlock := NewBlock([]byte(data), lastestBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
}

// AddBlockWithMining adds a block with mining and rewards
func (bc *Blockchain) AddBlockWithMining(data string, minerID string, difficulty int) (time.Duration, error) {
	latestBlock := bc.GetLatestBlock()
	if latestBlock == nil {
		return 0, fmt.Errorf("no latest block found")
	}

	newBlock := NewBlock([]byte(data), latestBlock.Hash)
	
	// Mine the block
	duration := newBlock.MineBlock(difficulty)
	
	// Add block to chain
	bc.Blocks = append(bc.Blocks, newBlock)
	
	// Calculate and add reward
	reward := bc.CalculateReward(difficulty)
	bc.AddMiningReward(minerID, len(bc.Blocks)-1, reward, difficulty)
	
	return duration, nil
}

// CalculateReward calculates mining reward based on difficulty
func (bc *Blockchain) CalculateReward(difficulty int) float64 {
	// Base reward + difficulty bonus
	baseReward := 10.0
	difficultyBonus := float64(difficulty) * 2.5
	return baseReward + difficultyBonus
}

// AddMiningReward adds a mining reward to the tracking system
func (bc *Blockchain) AddMiningReward(minerID string, blockIndex int, reward float64, difficulty int) {
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
		"total_blocks":    len(bc.Blocks),
		"total_rewards":   bc.TotalRewards,
		"miner_rewards":   minerStats,
		"miner_blocks":    blockCount,
		"reward_count":    len(bc.MiningRewards),
	}
}

func (bc *Blockchain) GetLatestBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) IsValid() bool {
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

func (bc *Blockchain) GetBlockByIndex(index int) (*Block, error) {
	if index < 0 || index >= len(bc.Blocks) {
		return nil, fmt.Errorf("block index %d out of range", index)
	}
	return bc.Blocks[index], nil
}

func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, block := range bc.Blocks {
		if bytes.Equal(hash, block.Hash) {
			return block
		}
	}
	return nil
}

func (bc *Blockchain) GetChainLength() int {
	return len(bc.Blocks)
}

// Print entire blockchain for debugging
func (bc *Blockchain) PrintBlockChain() {
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
	bc.Blocks = newBlockchain.Blocks
	return nil
}

func (bc *Blockchain) SaveToFile(filename string) error {
	jsonData, err := bc.ToJSON()
	if err != nil {
		return fmt.Errorf("Failed to convert blockchain to JSON: %w", err)
	}
	err = os.WriteFile(filename, jsonData, 0664)
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
	err = bc.FromJSON(data)
	if err != nil {
		return fmt.Errorf("Failed to load blockchain from file %s: %w", filename, err)
	}
	return nil
}

func (bc *Blockchain) ExportPrettyJSON() (string, error) {
	jsonData, err := bc.ToJSON()
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
