package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	genesis := NewGenesisBlock()

	genesis.Hash = genesis.CalculateHash()

	return &Blockchain{
		Blocks: []*Block{genesis},
	}
}

func (bc *Blockchain) AddBlock(data string) error {
	lastestBlock := bc.GetLatestBlock()

	newBlock := NewBlock([]byte(data), lastestBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
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
		return string(genesis.PrevHash) == "" && bytes.Equal(genesis.Hash, genesis.CalculateHash())
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
		fmt.Printf("PrevHash:%s\n", block.PrevHash)
		fmt.Printf("Hash:%s\n", block.Hash)
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
