package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliexe/blockChain/internal/crypto"
	"github.com/aliexe/blockChain/internal/transactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WalletTestSuite defines the test suite
type WalletTestSuite struct {
	suite.Suite
	tempDir string
}

// SetupSuite is called once before all tests
func (suite *WalletTestSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "wallet_test")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir
}

// TearDownSuite is called once after all tests
func (suite *WalletTestSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
}

// SetupTest is called before each test
func (suite *WalletTestSuite) SetupTest() {
	// Clean up temp directory before each test
	os.RemoveAll(suite.tempDir)
	os.MkdirAll(suite.tempDir, 0755)
}

func TestWalletTestSuite(t *testing.T) {
	suite.Run(t, new(WalletTestSuite))
}

func (suite *WalletTestSuite) TestNewWallet() {
	tests := []struct {
		name        string
		config      WalletConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid wallet",
			config: WalletConfig{
				Name:       "Test Wallet",
				Passphrase: "secure123",
			},
			expectError: false,
		},
		{
			name: "wallet without passphrase",
			config: WalletConfig{
				Name: "Test Wallet",
			},
			expectError: false,
		},
		{
			name: "empty name",
			config: WalletConfig{
				Name: "",
			},
			expectError: true,
			errorMsg:    "wallet name cannot be empty",
		},
		{
			name: "wallet with description",
			config: WalletConfig{
				Name:        "Test Wallet",
				Passphrase:  "secure123",
				Description: "Test wallet for unit testing",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			wallet, err := NewWallet(tt.config)

			if tt.expectError {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tt.errorMsg)
				assert.Nil(suite.T(), wallet)
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), wallet)
				assert.Equal(suite.T(), tt.config.Name, wallet.Name)
				assert.NotEmpty(suite.T(), wallet.Addresses)
				assert.Equal(suite.T(), tt.config.Passphrase != "", wallet.IsEncrypted())

				if tt.config.Description != "" {
					assert.Equal(suite.T(), tt.config.Description, wallet.Metadata["description"])
				}
			}
		})
	}
}

func (suite *WalletTestSuite) TestGenerateNewAddress() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	initialCount := wallet.GetAddressCount()

	// Generate new address
	newAddress, err := wallet.GenerateNewAddress()
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), newAddress)

	// Check address was added
	assert.Equal(suite.T(), initialCount+1, wallet.GetAddressCount())
	addresses := wallet.GetAddresses()
	assert.Contains(suite.T(), addresses, newAddress)

	// Verify key pair exists
	keyPair, err := wallet.GetKeyPair(newAddress)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), keyPair)
	assert.Equal(suite.T(), newAddress, keyPair.Address)
}

func (suite *WalletTestSuite) TestGenerateNewAddressEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Should fail on encrypted wallet
	_, err = wallet.GenerateNewAddress()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot generate address on encrypted wallet")
}

func (suite *WalletTestSuite) TestGetKeyPair() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	addresses := wallet.GetAddresses()
	require.Greater(suite.T(), len(addresses), 0)

	// Get existing key pair
	keyPair, err := wallet.GetKeyPair(addresses[0])
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), keyPair)
	assert.Equal(suite.T(), addresses[0], keyPair.Address)

	// Get non-existent key pair
	_, err = wallet.GetKeyPair("nonexistent_address")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "address not found")
}

func (suite *WalletTestSuite) TestGetKeyPairEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Should fail on encrypted wallet
	_, err = wallet.GetKeyPair(wallet.GetAddresses()[0])
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "wallet is encrypted")
}

func (suite *WalletTestSuite) TestGetAddresses() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	addresses := wallet.GetAddresses()
	assert.NotEmpty(suite.T(), addresses)

	// Generate more addresses
	_, err = wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	newAddresses := wallet.GetAddresses()
	assert.Equal(suite.T(), len(addresses)+1, len(newAddresses))
	assert.Contains(suite.T(), newAddresses, addresses[0])
}

func (suite *WalletTestSuite) TestGetAddressCount() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	initialCount := wallet.GetAddressCount()
	assert.Greater(suite.T(), initialCount, 0)

	// Generate new address
	_, err = wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), initialCount+1, wallet.GetAddressCount())
}

func (suite *WalletTestSuite) TestCalculateBalance() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Create mock UTXO set
	utxoSet := map[string]map[int]transactions.TxOutput{
		"tx1": {
			0: {Address: wallet.GetAddresses()[0], Amount: 1.5},
			1: {Address: "other_address", Amount: 2.0},
		},
		"tx2": {
			0: {Address: wallet.GetAddresses()[0], Amount: 0.5},
		},
	}

	balance, err := wallet.CalculateBalance(utxoSet)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2.0, balance) // 1.5 + 0.5
}

func (suite *WalletTestSuite) TestCalculateBalanceEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	utxoSet := map[string]map[int]transactions.TxOutput{}
	_, err = wallet.CalculateBalance(utxoSet)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot calculate balance on encrypted wallet")
}

func (suite *WalletTestSuite) TestGetAddressBalance() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	addresses := wallet.GetAddresses()
	utxoSet := map[string]map[int]transactions.TxOutput{
		"tx1": {
			0: {Address: addresses[0], Amount: 1.5},
			1: {Address: addresses[0], Amount: 0.5},
		},
	}

	balance, err := wallet.GetAddressBalance(addresses[0], utxoSet)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2.0, balance)

	// Test non-existent address
	_, err = wallet.GetAddressBalance("nonexistent", utxoSet)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "address not found")
}

func (suite *WalletTestSuite) TestGetAddressBalanceEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	utxoSet := map[string]map[int]transactions.TxOutput{}
	_, err = wallet.GetAddressBalance(wallet.GetAddresses()[0], utxoSet)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot calculate balance on encrypted wallet")
}

func (suite *WalletTestSuite) TestGetAddressBalanceEmptyUtxoSet() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	utxoSet := map[string]map[int]transactions.TxOutput{}
	balance, err := wallet.GetAddressBalance(wallet.GetAddresses()[0], utxoSet)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0.0, balance)
}

func (suite *WalletTestSuite) TestEncryptDecrypt() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Generate some addresses
	address1, err := wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)
	_, err = wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	passphrase := "secure123"

	// Encrypt wallet
	err = wallet.Encrypt(passphrase)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to encrypt again
	err = wallet.Encrypt(passphrase)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "already encrypted")

	// Should fail to get key pairs while encrypted
	_, err = wallet.GetKeyPair(address1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "wallet is encrypted")

	// Decrypt wallet
	err = wallet.Decrypt(passphrase)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), wallet.IsEncrypted())

	// Should be able to get key pairs again
	keyPair, err := wallet.GetKeyPair(address1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), address1, keyPair.Address)

	// Test wrong passphrase
	err = wallet.Encrypt(passphrase)
	require.NoError(suite.T(), err)
	err = wallet.Decrypt("wrongpass")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to decrypt wallet data")
}

func (suite *WalletTestSuite) TestSaveAndLoad() {
	config := WalletConfig{
		Name:        "Test Wallet",
		Description: "Test wallet",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Generate some addresses
	address1, err := wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)
	address2, err := wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	// Save wallet
	filename := filepath.Join(suite.tempDir, "test_wallet.json")
	err = wallet.SaveToFile(filename)
	assert.NoError(suite.T(), err)

	// Load wallet
	loadedWallet, err := LoadFromFile(filename)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), loadedWallet)

	// Verify wallet data
	assert.Equal(suite.T(), wallet.Name, loadedWallet.Name)
	assert.Equal(suite.T(), wallet.Metadata["description"], loadedWallet.Metadata["description"])
	assert.Equal(suite.T(), len(wallet.Addresses), len(loadedWallet.Addresses))

	// Verify addresses
	loadedAddresses := loadedWallet.GetAddresses()
	assert.Contains(suite.T(), loadedAddresses, address1)
	assert.Contains(suite.T(), loadedAddresses, address2)

	// Verify key pairs work
	keyPair, err := loadedWallet.GetKeyPair(address1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), address1, keyPair.Address)
}

func (suite *WalletTestSuite) TestSaveAndLoadEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Save encrypted wallet
	filename := filepath.Join(suite.tempDir, "encrypted_wallet.json")
	err = wallet.SaveToFile(filename)
	assert.NoError(suite.T(), err)

	// Load encrypted wallet
	loadedWallet, err := LoadFromFile(filename)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), loadedWallet.IsEncrypted())

	// Decrypt and verify
	err = loadedWallet.Decrypt("secure123")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), loadedWallet.IsEncrypted())

	// Verify addresses match
	assert.Equal(suite.T(), wallet.GetAddresses(), loadedWallet.GetAddresses())
}

func (suite *WalletTestSuite) TestBackup() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Create backup
	err = wallet.Backup(suite.tempDir)
	assert.NoError(suite.T(), err)

	// Check backup file exists
	backupFiles, err := filepath.Glob(filepath.Join(suite.tempDir, "*-backup-*.json"))
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), backupFiles, 1)

	// Restore from backup
	restoreWallet, err := Restore(backupFiles[0])
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), wallet.Name, restoreWallet.Name)
	assert.Equal(suite.T(), wallet.GetAddresses(), restoreWallet.GetAddresses())
}

func (suite *WalletTestSuite) TestGetInfo() {
	config := WalletConfig{
		Name:        "Test Wallet",
		Description: "Test description",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	info := wallet.GetInfo()
	assert.Equal(suite.T(), wallet.Name, info["name"])
	assert.Equal(suite.T(), wallet.CreatedAt, info["created_at"])
	assert.Equal(suite.T(), wallet.UpdatedAt, info["updated_at"])
	assert.Equal(suite.T(), len(wallet.Addresses), info["address_count"])
	assert.Equal(suite.T(), false, info["encrypted"])
	assert.Equal(suite.T(), len(wallet.KeyPairs), info["key_pair_count"])
	assert.Equal(suite.T(), "Test description", info["metadata"].(map[string]string)["description"])
}

func (suite *WalletTestSuite) TestValidate() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Valid wallet
	err = wallet.Validate()
	assert.NoError(suite.T(), err)

	// Test encrypted wallet validation
	err = wallet.Encrypt("test123")
	require.NoError(suite.T(), err)
	err = wallet.Validate()
	assert.NoError(suite.T(), err)
}

func (suite *WalletTestSuite) TestValidateInvalidWallet() {
	// Invalid wallet with empty name
	wallet := &Wallet{
		Name: "",
	}
	err := wallet.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "wallet name cannot be empty")

	// Invalid timestamps
	wallet = &Wallet{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
	err = wallet.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "created_at cannot be after updated_at")
}

func (suite *WalletTestSuite) TestSignTransaction() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Create transaction
	inputs := []transactions.TxInput{
		{TxID: "tx123", Index: 0},
	}
	outputs := []transactions.TxOutput{
		{Address: "recipient", Amount: 1.0},
	}
	tx := transactions.NewTransaction(inputs, outputs)

	// Create referenced output
	referencedOutputs := []transactions.TxOutput{
		{Address: wallet.GetAddresses()[0], Amount: 2.0},
	}

	// Sign transaction
	err = wallet.SignTransaction(tx, 0, referencedOutputs)
	assert.NoError(suite.T(), err)

	// Verify signature
	err = tx.VerifyInputSignature(0, referencedOutputs)
	assert.NoError(suite.T(), err)
}

func (suite *WalletTestSuite) TestSignTransactionEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Create transaction
	tx := transactions.NewTransaction([]transactions.TxInput{{TxID: "tx123", Index: 0}},
		[]transactions.TxOutput{{Address: "recipient", Amount: 1.0}})

	referencedOutputs := []transactions.TxOutput{{Address: wallet.GetAddresses()[0], Amount: 2.0}}

	// Should fail on encrypted wallet
	err = wallet.SignTransaction(tx, 0, referencedOutputs)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot sign transaction with encrypted wallet")
}

func (suite *WalletTestSuite) TestGetUnspentOutputs() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Create mock UTXO set
	utxoSet := map[string]map[int]transactions.TxOutput{
		"tx1": {
			0: {Address: wallet.GetAddresses()[0], Amount: 1.5},
			1: {Address: "other_address", Amount: 2.0},
		},
		"tx2": {
			0: {Address: wallet.GetAddresses()[0], Amount: 0.5},
		},
	}

	unspent := wallet.GetUnspentOutputs(utxoSet)
	assert.Len(suite.T(), unspent, 2) // 2 outputs for our wallet

	// Verify outputs
	totalAmount := 0.0
	for _, output := range unspent {
		totalAmount += output.Amount
		assert.Equal(suite.T(), wallet.GetAddresses()[0], output.Address)
		assert.NotEmpty(suite.T(), output.TxID)
		assert.GreaterOrEqual(suite.T(), output.Index, 0)
	}
	assert.Equal(suite.T(), 2.0, totalAmount)
}

func (suite *WalletTestSuite) TestGetUnspentOutputsEncrypted() {
	config := WalletConfig{
		Name:       "Test Wallet",
		Passphrase: "secure123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	utxoSet := map[string]map[int]transactions.TxOutput{}
	unspent := wallet.GetUnspentOutputs(utxoSet)
	assert.Len(suite.T(), unspent, 0)
}

func (suite *WalletTestSuite) TestGetUnspentOutputsEmptyUtxoSet() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	utxoSet := map[string]map[int]transactions.TxOutput{}
	unspent := wallet.GetUnspentOutputs(utxoSet)
	assert.Len(suite.T(), unspent, 0)
}

// Edge cases and error handling tests

func (suite *WalletTestSuite) TestEncryptEmptyWallet() {
	wallet := &Wallet{
		Name:      "Empty",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		KeyPairs:  make(map[string]*crypto.KeyPair),
		Addresses: []string{},
		Encrypted: false,
	}

	err := wallet.Encrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot encrypt empty wallet")
}

func (suite *WalletTestSuite) TestDecryptNotEncrypted() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "wallet is not encrypted")
}

func (suite *WalletTestSuite) TestSaveToFileInvalidPath() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Try to save to invalid path
	invalidPath := "/root/nonexistent/path/wallet.json"
	err = wallet.SaveToFile(invalidPath)
	assert.Error(suite.T(), err)
}

func (suite *WalletTestSuite) TestLoadFromFileNonExistent() {
	_, err := LoadFromFile("/nonexistent/wallet.json")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to read wallet file")
}

func (suite *WalletTestSuite) TestLoadFromFileInvalidJSON() {
	// Create file with invalid JSON
	invalidFile := filepath.Join(suite.tempDir, "invalid.json")
	err := os.WriteFile(invalidFile, []byte("invalid json"), 0644)
	require.NoError(suite.T(), err)

	_, err = LoadFromFile(invalidFile)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to unmarshal wallet storage")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletWithoutData() {
	// Create encrypted wallet storage without encrypted_data
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata:  make(map[string]string),
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "invalid_encrypted.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to missing data
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletInvalidHex() {
	// Create encrypted wallet storage with invalid hex in metadata
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",     // Valid hex for "abc"
			"salt":           "invalidhex", // Invalid hex
			"nonce":          "616263",
			"checksum":       "616263",
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "invalid_hex.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to invalid hex
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to decode salt")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletInvalidNonceHex() {
	// Create encrypted wallet storage with invalid nonce hex
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",
			"salt":           "616263",
			"nonce":          "invalidhex", // Invalid hex
			"checksum":       "616263",
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "invalid_nonce.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to invalid nonce hex
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to decode nonce")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletInvalidChecksum() {
	// Create encrypted wallet storage with wrong checksum
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",
			"salt":           "616263",
			"nonce":          "616263",
			"checksum":       "wrongchecksum", // Wrong checksum
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "invalid_checksum.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to wrong checksum
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "checksum verification failed")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletMissingSalt() {
	// Create encrypted wallet storage missing salt
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",
			"nonce":          "616263",
			"checksum":       "616263",
			// Missing salt
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "missing_salt.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to missing salt
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "salt not found")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletMissingNonce() {
	// Create encrypted wallet storage missing nonce
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",
			"salt":           "616263",
			"checksum":       "616263",
			// Missing nonce
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "missing_nonce.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to missing nonce
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "nonce not found")
}

func (suite *WalletTestSuite) TestLoadEncryptedWalletMissingChecksum() {
	// Create encrypted wallet storage missing checksum
	walletStorage := WalletStorage{
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addresses: []string{},
		Encrypted: true,
		Metadata: map[string]string{
			"encrypted_data": "616263",
			"salt":           "616263",
			"nonce":          "616263",
			// Missing checksum
		},
	}

	data, err := json.Marshal(walletStorage)
	require.NoError(suite.T(), err)

	invalidFile := filepath.Join(suite.tempDir, "missing_checksum.json")
	err = os.WriteFile(invalidFile, data, 0644)
	require.NoError(suite.T(), err)

	wallet, err := LoadFromFile(invalidFile)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Should fail to decrypt due to missing checksum
	err = wallet.Decrypt("test123")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "checksum not found")
}

func (suite *WalletTestSuite) TestSignTransactionInvalidInput() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	tx := transactions.NewTransaction([]transactions.TxInput{{TxID: "tx123", Index: 0}},
		[]transactions.TxOutput{{Address: "recipient", Amount: 1.0}})

	// Invalid input index
	referencedOutputs := []transactions.TxOutput{{Address: wallet.GetAddresses()[0], Amount: 2.0}}
	err = wallet.SignTransaction(tx, 5, referencedOutputs)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "input index 5 out of range")

	// Missing referenced output
	err = wallet.SignTransaction(tx, 0, []transactions.TxOutput{})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "referenced output not found")
}

func (suite *WalletTestSuite) TestSignTransactionAddressMismatch() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	tx := transactions.NewTransaction([]transactions.TxInput{{TxID: "tx123", Index: 0}},
		[]transactions.TxOutput{{Address: "recipient", Amount: 1.0}})

	// Referenced output with different address
	referencedOutputs := []transactions.TxOutput{{Address: "different_address", Amount: 2.0}}
	err = wallet.SignTransaction(tx, 0, referencedOutputs)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "no key pair found for address")
}

// Performance tests

func (suite *WalletTestSuite) BenchmarkGenerateNewAddress(b *testing.B) {
	config := WalletConfig{Name: "Benchmark Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wallet.GenerateNewAddress()
		if err != nil {
			b.Fatalf("Failed to generate address: %v", err)
		}
	}
}

func (suite *WalletTestSuite) BenchmarkCalculateBalance(b *testing.B) {
	config := WalletConfig{Name: "Benchmark Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(b, err)

	// Generate multiple addresses
	for i := 0; i < 100; i++ {
		_, err := wallet.GenerateNewAddress()
		require.NoError(b, err)
	}

	// Create large UTXO set
	utxoSet := make(map[string]map[int]transactions.TxOutput)
	for i := 0; i < 1000; i++ {
		txID := fmt.Sprintf("tx%d", i)
		utxoSet[txID] = map[int]transactions.TxOutput{
			0: {Address: wallet.GetAddresses()[i%len(wallet.GetAddresses())], Amount: 1.0},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wallet.CalculateBalance(utxoSet)
		if err != nil {
			b.Fatalf("Failed to calculate balance: %v", err)
		}
	}
}

func (suite *WalletTestSuite) BenchmarkEncryptDecrypt(b *testing.B) {
	config := WalletConfig{Name: "Benchmark Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(b, err)

	// Generate some addresses
	for i := 0; i < 10; i++ {
		_, err := wallet.GenerateNewAddress()
		require.NoError(b, err)
	}

	passphrase := "benchmark123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Encrypt
		err := wallet.Encrypt(passphrase)
		if err != nil {
			b.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt
		err = wallet.Decrypt(passphrase)
		if err != nil {
			b.Fatalf("Failed to decrypt: %v", err)
		}
	}
}

// Integration tests with crypto package

func (suite *WalletTestSuite) TestWalletWithExternalKeyPair() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Generate external key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(suite.T(), err)

	externalKeyPair, err := crypto.GetKeyPairFromPrivate(privateKey)
	require.NoError(suite.T(), err)

	// Add external key pair to wallet
	wallet.KeyPairs[externalKeyPair.Address] = externalKeyPair
	wallet.Addresses = append(wallet.Addresses, externalKeyPair.Address)
	wallet.UpdatedAt = time.Now()

	// Verify it works
	retrievedKeyPair, err := wallet.GetKeyPair(externalKeyPair.Address)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), externalKeyPair.Address, retrievedKeyPair.Address)
	assert.True(suite.T(), externalKeyPair.PublicKey.Equal(retrievedKeyPair.PublicKey))
}

// Test edge cases for balance calculation

func (suite *WalletTestSuite) TestCalculateBalanceWithEdgeCases() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	tests := []struct {
		name     string
		utxoSet  map[string]map[int]transactions.TxOutput
		expected float64
	}{
		{
			name:     "empty UTXO set",
			utxoSet:  map[string]map[int]transactions.TxOutput{},
			expected: 0.0,
		},
		{
			name: "no matching addresses",
			utxoSet: map[string]map[int]transactions.TxOutput{
				"tx1": {0: {Address: "other_address", Amount: 1.0}},
			},
			expected: 0.0,
		},
		{
			name: "very small amounts",
			utxoSet: map[string]map[int]transactions.TxOutput{
				"tx1": {0: {Address: wallet.GetAddresses()[0], Amount: 0.00000001}},
			},
			expected: 0.00000001,
		},
		{
			name: "very large amounts",
			utxoSet: map[string]map[int]transactions.TxOutput{
				"tx1": {0: {Address: wallet.GetAddresses()[0], Amount: math.MaxFloat64 / 2}},
			},
			expected: math.MaxFloat64 / 2,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			balance, err := wallet.CalculateBalance(tt.utxoSet)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expected, balance)
		})
	}
}

func (suite *WalletTestSuite) TestValidateUnencryptedWallet() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Add more addresses
	_, err = wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)
	_, err = wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	// Valid wallet
	err = wallet.Validate()
	assert.NoError(suite.T(), err)

	// Save original addresses
	originalAddresses := make([]string, len(wallet.Addresses))
	copy(originalAddresses, wallet.Addresses)

	// Manually corrupt wallet to test validation - mismatch counts
	wallet.Addresses = wallet.Addresses[:1] // Remove addresses but keep key pairs
	err = wallet.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "key pair count")

	// Restore addresses and corrupt differently - make counts match but with wrong address
	wallet.Addresses = make([]string, len(originalAddresses))
	copy(wallet.Addresses, originalAddresses)

	// Replace one key pair with wrong address
	wrongKeyPair, _ := crypto.NewKeyPair()
	delete(wallet.KeyPairs, wallet.Addresses[0])        // Remove the correct one
	wallet.KeyPairs[wallet.Addresses[0]] = wrongKeyPair // Add with wrong keypair
	err = wallet.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "address mismatch")
}

func (suite *WalletTestSuite) TestIsEncrypted() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Initially not encrypted
	assert.False(suite.T(), wallet.IsEncrypted())

	// Encrypt wallet
	err = wallet.Encrypt("test123")
	require.NoError(suite.T(), err)
	assert.True(suite.T(), wallet.IsEncrypted())

	// Decrypt wallet
	err = wallet.Decrypt("test123")
	require.NoError(suite.T(), err)
	assert.False(suite.T(), wallet.IsEncrypted())
}

func (suite *WalletTestSuite) TestWalletMetadata() {
	config := WalletConfig{
		Name:        "Test Wallet",
		Description: "Test description",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Test initial metadata
	assert.Equal(suite.T(), "Test description", wallet.Metadata["description"])

	// Add custom metadata
	wallet.Metadata["custom_field"] = "custom_value"
	assert.Equal(suite.T(), "custom_value", wallet.Metadata["custom_field"])

	// Test that metadata persists through save/load
	filename := filepath.Join(suite.tempDir, "metadata_test.json")
	err = wallet.SaveToFile(filename)
	require.NoError(suite.T(), err)

	loadedWallet, err := LoadFromFile(filename)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test description", loadedWallet.Metadata["description"])
	assert.Equal(suite.T(), "custom_value", loadedWallet.Metadata["custom_field"])
}

func (suite *WalletTestSuite) TestMultipleAddressesBalance() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Generate multiple addresses
	addresses := make([]string, 5)
	for i := 0; i < 5; i++ {
		if i > 0 {
			addresses[i], err = wallet.GenerateNewAddress()
			require.NoError(suite.T(), err)
		} else {
			addresses[i] = wallet.GetAddresses()[0]
		}
	}

	// Create UTXO set with outputs for different addresses
	utxoSet := make(map[string]map[int]transactions.TxOutput)
	for i, addr := range addresses {
		txID := fmt.Sprintf("tx%d", i)
		utxoSet[txID] = map[int]transactions.TxOutput{
			0: {Address: addr, Amount: float64(i+1) * 0.5},
		}
	}

	// Test total balance
	totalBalance, err := wallet.CalculateBalance(utxoSet)
	assert.NoError(suite.T(), err)
	expectedTotal := 0.5 + 1.0 + 1.5 + 2.0 + 2.5 // 7.5
	assert.Equal(suite.T(), expectedTotal, totalBalance)

	// Test individual address balances
	for i, addr := range addresses {
		balance, err := wallet.GetAddressBalance(addr, utxoSet)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), float64(i+1)*0.5, balance)
	}
}

func (suite *WalletTestSuite) TestBackupAndRestoreEncrypted() {
	config := WalletConfig{
		Name:       "Encrypted Test Wallet",
		Passphrase: "backup123",
	}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Wallet already has one address from NewWallet
	// No need to generate more since it's encrypted

	// Create backup
	err = wallet.Backup(suite.tempDir)
	assert.NoError(suite.T(), err)

	// Find backup file
	backupFiles, err := filepath.Glob(filepath.Join(suite.tempDir, "*-backup-*.json"))
	require.NoError(suite.T(), err)
	require.Len(suite.T(), backupFiles, 1)

	// Restore from backup
	restoreWallet, err := Restore(backupFiles[0])
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), restoreWallet.IsEncrypted())

	// Decrypt and verify
	err = restoreWallet.Decrypt("backup123")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), wallet.Name, restoreWallet.Name)
	assert.Equal(suite.T(), wallet.GetAddresses(), restoreWallet.GetAddresses())
}

func (suite *WalletTestSuite) TestSignAllInputsTransaction() {
	config := WalletConfig{Name: "Test Wallet"}
	wallet, err := NewWallet(config)
	require.NoError(suite.T(), err)

	// Generate additional address for multi-input transaction
	secondAddr, err := wallet.GenerateNewAddress()
	require.NoError(suite.T(), err)

	// Create transaction with multiple inputs
	inputs := []transactions.TxInput{
		{TxID: "tx1", Index: 0},
		{TxID: "tx2", Index: 0},
	}
	outputs := []transactions.TxOutput{
		{Address: "recipient", Amount: 1.5},
	}
	tx := transactions.NewTransaction(inputs, outputs)

	// Create referenced outputs
	referencedOutputs := []transactions.TxOutput{
		{Address: wallet.GetAddresses()[0], Amount: 1.0},
		{Address: secondAddr, Amount: 1.0},
	}

	// Sign both inputs
	err = wallet.SignTransaction(tx, 0, referencedOutputs)
	assert.NoError(suite.T(), err)
	err = wallet.SignTransaction(tx, 1, referencedOutputs)
	assert.NoError(suite.T(), err)

	// Verify both signatures
	err = tx.VerifyInputSignature(0, referencedOutputs)
	assert.NoError(suite.T(), err)
	err = tx.VerifyInputSignature(1, referencedOutputs)
	assert.NoError(suite.T(), err)
}
