package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyPair represents a cryptographic key pair
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey `json:"-"`
	PublicKey  *ecdsa.PublicKey  `json:"-"`
	Address    string            `json:"address"`
}

// KeyStorage handles serialization and storage of keys
type KeyStorage struct {
	PrivateKeyPEM string `json:"private_key_pem"`
	PublicKeyPEM  string `json:"public_key_pem"`
	Address       string `json:"address"`
}

// NewKeyPair generates a new ECDSA key pair using secp256k1 curve
func NewKeyPair() (*KeyPair, error) {
	// Use secp256k1 curve (same as Bitcoin)
	curve := elliptic.P256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey := &privateKey.PublicKey
	address, err := generateAddress(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// generateAddress creates an address from a public key
func generateAddress(publicKey *ecdsa.PublicKey) (string, error) {
	// Serialize public key in uncompressed format (0x04 + X + Y)
	pubKeyBytes := make([]byte, 0, 65)
	pubKeyBytes = append(pubKeyBytes, 0x04)
	pubKeyBytes = append(pubKeyBytes, publicKey.X.Bytes()...)
	pubKeyBytes = append(pubKeyBytes, publicKey.Y.Bytes()...)

	// Double SHA256 for better collision resistance (like Bitcoin)
	hash1 := sha256.Sum256(pubKeyBytes)
	hash2 := sha256.Sum256(hash1[:])

	// Take first 20 bytes of double hash as address bytes
	addressBytes := hash2[:20]

	// Calculate checksum by double hashing the address bytes
	checksumHash1 := sha256.Sum256(addressBytes)
	checksumHash2 := sha256.Sum256(checksumHash1[:])
	checksum := checksumHash2[:4]

	// Combine address bytes and checksum
	fullAddress := append(addressBytes, checksum...)

	// Convert to hex string with prefix
	address := "0x" + hex.EncodeToString(fullAddress)

	return address, nil
}

// SerializePrivateKey converts private key to PEM format
func SerializePrivateKey(privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("private key is nil")
	}
	derBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	})

	return string(privateKeyPEM), nil
}

// SerializePublicKey converts public key to PEM format
func SerializePublicKey(publicKey *ecdsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", fmt.Errorf("public key is nil")
	}
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	})

	return string(publicKeyPEM), nil
}

// DeserializePrivateKey converts PEM format back to private key
func DeserializePrivateKey(pemData string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}

// DeserializePublicKey converts PEM format back to public key
func DeserializePublicKey(pemData string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}

	return ecPublicKey, nil
}

// ToStorage converts KeyPair to KeyStorage for serialization
func (kp *KeyPair) ToStorage() (*KeyStorage, error) {
	privateKeyPEM, err := SerializePrivateKey(kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize private key: %w", err)
	}

	publicKeyPEM, err := SerializePublicKey(kp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize public key: %w", err)
	}

	return &KeyStorage{
		PrivateKeyPEM: privateKeyPEM,
		PublicKeyPEM:  publicKeyPEM,
		Address:       kp.Address,
	}, nil
}

// FromStorage converts KeyStorage back to KeyPair
func FromStorage(storage *KeyStorage) (*KeyPair, error) {
	privateKey, err := DeserializePrivateKey(storage.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize private key: %w", err)
	}

	publicKey, err := DeserializePublicKey(storage.PublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize public key: %w", err)
	}

	// Verify that the public key matches the private key
	if !privateKey.PublicKey.Equal(publicKey) {
		return nil, fmt.Errorf("public key does not match private key")
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    storage.Address,
	}, nil
}

// SaveToFile saves the key pair to a file
func (kp *KeyPair) SaveToFile(filename string) error {
	storage, err := kp.ToStorage()
	if err != nil {
		return fmt.Errorf("failed to convert key pair to storage format: %w", err)
	}

	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key storage: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// LoadFromFile loads a key pair from a file
func LoadFromFile(filename string) (*KeyPair, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	var storage KeyStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key storage: %w", err)
	}

	return FromStorage(&storage)
}

// GetPublicKeyBytes returns the public key as bytes (uncompressed format)
func (kp *KeyPair) GetPublicKeyBytes() []byte {
	// Uncompressed format: 0x04 + X + Y
	pubKeyBytes := make([]byte, 0, 65)
	pubKeyBytes = append(pubKeyBytes, 0x04)
	pubKeyBytes = append(pubKeyBytes, kp.PublicKey.X.Bytes()...)
	pubKeyBytes = append(pubKeyBytes, kp.PublicKey.Y.Bytes()...)
	return pubKeyBytes
}

// GetPrivateKeyBytes returns the private key as bytes
func (kp *KeyPair) GetPrivateKeyBytes() []byte {
	return kp.PrivateKey.D.Bytes()
}

// ValidateAddress checks if an address is valid
func ValidateAddress(address string) bool {
	if !strings.HasPrefix(address, "0x") {
		return false
	}

	hexPart := strings.TrimPrefix(address, "0x")
	if len(hexPart) != 48 { // 24 bytes = 48 hex characters (20 address + 4 checksum)
		return false
	}

	decodedBytes, err := hex.DecodeString(hexPart)
	if err != nil {
		return false
	}

	// Verify checksum
	if len(decodedBytes) != 24 {
		return false
	}

	addressBytes := decodedBytes[:20]
	checksum := decodedBytes[20:24]

	// Recalculate checksum
	hash1 := sha256.Sum256(addressBytes)
	hash2 := sha256.Sum256(hash1[:])
	expectedChecksum := hash2[:4]

	// Compare checksums
	return bytes.Equal(checksum, expectedChecksum)
}

// DerivePublicKey derives public key from private key
func DerivePublicKey(privateKey *ecdsa.PrivateKey) *ecdsa.PublicKey {
	return &privateKey.PublicKey
}

// GetKeyPairFromPrivate creates a KeyPair from just a private key
func GetKeyPairFromPrivate(privateKey *ecdsa.PrivateKey) (*KeyPair, error) {
	publicKey := DerivePublicKey(privateKey)
	address, err := generateAddress(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// GenerateMultipleKeys generates multiple key pairs for testing
func GenerateMultipleKeys(count int) ([]*KeyPair, error) {
	keys := make([]*KeyPair, count)

	for i := 0; i < count; i++ {
		keyPair, err := NewKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair %d: %w", i, err)
		}
		keys[i] = keyPair
	}

	return keys, nil
}

// CompareAddresses safely compares two addresses
func CompareAddresses(addr1, addr2 string) bool {
	return strings.EqualFold(addr1, addr2)
}

// IsValidKeyPair checks if a key pair is valid
func (kp *KeyPair) IsValidKeyPair() bool {
	if kp.PrivateKey == nil || kp.PublicKey == nil {
		return false
	}

	// Check if public key matches private key
	derivedPubKey := DerivePublicKey(kp.PrivateKey)
	if !derivedPubKey.Equal(kp.PublicKey) {
		return false
	}

	// Check if address is valid
	if !ValidateAddress(kp.Address) {
		return false
	}

	// Check if address matches public key
	expectedAddr, err := generateAddress(kp.PublicKey)
	if err != nil {
		return false
	}

	return CompareAddresses(kp.Address, expectedAddr)
}

// GetKeyInfo returns information about the key pair
func (kp *KeyPair) GetKeyInfo() map[string]interface{} {
	return map[string]interface{}{
		"address":           kp.Address,
		"curve":             kp.PrivateKey.Curve.Params().Name,
		"public_key_x":      kp.PublicKey.X.String(),
		"public_key_y":      kp.PublicKey.Y.String(),
		"private_key_d":     kp.PrivateKey.D.String(),
		"public_key_bytes":  hex.EncodeToString(kp.GetPublicKeyBytes()),
		"private_key_bytes": hex.EncodeToString(kp.GetPrivateKeyBytes()),
	}
}
