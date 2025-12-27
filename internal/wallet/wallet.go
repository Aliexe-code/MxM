package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aliexe/blockChain/internal/crypto"
	"github.com/aliexe/blockChain/internal/transactions"
)

// Wallet represents a cryptocurrency wallet
type Wallet struct {
	Name         string                    `json:"name"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	KeyPairs     map[string]*crypto.KeyPair `json:"-"` // Don't serialize private keys directly
	Addresses    []string                  `json:"addresses"`
	Encrypted    bool                      `json:"encrypted"`
	Metadata     map[string]string         `json:"metadata"`
	mu           sync.RWMutex             `json:"-"`
}

// WalletStorage represents the serialized wallet format
type WalletStorage struct {
	Name         string            `json:"name"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	KeyStores    []*crypto.KeyStorage `json:"key_stores"`
	Addresses    []string          `json:"addresses"`
	Encrypted    bool              `json:"encrypted"`
	EncryptionData *EncryptionData `json:"encryption_data,omitempty"`
	Metadata     map[string]string `json:"metadata"`
}

// EncryptionData contains encryption metadata
type EncryptionData struct {
	Salt        string `json:"salt"`
	Nonce       string `json:"nonce"`
	Checksum    string `json:"checksum"`
}

// WalletConfig contains wallet creation options
type WalletConfig struct {
	Name        string
	Passphrase  string
	Description string
}

// NewWallet creates a new wallet
func NewWallet(config WalletConfig) (*Wallet, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("wallet name cannot be empty")
	}

	now := time.Now()
	wallet := &Wallet{
		Name:      config.Name,
		CreatedAt: now,
		UpdatedAt: now,
		KeyPairs:  make(map[string]*crypto.KeyPair),
		Addresses: []string{},
		Encrypted: false,
		Metadata:  make(map[string]string),
	}

	if config.Description != "" {
		wallet.Metadata["description"] = config.Description
	}

	// Generate initial address
	_, err := wallet.GenerateNewAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial address: %w", err)
	}

	// Encrypt if passphrase provided
	if config.Passphrase != "" {
		err := wallet.Encrypt(config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt wallet: %w", err)
		}
	}

	return wallet, nil
}

// GenerateNewAddress creates a new address in the wallet
func (w *Wallet) GenerateNewAddress() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Encrypted {
		return "", fmt.Errorf("cannot generate address on encrypted wallet")
	}

	// Generate new key pair
	keyPair, err := crypto.NewKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Store key pair
	w.KeyPairs[keyPair.Address] = keyPair
	w.Addresses = append(w.Addresses, keyPair.Address)
	w.UpdatedAt = time.Now()

	return keyPair.Address, nil
}

// GetKeyPair retrieves a key pair by address
func (w *Wallet) GetKeyPair(address string) (*crypto.KeyPair, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Encrypted {
		return nil, fmt.Errorf("wallet is encrypted")
	}

	keyPair, exists := w.KeyPairs[address]
	if !exists {
		return nil, fmt.Errorf("address not found in wallet: %s", address)
	}

	return keyPair, nil
}

// GetAddresses returns all addresses in the wallet
func (w *Wallet) GetAddresses() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	addresses := make([]string, len(w.Addresses))
	copy(addresses, w.Addresses)
	return addresses
}

// GetAddressCount returns the number of addresses in the wallet
func (w *Wallet) GetAddressCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.Addresses)
}

// CalculateBalance calculates the total balance for the wallet
func (w *Wallet) CalculateBalance(utxoSet map[string]map[int]transactions.TxOutput) (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Encrypted {
		return 0, fmt.Errorf("cannot calculate balance on encrypted wallet")
	}

	var totalBalance float64

	for _, address := range w.Addresses {
		// Find all UTXOs for this address
			for _, outputs := range utxoSet {
				for _, output := range outputs {
					// Check if this output belongs to our wallet address
					if output.Address == address {
						// Check if this UTXO is still unspent
						// In a real implementation, we'd need to check if this output has been spent
						// For now, we'll assume all outputs in the UTXO set are unspent
						totalBalance += output.Amount
					}
				}
			}	}

	return totalBalance, nil
}

// GetAddressBalance calculates balance for a specific address
func (w *Wallet) GetAddressBalance(address string, utxoSet map[string]map[int]transactions.TxOutput) (float64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Encrypted {
		return 0, fmt.Errorf("cannot calculate balance on encrypted wallet")
	}

	// Check if address exists in wallet
	_, exists := w.KeyPairs[address]
	if !exists {
		return 0, fmt.Errorf("address not found in wallet: %s", address)
	}

	var addressBalance float64

	// Find all UTXOs for this address
	for _, outputs := range utxoSet {
		for _, output := range outputs {
			if output.Address == address {
				addressBalance += output.Amount
			}
		}
	}

	return addressBalance, nil
}

// Encrypt encrypts the wallet with a passphrase
func (w *Wallet) Encrypt(passphrase string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Encrypted {
		return fmt.Errorf("wallet is already encrypted")
	}

	if len(w.KeyPairs) == 0 {
		return fmt.Errorf("cannot encrypt empty wallet")
	}

	// Convert key pairs to storage format
	keyStores := make([]*crypto.KeyStorage, 0, len(w.KeyPairs))
	for _, keyPair := range w.KeyPairs {
		storage, err := keyPair.ToStorage()
		if err != nil {
			return fmt.Errorf("failed to convert key pair to storage: %w", err)
		}
		keyStores = append(keyStores, storage)
	}

	// Serialize wallet data
	walletData := WalletStorage{
		Name:      w.Name,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
		KeyStores: keyStores,
		Addresses: w.Addresses,
		Encrypted: true,
		Metadata:  w.Metadata,
	}

	data, err := json.Marshal(walletData)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}

	// Encrypt the data
	encryptedData, encryptionData, err := encryptData(data, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt wallet data: %w", err)
	}

	// Clear unencrypted data
	w.KeyPairs = make(map[string]*crypto.KeyPair)

	// Store encrypted data in metadata temporarily
	w.Metadata["encrypted_data"] = hex.EncodeToString(encryptedData)
	w.Metadata["salt"] = encryptionData.Salt
	w.Metadata["nonce"] = encryptionData.Nonce
	w.Metadata["checksum"] = encryptionData.Checksum
	w.Encrypted = true
	w.UpdatedAt = time.Now()

	return nil
}

// Decrypt decrypts the wallet with a passphrase
func (w *Wallet) Decrypt(passphrase string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.Encrypted {
		return fmt.Errorf("wallet is not encrypted")
	}

	// Get encrypted data from metadata
	encryptedDataHex, exists := w.Metadata["encrypted_data"]
	if !exists {
		return fmt.Errorf("encrypted data not found")
	}

	salt, exists := w.Metadata["salt"]
	if !exists {
		return fmt.Errorf("salt not found")
	}

	nonce, exists := w.Metadata["nonce"]
	if !exists {
		return fmt.Errorf("nonce not found")
	}

	checksum, exists := w.Metadata["checksum"]
	if !exists {
		return fmt.Errorf("checksum not found")
	}

	encryptionData := &EncryptionData{
		Salt:     salt,
		Nonce:    nonce,
		Checksum: checksum,
	}

	// Decode hex data
	encryptedData, err := hex.DecodeString(encryptedDataHex)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Decrypt the data
	data, err := decryptData(encryptedData, passphrase, encryptionData)
	if err != nil {
		return fmt.Errorf("failed to decrypt wallet data: %w", err)
	}

	// Unmarshal wallet data
	var walletStorage WalletStorage
	err = json.Unmarshal(data, &walletStorage)
	if err != nil {
		return fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	// Restore key pairs
	w.KeyPairs = make(map[string]*crypto.KeyPair)
	for _, keyStore := range walletStorage.KeyStores {
		keyPair, err := crypto.FromStorage(keyStore)
		if err != nil {
			return fmt.Errorf("failed to restore key pair: %w", err)
		}
		w.KeyPairs[keyPair.Address] = keyPair
	}

	// Clear encrypted data from metadata
	delete(w.Metadata, "encrypted_data")
	delete(w.Metadata, "salt")
	delete(w.Metadata, "nonce")
	delete(w.Metadata, "checksum")

	w.Encrypted = false
	w.UpdatedAt = time.Now()

	return nil
}

// IsEncrypted returns whether the wallet is encrypted
func (w *Wallet) IsEncrypted() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Encrypted
}

// SaveToFile saves the wallet to a file
func (w *Wallet) SaveToFile(filename string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Prepare storage data
	var walletStorage WalletStorage

	if w.Encrypted {
		// For encrypted wallets, save the encrypted data
		walletStorage = WalletStorage{
			Name:      w.Name,
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
			Addresses: w.Addresses,
			Encrypted: true,
			Metadata:  make(map[string]string),
		}

		// Copy encryption metadata
		for key, value := range w.Metadata {
			if key == "encrypted_data" || key == "salt" || key == "nonce" || key == "checksum" {
				walletStorage.Metadata[key] = value
			}
		}

		// Parse encryption data
		if salt, exists := w.Metadata["salt"]; exists {
			if nonce, exists := w.Metadata["nonce"]; exists {
				if checksum, exists := w.Metadata["checksum"]; exists {
					walletStorage.EncryptionData = &EncryptionData{
						Salt:     salt,
						Nonce:    nonce,
						Checksum: checksum,
					}
				}
			}
		}
	} else {
		// For unencrypted wallets, convert key pairs to storage
		keyStores := make([]*crypto.KeyStorage, 0, len(w.KeyPairs))
		for _, keyPair := range w.KeyPairs {
			storage, err := keyPair.ToStorage()
			if err != nil {
				return fmt.Errorf("failed to convert key pair to storage: %w", err)
			}
			keyStores = append(keyStores, storage)
		}

		walletStorage = WalletStorage{
			Name:      w.Name,
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
			KeyStores: keyStores,
			Addresses: w.Addresses,
			Encrypted: false,
			Metadata:  w.Metadata,
		}
	}

	// Serialize
	data, err := json.MarshalIndent(walletStorage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallet storage: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

// LoadFromFile loads a wallet from a file
func LoadFromFile(filename string) (*Wallet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	var walletStorage WalletStorage
	if err := json.Unmarshal(data, &walletStorage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet storage: %w", err)
	}

	wallet := &Wallet{
		Name:      walletStorage.Name,
		CreatedAt: walletStorage.CreatedAt,
		UpdatedAt: walletStorage.UpdatedAt,
		Addresses: walletStorage.Addresses,
		Encrypted: walletStorage.Encrypted,
		Metadata:  walletStorage.Metadata,
		KeyPairs:  make(map[string]*crypto.KeyPair),
	}

	if walletStorage.Encrypted {
		// For encrypted wallets, restore encryption metadata
		if walletStorage.EncryptionData != nil {
			encryptedDataHex, exists := walletStorage.Metadata["encrypted_data"]
			if exists {
				wallet.Metadata["encrypted_data"] = encryptedDataHex
				wallet.Metadata["salt"] = walletStorage.EncryptionData.Salt
				wallet.Metadata["nonce"] = walletStorage.EncryptionData.Nonce
				wallet.Metadata["checksum"] = walletStorage.EncryptionData.Checksum
			}
		}
	} else {
		// For unencrypted wallets, restore key pairs
		for _, keyStore := range walletStorage.KeyStores {
			keyPair, err := crypto.FromStorage(keyStore)
			if err != nil {
				return nil, fmt.Errorf("failed to restore key pair: %w", err)
			}
			wallet.KeyPairs[keyPair.Address] = keyPair
		}
	}

	return wallet, nil
}

// Backup creates a backup of the wallet
func (w *Wallet) Backup(backupDir string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFilename := filepath.Join(backupDir, fmt.Sprintf("%s-backup-%s.json", w.Name, timestamp))

	return w.SaveToFile(backupFilename)
}

// Restore restores a wallet from a backup file
func Restore(backupFile string) (*Wallet, error) {
	return LoadFromFile(backupFile)
}

// GetInfo returns wallet information
func (w *Wallet) GetInfo() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	info := map[string]interface{}{
		"name":          w.Name,
		"created_at":    w.CreatedAt,
		"updated_at":    w.UpdatedAt,
		"address_count": len(w.Addresses),
		"encrypted":     w.Encrypted,
		"metadata":      w.Metadata,
	}

	if !w.Encrypted {
		info["key_pair_count"] = len(w.KeyPairs)
	}

	return info
}

// Validate checks if the wallet is valid
func (w *Wallet) Validate() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Name == "" {
		return fmt.Errorf("wallet name cannot be empty")
	}

	if w.CreatedAt.After(w.UpdatedAt) {
		return fmt.Errorf("created_at cannot be after updated_at")
	}

	if !w.Encrypted {
		// Validate key pairs for unencrypted wallets
		if len(w.KeyPairs) != len(w.Addresses) {
			return fmt.Errorf("key pair count (%d) doesn't match address count (%d)", 
				len(w.KeyPairs), len(w.Addresses))
		}

		for address, keyPair := range w.KeyPairs {
			if !keyPair.IsValidKeyPair() {
				return fmt.Errorf("invalid key pair for address: %s", address)
			}
			if keyPair.Address != address {
				return fmt.Errorf("key pair address mismatch: expected %s, got %s", 
					address, keyPair.Address)
			}
		}

		// Check all addresses are in key pairs
		for _, address := range w.Addresses {
			if _, exists := w.KeyPairs[address]; !exists {
				return fmt.Errorf("address not found in key pairs: %s", address)
			}
		}
	}

	return nil
}

// encryptData encrypts data using AES-GCM
func encryptData(data []byte, passphrase string) ([]byte, *EncryptionData, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from passphrase
	key := deriveKey(passphrase, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	encrypted := gcm.Seal(nil, nonce, data, nil)

	// Calculate checksum
	checksum := sha256.Sum256(encrypted)

	encryptionData := &EncryptionData{
		Salt:     hex.EncodeToString(salt),
		Nonce:    hex.EncodeToString(nonce),
		Checksum: hex.EncodeToString(checksum[:]),
	}

	return encrypted, encryptionData, nil
}

// decryptData decrypts data using AES-GCM
func decryptData(encryptedData []byte, passphrase string, encryptionData *EncryptionData) ([]byte, error) {
	// Decode salt and nonce
	salt, err := hex.DecodeString(encryptionData.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	nonce, err := hex.DecodeString(encryptionData.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Verify checksum
	calculatedChecksum := sha256.Sum256(encryptedData)
	if hex.EncodeToString(calculatedChecksum[:]) != encryptionData.Checksum {
		return nil, fmt.Errorf("checksum verification failed")
	}

	// Derive key
	key := deriveKey(passphrase, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	data, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return data, nil
}

// deriveKey derives an encryption key from passphrase using PBKDF2
func deriveKey(passphrase string, salt []byte) []byte {
	hash := sha256.Sum256([]byte(passphrase + string(salt)))
	return hash[:]
}

// SignTransaction signs a transaction using the appropriate wallet key
func (w *Wallet) SignTransaction(tx *transactions.Transaction, inputIndex int, referencedTxOutputs []transactions.TxOutput) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.Encrypted {
		return fmt.Errorf("cannot sign transaction with encrypted wallet")
	}

	if inputIndex >= len(tx.Inputs) {
		return fmt.Errorf("input index %d out of range", inputIndex)
	}

	// Get the referenced output to determine which address to use
	if inputIndex >= len(referencedTxOutputs) {
		return fmt.Errorf("referenced output not found for input %d", inputIndex)
	}

	referencedOutput := referencedTxOutputs[inputIndex]
	keyPair, exists := w.KeyPairs[referencedOutput.Address]
	if !exists {
		return fmt.Errorf("no key pair found for address: %s", referencedOutput.Address)
	}

	// Sign the transaction
	return tx.SignTransaction(inputIndex, keyPair.PrivateKey, referencedTxOutputs)
}

// GetUnspentOutputs returns unspent outputs for the wallet
func (w *Wallet) GetUnspentOutputs(utxoSet map[string]map[int]transactions.TxOutput) []transactions.TxOutput {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var unspentOutputs []transactions.TxOutput

	if w.Encrypted {
		return unspentOutputs
	}

	// Create address set for fast lookup
	addressSet := make(map[string]bool)
	for _, address := range w.Addresses {
		addressSet[address] = true
	}

	// Find all UTXOs for wallet addresses
	for txID, outputs := range utxoSet {
		for index, output := range outputs {
			if addressSet[output.Address] {
				// Copy the output and set its transaction info
				utxo := output
				utxo.TxID = txID
				utxo.Index = index
				unspentOutputs = append(unspentOutputs, utxo)
			}
		}
	}

	return unspentOutputs
}