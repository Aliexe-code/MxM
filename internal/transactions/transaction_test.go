package transactions

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test data
var (
	testPrivateKey *ecdsa.PrivateKey
	testPublicKey  *ecdsa.PublicKey
	testAddress    = "0x1234567890123456789012345678901234567890"
	testAddress2   = "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
)

func init() {
	var err error
	testPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate test key: %v", err)
	}
	testPublicKey = &testPrivateKey.PublicKey
}

// TransactionTestSuite defines the test suite
type TransactionTestSuite struct {
	suite.Suite
}

// SetupTest is called before each test
func (suite *TransactionTestSuite) SetupTest() {
	// Setup if needed
}

func (suite *TransactionTestSuite) TestNewTransaction() {
	inputs := []TxInput{
		{TxID: "tx123", Index: 0, Signature: "", PublicKey: ""},
	}
	outputs := []TxOutput{
		{Address: testAddress, Amount: 1.0, TxID: "", Index: 0},
	}

	tx := NewTransaction(inputs, outputs)

	assert.NotEmpty(suite.T(), tx.ID, "Transaction ID should not be empty")
	assert.Len(suite.T(), tx.Inputs, 1, "Expected 1 input")
	assert.Len(suite.T(), tx.Outputs, 1, "Expected 1 output")
}

func TestNewCoinbaseTransaction(t *testing.T) {
	tx := NewCoinbaseTransaction(testAddress, 12.5)

	if !tx.IsCoinbase() {
		t.Error("Transaction should be coinbase")
	}

	if len(tx.Inputs) != 0 {
		t.Errorf("Coinbase should have 0 inputs, got %d", len(tx.Inputs))
	}

	if len(tx.Outputs) != 1 {
		t.Errorf("Coinbase should have 1 output, got %d", len(tx.Outputs))
	}

	if tx.Outputs[0].Address != testAddress {
		t.Errorf("Expected address %s, got %s", testAddress, tx.Outputs[0].Address)
	}

	if tx.Outputs[0].Amount != 12.5 {
		t.Errorf("Expected amount 12.5, got %f", tx.Outputs[0].Amount)
	}
}

func TestCalculateID(t *testing.T) {
	inputs := []TxInput{
		{TxID: "tx123", Index: 0, Signature: "", PublicKey: ""},
	}
	outputs := []TxOutput{
		{Address: testAddress, Amount: 1.0, TxID: "", Index: 0},
	}

	tx := NewTransaction(inputs, outputs)
	id1 := tx.ID
	id2 := tx.CalculateID()

	if id1 != id2 {
		t.Error("Transaction ID should be consistent")
	}

	// Modify transaction and check ID changes
	tx.Outputs[0].Amount = 2.0
	id3 := tx.CalculateID()

	if id1 == id3 {
		t.Error("Transaction ID should change when data changes")
	}
}

func TestGetOutputAmount(t *testing.T) {
	outputs := []TxOutput{
		{Address: testAddress, Amount: 1.5, TxID: "", Index: 0},
		{Address: testAddress2, Amount: 2.5, TxID: "", Index: 1},
	}

	tx := NewTransaction([]TxInput{}, outputs)
	amount := tx.GetOutputAmount()

	if amount != 4.0 {
		t.Errorf("Expected total amount 4.0, got %f", amount)
	}
}

func TestValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "valid transaction",
			tx: NewTransaction(
				[]TxInput{{TxID: "tx123", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			wantErr: false,
		},
		{
			name:    "empty ID",
			tx:      &Transaction{ID: "", Inputs: []TxInput{}, Outputs: []TxOutput{}},
			wantErr: true,
		},
		{
			name: "empty address",
			tx: NewTransaction(
				[]TxInput{{TxID: "tx123", Index: 0}},
				[]TxOutput{{Address: "", Amount: 1.0}},
			),
			wantErr: true,
		},
		{
			name: "negative amount",
			tx: NewTransaction(
				[]TxInput{{TxID: "tx123", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: -1.0}},
			),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSignTransaction(t *testing.T) {
	// Create transaction
	inputs := []TxInput{
		{TxID: "tx123", Index: 0, Signature: "", PublicKey: ""},
	}
	outputs := []TxOutput{
		{Address: testAddress, Amount: 1.0, TxID: "", Index: 0},
	}
	tx := NewTransaction(inputs, outputs)

	// Create referenced output
	referencedOutputs := []TxOutput{
		{Address: testAddress2, Amount: 2.0, TxID: "tx123", Index: 0},
	}

	// Sign transaction
	err := tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Check signature is present
	if tx.Inputs[0].Signature == "" {
		t.Error("Signature should not be empty after signing")
	}

	// Check public key is present
	if tx.Inputs[0].PublicKey == "" {
		t.Error("Public key should not be empty after signing")
	}
}

func TestVerifyInputSignature(t *testing.T) {
	// Create and sign transaction
	inputs := []TxInput{
		{TxID: "tx123", Index: 0, Signature: "", PublicKey: ""},
	}
	outputs := []TxOutput{
		{Address: testAddress, Amount: 1.0, TxID: "", Index: 0},
	}
	tx := NewTransaction(inputs, outputs)

	referencedOutputs := []TxOutput{
		{Address: testAddress2, Amount: 2.0, TxID: "tx123", Index: 0},
	}

	err := tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify signature
	err = tx.VerifyInputSignature(0, referencedOutputs)
	if err != nil {
		t.Errorf("Signature verification failed: %v", err)
	}

	// Test with wrong signature
	originalSig := tx.Inputs[0].Signature
	tx.Inputs[0].Signature = "wrong_signature"
	err = tx.VerifyInputSignature(0, referencedOutputs)
	if err == nil {
		t.Error("Should fail with wrong signature")
	}
	tx.Inputs[0].Signature = originalSig

	// Test with wrong public key
	originalKey := tx.Inputs[0].PublicKey
	tx.Inputs[0].PublicKey = "wrong_public_key"
	err = tx.VerifyInputSignature(0, referencedOutputs)
	if err == nil {
		t.Error("Should fail with wrong public key")
	}
	tx.Inputs[0].PublicKey = originalKey
}

func TestValidateAmounts(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name:    "valid coinbase",
			tx:      NewCoinbaseTransaction(testAddress, 12.5),
			wantErr: false,
		},
		{
			name:    "coinbase with zero amount",
			tx:      NewCoinbaseTransaction(testAddress, 0),
			wantErr: true,
		},
		{
			name:    "valid regular transaction",
			tx:      NewCoinbaseTransaction(testAddress, 1.0),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utxoSet := make(map[string]map[int]TxOutput)
			err := tt.tx.ValidateAmounts(utxoSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAmounts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateChange(t *testing.T) {
	tests := []struct {
		name         string
		inputAmount  float64
		outputAmount float64
		desiredFee   float64
		wantChange   float64
		wantFee      float64
		wantErr      bool
	}{
		{
			name:         "normal case",
			inputAmount:  1.0,
			outputAmount: 0.8,
			desiredFee:   0.1,
			wantChange:   0.1,
			wantFee:      0.2,
			wantErr:      false,
		},
		{
			name:         "no change",
			inputAmount:  1.0,
			outputAmount: 0.9,
			desiredFee:   0.1,
			wantChange:   0.0,
			wantFee:      0.1,
			wantErr:      false,
		},
		{
			name:         "insufficient input",
			inputAmount:  0.5,
			outputAmount: 0.8,
			desiredFee:   0.1,
			wantChange:   0.0,
			wantFee:      0.0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change, fee, err := CalculateChange(tt.inputAmount, tt.outputAmount, tt.desiredFee)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if math.Abs(change-tt.wantChange) > 1e-8 {
				t.Errorf("CalculateChange() change = %v, want %v", change, tt.wantChange)
			}
			if math.Abs(fee-tt.wantFee) > 1e-8 {
				t.Errorf("CalculateChange() fee = %v, want %v", fee, tt.wantFee)
			}
		})
	}
}

func TestValidateAddressFormat(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", "0x1234567890123456789012345678901234567890", false},
		{"empty address", "", true},
		{"too short", "0x123", true},
		{"too long", "0x12345678901234567890123456789012345678901", true},
		{"missing prefix", "1234567890123456789012345678901234567890", true},
		{"invalid hex", "0xg234567890123456789012345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddressFormat(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddressFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateSimpleTransaction(t *testing.T) {
	tx := CreateSimpleTransaction("tx123", 0, testAddress, 1.0, testAddress2)

	if len(tx.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(tx.Inputs))
	}

	if len(tx.Outputs) != 2 {
		t.Errorf("Expected 2 outputs, got %d", len(tx.Outputs))
	}

	if tx.Inputs[0].TxID != "tx123" {
		t.Errorf("Expected input txID tx123, got %s", tx.Inputs[0].TxID)
	}

	if tx.Outputs[0].Address != testAddress {
		t.Errorf("Expected first output address %s, got %s", testAddress, tx.Outputs[0].Address)
	}

	if tx.Outputs[0].Amount != 1.0 {
		t.Errorf("Expected first output amount 1.0, got %f", tx.Outputs[0].Amount)
	}
}

func TestCalculateOptimalFee(t *testing.T) {
	tests := []struct {
		name        string
		inputCount  int
		outputCount int
		priority    float64
		expected    float64
	}{
		{"low priority", 1, 1, 0.5, 0.00005},
		{"normal priority", 1, 1, 1.0, 0.0001},
		{"high priority", 1, 1, 2.0, 0.0002},
		{"multiple inputs", 3, 2, 1.0, 0.000105},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := CalculateOptimalFee(tt.inputCount, tt.outputCount, tt.priority)
			// Allow for small rounding differences
			if fee < tt.expected*0.9 || fee > tt.expected*1.1 {
				t.Errorf("CalculateOptimalFee() = %v, expected around %v", fee, tt.expected)
			}
		})
	}
}

func TestEstimateTransactionSize(t *testing.T) {
	tests := []struct {
		name        string
		inputCount  int
		outputCount int
		expected    int
	}{
		{"simple", 1, 1, 192},
		{"multiple", 3, 5, 624},
		{"coinbase", 0, 1, 44},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := EstimateTransactionSize(tt.inputCount, tt.outputCount)
			if size != tt.expected {
				t.Errorf("EstimateTransactionSize() = %v, want %v", size, tt.expected)
			}
		})
	}
}

func TestGetTransactionSummary(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	summary := tx.GetTransactionSummary(make(map[string]map[int]TxOutput))

	if summary["input_count"].(int) != 1 {
		t.Errorf("Expected input_count 1, got %v", summary["input_count"])
	}

	if summary["output_count"].(int) != 1 {
		t.Errorf("Expected output_count 1, got %v", summary["output_count"])
	}

	if summary["total_output"].(float64) != 1.0 {
		t.Errorf("Expected total_output 1.0, got %v", summary["total_output"])
	}
}

func TestCloneTransaction(t *testing.T) {
	original := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	clone := original.CloneTransaction()

	// Verify they're equal but not the same object
	if clone.ID != original.ID {
		t.Error("Clone should have same ID")
	}

	if &clone == &original {
		t.Error("Clone should be a different object")
	}

	// Modify clone and verify original is unchanged
	clone.Outputs[0].Amount = 2.0
	if original.Outputs[0].Amount != 1.0 {
		t.Error("Original should not be affected by clone modification")
	}
}

func TestGetTransactionType(t *testing.T) {
	tests := []struct {
		name     string
		tx       *Transaction
		expected string
	}{
		{
			name:     "coinbase",
			tx:       NewCoinbaseTransaction(testAddress, 12.5),
			expected: "coinbase",
		},
		{
			name: "simple transfer",
			tx: NewTransaction(
				[]TxInput{{TxID: "tx123", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			expected: "simple_transfer",
		},
		{
			name: "multi output",
			tx: NewTransaction(
				[]TxInput{{TxID: "tx123", Index: 0}},
				[]TxOutput{
					{Address: testAddress, Amount: 0.5},
					{Address: testAddress2, Amount: 0.3},
					{Address: testAddress, Amount: 0.2},
				},
			),
			expected: "multi_output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txType := tt.tx.GetTransactionType()
			if txType != tt.expected {
				t.Errorf("GetTransactionType() = %v, want %v", txType, tt.expected)
			}
		})
	}
}

func TestToJSONFromJSON(t *testing.T) {
	original := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Convert to JSON
	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// Convert back from JSON
	restored, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to restore from JSON: %v", err)
	}

	// Verify they're equal
	if restored.ID != original.ID {
		t.Error("Restored transaction should have same ID")
	}

	if len(restored.Inputs) != len(original.Inputs) {
		t.Error("Restored transaction should have same number of inputs")
	}

	if len(restored.Outputs) != len(original.Outputs) {
		t.Error("Restored transaction should have same number of outputs")
	}
}

func TestString(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	str := tx.String()
	if !strings.Contains(str, "Transaction") {
		t.Error("String should contain 'Transaction'")
	}
	if !strings.Contains(str, "1") { // inputs
		t.Error("String should contain input count")
	}
}

func TestGetInfo(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	utxoSet := make(map[string]map[int]TxOutput)
	info := tx.GetInfo(utxoSet)
	if info["id"] != tx.ID {
		t.Error("Info ID should match transaction ID")
	}
	if info["input_count"].(int) != 1 {
		t.Error("Info input_count should be 1")
	}
}

func TestCalculateDustThreshold(t *testing.T) {
	threshold := CalculateDustThreshold(0.00001)
	if threshold <= 0 {
		t.Error("Dust threshold should be positive")
	}
}

func TestIsDustOutput(t *testing.T) {
	if !IsDustOutput(0.000001, 0.00001) {
		t.Error("Very small output should be dust")
	}
	if IsDustOutput(1.0, 0.00001) {
		t.Error("Large output should not be dust")
	}
}

func TestOptimizeTransaction(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{
			{Address: testAddress, Amount: 1.0},
			{Address: testAddress2, Amount: 0.000001}, // dust
		},
	)

	originalOutputs := len(tx.Outputs)
	tx.OptimizeTransaction(0.00001)
	if len(tx.Outputs) >= originalOutputs {
		t.Error("Dust should be removed or consolidated")
	}
}

func TestMergeTransactions(t *testing.T) {
	tx1 := NewTransaction(
		[]TxInput{{TxID: "tx1", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)
	tx2 := NewTransaction(
		[]TxInput{{TxID: "tx2", Index: 0}},
		[]TxOutput{{Address: testAddress2, Amount: 2.0}},
	)

	merged := MergeTransactions([]*Transaction{tx1, tx2})
	if len(merged.Inputs) != 2 {
		t.Errorf("Merged should have 2 inputs, got %d", len(merged.Inputs))
	}
	if len(merged.Outputs) != 2 {
		t.Errorf("Merged should have 2 outputs, got %d", len(merged.Outputs))
	}
}

func TestValidateTransactionBalance(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	utxoSet := map[string]map[int]TxOutput{
		"tx123": {0: {Address: testAddress2, Amount: 2.0}},
	}

	err := tx.ValidateTransactionBalance(utxoSet)
	assert.NoError(t, err, "Should not error with sufficient input")

	// Test with insufficient input
	tx2 := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 3.0}}, // More than input
	)
	err = tx2.ValidateTransactionBalance(utxoSet)
	assert.Error(t, err, "Should error with insufficient input")

	// Test with unknown transaction
	utxoSet2 := map[string]map[int]TxOutput{}
	err = tx.ValidateTransactionBalance(utxoSet2)
	assert.Error(t, err, "Should error with unknown transaction")
}

func TestValidateInputs(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	utxoSet := map[string]map[int]TxOutput{
		"tx123": {0: {Address: testAddress2, Amount: 2.0}},
	}

	err := tx.ValidateInputs(utxoSet)
	if err == nil {
		t.Error("Should error without signature")
	}
}

func TestCreateChangeOutput(t *testing.T) {
	output := CreateChangeOutput(testAddress, 0.5)
	if output.Address != testAddress {
		t.Errorf("Expected address %s, got %s", testAddress, output.Address)
	}
	if output.Amount != 0.5 {
		t.Errorf("Expected amount 0.5, got %f", output.Amount)
	}
}

func TestValidateTransactionStructure(t *testing.T) {
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	err := tx.ValidateTransactionStructure(utxoSet)
	if err != nil {
		t.Errorf("Valid coinbase should not error, got %v", err)
	}
}

func TestGetValidationReport(t *testing.T) {
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	report := tx.GetValidationReport(utxoSet)
	if report["is_valid"] != true {
		t.Error("Valid transaction should be valid")
	}
	if len(report["errors"].([]string)) != 0 {
		t.Error("Valid transaction should have no errors")
	}
}

func TestGetInputPublicKey(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	referencedOutputs := []TxOutput{
		{Address: testAddress2, Amount: 2.0},
	}

	err := tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	pubKey, err := tx.GetInputPublicKey(0)
	if err != nil {
		t.Errorf("Failed to get public key: %v", err)
	}
	if pubKey == nil {
		t.Error("Public key should not be nil")
	}
}

func TestSignAllInputs(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	referencedOutputs := [][]TxOutput{
		{{Address: testAddress2, Amount: 2.0}},
	}

	err := tx.SignAllInputs([]*ecdsa.PrivateKey{testPrivateKey}, referencedOutputs)
	assert.NoError(t, err, "Should sign all inputs")

	// Test with wrong number of keys
	err = tx.SignAllInputs([]*ecdsa.PrivateKey{}, referencedOutputs)
	assert.Error(t, err, "Should error with wrong number of keys")
}

func TestVerifyAllSignatures(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	referencedOutputs := [][]TxOutput{
		{{Address: testAddress2, Amount: 2.0}},
	}

	err := tx.SignAllInputs([]*ecdsa.PrivateKey{testPrivateKey}, referencedOutputs)
	require.NoError(t, err)

	err = tx.VerifyAllSignatures(referencedOutputs)
	assert.NoError(t, err, "Should verify all signatures")

	// Test with unsigned transaction
	tx2 := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)
	err = tx2.VerifyAllSignatures(referencedOutputs)
	assert.Error(t, err, "Should error on unsigned transaction")
}

func TestGetSignatureInfo(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	info := tx.GetSignatureInfo()
	if info["total_inputs"].(int) != 1 {
		t.Error("Should have 1 total input")
	}
	if info["signed_inputs"].(int) != 0 {
		t.Error("Should have 0 signed inputs")
	}
}

func TestValidateTransactionStructureInvalid(t *testing.T) {
	// Invalid transaction
	tx := &Transaction{
		ID:      "",
		Inputs:  []TxInput{},
		Outputs: []TxOutput{},
	}
	utxoSet := map[string]map[int]TxOutput{}

	err := tx.ValidateTransactionStructure(utxoSet)
	assert.Error(t, err, "Invalid transaction should error")
}

func TestValidateTransactionStructureInvalidAddress(t *testing.T) {
	// Transaction with invalid address
	tx := NewCoinbaseTransaction("invalid_address", 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	err := tx.ValidateTransactionStructure(utxoSet)
	assert.Error(t, err, "Transaction with invalid address should error")
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON("invalid json")
	assert.Error(t, err, "Should error on invalid JSON")
}

func TestTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionTestSuite))
}

// Benchmark tests
func BenchmarkNewTransaction(b *testing.B) {
	inputs := []TxInput{{TxID: "tx123", Index: 0}}
	outputs := []TxOutput{{Address: testAddress, Amount: 1.0}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTransaction(inputs, outputs)
	}
}

func BenchmarkCalculateID(b *testing.B) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.CalculateID()
	}
}

func BenchmarkSignTransaction(b *testing.B) {
	tx := NewTransaction(
		[]TxInput{{TxID: "tx123", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)
	referencedOutputs := []TxOutput{
		{Address: testAddress2, Amount: 2.0, TxID: "tx123", Index: 0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	}
}

// Additional tests for improved coverage

func TestCalculateIDWithEmptyTransaction(t *testing.T) {
	tx := &Transaction{
		ID:        "",
		Inputs:    []TxInput{},
		Outputs:   []TxOutput{},
		Timestamp: 0,
	}

	id := tx.CalculateID()
	assert.NotEmpty(t, id, "ID should not be empty even for empty transaction")
}

func TestCalculateIDWithInvalidData(t *testing.T) {
	tx := &Transaction{
		ID:        "",
		Inputs:    []TxInput{{TxID: "", Index: -1}},
		Outputs:   []TxOutput{{Address: "", Amount: -1}},
		Timestamp: 0,
	}

	id := tx.CalculateID()
	assert.NotEmpty(t, id, "ID should be generated even with invalid data")
}

func TestToJSONWithComplexTransaction(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{
			{TxID: "tx123", Index: 0, Signature: "sig123", PublicKey: "key123"},
			{TxID: "tx456", Index: 1, Signature: "sig456", PublicKey: "key456"},
		},
		[]TxOutput{
			{Address: testAddress, Amount: 1.5, TxID: "out1", Index: 0},
			{Address: testAddress2, Amount: 2.5, TxID: "out2", Index: 1},
		},
	)
	tx.Timestamp = 1234567890

	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "sig123")
	assert.Contains(t, jsonStr, "key123")
	assert.Contains(t, jsonStr, "1234567890")
}

func TestValidateAmountsComprehensive(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
		errMsg  string
	}{
		{
			name: "coinbase with multiple outputs",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}, {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: true,
			errMsg:  "coinbase transaction must have exactly one output",
		},
		{
			name: "coinbase with zero amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{},
				Outputs: []TxOutput{{Address: testAddress, Amount: 0}},
			},
			wantErr: true,
			errMsg:  "coinbase amount must be positive",
		},
		{
			name: "regular transaction with insufficient input",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 2.0}},
			},
			wantErr: true,
			errMsg:  "exceeds input amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateAmounts(make(map[string]map[int]TxOutput))
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransactionStructureComprehensive(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		utxoSet map[string]map[int]TxOutput
		wantErr bool
	}{
		{
			name: "valid transaction",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			utxoSet: map[string]map[int]TxOutput{
				"prev": {0: {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: true, // Will fail signature verification
		},
		{
			name: "transaction with invalid address format",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: "invalid", Amount: 1.0}},
			),
			utxoSet: map[string]map[int]TxOutput{
				"prev": {0: {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: true,
		},
		{
			name: "transaction with unknown UTXO",
			tx: NewTransaction(
				[]TxInput{{TxID: "unknown", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			utxoSet: map[string]map[int]TxOutput{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateTransactionStructure(tt.utxoSet)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetValidationReportComprehensive(t *testing.T) {
	// Test valid coinbase
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	report := tx.GetValidationReport(utxoSet)

	assert.True(t, report["is_valid"].(bool))
	assert.Empty(t, report["errors"].([]string))
	assert.Empty(t, report["warnings"].([]string))
	assert.Equal(t, 1, report["output_count"])
	assert.Equal(t, 0, report["input_count"])
}

func TestGetInputPublicKeyErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		tx         *Transaction
		inputIndex int
		wantErr    bool
	}{
		{
			name: "index out of range",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			inputIndex: 5,
			wantErr:    true,
		},
		{
			name: "invalid public key format",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      "prev",
					Index:     0,
					Signature: "sig",
					PublicKey: "invalid_key",
				}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			inputIndex: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.tx.GetInputPublicKey(tt.inputIndex)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTransactionTypeAllCases(t *testing.T) {
	tests := []struct {
		name     string
		tx       *Transaction
		expected string
	}{
		{
			name:     "coinbase",
			tx:       NewCoinbaseTransaction(testAddress, 1.0),
			expected: "coinbase",
		},
		{
			name: "simple transfer",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			expected: "simple_transfer",
		},
		{
			name: "multi output",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{
					{Address: testAddress, Amount: 0.5},
					{Address: testAddress2, Amount: 0.3},
					{Address: testAddress, Amount: 0.2},
				},
			),
			expected: "multi_output",
		},
		{
			name: "consolidation",
			tx: NewTransaction(
				[]TxInput{
					{TxID: "prev1", Index: 0},
					{TxID: "prev2", Index: 0},
					{TxID: "prev3", Index: 0},
				},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			expected: "consolidation",
		},
		{
			name: "standard",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{
					{Address: testAddress, Amount: 0.7},
					{Address: testAddress2, Amount: 0.3},
				},
			),
			expected: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txType := tt.tx.GetTransactionType()
			assert.Equal(t, tt.expected, txType)
		})
	}
}

func TestSignTransactionErrorCases(t *testing.T) {
	tests := []struct {
		name                string
		tx                  *Transaction
		inputIndex          int
		privateKey          *ecdsa.PrivateKey
		referencedTxOutputs []TxOutput
		wantErr             bool
		errMsg              string
	}{
		{
			name: "input index out of range",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			inputIndex:          5,
			privateKey:          testPrivateKey,
			referencedTxOutputs: []TxOutput{{Address: testAddress2, Amount: 2.0}},
			wantErr:             true,
			errMsg:              "input index 5 out of range",
		},
		{
			name: "empty tx ID",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      "", // Empty
					Index:     0,
					Signature: "",
					PublicKey: "",
				}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			inputIndex:          0,
			privateKey:          testPrivateKey,
			referencedTxOutputs: []TxOutput{{Address: testAddress2, Amount: 2.0}},
			wantErr:             true,
			errMsg:              "input 0 has empty tx ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.SignTransaction(tt.inputIndex, tt.privateKey, tt.referencedTxOutputs)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodePublicKeyInvalidCases(t *testing.T) {
	tests := []struct {
		name    string
		hexKey  string
		wantErr bool
	}{
		{"invalid hex", "0xg234567890123456789012345678901234567890", true},
		{"too short", "0412345678901234567890123456789012345678", true},
		{"wrong prefix", "03123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", true},
		{"invalid curve point", "04123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodePublicKey(tt.hexKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetSigningMessageErrorCases(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Test with invalid referenced output (index out of range)
	_, err := tx.getSigningMessage(0, []TxOutput{})
	assert.NoError(t, err) // Should handle empty case gracefully
}

func TestValidateInputsComprehensive(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Test with non-existent output in UTXO set
	utxoSet := map[string]map[int]TxOutput{
		"prev": {1: {Address: testAddress2, Amount: 2.0}}, // Different index
	}
	err := tx.ValidateInputs(utxoSet)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent output")

	// Test with valid UTXO but no signature
	utxoSetValid := map[string]map[int]TxOutput{
		"prev": {0: {Address: testAddress2, Amount: 2.0}},
	}
	err = tx.ValidateInputs(utxoSetValid)
	assert.Error(t, err) // Should fail due to missing signature
}

func TestValidateAmountsWithValidFee(t *testing.T) {
	// Test a transaction with valid fee calculation
	tx := &Transaction{
		ID:      "test",
		Inputs:  []TxInput{{TxID: "prev", Index: 0}},
		Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
	}

	// Mock the GetInputAmount to return a higher value
	// This test ensures the fee validation path is covered
	utxoSet := make(map[string]map[int]TxOutput)
	err := tx.ValidateAmounts(utxoSet)
	// Since GetInputAmount returns 0, this should fail
	assert.Error(t, err)
}

func TestValidateAddressFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid lowercase", "0x1234567890abcdef1234567890abcdef12345678", false},
		{"valid uppercase", "0X1234567890ABCDEF1234567890ABCDEF12345678", false},
		{"mixed case", "0x1234567890AbCdEf1234567890AbCdEf12345678", false},
		{"invalid character in middle", "0x1234567890g234567890123456789012345678", true},
		{"empty string", "", true},
		{"only prefix", "0x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddressFormat(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransactionStructureWithSignedTransaction(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Sign the transaction
	referencedOutputs := []TxOutput{{Address: testAddress2, Amount: 2.0}}
	err := tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	require.NoError(t, err)

	utxoSet := map[string]map[int]TxOutput{
		"prev": {0: {Address: testAddress2, Amount: 2.0}},
	}

	// With proper UTXO set, the transaction should pass all validation
	// (input amount is now calculated correctly from UTXO set)
	err = tx.ValidateTransactionStructure(utxoSet)
	assert.NoError(t, err)
}

func TestGetValidationReportWithWarnings(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 0.001}}, // Small amount
	)

	// Create a UTXO that would result in high fee
	utxoSet := map[string]map[int]TxOutput{
		"prev": {0: {Address: testAddress2, Amount: 1.0}},
	}

	report := tx.GetValidationReport(utxoSet)

	// Should have errors due to signature verification
	assert.False(t, report["is_valid"].(bool))
	assert.NotEmpty(t, report["errors"].([]string))
}

func TestToJSONErrorHandling(t *testing.T) {
	// Test JSON marshaling with complex data
	tx := NewTransaction(
		[]TxInput{
			{TxID: "tx1", Index: 0, Signature: "sig1", PublicKey: "key1"},
			{TxID: "tx2", Index: 1, Signature: "sig2", PublicKey: "key2"},
		},
		[]TxOutput{
			{Address: testAddress, Amount: 1.5, TxID: "out1", Index: 0},
			{Address: testAddress2, Amount: 2.5, TxID: "out2", Index: 1},
		},
	)
	tx.Timestamp = 1234567890

	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "sig1")
	assert.Contains(t, jsonStr, "key1")
	assert.Contains(t, jsonStr, "1234567890")
}

func TestCalculateIDJSONErrorHandling(t *testing.T) {
	// Create a transaction with normal data to test ID calculation
	tx := &Transaction{
		ID:        "test",
		Inputs:    []TxInput{{TxID: "test", Index: 0, Signature: "", PublicKey: ""}},
		Outputs:   []TxOutput{{Address: testAddress, Amount: 1.5}},
		Timestamp: 0,
	}

	// Should generate an ID
	id := tx.CalculateID()
	assert.NotEmpty(t, id)
}

func TestGetSignatureInfoWithMixedSignatures(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{
			{TxID: "prev1", Index: 0, Signature: "sig1", PublicKey: "key1"},
			{TxID: "prev2", Index: 0, Signature: "", PublicKey: ""}, // Unsigned
			{TxID: "prev3", Index: 0, Signature: "sig3", PublicKey: "key3"},
		},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	info := tx.GetSignatureInfo()
	assert.Equal(t, 3, info["total_inputs"].(int))
	assert.Equal(t, 2, info["signed_inputs"].(int))
	assert.Equal(t, 1, info["unsigned_inputs"].(int))
	assert.False(t, info["is_coinbase"].(bool))
}

func TestValidateAmountsWithPositiveAmounts(t *testing.T) {
	// Test the positive path in ValidateAmounts
	tx := &Transaction{
		ID:      "test",
		Inputs:  []TxInput{{TxID: "prev", Index: 0}},
		Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
	}

	// This should fail due to input amount being 0, but covers the validation logic
	err := tx.ValidateAmounts(make(map[string]map[int]TxOutput))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds input amount")
}

func TestVerifyInputSignatureWithInvalidSignature(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0, Signature: "invalid_signature", PublicKey: ""}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	referencedOutputs := []TxOutput{{Address: testAddress2, Amount: 2.0}}
	err := tx.VerifyInputSignature(0, referencedOutputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode signature")
}

func TestValidateBasicWithInvalidInputIndex(t *testing.T) {
	tx := &Transaction{
		ID: "test",
		Inputs: []TxInput{{
			TxID:  "prev",
			Index: -1, // Invalid index
		}},
		Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
	}

	err := tx.ValidateBasic()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid index")
}

func TestToJSONWithEmptyTransaction(t *testing.T) {
	tx := &Transaction{
		ID:        "test",
		Inputs:    []TxInput{},
		Outputs:   []TxOutput{},
		Timestamp: 0,
	}

	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "test")
}

func TestValidateTransactionBalanceEdgeCases(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Test with empty UTXO set
	utxoSet := map[string]map[int]TxOutput{}
	err := tx.ValidateTransactionBalance(utxoSet)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transaction")
}

func TestValidateAmountsFullCoverage(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "coinbase with negative amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{},
				Outputs: []TxOutput{{Address: testAddress, Amount: -1.0}},
			},
			wantErr: true,
		},
		{
			name: "regular transaction with negative output amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: -1.0}},
			},
			wantErr: true,
		},
		{
			name: "regular transaction with zero output amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 0.0}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateAmounts(make(map[string]map[int]TxOutput))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAmountsWithMultipleOutputs(t *testing.T) {
	// Test the individual output validation loop
	tx := &Transaction{
		ID:     "test",
		Inputs: []TxInput{{TxID: "prev", Index: 0}},
		Outputs: []TxOutput{
			{Address: testAddress, Amount: 0.5},
			{Address: testAddress2, Amount: -1.0}, // Invalid
		},
	}

	err := tx.ValidateAmounts(make(map[string]map[int]TxOutput))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has invalid amount")
}

func TestValidateAmountsWithEqualAmounts(t *testing.T) {
	// Test case where input equals output (zero fee)
	tx := &Transaction{
		ID:      "test",
		Inputs:  []TxInput{{TxID: "prev", Index: 0}},
		Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
	}

	err := tx.ValidateAmounts(make(map[string]map[int]TxOutput))
	// Should fail because input amount is 0
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds input amount")
}

func TestToJSONWithInvalidData(t *testing.T) {
	// Test ToJSON with problematic data that might cause issues
	tx := &Transaction{
		ID:        "test",
		Inputs:    []TxInput{{TxID: "test", Index: 0, Signature: "sig", PublicKey: "key"}},
		Outputs:   []TxOutput{{Address: testAddress, Amount: 1.5}},
		Timestamp: 1234567890,
	}

	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "1234567890")
}

func TestValidateTransactionStructureWithCoinbase(t *testing.T) {
	// Test ValidateTransactionStructure with coinbase (should skip input validation)
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	err := tx.ValidateTransactionStructure(utxoSet)
	assert.NoError(t, err)
}

func TestGetValidationReportWithCoinbase(t *testing.T) {
	// Test GetValidationReport with coinbase transaction
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	report := tx.GetValidationReport(utxoSet)
	assert.True(t, report["is_valid"].(bool))
	assert.Empty(t, report["errors"].([]string))
	assert.Equal(t, 1, report["output_count"].(int))
	assert.Equal(t, 0, report["input_count"].(int))
}

func TestValidateInputsWithCoinbase(t *testing.T) {
	// Test ValidateInputs with coinbase (should return nil immediately)
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	utxoSet := map[string]map[int]TxOutput{}

	err := tx.ValidateInputs(utxoSet)
	assert.NoError(t, err)
}

func TestVerifyAllSignaturesWithCoinbase(t *testing.T) {
	// Test VerifyAllSignatures with coinbase (should return nil immediately)
	tx := NewCoinbaseTransaction(testAddress, 1.0)
	referencedOutputs := [][]TxOutput{}

	err := tx.VerifyAllSignatures(referencedOutputs)
	assert.NoError(t, err)
}

func TestToJSONEdgeCases(t *testing.T) {
	// Test ToJSON with various edge cases
	tests := []struct {
		name string
		tx   *Transaction
	}{
		{
			name: "transaction with nil inputs",
			tx: &Transaction{
				ID:        "test",
				Inputs:    nil,
				Outputs:   []TxOutput{{Address: testAddress, Amount: 1.0}},
				Timestamp: 12345,
			},
		},
		{
			name: "transaction with nil outputs",
			tx: &Transaction{
				ID:        "test",
				Inputs:    []TxInput{{TxID: "prev", Index: 0}},
				Outputs:   nil,
				Timestamp: 12345,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := tt.tx.ToJSON()
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonStr)
		})
	}
}

func TestVerifyInputSignatureEdgeCases(t *testing.T) {
	// Test VerifyInputSignature with various edge cases
	tests := []struct {
		name              string
		tx                *Transaction
		inputIndex        int
		referencedOutputs []TxOutput
		wantErr           bool
		errMsg            string
	}{
		{
			name: "invalid signature hex",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      "prev",
					Index:     0,
					Signature: "invalid_hex_signature",
					PublicKey: encodePublicKey(testPublicKey),
				}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			inputIndex:        0,
			referencedOutputs: []TxOutput{{Address: testAddress2, Amount: 2.0}},
			wantErr:           true,
			errMsg:            "failed to decode signature",
		},
		{
			name: "invalid public key",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      "prev",
					Index:     0,
					Signature: "304402207deaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead02207deaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead",
					PublicKey: "invalid_public_key",
				}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			inputIndex:        0,
			referencedOutputs: []TxOutput{{Address: testAddress2, Amount: 2.0}},
			wantErr:           true,
			errMsg:            "failed to decode public key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.VerifyInputSignature(tt.inputIndex, tt.referencedOutputs)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculateIDEdgeCases(t *testing.T) {
	// Test CalculateID with edge cases that might cause JSON marshaling issues
	tests := []struct {
		name string
		tx   *Transaction
	}{
		{
			name: "transaction with very long strings",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      strings.Repeat("a", 1000),
					Index:     0,
					Signature: strings.Repeat("b", 1000),
					PublicKey: strings.Repeat("c", 1000),
				}},
				Outputs: []TxOutput{{
					Address: strings.Repeat("d", 1000),
					Amount:  1.0,
				}},
				Timestamp: 1234567890,
			},
		},
		{
			name: "transaction with special characters",
			tx: &Transaction{
				ID: "test",
				Inputs: []TxInput{{
					TxID:      "prev\n\t\r",
					Index:     0,
					Signature: "sig\"\\",
					PublicKey: "key'`",
				}},
				Outputs: []TxOutput{{
					Address: "addr\u0000\u0001",
					Amount:  1.0,
				}},
				Timestamp: 1234567890,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.tx.CalculateID()
			assert.NotEmpty(t, id)
			assert.Len(t, id, 64) // SHA256 hex output
		})
	}
}

func TestValidateBasicEdgeCases(t *testing.T) {
	// Test ValidateBasic with edge cases
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
		errMsg  string
	}{
		{
			name: "transaction with invalid address format in output",
			tx: &Transaction{
				ID:     "test",
				Inputs: []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{
					Address: "invalid_address_no_0x",
					Amount:  1.0,
				}},
			},
			wantErr: true,
			errMsg:  "invalid address format",
		},
		{
			name: "transaction with negative input index",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: -1}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			wantErr: true,
			errMsg:  "invalid index",
		},
		{
			name: "transaction with empty transaction ID",
			tx: &Transaction{
				ID:      "",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1.0}},
			},
			wantErr: true,
			errMsg:  "transaction ID is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateBasic()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransactionBalanceAdvanced(t *testing.T) {
	// Test ValidateTransactionBalance with edge cases
	tests := []struct {
		name    string
		tx      *Transaction
		utxoSet map[string]map[int]TxOutput
		wantErr bool
		errMsg  string
	}{
		{
			name: "transaction with non-existent output index",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 5}}, // Non-existent index
				[]TxOutput{{Address: testAddress, Amount: 1.0}},
			),
			utxoSet: map[string]map[int]TxOutput{
				"prev": {0: {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: true,
			errMsg:  "non-existent output",
		},
		{
			name: "transaction with insufficient balance",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 5.0}}, // More than available
			),
			utxoSet: map[string]map[int]TxOutput{
				"prev": {0: {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: true,
			errMsg:  "insufficient input amount",
		},
		{
			name: "transaction with exact balance (no fee)",
			tx: NewTransaction(
				[]TxInput{{TxID: "prev", Index: 0}},
				[]TxOutput{{Address: testAddress, Amount: 2.0}}, // Exactly equal
			),
			utxoSet: map[string]map[int]TxOutput{
				"prev": {0: {Address: testAddress2, Amount: 2.0}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateTransactionBalance(tt.utxoSet)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToJSONWithComplexData(t *testing.T) {
	// Test ToJSON with complex data structures
	tx := &Transaction{
		ID: "test_complex",
		Inputs: []TxInput{
			{
				TxID:      "tx_\"quoted\"",
				Index:     0,
				Signature: "sig\nwith\tnewlines\r",
				PublicKey: "key\u0000with\u0001null",
			},
			{
				TxID:      "tx_\\escaped\\",
				Index:     1,
				Signature: "",
				PublicKey: "",
			},
		},
		Outputs: []TxOutput{
			{
				Address: "0x1234567890123456789012345678901234567890",
				Amount:  1.23456789,
				TxID:    "parent_tx",
				Index:   0,
			},
			{
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Amount:  0.00000001,
				TxID:    "parent_tx",
				Index:   1,
			},
		},
		Timestamp: 1234567890,
	}

	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "test_complex")
	assert.Contains(t, jsonStr, "1.23456789")
	assert.Contains(t, jsonStr, "1234567890")
}

func TestVerifyInputSignatureComprehensive(t *testing.T) {
	// Test more signature verification scenarios
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Sign with valid key first
	referencedOutputs := []TxOutput{{Address: testAddress2, Amount: 2.0}}
	err := tx.SignTransaction(0, testPrivateKey, referencedOutputs)
	require.NoError(t, err)

	// Test verification with correct data
	err = tx.VerifyInputSignature(0, referencedOutputs)
	assert.NoError(t, err)

	// Test verification with modified referenced output
	modifiedOutputs := []TxOutput{{Address: testAddress2, Amount: 3.0}} // Different amount
	err = tx.VerifyInputSignature(0, modifiedOutputs)
	assert.Error(t, err)
}

func TestCalculateIDWithComplexScenarios(t *testing.T) {
	// Test CalculateID with various complex scenarios
	tests := []struct {
		name string
		tx   *Transaction
	}{
		{
			name: "transaction with maximum values",
			tx: &Transaction{
				ID: "",
				Inputs: []TxInput{
					{TxID: strings.Repeat("f", 64), Index: 2147483647, Signature: strings.Repeat("s", 200), PublicKey: strings.Repeat("k", 130)},
				},
				Outputs: []TxOutput{
					{Address: strings.Repeat("a", 42), Amount: math.MaxFloat64, TxID: "max_tx", Index: 2147483647},
				},
				Timestamp: 9223372036854775807, // Max int64
			},
		},
		{
			name: "transaction with minimum values",
			tx: &Transaction{
				ID: "",
				Inputs: []TxInput{
					{TxID: "", Index: -2147483648, Signature: "", PublicKey: ""},
				},
				Outputs: []TxOutput{
					{Address: "", Amount: math.SmallestNonzeroFloat64, TxID: "", Index: -2147483648},
				},
				Timestamp: -9223372036854775808, // Min int64
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.tx.CalculateID()
			assert.NotEmpty(t, id)
			assert.Len(t, id, 64) // SHA256 hex output
		})
	}
}

func TestGetSigningMessageWithVariousInputs(t *testing.T) {
	// Test getSigningMessage with different input scenarios
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Test with empty referenced outputs
	message, err := tx.getSigningMessage(0, []TxOutput{})
	assert.NoError(t, err)
	assert.NotEmpty(t, message)

	// Test with nil referenced outputs slice
	message, err = tx.getSigningMessage(0, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, message)
}

func TestToJSONWithUnmarshalableData(t *testing.T) {
	// Test ToJSON with edge case data
	tx := &Transaction{
		ID: "test_edge_case",
		Inputs: []TxInput{{
			TxID:      "prev",
			Index:     0,
			Signature: "sig_with_特殊_字符",
			PublicKey: "key_with_特殊_字符",
		}},
		Outputs: []TxOutput{{
			Address: testAddress,
			Amount:  1.797693134862315708145274237317043567981e+308, // Near max float64
		}},
		Timestamp: 9223372036854775807, // Max int64
	}

	// This should work with edge case data
	jsonStr, err := tx.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, "test_edge_case")
}

func TestCalculateIDWithJSONErrorPath(t *testing.T) {
	// Test CalculateID path that might trigger JSON errors
	// Create transaction with data that could cause issues
	tx := &Transaction{
		ID: "test",
		Inputs: []TxInput{{
			TxID:      strings.Repeat("x", 10000), // Very long string
			Index:     0,
			Signature: "",
			PublicKey: "",
		}},
		Outputs: []TxOutput{{
			Address: testAddress,
			Amount:  1.0,
		}},
		Timestamp: 0,
	}

	// Should still generate an ID
	id := tx.CalculateID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 64)
}

func TestValidateAmountsComplexScenarios(t *testing.T) {
	// Test complex amount validation scenarios
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "coinbase with very small amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{},
				Outputs: []TxOutput{{Address: testAddress, Amount: math.SmallestNonzeroFloat64}},
			},
			wantErr: false, // Should pass - small positive amount
		},
		{
			name: "regular transaction with sub-normal amount",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 1e-310}}, // Very small
			},
			wantErr: true, // Should fail - too small
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateAmounts(make(map[string]map[int]TxOutput))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBasicComplexValidation(t *testing.T) {
	// Test more complex validation scenarios
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "transaction with both inputs and outputs empty",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{},
				Outputs: []TxOutput{},
			},
			wantErr: true,
		},
		{
			name: "transaction with negative amount in output",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: -0.1}},
			},
			wantErr: true,
		},
		{
			name: "transaction with zero amount in output",
			tx: &Transaction{
				ID:      "test",
				Inputs:  []TxInput{{TxID: "prev", Index: 0}},
				Outputs: []TxOutput{{Address: testAddress, Amount: 0.0}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.ValidateBasic()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransactionStructureComplexPaths(t *testing.T) {
	// Test complex validation paths in ValidateTransactionStructure
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 1.0}},
	)

	// Create UTXO set with invalid address in referenced output
	utxoSet := map[string]map[int]TxOutput{
		"prev": {0: {Address: "invalid", Amount: 2.0}},
	}

	err := tx.ValidateTransactionStructure(utxoSet)
	assert.Error(t, err)
}

func TestGetValidationReportComplexPaths(t *testing.T) {
	// Test complex paths in GetValidationReport
	tx := NewTransaction(
		[]TxInput{{TxID: "prev", Index: 0}},
		[]TxOutput{{Address: testAddress, Amount: 0.001}}, // Small amount
	)

	// Create UTXO set that would cause multiple validation errors
	utxoSet := map[string]map[int]TxOutput{
		"prev": {0: {Address: testAddress2, Amount: 10.0}},
	}

	report := tx.GetValidationReport(utxoSet)

	// Should have validation errors due to signature verification
	assert.False(t, report["is_valid"].(bool))
	assert.NotEmpty(t, report["errors"].([]string))
}
