package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Block struct {
	Timestamp int64  `json:"timestamp"`
	Data      []byte `json:"data"`
	PrevHash  []byte `json:"prev_hash"`
	Hash      []byte `json:"hash"`
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
