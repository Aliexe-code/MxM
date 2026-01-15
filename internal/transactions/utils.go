package transactions

import (
	"fmt"
	"math"
)

// CreateSimpleTransaction creates a simple transaction with one input and one output
func CreateSimpleTransaction(fromTxID string, fromIndex int, toAddress string, amount float64, changeAddress string) *Transaction {
	input := TxInput{
		TxID:      fromTxID,
		Index:     fromIndex,
		Signature: "",
		PublicKey: "",
	}

	output := TxOutput{
		Address: toAddress,
		Amount:  amount,
		TxID:    "",
		Index:   0,
	}

	tx := NewTransaction([]TxInput{input}, []TxOutput{output})

	// Add change output if change address is different
	if changeAddress != "" && changeAddress != toAddress {
		changeOutput := TxOutput{
			Address: changeAddress,
			Amount:  0, // Will be calculated later
			TxID:    "",
			Index:   1,
		}
		tx.Outputs = append(tx.Outputs, changeOutput)
	}

	return tx
}

// CalculateOptimalFee calculates the optimal transaction fee based on size and priority
func CalculateOptimalFee(inputCount, outputCount int, priority float64) float64 {
	// Base fee calculation
	baseFee := 0.0001 // Base fee in BTC-like units

	// Size-based component (approximate)
	size := inputCount*148 + outputCount*34 + 10 // Approximate transaction size
	sizeFee := float64(size) * 0.00000001        // 1 satoshi per byte

	// Priority multiplier (0.1 = low priority, 1.0 = normal, 10.0 = high priority)
	priorityMultiplier := math.Max(0.1, math.Min(10.0, priority))

	totalFee := (baseFee + sizeFee) * priorityMultiplier

	// Round to 8 decimal places (like Bitcoin)
	return math.Round(totalFee*1e8) / 1e8
}

// EstimateTransactionSize estimates the size of a transaction in bytes
func EstimateTransactionSize(inputCount, outputCount int) int {
	// Approximate sizes:
	// - Transaction header: ~10 bytes
	// - Each input: ~148 bytes (with signature)
	// - Each output: ~34 bytes
	// - Coinbase transactions are slightly different

	return 10 + inputCount*148 + outputCount*34
}

// CalculateDustThreshold calculates the minimum output amount to avoid dust
func CalculateDustThreshold(feeRate float64) float64 {
	// Dust is an output that costs more to spend than it's worth
	// Calculation based on the cost to spend the output

	// Cost to spend: ~148 bytes (input size) * fee rate
	costToSpend := 148 * feeRate

	// Add small buffer
	dustThreshold := costToSpend * 3

	return math.Round(dustThreshold*1e8) / 1e8
}

// IsDustOutput checks if an output is considered dust
func IsDustOutput(amount, feeRate float64) bool {
	dustThreshold := CalculateDustThreshold(feeRate)
	return amount < dustThreshold
}

// OptimizeTransaction optimizes a transaction by removing dust outputs and consolidating amounts
func (tx *Transaction) OptimizeTransaction(minOutputValue float64) {
	var optimizedOutputs []TxOutput
	var totalDust float64

	// Filter out dust outputs
	for _, output := range tx.Outputs {
		if output.Amount >= minOutputValue {
			optimizedOutputs = append(optimizedOutputs, output)
		} else {
			totalDust += output.Amount
		}
	}

	// If there's dust to consolidate and we have at least one output
	if totalDust > 0 && len(optimizedOutputs) > 0 {
		// Add dust to the first output
		optimizedOutputs[0].Amount += totalDust
	}

	tx.Outputs = optimizedOutputs
	tx.ID = tx.CalculateID() // Recalculate ID after changes
}

// GetTransactionSummary returns a summary of transaction statistics
func (tx *Transaction) GetTransactionSummary(utxoSet map[string]map[int]TxOutput) map[string]interface{} {
	fee := tx.GetFee(utxoSet)
	estimatedSize := EstimateTransactionSize(len(tx.Inputs), len(tx.Outputs))
	feeRate := fee / float64(estimatedSize)

	summary := map[string]interface{}{
		"tx_id":          tx.ID,
		"is_coinbase":    tx.IsCoinbase(),
		"input_count":    len(tx.Inputs),
		"output_count":   len(tx.Outputs),
		"total_input":    tx.GetInputAmount(utxoSet),
		"total_output":   tx.GetOutputAmount(),
		"fee":            fee,
		"estimated_size": estimatedSize,
		"fee_rate":       feeRate,
		"timestamp":      tx.Timestamp,
	}

	// Add input/output details
	var inputs []map[string]interface{}
	for _, input := range tx.Inputs {
		inputs = append(inputs, map[string]interface{}{
			"tx_id":   input.TxID,
			"index":   input.Index,
			"has_sig": input.Signature != "",
			"has_key": input.PublicKey != "",
		})
	}
	summary["inputs"] = inputs

	var outputs []map[string]interface{}
	for _, output := range tx.Outputs {
		outputs = append(outputs, map[string]interface{}{
			"address": output.Address,
			"amount":  output.Amount,
			"index":   output.Index,
		})
	}
	summary["outputs"] = outputs

	return summary
}

// CloneTransaction creates a deep copy of a transaction
func (tx *Transaction) CloneTransaction() *Transaction {
	clone := &Transaction{
		ID:        tx.ID,
		Inputs:    make([]TxInput, len(tx.Inputs)),
		Outputs:   make([]TxOutput, len(tx.Outputs)),
		Timestamp: tx.Timestamp,
	}

	// Copy inputs
	for i, input := range tx.Inputs {
		clone.Inputs[i] = TxInput{
			TxID:      input.TxID,
			Index:     input.Index,
			Signature: input.Signature,
			PublicKey: input.PublicKey,
		}
	}

	// Copy outputs
	for i, output := range tx.Outputs {
		clone.Outputs[i] = TxOutput{
			Address: output.Address,
			Amount:  output.Amount,
			TxID:    output.TxID,
			Index:   output.Index,
		}
	}

	return clone
}

// MergeTransactions merges multiple transactions into one (for consolidation)
func MergeTransactions(transactions []*Transaction) *Transaction {
	var allInputs []TxInput
	var allOutputs []TxOutput

	for _, tx := range transactions {
		allInputs = append(allInputs, tx.Inputs...)
		allOutputs = append(allOutputs, tx.Outputs...)
	}

	mergedTx := NewTransaction(allInputs, allOutputs)
	return mergedTx
}

// ValidateTransactionBalance checks if transaction inputs and outputs balance properly
func (tx *Transaction) ValidateTransactionBalance(utxoSet map[string]map[int]TxOutput) error {
	if tx.IsCoinbase() {
		return nil // Coinbase transactions don't need balance validation
	}

	// Calculate actual input amount from UTXO set
	var actualInputAmount float64
	for _, input := range tx.Inputs {
		if outputs, exists := utxoSet[input.TxID]; exists {
			if output, exists := outputs[input.Index]; exists {
				actualInputAmount += output.Amount
			} else {
				return fmt.Errorf("input references non-existent output %d in tx %s",
					input.Index, input.TxID)
			}
		} else {
			return fmt.Errorf("input references unknown transaction %s", input.TxID)
		}
	}

	outputAmount := tx.GetOutputAmount()

	// Check balance
	if actualInputAmount < outputAmount {
		return fmt.Errorf("insufficient input amount: have %.8f, need %.8f",
			actualInputAmount, outputAmount)
	}

	return nil
}

// GetTransactionType determines the type of transaction
func (tx *Transaction) GetTransactionType() string {
	if tx.IsCoinbase() {
		return "coinbase"
	}

	if len(tx.Inputs) == 1 && len(tx.Outputs) == 1 {
		return "simple_transfer"
	}

	if len(tx.Outputs) > 2 {
		return "multi_output"
	}

	if len(tx.Inputs) > 1 {
		return "consolidation"
	}

	return "standard"
}
