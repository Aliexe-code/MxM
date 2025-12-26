package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		Timestamp int64  `json:"timestamp"`
		Data      []byte `json:"data"`
		PrevHash  []byte `json:"prev_hash"`
		Hash      []byte `json:"hash"`
	}{
		Timestamp: b.Timestamp,
		Data:      []byte(b.Data),
		PrevHash:  []byte(b.PrevHash),
		Hash:      []byte(b.Hash),
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	type Alias Block

	aux := struct {
		Timestamp int64  `json:"timestamp"`
		Data      []byte `json:"data"`
		PrevHash  []byte `json:"prev_hash"`
		Hash      []byte `json:"hash"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("Failed to unmarshal block json:%w", err)
	}
	b.Timestamp = aux.Timestamp
	b.Data = []byte(aux.Data)
	b.PrevHash = []byte(aux.PrevHash)
	b.Hash = []byte(aux.Hash)

	return nil
}
