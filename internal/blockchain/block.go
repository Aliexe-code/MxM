package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Blockchain struct {
	Blocks []*Block
}

type Block struct {
	Timestamp int64  `json:"timestamp"`
	Data      []byte `json:"data"`
	PrevHash  []byte `json:"prev_hash"`
	Hash      []byte `json:"hash"`
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

func (b *Block) CalculateHash() []byte {
	record := fmt.Sprintf("%d %s%s", b.Timestamp, b.Data, b.PrevHash)

	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)

	return []byte(hex.EncodeToString(hashed))
}

func NewGenesisBlock() *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		Data:      []byte("Genesis Block"),
		PrevHash:  []byte{},
	}
	block.Hash = block.CalculateHash()
	return block

}

func NewBlock(data []byte, prevHash []byte) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		Data:      data,
		PrevHash:  prevHash,
		Hash:      []byte{},
	}
	block.Hash = block.CalculateHash()
	return block
}
