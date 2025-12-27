package transactions

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUTXOSetSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	assert.NotNil(t, utxoSet)
	assert.Equal(t, 0, utxoSet.GetCount())
	assert.Equal(t, 0.0, utxoSet.GetTotalValue())
}

func TestUTXOKeyStringSimple(t *testing.T) {
	key := UTXOKey{TxID: "tx123", Index: 0}
	assert.Equal(t, "tx123:0", key.String())
}

func TestAddSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	output := TxOutput{Address: "addr1", Amount: 1.5}

	err := utxoSet.Add("tx1", 0, output)
	assert.NoError(t, err)
	assert.Equal(t, 1, utxoSet.GetCount())
	assert.Equal(t, 1.5, utxoSet.GetTotalValue())

	// Test adding duplicate UTXO
	err = utxoSet.Add("tx1", 0, output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UTXO already exists")
}

func TestSpendSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	output := TxOutput{Address: "addr1", Amount: 1.5}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Spend the UTXO
	err = utxoSet.Spend("tx1", 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, utxoSet.GetCount())
	assert.Equal(t, 0.0, utxoSet.GetTotalValue())

	// Test spending non-existent UTXO
	err = utxoSet.Spend("tx1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UTXO not found")
}

func TestGetSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	output := TxOutput{Address: "addr1", Amount: 1.5}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Get existing UTXO
	retrieved, err := utxoSet.Get("tx1", 0)
	assert.NoError(t, err)
	assert.Equal(t, "addr1", retrieved.Address)
	assert.Equal(t, 1.5, retrieved.Amount)
	assert.Equal(t, "tx1", retrieved.TxID)
	assert.Equal(t, 0, retrieved.Index)

	// Get non-existent UTXO
	_, err = utxoSet.Get("tx2", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UTXO not found")
}

func TestExistsSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	output := TxOutput{Address: "addr1", Amount: 1.5}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Test existing UTXO
	assert.True(t, utxoSet.Exists("tx1", 0))

	// Test non-existent UTXO
	assert.False(t, utxoSet.Exists("tx2", 0))
}

func TestGetByAddressSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for different addresses
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
		{Address: "addr1", Amount: 1.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Get UTXOs for addr1
	addr1UTXOs := utxoSet.GetByAddress("addr1")
	assert.Len(t, addr1UTXOs, 2)

	// Get UTXOs for addr2
	addr2UTXOs := utxoSet.GetByAddress("addr2")
	assert.Len(t, addr2UTXOs, 1)

	// Get UTXOs for non-existent address
	emptyUTXOs := utxoSet.GetByAddress("addr3")
	assert.Len(t, emptyUTXOs, 0)
}

func TestValidateTransactionSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Create valid transaction
	tx := NewTransaction(
		[]TxInput{{TxID: "tx1", Index: 0}},
		[]TxOutput{{Address: "addr3", Amount: 0.9}},
	)

	err := utxoSet.ValidateTransaction(tx)
	assert.NoError(t, err)

	// Test transaction with non-existent UTXO
	invalidTx := NewTransaction(
		[]TxInput{{TxID: "tx2", Index: 0}},
		[]TxOutput{{Address: "addr3", Amount: 0.9}},
	)

	err = utxoSet.ValidateTransaction(invalidTx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent UTXO")

	// Test coinbase transaction (should always be valid)
	coinbaseTx := NewCoinbaseTransaction("addr1", 1.0)
	err = utxoSet.ValidateTransaction(coinbaseTx)
	assert.NoError(t, err)
}

func TestProcessTransactionSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add initial UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	initialCount := utxoSet.GetCount()
	initialValue := utxoSet.GetTotalValue()

	// Create and process transaction
	tx := NewTransaction(
		[]TxInput{{TxID: "tx1", Index: 0}},
		[]TxOutput{{Address: "addr3", Amount: 0.8}, {Address: "addr1", Amount: 0.1}},
	)

	tx.ID = "tx2" // Set transaction ID

	err := utxoSet.ProcessTransaction(tx)
	assert.NoError(t, err)

	// Check that inputs were spent and outputs added
	assert.Equal(t, initialCount+1, utxoSet.GetCount())        // 1 spent, 2 added = +1
	assert.Equal(t, initialValue-0.1, utxoSet.GetTotalValue()) // Fee of 0.1
}

func TestGetBalanceSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for different addresses
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
		{Address: "addr1", Amount: 1.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	assert.Equal(t, 2.5, utxoSet.GetBalance("addr1"))
	assert.Equal(t, 2.0, utxoSet.GetBalance("addr2"))
	assert.Equal(t, 0.0, utxoSet.GetBalance("addr3"))
}

func TestSelectForAmountSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for an address
	outputs := []TxOutput{
		{Address: "addr1", Amount: 0.5},
		{Address: "addr1", Amount: 1.0},
		{Address: "addr1", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Select UTXOs for 1.2 amount
	selected, total, err := utxoSet.SelectForAmount(1.2, "addr1")
	assert.NoError(t, err)
	assert.Len(t, selected, 2)  // Selects until target is reached
	assert.Equal(t, 1.5, total) // 0.5 + 1.0

	// Test insufficient funds
	_, _, err = utxoSet.SelectForAmount(5.0, "addr1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestClearSimple(t *testing.T) {
	utxoSet := NewUTXOSet()
	output := TxOutput{Address: "addr1", Amount: 1.0}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	assert.Equal(t, 1, utxoSet.GetCount())

	utxoSet.Clear()
	assert.Equal(t, 0, utxoSet.GetCount())
	assert.Equal(t, 0.0, utxoSet.GetTotalValue())
}

func TestCloneSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	clone := utxoSet.Clone()
	assert.Equal(t, utxoSet.GetCount(), clone.GetCount())
	assert.Equal(t, utxoSet.GetTotalValue(), clone.GetTotalValue())

	// Modify original and verify clone is unchanged
	utxoSet.Spend("tx1", 0)
	assert.Equal(t, 1, utxoSet.GetCount())
	assert.Equal(t, 2, clone.GetCount())
}

func TestGetStatsSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for different addresses
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
		{Address: "addr1", Amount: 1.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	stats := utxoSet.GetStats()
	assert.Equal(t, 3, stats["total_count"])
	assert.Equal(t, 4.5, stats["total_value"])
	assert.Equal(t, 2, stats["address_count"])
	assert.Equal(t, 1.5, stats["average_utxo_value"])
}

func TestFindUTXOsForAmountSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for an address
	outputs := []TxOutput{
		{Address: "addr1", Amount: 0.5},
		{Address: "addr1", Amount: 1.0},
		{Address: "addr1", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Find UTXOs for 1.2 amount
	selected, total, err := utxoSet.FindUTXOsForAmount(1.2, "addr1")
	assert.NoError(t, err)
	assert.Len(t, selected, 1) // Should select 2.0 (single sufficient UTXO)
	assert.Equal(t, 2.0, total)

	// Test insufficient funds
	_, _, err = utxoSet.FindUTXOsForAmount(5.0, "addr1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestValidateUTXOSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add a valid UTXO
	output := TxOutput{Address: "addr1", Amount: 1.0}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Validate existing UTXO
	err = utxoSet.ValidateUTXO("tx1", 0)
	assert.NoError(t, err)

	// Validate non-existent UTXO
	err = utxoSet.ValidateUTXO("tx2", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UTXO not found")
}

func TestAddBatchSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	utxos := map[string]map[int]TxOutput{
		"tx1": {
			0: {Address: "addr1", Amount: 1.0},
			1: {Address: "addr2", Amount: 2.0},
		},
		"tx2": {
			0: {Address: "addr3", Amount: 1.5},
		},
	}

	err := utxoSet.AddBatch(utxos)
	assert.NoError(t, err)
	assert.Equal(t, 3, utxoSet.GetCount())
	assert.Equal(t, 4.5, utxoSet.GetTotalValue())
}

func TestHasSufficientBalanceSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs for an address
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr1", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Test sufficient balance
	assert.True(t, utxoSet.HasSufficientBalance("addr1", 2.5))
	assert.True(t, utxoSet.HasSufficientBalance("addr1", 3.0))

	// Test insufficient balance
	assert.False(t, utxoSet.HasSufficientBalance("addr1", 3.1))

	// Test non-existent address
	assert.False(t, utxoSet.HasSufficientBalance("nonexistent", 1.0))
}

func TestDoubleSpendingPreventionSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add a UTXO
	output := TxOutput{Address: "addr1", Amount: 2.0}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Create transaction with double spend
	doubleSpendTx := NewTransaction(
		[]TxInput{
			{TxID: "tx1", Index: 0},
			{TxID: "tx1", Index: 0}, // Same UTXO twice
		},
		[]TxOutput{{Address: "addr2", Amount: 1.5}},
	)

	err = utxoSet.ValidateTransaction(doubleSpendTx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "double spend")
}

func TestUTXOSetUpdateSimple(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add initial UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Process transaction that spends one UTXO and creates two new ones
	tx := NewTransaction(
		[]TxInput{{TxID: "tx1", Index: 0}}, // Spend 1.0
		[]TxOutput{
			{Address: "addr3", Amount: 0.7},
			{Address: "addr1", Amount: 0.2}, // Change
		},
	)
	tx.ID = "tx2"

	err := utxoSet.ProcessTransaction(tx)
	assert.NoError(t, err)

	// Verify UTXO set was updated correctly
	assert.Equal(t, 3, utxoSet.GetCount())                  // 1 spent, 2 added
	assert.InDelta(t, 2.9, utxoSet.GetTotalValue(), 0.0001) // 3.0 - 0.1 fee

	// Verify old UTXO is gone
	assert.False(t, utxoSet.Exists("tx1", 0))
	assert.True(t, utxoSet.Exists("tx1", 1)) // Still exists

	// Verify new UTXOs exist
	assert.True(t, utxoSet.Exists("tx2", 0))
	assert.True(t, utxoSet.Exists("tx2", 1))
}

func TestUTXOEdgeCases(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test empty UTXO set operations
	_, err := utxoSet.Get("tx1", 0)
	assert.Error(t, err)

	err = utxoSet.Spend("tx1", 0)
	assert.Error(t, err)

	// Test zero and negative amounts
	zeroOutput := TxOutput{Address: "addr1", Amount: 0.0}
	err = utxoSet.Add("tx1", 0, zeroOutput)
	assert.NoError(t, err)

	err = utxoSet.ValidateUTXO("tx1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTXO amount")

	negativeOutput := TxOutput{Address: "addr1", Amount: -1.0}
	err = utxoSet.Add("tx2", 0, negativeOutput)
	assert.NoError(t, err)

	err = utxoSet.ValidateUTXO("tx2", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTXO amount")

	// Test empty address
	emptyAddrOutput := TxOutput{Address: "", Amount: 1.0}
	err = utxoSet.Add("tx3", 0, emptyAddrOutput)
	assert.NoError(t, err)

	err = utxoSet.ValidateUTXO("tx3", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTXO address")
}

func TestUTXOLargeValues(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test with very large amounts
	largeAmount := math.MaxFloat64 / 2
	output := TxOutput{Address: "addr1", Amount: largeAmount}

	err := utxoSet.Add("tx1", 0, output)
	assert.NoError(t, err)
	assert.Equal(t, largeAmount, utxoSet.GetTotalValue())
	assert.Equal(t, largeAmount, utxoSet.GetBalance("addr1"))
}

func TestUTXOMultipleTransactions(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add initial funding UTXO
	fundingOutput := TxOutput{Address: "addr1", Amount: 10.0}
	err := utxoSet.Add("funding", 0, fundingOutput)
	require.NoError(t, err)

	// First transaction: split funding into multiple UTXOs
	tx1 := NewTransaction(
		[]TxInput{{TxID: "funding", Index: 0}},
		[]TxOutput{
			{Address: "addr1", Amount: 3.0},
			{Address: "addr2", Amount: 4.0},
			{Address: "addr3", Amount: 2.5},
		},
	)
	tx1.ID = "tx1"

	err = utxoSet.ProcessTransaction(tx1)
	require.NoError(t, err)

	// Second transaction: spend from addr2 and addr3
	tx2 := NewTransaction(
		[]TxInput{
			{TxID: "tx1", Index: 1}, // 4.0 from addr2
			{TxID: "tx1", Index: 2}, // 2.5 from addr3
		},
		[]TxOutput{
			{Address: "addr4", Amount: 5.0},
			{Address: "addr2", Amount: 1.0}, // Change
		},
	)
	tx2.ID = "tx2"

	err = utxoSet.ProcessTransaction(tx2)
	require.NoError(t, err)

	// Verify final state
	assert.Equal(t, 3, utxoSet.GetCount())
	assert.Equal(t, 9.0, utxoSet.GetTotalValue()) // 10.0 - 1.0 total fees

	// Verify balances
	assert.Equal(t, 3.0, utxoSet.GetBalance("addr1"))
	assert.Equal(t, 1.0, utxoSet.GetBalance("addr2"))
	assert.Equal(t, 0.0, utxoSet.GetBalance("addr3"))
	assert.Equal(t, 5.0, utxoSet.GetBalance("addr4"))
}

func TestUTXOPruning(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
		{Address: "addr3", Amount: 1.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Create list of spent UTXOs
	spentUTXOs := []UTXOKey{
		{TxID: "tx1", Index: 0},
		{TxID: "tx1", Index: 2},
	}

	err := utxoSet.PruneSpent(spentUTXOs)
	assert.NoError(t, err)
	assert.Equal(t, 1, utxoSet.GetCount())
	assert.Equal(t, 2.0, utxoSet.GetTotalValue())

	// Verify remaining UTXO
	exists := utxoSet.Exists("tx1", 1)
	assert.True(t, exists)
}

func TestUTXOGetByAmount(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs with different amounts
	outputs := []TxOutput{
		{Address: "addr1", Amount: 0.5},
		{Address: "addr2", Amount: 2.0},
		{Address: "addr1", Amount: 1.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Get UTXOs with amount >= 1.0
	highValueUTXOs := utxoSet.GetByAmount(1.0)
	assert.Len(t, highValueUTXOs, 2)

	// Get UTXOs with amount >= 2.0
	veryHighValueUTXOs := utxoSet.GetByAmount(2.0)
	assert.Len(t, veryHighValueUTXOs, 1)

	// Get UTXOs with amount > 2.0
	noUTXOs := utxoSet.GetByAmount(2.1)
	assert.Len(t, noUTXOs, 0)
}

func TestUTXOGetKeys(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	keys := utxoSet.GetKeys()
	assert.Len(t, keys, 2)

	// Verify keys
	keySet := make(map[UTXOKey]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	assert.True(t, keySet[UTXOKey{TxID: "tx1", Index: 0}])
	assert.True(t, keySet[UTXOKey{TxID: "tx1", Index: 1}])
}

func TestUTXOGetByRange(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs with different amounts
	outputs := []TxOutput{
		{Address: "addr1", Amount: 0.5},
		{Address: "addr2", Amount: 1.5},
		{Address: "addr3", Amount: 2.5},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Get UTXOs in range [1.0, 2.0]
	rangeUTXOs := utxoSet.GetUTXOsByRange(1.0, 2.0)
	assert.Len(t, rangeUTXOs, 1)
	assert.Equal(t, 1.5, rangeUTXOs[0].Amount)

	// Get UTXOs in range [0.0, 1.0]
	rangeUTXOs = utxoSet.GetUTXOsByRange(0.0, 1.0)
	assert.Len(t, rangeUTXOs, 1)
	assert.Equal(t, 0.5, rangeUTXOs[0].Amount)

	// Get UTXOs in range [3.0, 4.0] (empty)
	rangeUTXOs = utxoSet.GetUTXOsByRange(3.0, 4.0)
	assert.Len(t, rangeUTXOs, 0)
}

func TestUTXOComprehensiveOperations(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test GetAll on empty set
	all := utxoSet.GetAll()
	assert.Len(t, all, 0)

	// Add UTXOs and test GetAll
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr2", Amount: 2.0},
	}
	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	all = utxoSet.GetAll()
	assert.Len(t, all, 2)

	// Test GetCount and GetTotalValue
	assert.Equal(t, 2, utxoSet.GetCount())
	assert.Equal(t, 3.0, utxoSet.GetTotalValue())

	// Test Clear
	utxoSet.Clear()
	assert.Equal(t, 0, utxoSet.GetCount())
	assert.Equal(t, 0.0, utxoSet.GetTotalValue())
}

func TestUTXOValidationEdgeCases(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXO with invalid data for validation
	invalidOutput := TxOutput{Address: "addr1", Amount: -1.0}
	err := utxoSet.Add("tx1", 0, invalidOutput)
	require.NoError(t, err)

	// Validate should fail for negative amount
	err = utxoSet.ValidateUTXO("tx1", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTXO amount")

	// Add UTXO with empty address
	emptyAddrOutput := TxOutput{Address: "", Amount: 1.0}
	err = utxoSet.Add("tx2", 0, emptyAddrOutput)
	require.NoError(t, err)

	// Validate should fail for empty address
	err = utxoSet.ValidateUTXO("tx2", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTXO address")
}

func TestUTXOTransactionValidationEdgeCases(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	output := TxOutput{Address: "addr1", Amount: 2.0}
	err := utxoSet.Add("tx1", 0, output)
	require.NoError(t, err)

	// Test transaction with invalid input index
	invalidTx := NewTransaction(
		[]TxInput{{TxID: "tx1", Index: 5}}, // Non-existent index
		[]TxOutput{{Address: "addr2", Amount: 1.0}},
	)
	err = utxoSet.ValidateTransaction(invalidTx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent UTXO")

	// Test transaction with empty inputs (non-coinbase)
	emptyInputTx := NewTransaction(
		[]TxInput{},
		[]TxOutput{{Address: "addr2", Amount: 1.0}},
	)
	err = utxoSet.ValidateTransaction(emptyInputTx)
	assert.NoError(t, err) // Empty inputs are technically valid
}

func TestUTXOProcessingEdgeCases(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test processing coinbase transaction
	coinbaseTx := NewCoinbaseTransaction("addr1", 1.0)
	coinbaseTx.ID = "coinbase1"

	err := utxoSet.ProcessTransaction(coinbaseTx)
	assert.NoError(t, err)
	assert.Equal(t, 1, utxoSet.GetCount())
	assert.True(t, utxoSet.Exists("coinbase1", 0))

	// Test processing transaction that fails validation
	invalidTx := NewTransaction(
		[]TxInput{{TxID: "nonexistent", Index: 0}},
		[]TxOutput{{Address: "addr2", Amount: 1.0}},
	)
	invalidTx.ID = "invalid1"

	err = utxoSet.ProcessTransaction(invalidTx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction validation failed")
}

func TestUTXOSelectionStrategies(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add UTXOs with varying amounts
	outputs := []TxOutput{
		{Address: "addr1", Amount: 0.1},
		{Address: "addr1", Amount: 0.5},
		{Address: "addr1", Amount: 1.0},
		{Address: "addr1", Amount: 2.0},
		{Address: "addr1", Amount: 5.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	// Test exact match selection
	selected, total, err := utxoSet.FindUTXOsForAmount(1.0, "addr1")
	assert.NoError(t, err)
	assert.Len(t, selected, 1)
	assert.Equal(t, 1.0, total)

	// Test selection requiring multiple UTXOs
	selected, total, err = utxoSet.FindUTXOsForAmount(6.0, "addr1")
	assert.NoError(t, err)
	assert.Greater(t, len(selected), 1)
	assert.GreaterOrEqual(t, total, 6.0)

	// Test selection with insufficient funds
	_, _, err = utxoSet.FindUTXOsForAmount(10.0, "addr1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
}

func TestUTXOStatisticsAndAnalytics(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test stats on empty set
	stats := utxoSet.GetStats()
	assert.Equal(t, 0, stats["total_count"])
	assert.Equal(t, 0.0, stats["total_value"])
	assert.Equal(t, 0, stats["address_count"])
	assert.Equal(t, 0.0, stats["average_utxo_value"])

	// Add UTXOs for multiple addresses
	outputs := []TxOutput{
		{Address: "addr1", Amount: 1.0},
		{Address: "addr1", Amount: 2.0},
		{Address: "addr2", Amount: 3.0},
		{Address: "addr3", Amount: 4.0},
	}

	for i, output := range outputs {
		err := utxoSet.Add("tx1", i, output)
		require.NoError(t, err)
	}

	stats = utxoSet.GetStats()
	assert.Equal(t, 4, stats["total_count"])
	assert.Equal(t, 10.0, stats["total_value"])
	assert.Equal(t, 3, stats["address_count"])
	assert.Equal(t, 2.5, stats["average_utxo_value"])

	// Verify address-specific stats
	addressCounts := stats["address_counts"].(map[string]int)
	assert.Equal(t, 2, addressCounts["addr1"])
	assert.Equal(t, 1, addressCounts["addr2"])
	assert.Equal(t, 1, addressCounts["addr3"])

	addressValues := stats["address_values"].(map[string]float64)
	assert.Equal(t, 3.0, addressValues["addr1"])
	assert.Equal(t, 3.0, addressValues["addr2"])
	assert.Equal(t, 4.0, addressValues["addr3"])
}

func TestUTXOBatchOperations(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test AddBatch with multiple transactions
	batchUTXOs := map[string]map[int]TxOutput{
		"tx1": {
			0: {Address: "addr1", Amount: 1.0},
			1: {Address: "addr2", Amount: 2.0},
		},
		"tx2": {
			0: {Address: "addr3", Amount: 3.0},
			1: {Address: "addr1", Amount: 1.5},
		},
		"tx3": {
			0: {Address: "addr2", Amount: 2.5},
		},
	}

	err := utxoSet.AddBatch(batchUTXOs)
	assert.NoError(t, err)
	assert.Equal(t, 5, utxoSet.GetCount())
	assert.Equal(t, 10.0, utxoSet.GetTotalValue())

	// Test AddBatch with duplicate UTXOs
	err = utxoSet.AddBatch(batchUTXOs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UTXO already exists")

	// Test PruneSpent with multiple UTXOs
	spentUTXOs := []UTXOKey{
		{TxID: "tx1", Index: 0},
		{TxID: "tx2", Index: 0},
		{TxID: "tx3", Index: 0},
	}

	err = utxoSet.PruneSpent(spentUTXOs)
	assert.NoError(t, err)
	assert.Equal(t, 2, utxoSet.GetCount())
	assert.Equal(t, 3.5, utxoSet.GetTotalValue())
}

func TestUTXOComplexTransactionScenarios(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Create initial funding
	fundingUTXOs := map[string]map[int]TxOutput{
		"funding": {
			0: {Address: "alice", Amount: 10.0},
			1: {Address: "bob", Amount: 5.0},
			2: {Address: "charlie", Amount: 3.0},
		},
	}

	err := utxoSet.AddBatch(fundingUTXOs)
	require.NoError(t, err)

	// Complex transaction: Alice pays Bob and Charlie
	tx1 := NewTransaction(
		[]TxInput{{TxID: "funding", Index: 0}}, // Alice spends 10.0
		[]TxOutput{
			{Address: "bob", Amount: 6.0},
			{Address: "charlie", Amount: 2.0},
			{Address: "alice", Amount: 1.5}, // Change
		},
	)
	tx1.ID = "tx1"

	err = utxoSet.ProcessTransaction(tx1)
	assert.NoError(t, err)

	// Verify state
	assert.Equal(t, 5, utxoSet.GetCount())         // 3 spent, 3 added = +0
	assert.Equal(t, 17.5, utxoSet.GetTotalValue()) // 18.0 - 0.5 fee

	// Another transaction: Bob pays Dave
	tx2 := NewTransaction(
		[]TxInput{{TxID: "funding", Index: 1}}, // Bob spends 5.0
		[]TxOutput{
			{Address: "dave", Amount: 4.0},
			{Address: "bob", Amount: 0.8}, // Change
		},
	)
	tx2.ID = "tx2"

	err = utxoSet.ProcessTransaction(tx2)
	assert.NoError(t, err)

	// Verify final balances
	assert.Equal(t, 1.5, utxoSet.GetBalance("alice"))
	assert.Equal(t, 6.8, utxoSet.GetBalance("bob"))     // 1.0 + 5.0 + 0.8
	assert.Equal(t, 5.0, utxoSet.GetBalance("charlie")) // 2.0 + 3.0
	assert.Equal(t, 4.0, utxoSet.GetBalance("dave"))
}

func TestUTXOErrorHandlingAndRecovery(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test various error conditions
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "spend non-existent UTXO",
			testFunc: func(t *testing.T) {
				err := utxoSet.Spend("nonexistent", 0)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "UTXO not found")
			},
		},
		{
			name: "get non-existent UTXO",
			testFunc: func(t *testing.T) {
				_, err := utxoSet.Get("nonexistent", 0)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "UTXO not found")
			},
		},
		{
			name: "validate non-existent UTXO",
			testFunc: func(t *testing.T) {
				err := utxoSet.ValidateUTXO("nonexistent", 0)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "UTXO not found")
			},
		},
		{
			name: "select from non-existent address",
			testFunc: func(t *testing.T) {
				_, _, err := utxoSet.SelectForAmount(1.0, "nonexistent")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no UTXOs found for address")
			},
		},
		{
			name: "find for non-existent address",
			testFunc: func(t *testing.T) {
				_, _, err := utxoSet.FindUTXOsForAmount(1.0, "nonexistent")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no UTXOs found")
			},
		},
		{
			name: "invalid target amount",
			testFunc: func(t *testing.T) {
				_, _, err := utxoSet.SelectForAmount(-1.0, "addr1")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "target amount must be positive")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestUTXOMemoryEfficiency(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Test with large number of UTXOs
	const numUTXOs = 1000
	for i := 0; i < numUTXOs; i++ {
		output := TxOutput{
			Address: fmt.Sprintf("addr%d", i%10),
			Amount:  float64(i) + 0.5,
		}
		err := utxoSet.Add(fmt.Sprintf("tx%d", i), 0, output)
		require.NoError(t, err)
	}

	assert.Equal(t, numUTXOs, utxoSet.GetCount())

	// Test efficient lookups
	for i := 0; i < 100; i++ {
		exists := utxoSet.Exists(fmt.Sprintf("tx%d", i), 0)
		assert.True(t, exists)
	}

	// Test balance calculations
	totalBalance := 0.0
	for i := 0; i < 10; i++ {
		balance := utxoSet.GetBalance(fmt.Sprintf("addr%d", i))
		totalBalance += balance
		assert.Greater(t, balance, 0.0)
	}

	assert.Equal(t, utxoSet.GetTotalValue(), totalBalance)

	// Test efficient cloning
	clone := utxoSet.Clone()
	assert.Equal(t, utxoSet.GetCount(), clone.GetCount())
	assert.Equal(t, utxoSet.GetTotalValue(), clone.GetTotalValue())

	// Modify original and verify clone is unchanged
	utxoSet.Spend("tx0", 0)
	assert.Equal(t, numUTXOs-1, utxoSet.GetCount())
	assert.Equal(t, numUTXOs, clone.GetCount())
}
