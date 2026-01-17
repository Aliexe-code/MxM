package transactions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type TxOutput struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	TxID    string  `json:"tx_id"`
	Index   int     `json:"index"`
}

type TxInput struct {
	TxID      string `json:"tx_id"`
	Index     int    `json:"index"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}
type Transaction struct {
	ID        string     `json:"id"`
	Inputs    []TxInput  `json:"inputs"`
	Outputs   []TxOutput `json:"outputs"`
	Timestamp int64      `json:"timestamp"`
}

func NewTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: 0, // Will be set when finalized
	}
	tx.ID = tx.CalculateID()
	return tx
}
func NewCoinbaseTransaction(toAddress string, amount float64) *Transaction {
	// Coinbase has no inputs (special case)
	output := TxOutput{
		Address: toAddress,
		Amount:  amount,
		TxID:    "", // Will be set after ID calculation
		Index:   0,
	}

	tx := &Transaction{
		Inputs:    []TxInput{}, // Empty for coinbase
		Outputs:   []TxOutput{output},
		Timestamp: 0,
	}

	tx.ID = tx.CalculateID()

	// Update output with the transaction ID
	for i := range tx.Outputs {
		tx.Outputs[i].TxID = tx.ID
		tx.Outputs[i].Index = i
	}

	return tx
}
func (tx *Transaction) CalculateID() string {
	// Create a copy for ID calculation (without ID field)
	txCopy := &Transaction{
		Inputs:    tx.Inputs,
		Outputs:   tx.Outputs,
		Timestamp: tx.Timestamp,
	}

	// Serialize to JSON (sorted for consistency)
	jsonData, err := json.Marshal(txCopy)
	if err != nil {
		return ""
	}

	// Double SHA256 (like Bitcoin)
	firstHash := sha256.Sum256(jsonData)
	secondHash := sha256.Sum256(firstHash[:])

	return hex.EncodeToString(secondHash[:])
}
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 0
}
func (tx *Transaction) GetInputAmount(utxoSet map[string]map[int]TxOutput) float64 {
	var amount float64
	for _, input := range tx.Inputs {
		// Look up the referenced output in the UTXO set
		if outputs, exists := utxoSet[input.TxID]; exists {
			if output, exists := outputs[input.Index]; exists {
				amount += output.Amount
			}
		}
	}
	return amount
}
func (tx *Transaction) GetOutputAmount() float64 {
	var amount float64
	for _, output := range tx.Outputs {
		amount += output.Amount
	}
	return amount
}

func (tx *Transaction) GetFee(utxoSet map[string]map[int]TxOutput) float64 {
	if tx.IsCoinbase() {
		return 0 // Coinbase transactions have no fee
	}
	return tx.GetInputAmount(utxoSet) - tx.GetOutputAmount()
}
func (tx *Transaction) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction to JSON: %w", err)
	}
	return string(jsonData), nil
}

// FromJSON creates transaction from JSON string
func FromJSON(jsonStr string) (*Transaction, error) {
	var tx Transaction
	err := json.Unmarshal([]byte(jsonStr), &tx)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction from JSON: %w", err)
	}
	return &tx, nil
}

// ValidateBasic validates basic transaction structure
func (tx *Transaction) ValidateBasic() error {
	if tx.ID == "" {
		return fmt.Errorf("transaction ID is empty")
	}

	if len(tx.Inputs) == 0 && len(tx.Outputs) == 0 {
		return fmt.Errorf("transaction must have inputs or outputs")
	}

	// Validate outputs
	for i, output := range tx.Outputs {
		if output.Address == "" {
			return fmt.Errorf("output %d has empty address", i)
		}
		if output.Amount <= 0 {
			return fmt.Errorf("output %d has invalid amount: %f", i, output.Amount)
		}
		if !strings.HasPrefix(output.Address, "0x") {
			return fmt.Errorf("output %d has invalid address format: %s", i, output.Address)
		}
	}

	// Validate inputs (except for coinbase)
	if !tx.IsCoinbase() {
		for i, input := range tx.Inputs {
			if input.TxID == "" {
				return fmt.Errorf("input %d has empty tx ID", i)
			}
			if input.Index < 0 {
				return fmt.Errorf("input %d has invalid index: %d", i, input.Index)
			}
		}
	}

	return nil
}

// String returns a string representation of the transaction
func (tx *Transaction) String() string {
	return fmt.Sprintf("Transaction{ID: %s, Inputs: %d, Outputs: %d, Amount: %.8f}",
		tx.ID[:8]+"...", len(tx.Inputs), len(tx.Outputs), tx.GetOutputAmount())
}

// GetInfo returns detailed information about the transaction
func (tx *Transaction) GetInfo(utxoSet map[string]map[int]TxOutput) map[string]interface{} {
	return map[string]interface{}{
		"id":            tx.ID,
		"is_coinbase":   tx.IsCoinbase(),
		"input_count":   len(tx.Inputs),
		"output_count":  len(tx.Outputs),
		"input_amount":  tx.GetInputAmount(utxoSet),
		"output_amount": tx.GetOutputAmount(),
		"fee":           tx.GetFee(utxoSet),
		"timestamp":     tx.Timestamp,
	}
}
