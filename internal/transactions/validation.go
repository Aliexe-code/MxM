package transactions

import (
	"fmt"
	"math"
)

// ValidateAmounts validates transaction amounts and sums
func (tx *Transaction) ValidateAmounts() error {
	if tx.IsCoinbase() {
		if len(tx.Outputs) != 1 {
			return fmt.Errorf("coinbase transaction must have exactly one output")
		}
		if tx.Outputs[0].Amount <= 0 {
			return fmt.Errorf("coinbase amount must be positive")
		}
		return nil
	}

	// Calculate total input and output amounts
	inputAmount := tx.GetInputAmount()
	outputAmount := tx.GetOutputAmount()

	// Check that output amount doesn't exceed input amount
	if outputAmount > inputAmount {
		return fmt.Errorf("output amount (%.8f) exceeds input amount (%.8f)",
			outputAmount, inputAmount)
	}

	// Check that fee is non-negative
	fee := tx.GetFee()
	if fee < 0 {
		return fmt.Errorf("transaction fee cannot be negative: %.8f", fee)
	}

	// Validate individual output amounts
	for i, output := range tx.Outputs {
		if output.Amount <= 0 {
			return fmt.Errorf("output %d has invalid amount: %.8f", i, output.Amount)
		}
		if math.IsNaN(outputAmount) || math.IsInf(outputAmount, 0) {
			return fmt.Errorf("output %d has invalid amount value: %f", i, output.Amount)
		}
	}

	return nil
}

// ValidateInputs validates transaction inputs against available UTXOs
func (tx *Transaction) ValidateInputs(utxoSet map[string]map[int]TxOutput) error {
	if tx.IsCoinbase() {
		return nil // Coinbase transactions have no inputs to validate
	}

	for i, input := range tx.Inputs {
		// Check if the referenced transaction exists in UTXO set
		outputs, exists := utxoSet[input.TxID]
		if !exists {
			return fmt.Errorf("input %d references unknown transaction: %s", i, input.TxID)
		}

		// Check if the referenced output exists
		output, exists := outputs[input.Index]
		if !exists {
			return fmt.Errorf("input %d references non-existent output %d in transaction %s",
				i, input.Index, input.TxID)
		}

		// Validate the signature
		err := tx.VerifyInputSignature(i, []TxOutput{output})
		if err != nil {
			return fmt.Errorf("input %d signature verification failed: %w", i, err)
		}
	}

	return nil
}

// CalculateChange calculates the change amount for a transaction
func CalculateChange(inputAmount, outputAmount, desiredFee float64) (float64, float64, error) {
	if inputAmount <= outputAmount {
		return 0, 0, fmt.Errorf("input amount (%.8f) must be greater than output amount (%.8f)",
			inputAmount, outputAmount)
	}

	// Calculate actual fee (input - output)
	actualFee := inputAmount - outputAmount

	// Calculate change (if any)
	change := inputAmount - outputAmount - desiredFee

	if change < 0 {
		change = 0
	}

	return change, actualFee, nil
}

// CreateChangeOutput creates a change output
func CreateChangeOutput(address string, amount float64) TxOutput {
	return TxOutput{
		Address: address,
		Amount:  amount,
		TxID:    "", // Will be set when transaction is finalized
		Index:   -1, // Will be set when transaction is finalized
	}
}

// ValidateAddressFormat validates a blockchain address format
func ValidateAddressFormat(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if len(address) != 42 { // 0x + 40 hex characters
		return fmt.Errorf("address must be 42 characters long, got %d", len(address))
	}

	if !HasHexPrefix(address) {
		return fmt.Errorf("address must start with 0x prefix")
	}

	// Validate hex characters
	for _, char := range address[2:] {
		if !isHexChar(byte(char)) {
			return fmt.Errorf("address contains invalid hex character: %c", char)
		}
	}

	return nil
}

// HasHexPrefix checks if string has 0x prefix
func HasHexPrefix(s string) bool {
	return len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X')
}

// isHexChar checks if character is a valid hex character
func isHexChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// ValidateTransactionStructure performs comprehensive validation
func (tx *Transaction) ValidateTransactionStructure(utxoSet map[string]map[int]TxOutput) error {
	// Basic validation
	if err := tx.ValidateBasic(); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	// Amount validation
	if err := tx.ValidateAmounts(); err != nil {
		return fmt.Errorf("amount validation failed: %w", err)
	}

	// Input validation against UTXO set
	if err := tx.ValidateInputs(utxoSet); err != nil {
		return fmt.Errorf("input validation failed: %w", err)
	}

	// Validate address formats
	for i, output := range tx.Outputs {
		if err := ValidateAddressFormat(output.Address); err != nil {
			return fmt.Errorf("output %d address validation failed: %w", i, err)
		}
	}

	return nil
}

// GetValidationReport returns a detailed validation report
func (tx *Transaction) GetValidationReport(utxoSet map[string]map[int]TxOutput) map[string]interface{} {
	report := map[string]interface{}{
		"is_valid":     false,
		"errors":       []string{},
		"warnings":     []string{},
		"input_count":  len(tx.Inputs),
		"output_count": len(tx.Outputs),
		"total_amount": tx.GetOutputAmount(),
		"fee":          tx.GetFee(),
		"is_coinbase":  tx.IsCoinbase(),
	}

	// Perform validation checks
	if err := tx.ValidateBasic(); err != nil {
		report["errors"] = append(report["errors"].([]string), fmt.Sprintf("Basic: %v", err))
	}

	if err := tx.ValidateAmounts(); err != nil {
		report["errors"] = append(report["errors"].([]string), fmt.Sprintf("Amounts: %v", err))
	}

	if err := tx.ValidateInputs(utxoSet); err != nil {
		report["errors"] = append(report["errors"].([]string), fmt.Sprintf("Inputs: %v", err))
	}

	// Check for warnings
	if tx.GetFee() > 0.01 {
		report["warnings"] = append(report["warnings"].([]string), "High transaction fee")
	}

	if len(tx.Outputs) > 10 {
		report["warnings"] = append(report["warnings"].([]string), "Many outputs may affect performance")
	}

	// Set final validation status
	report["is_valid"] = len(report["errors"].([]string)) == 0

	return report
}
