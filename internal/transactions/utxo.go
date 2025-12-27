package transactions

import (
	"fmt"
	"sync"
)

// UTXOKey represents a unique identifier for a UTXO
type UTXOKey struct {
	TxID  string
	Index int
}

// String returns a string representation of the UTXO key
func (k UTXOKey) String() string {
	return fmt.Sprintf("%s:%d", k.TxID, k.Index)
}

// UTXOSet represents the set of all unspent transaction outputs
type UTXOSet struct {
	utxos      map[UTXOKey]TxOutput
	mu         sync.RWMutex
	totalCount int
	totalValue float64
}

// NewUTXOSet creates a new empty UTXO set
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{
		utxos: make(map[UTXOKey]TxOutput),
	}
}

// Add adds a new UTXO to the set
func (us *UTXOSet) Add(txID string, index int, output TxOutput) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	key := UTXOKey{TxID: txID, Index: index}

	// Check if UTXO already exists
	if _, exists := us.utxos[key]; exists {
		return fmt.Errorf("UTXO already exists: %s", key.String())
	}

	// Set transaction info on output
	output.TxID = txID
	output.Index = index

	// Add to set
	us.utxos[key] = output
	us.totalCount++
	us.totalValue += output.Amount

	return nil
}

// Spend removes a UTXO from the set (when it's spent as an input)
func (us *UTXOSet) Spend(txID string, index int) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	key := UTXOKey{TxID: txID, Index: index}

	output, exists := us.utxos[key]
	if !exists {
		return fmt.Errorf("UTXO not found: %s", key.String())
	}

	// Remove from set
	delete(us.utxos, key)
	us.totalCount--
	us.totalValue -= output.Amount

	return nil
}

// Get retrieves a UTXO from the set
func (us *UTXOSet) Get(txID string, index int) (TxOutput, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	key := UTXOKey{TxID: txID, Index: index}

	output, exists := us.utxos[key]
	if !exists {
		return TxOutput{}, fmt.Errorf("UTXO not found: %s", key.String())
	}

	return output, nil
}

// Exists checks if a UTXO exists in the set
func (us *UTXOSet) Exists(txID string, index int) bool {
	us.mu.RLock()
	defer us.mu.RUnlock()

	key := UTXOKey{TxID: txID, Index: index}
	_, exists := us.utxos[key]
	return exists
}

// GetAll returns all UTXOs in the set
func (us *UTXOSet) GetAll() map[UTXOKey]TxOutput {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[UTXOKey]TxOutput, len(us.utxos))
	for key, output := range us.utxos {
		result[key] = output
	}

	return result
}

// GetByAddress returns all UTXOs for a specific address
func (us *UTXOSet) GetByAddress(address string) []TxOutput {
	us.mu.RLock()
	defer us.mu.RUnlock()

	var outputs []TxOutput
	for _, output := range us.utxos {
		if output.Address == address {
			outputs = append(outputs, output)
		}
	}

	return outputs
}

// GetByAmount returns UTXOs greater than or equal to the specified amount
func (us *UTXOSet) GetByAmount(minAmount float64) []TxOutput {
	us.mu.RLock()
	defer us.mu.RUnlock()

	var outputs []TxOutput
	for _, output := range us.utxos {
		if output.Amount >= minAmount {
			outputs = append(outputs, output)
		}
	}

	return outputs
}

// SelectForAmount selects UTXOs to cover the specified amount
// Returns selected UTXOs and the total amount selected
func (us *UTXOSet) SelectForAmount(targetAmount float64, address string) ([]TxOutput, float64, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	if targetAmount <= 0 {
		return nil, 0, fmt.Errorf("target amount must be positive")
	}

	// Get all UTXOs for the specified address
	var addressUTXOs []TxOutput
	for _, output := range us.utxos {
		if output.Address == address {
			addressUTXOs = append(addressUTXOs, output)
		}
	}

	if len(addressUTXOs) == 0 {
		return nil, 0, fmt.Errorf("no UTXOs found for address: %s", address)
	}

	// Sort by amount ascending (select smallest first)
	for i := 0; i < len(addressUTXOs)-1; i++ {
		for j := i + 1; j < len(addressUTXOs); j++ {
			if addressUTXOs[i].Amount > addressUTXOs[j].Amount {
				addressUTXOs[i], addressUTXOs[j] = addressUTXOs[j], addressUTXOs[i]
			}
		}
	}

	var selected []TxOutput
	var totalAmount float64

	// Select UTXOs until we reach the target amount
	for _, output := range addressUTXOs {
		selected = append(selected, output)
		totalAmount += output.Amount
		if totalAmount >= targetAmount {
			break
		}
	}

	if totalAmount < targetAmount {
		return nil, 0, fmt.Errorf("insufficient funds: have %.8f, need %.8f", totalAmount, targetAmount)
	}

	return selected, totalAmount, nil
}

// ValidateTransaction validates a transaction against the UTXO set
func (us *UTXOSet) ValidateTransaction(tx *Transaction) error {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Coinbase transactions don't need UTXO validation
	if tx.IsCoinbase() {
		return nil
	}

	// Check that all inputs reference existing UTXOs
	for i, input := range tx.Inputs {
		key := UTXOKey{TxID: input.TxID, Index: input.Index}

		if _, exists := us.utxos[key]; !exists {
			return fmt.Errorf("input %d references non-existent UTXO: %s", i, key.String())
		}
	}

	// Check for double spending within this transaction
	inputKeys := make(map[UTXOKey]bool)
	for i, input := range tx.Inputs {
		key := UTXOKey{TxID: input.TxID, Index: input.Index}
		if inputKeys[key] {
			return fmt.Errorf("input %d attempts double spend of UTXO: %s", i, key.String())
		}
		inputKeys[key] = true
	}

	return nil
}

// ProcessTransaction processes a transaction, spending inputs and adding outputs
func (us *UTXOSet) ProcessTransaction(tx *Transaction) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	// Validate transaction first
	if err := us.validateTransactionLocked(tx); err != nil {
		return fmt.Errorf("transaction validation failed: %w", err)
	}

	// Spend inputs (remove UTXOs)
	for _, input := range tx.Inputs {
		key := UTXOKey{TxID: input.TxID, Index: input.Index}
		output, exists := us.utxos[key]
		if !exists {
			// This should not happen if validation passed
			return fmt.Errorf("UTXO not found during processing: %s", key.String())
		}

		delete(us.utxos, key)
		us.totalCount--
		us.totalValue -= output.Amount
	}

	// Add outputs (create new UTXOs)
	for i, output := range tx.Outputs {
		key := UTXOKey{TxID: tx.ID, Index: i}

		// Set transaction info on output
		output.TxID = tx.ID
		output.Index = i

		us.utxos[key] = output
		us.totalCount++
		us.totalValue += output.Amount
	}

	return nil
}

// validateTransactionLocked validates a transaction (assumes lock is held)
func (us *UTXOSet) validateTransactionLocked(tx *Transaction) error {
	// Coinbase transactions don't need UTXO validation
	if tx.IsCoinbase() {
		return nil
	}

	// Check that all inputs reference existing UTXOs
	for i, input := range tx.Inputs {
		key := UTXOKey{TxID: input.TxID, Index: input.Index}

		if _, exists := us.utxos[key]; !exists {
			return fmt.Errorf("input %d references non-existent UTXO: %s", i, key.String())
		}
	}

	// Check for double spending within this transaction
	inputKeys := make(map[UTXOKey]bool)
	for i, input := range tx.Inputs {
		key := UTXOKey{TxID: input.TxID, Index: input.Index}
		if inputKeys[key] {
			return fmt.Errorf("input %d attempts double spend of UTXO: %s", i, key.String())
		}
		inputKeys[key] = true
	}

	return nil
}

// GetCount returns the number of UTXOs in the set
func (us *UTXOSet) GetCount() int {
	us.mu.RLock()
	defer us.mu.RUnlock()
	return us.totalCount
}

// GetTotalValue returns the total value of all UTXOs in the set
func (us *UTXOSet) GetTotalValue() float64 {
	us.mu.RLock()
	defer us.mu.RUnlock()
	return us.totalValue
}

// GetBalance returns the balance for a specific address
func (us *UTXOSet) GetBalance(address string) float64 {
	us.mu.RLock()
	defer us.mu.RUnlock()

	var balance float64
	for _, output := range us.utxos {
		if output.Address == address {
			balance += output.Amount
		}
	}

	return balance
}

// Clear removes all UTXOs from the set
func (us *UTXOSet) Clear() {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.utxos = make(map[UTXOKey]TxOutput)
	us.totalCount = 0
	us.totalValue = 0
}

// Clone creates a deep copy of the UTXO set
func (us *UTXOSet) Clone() *UTXOSet {
	us.mu.RLock()
	defer us.mu.RUnlock()

	clone := &UTXOSet{
		utxos:      make(map[UTXOKey]TxOutput, len(us.utxos)),
		totalCount: us.totalCount,
		totalValue: us.totalValue,
	}

	for key, output := range us.utxos {
		clone.utxos[key] = output
	}

	return clone
}

// GetStats returns statistics about the UTXO set
func (us *UTXOSet) GetStats() map[string]interface{} {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Calculate address distribution
	addressCounts := make(map[string]int)
	addressValues := make(map[string]float64)

	for _, output := range us.utxos {
		addressCounts[output.Address]++
		addressValues[output.Address] += output.Amount
	}

	return map[string]interface{}{
		"total_count":    us.totalCount,
		"total_value":    us.totalValue,
		"address_count":  len(addressCounts),
		"address_counts": addressCounts,
		"address_values": addressValues,
		"average_utxo_value": func() float64 {
			if us.totalCount == 0 {
				return 0
			}
			return us.totalValue / float64(us.totalCount)
		}(),
	}
}

// FindUTXOsForAmount finds the minimum number of UTXOs needed to reach the target amount
// Uses a simple greedy algorithm (can be improved with more sophisticated selection)
func (us *UTXOSet) FindUTXOsForAmount(targetAmount float64, address string) ([]TxOutput, float64, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	if targetAmount <= 0 {
		return nil, 0, fmt.Errorf("target amount must be positive")
	}

	// Get all UTXOs for the address
	var addressUTXOs []TxOutput
	for _, output := range us.utxos {
		if output.Address == address {
			addressUTXOs = append(addressUTXOs, output)
		}
	}

	if len(addressUTXOs) == 0 {
		return nil, 0, fmt.Errorf("no UTXOs found for address: %s", address)
	}

	// Sort by amount ascending for consistent selection
	for i := 0; i < len(addressUTXOs)-1; i++ {
		for j := i + 1; j < len(addressUTXOs); j++ {
			if addressUTXOs[i].Amount > addressUTXOs[j].Amount {
				addressUTXOs[i], addressUTXOs[j] = addressUTXOs[j], addressUTXOs[i]
			}
		}
	}

	// Sort by amount descending (simple greedy approach)
	// In a real implementation, you might want more sophisticated selection
	var selected []TxOutput
	var totalAmount float64

	// First try to find exact matches or larger amounts
	for _, output := range addressUTXOs {
		if output.Amount >= targetAmount {
			selected = []TxOutput{output}
			totalAmount = output.Amount
			return selected, totalAmount, nil
		}
	}

	// If no single UTXO is sufficient, combine multiple UTXOs
	// Sort by amount descending for greedy selection
	for i := 0; i < len(addressUTXOs)-1; i++ {
		for j := i + 1; j < len(addressUTXOs); j++ {
			if addressUTXOs[i].Amount < addressUTXOs[j].Amount {
				addressUTXOs[i], addressUTXOs[j] = addressUTXOs[j], addressUTXOs[i]
			}
		}
	}

	// Select UTXOs until we reach the target amount
	for _, output := range addressUTXOs {
		selected = append(selected, output)
		totalAmount += output.Amount
		if totalAmount >= targetAmount {
			break
		}
	}

	if totalAmount < targetAmount {
		return nil, 0, fmt.Errorf("insufficient funds: have %.8f, need %.8f", totalAmount, targetAmount)
	}

	return selected, totalAmount, nil
}

// ValidateUTXO validates a single UTXO
func (us *UTXOSet) ValidateUTXO(txID string, index int) error {
	us.mu.RLock()
	defer us.mu.RUnlock()

	key := UTXOKey{TxID: txID, Index: index}

	output, exists := us.utxos[key]
	if !exists {
		return fmt.Errorf("UTXO not found: %s", key.String())
	}

	// Validate output
	if output.Amount <= 0 {
		return fmt.Errorf("invalid UTXO amount: %.8f", output.Amount)
	}

	if output.Address == "" {
		return fmt.Errorf("invalid UTXO address: empty")
	}

	return nil
}

// PruneSpent removes UTXOs that are referenced in the provided spent list
func (us *UTXOSet) PruneSpent(spentUTXOs []UTXOKey) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	for _, key := range spentUTXOs {
		output, exists := us.utxos[key]
		if !exists {
			continue // UTXO already spent or doesn't exist
		}

		delete(us.utxos, key)
		us.totalCount--
		us.totalValue -= output.Amount
	}

	return nil
}

// AddBatch adds multiple UTXOs in a single operation
func (us *UTXOSet) AddBatch(utxos map[string]map[int]TxOutput) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	for txID, outputs := range utxos {
		for index, output := range outputs {
			key := UTXOKey{TxID: txID, Index: index}

			// Check if UTXO already exists
			if _, exists := us.utxos[key]; exists {
				return fmt.Errorf("UTXO already exists: %s", key.String())
			}

			// Set transaction info on output
			output.TxID = txID
			output.Index = index

			// Add to set
			us.utxos[key] = output
			us.totalCount++
			us.totalValue += output.Amount
		}
	}

	return nil
}

// GetKeys returns all UTXO keys in the set
func (us *UTXOSet) GetKeys() []UTXOKey {
	us.mu.RLock()
	defer us.mu.RUnlock()

	keys := make([]UTXOKey, 0, len(us.utxos))
	for key := range us.utxos {
		keys = append(keys, key)
	}

	return keys
}

// HasSufficientBalance checks if an address has sufficient balance
func (us *UTXOSet) HasSufficientBalance(address string, amount float64) bool {
	us.mu.RLock()
	defer us.mu.RUnlock()

	balance := us.GetBalance(address)
	return balance >= amount
}

// GetUTXOsByRange returns UTXOs within the specified amount range
func (us *UTXOSet) GetUTXOsByRange(minAmount, maxAmount float64) []TxOutput {
	us.mu.RLock()
	defer us.mu.RUnlock()

	var outputs []TxOutput
	for _, output := range us.utxos {
		if output.Amount >= minAmount && output.Amount <= maxAmount {
			outputs = append(outputs, output)
		}
	}

	return outputs
}
