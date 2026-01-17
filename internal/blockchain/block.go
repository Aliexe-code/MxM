package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

type Block struct {
	Timestamp  int64  `json:"timestamp"`
	Data       []byte `json:"data"`
	PrevHash   []byte `json:"prev_hash"`
	Hash       []byte `json:"hash"`
	Nonce      uint32 `json:"nonce"`
	Difficulty int    `json:"difficulty"`
}

func (b *Block) CalculateHash() []byte {
	data := b.PrevHash
	data = append(data, b.Data...)
	data = append(data, []byte(fmt.Sprintf("%d", b.Timestamp))...)
	data = append(data, []byte(fmt.Sprintf("%d", b.Difficulty))...)
	data = append(data, []byte(fmt.Sprintf("%d", b.Nonce))...)

	h := sha256.Sum256(data)
	return h[:]
}

func NewGenesisBlock() *Block {
	block := &Block{
		Timestamp:  time.Now().Unix(),
		Data:       []byte("Genesis Block"),
		PrevHash:   []byte{},
		Nonce:      0,
		Difficulty: DefaultDifficulty,
	}
	block.Hash = block.CalculateHash()
	return block

}

func NewBlock(data []byte, prevHash []byte) *Block {
	block := &Block{
		Timestamp:  time.Now().Unix(),
		Data:       data,
		PrevHash:   prevHash,
		Hash:       []byte{},
		Nonce:      0,
		Difficulty: DefaultDifficulty,
	}
	block.Hash = block.CalculateHash()
	return block
}

func (b *Block) MineBlock(difficulty int) time.Duration {
	pow := NewProofOfWork(b, difficulty)
	nonce, hash, duration := pow.Run(context.Background())
	// Always set the difficulty, even if mining fails
	b.Difficulty = difficulty
	if hash != nil {
		b.Nonce = nonce
		b.Hash = hash
	}
	return duration
}

// MineBlockCancellable returns a ProofOfWork instance that can be cancelled
func (b *Block) MineBlockCancellable(difficulty int) (*ProofOfWork, func() time.Duration) {
	pow := NewProofOfWork(b, difficulty)

	// Return the mining function that can be called to start mining
	miningFunc := func() time.Duration {
		nonce, hash, duration := pow.Run(context.Background())
		// Always set the difficulty, even if mining fails
		b.Difficulty = difficulty
		if hash != nil {
			b.Nonce = nonce
			b.Hash = hash
		} else {
			fmt.Printf("Failed to mine Block: %s\n", string(b.Data))
		}
		return duration
	}

	return pow, miningFunc
}

func (b *Block) IsValidProof() bool {
	pow := NewProofOfWork(b, b.Difficulty)
	return pow.Validate()
}

func (b *Block) MarshalJSON() ([]byte, error) {
	type Alias Block
	return json.Marshal(&struct {
		Timestamp  int64  `json:"timestamp"`
		Data       []byte `json:"data"`
		PrevHash   []byte `json:"prev_hash"`
		Hash       []byte `json:"hash"`
		Nonce      uint32 `json:"nonce"`
		Difficulty int    `json:"difficulty"`
	}{
		Timestamp:  b.Timestamp,
		Data:       []byte(b.Data),
		PrevHash:   []byte(b.PrevHash),
		Hash:       []byte(b.Hash),
		Nonce:      b.Nonce,
		Difficulty: b.Difficulty,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	type Alias Block

	aux := struct {
		Timestamp  int64  `json:"timestamp"`
		Data       []byte `json:"data"`
		PrevHash   []byte `json:"prev_hash"`
		Hash       []byte `json:"hash"`
		Nonce      uint32 `json:"nonce"`
		Difficulty int    `json:"difficulty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("Failed to unmarshal block json:%w", err)
	}
	b.Timestamp = aux.Timestamp
	b.Data = []byte(aux.Data)
	b.PrevHash = []byte(aux.PrevHash)
	b.Hash = []byte(aux.Hash)
	b.Nonce = aux.Nonce
	b.Difficulty = aux.Difficulty
	return nil
}
