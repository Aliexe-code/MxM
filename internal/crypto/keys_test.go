package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewKeyPair(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if keyPair.PrivateKey == nil {
		t.Fatal("Private key is nil")
	}

	if keyPair.PublicKey == nil {
		t.Fatal("Public key is nil")
	}

	if keyPair.Address == "" {
		t.Fatal("Address is empty")
	}

	if !strings.HasPrefix(keyPair.Address, "0x") {
		t.Errorf("Address should start with 0x, got: %s", keyPair.Address)
	}

	if len(keyPair.Address) != 42 { // 0x + 40 hex chars
		t.Errorf("Address should be 42 characters long, got: %d", len(keyPair.Address))
	}
}

func TestGenerateAddress(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	address := keyPair.Address
	if !ValidateAddress(address) {
		t.Errorf("Generated address is invalid: %s", address)
	}

	// Test address generation from public key directly
	derivedAddr, err := generateAddress(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to generate address from public key: %v", err)
	}

	if derivedAddr != address {
		t.Errorf("Address mismatch: expected %s, got %s", address, derivedAddr)
	}
}

func TestValidateAddress(t *testing.T) {
	validAddresses := []string{
		"0x1234567890123456789012345678901234567890",
		"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		"0x0000000000000000000000000000000000000000",
	}

	invalidAddresses := []string{
		"1234567890123456789012345678901234567890",    // missing 0x
		"0x123456789012345678901234567890123456789",   // too short
		"0x12345678901234567890123456789012345678900", // too long
		"0xg234567890123456789012345678901234567890",  // invalid hex
		"",   // empty
		"0x", // just prefix
	}

	for _, addr := range validAddresses {
		if !ValidateAddress(addr) {
			t.Errorf("Valid address rejected: %s", addr)
		}
	}

	for _, addr := range invalidAddresses {
		if ValidateAddress(addr) {
			t.Errorf("Invalid address accepted: %s", addr)
		}
	}
}

func TestSerializeDeserializePrivateKey(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Serialize private key
	pemData, err := SerializePrivateKey(keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to serialize private key: %v", err)
	}

	if !strings.Contains(pemData, "EC PRIVATE KEY") {
		t.Errorf("PEM data should contain 'EC PRIVATE KEY', got: %s", pemData[:50])
	}

	// Deserialize private key
	deserializedKey, err := DeserializePrivateKey(pemData)
	if err != nil {
		t.Fatalf("Failed to deserialize private key: %v", err)
	}

	// Compare keys
	if keyPair.PrivateKey.D.Cmp(deserializedKey.D) != 0 {
		t.Error("Private keys don't match after serialization/deserialization")
	}

	if !keyPair.PrivateKey.PublicKey.Equal(&deserializedKey.PublicKey) {
		t.Error("Public keys don't match after private key serialization/deserialization")
	}
}

func TestSerializeDeserializePublicKey(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Serialize public key
	pemData, err := SerializePublicKey(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to serialize public key: %v", err)
	}

	if !strings.Contains(pemData, "PUBLIC KEY") {
		t.Errorf("PEM data should contain 'PUBLIC KEY', got: %s", pemData[:50])
	}

	// Deserialize public key
	deserializedKey, err := DeserializePublicKey(pemData)
	if err != nil {
		t.Fatalf("Failed to deserialize public key: %v", err)
	}

	// Compare keys
	if !keyPair.PublicKey.Equal(deserializedKey) {
		t.Error("Public keys don't match after serialization/deserialization")
	}
}

func TestKeyStorageConversion(t *testing.T) {
	originalKeyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Convert to storage format
	storage, err := originalKeyPair.ToStorage()
	if err != nil {
		t.Fatalf("Failed to convert to storage format: %v", err)
	}

	// Convert back from storage format
	restoredKeyPair, err := FromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to restore from storage format: %v", err)
	}

	// Verify all components match
	if originalKeyPair.PrivateKey.D.Cmp(restoredKeyPair.PrivateKey.D) != 0 {
		t.Error("Private keys don't match after storage conversion")
	}

	if !originalKeyPair.PublicKey.Equal(restoredKeyPair.PublicKey) {
		t.Error("Public keys don't match after storage conversion")
	}

	if originalKeyPair.Address != restoredKeyPair.Address {
		t.Error("Addresses don't match after storage conversion")
	}
}

func TestSaveLoadKeyPair(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create temporary directory
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "test_key.json")

	// Save key pair
	err = keyPair.SaveToFile(filename)
	if err != nil {
		t.Fatalf("Failed to save key pair: %v", err)
	}

	// Verify file exists and has correct permissions
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions should be 0600, got: %v", info.Mode().Perm())
	}

	// Load key pair
	loadedKeyPair, err := LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load key pair: %v", err)
	}

	// Verify loaded key pair matches original
	if keyPair.PrivateKey.D.Cmp(loadedKeyPair.PrivateKey.D) != 0 {
		t.Error("Private keys don't match after save/load")
	}

	if !keyPair.PublicKey.Equal(loadedKeyPair.PublicKey) {
		t.Error("Public keys don't match after save/load")
	}

	if keyPair.Address != loadedKeyPair.Address {
		t.Error("Addresses don't match after save/load")
	}
}

func TestDerivePublicKey(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	derivedPublicKey := DerivePublicKey(keyPair.PrivateKey)
	if !derivedPublicKey.Equal(keyPair.PublicKey) {
		t.Error("Derived public key doesn't match original public key")
	}
}

func TestGetKeyPairFromPrivate(t *testing.T) {
	originalKeyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	derivedKeyPair, err := GetKeyPairFromPrivate(originalKeyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create key pair from private key: %v", err)
	}

	if originalKeyPair.PrivateKey.D.Cmp(derivedKeyPair.PrivateKey.D) != 0 {
		t.Error("Private keys don't match")
	}

	if !originalKeyPair.PublicKey.Equal(derivedKeyPair.PublicKey) {
		t.Error("Public keys don't match")
	}

	if originalKeyPair.Address != derivedKeyPair.Address {
		t.Error("Addresses don't match")
	}
}

func TestGenerateMultipleKeys(t *testing.T) {
	count := 5
	keys, err := GenerateMultipleKeys(count)
	if err != nil {
		t.Fatalf("Failed to generate multiple keys: %v", err)
	}

	if len(keys) != count {
		t.Errorf("Expected %d keys, got %d", count, len(keys))
	}

	// Check all keys are unique
	addresses := make(map[string]bool)
	for _, key := range keys {
		if addresses[key.Address] {
			t.Errorf("Duplicate address found: %s", key.Address)
		}
		addresses[key.Address] = true

		if !key.IsValidKeyPair() {
			t.Errorf("Invalid key pair with address: %s", key.Address)
		}
	}
}

func TestCompareAddresses(t *testing.T) {
	addr1 := "0x1234567890123456789012345678901234567890"
	addr2 := "0x1234567890123456789012345678901234567890"
	addr3 := "0x1234567890123456789012345678901234567891"
	addr4 := "0X1234567890123456789012345678901234567890" // uppercase

	if !CompareAddresses(addr1, addr2) {
		t.Error("Same addresses should be equal")
	}

	if CompareAddresses(addr1, addr3) {
		t.Error("Different addresses should not be equal")
	}

	if !CompareAddresses(addr1, addr4) {
		t.Error("Addresses should be case insensitive")
	}
}

func TestIsValidKeyPair(t *testing.T) {
	validKeyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if !validKeyPair.IsValidKeyPair() {
		t.Error("Valid key pair should pass validation")
	}

	// Test invalid key pair (nil private key)
	invalidKeyPair := &KeyPair{
		PrivateKey: nil,
		PublicKey:  validKeyPair.PublicKey,
		Address:    validKeyPair.Address,
	}

	if invalidKeyPair.IsValidKeyPair() {
		t.Error("Key pair with nil private key should be invalid")
	}

	// Test key pair with mismatched public key
	mismatchedKeyPair, _ := NewKeyPair()
	invalidKeyPair2 := &KeyPair{
		PrivateKey: validKeyPair.PrivateKey,
		PublicKey:  mismatchedKeyPair.PublicKey,
		Address:    validKeyPair.Address,
	}

	if invalidKeyPair2.IsValidKeyPair() {
		t.Error("Key pair with mismatched public key should be invalid")
	}

	// Test key pair with invalid address
	invalidAddrKeyPair := &KeyPair{
		PrivateKey: validKeyPair.PrivateKey,
		PublicKey:  validKeyPair.PublicKey,
		Address:    "invalid_address",
	}

	if invalidAddrKeyPair.IsValidKeyPair() {
		t.Error("Key pair with invalid address should be invalid")
	}
}

func TestGetKeyInfo(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	info := keyPair.GetKeyInfo()

	expectedKeys := []string{
		"address",
		"curve",
		"public_key_x",
		"public_key_y",
		"private_key_d",
		"public_key_bytes",
		"private_key_bytes",
	}

	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Missing key in info: %s", key)
		}
	}

	if info["address"] != keyPair.Address {
		t.Error("Address in info doesn't match")
	}

	if info["curve"] != "P-256" {
		t.Errorf("Expected curve P-256, got: %s", info["curve"])
	}
}

func TestGetPublicKeyBytes(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	pubKeyBytes := keyPair.GetPublicKeyBytes()
	if len(pubKeyBytes) == 0 {
		t.Error("Public key bytes should not be empty")
	}

	// Verify the bytes can be unmarshaled back to a public key
	curve := elliptic.P256()
	if len(pubKeyBytes) != 65 || pubKeyBytes[0] != 0x04 {
		t.Error("Invalid public key byte format")
	}

	// Extract X and Y coordinates
	x := new(big.Int).SetBytes(pubKeyBytes[1:33])
	y := new(big.Int).SetBytes(pubKeyBytes[33:65])

	// Verify the point is on the curve
	if !curve.IsOnCurve(x, y) {
		t.Error("Public key point is not on the curve")
	}

	reconstructedPubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	if !keyPair.PublicKey.Equal(reconstructedPubKey) {
		t.Error("Reconstructed public key doesn't match original")
	}
}

func TestGetPrivateKeyBytes(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	privKeyBytes := keyPair.GetPrivateKeyBytes()
	if len(privKeyBytes) == 0 {
		t.Error("Private key bytes should not be empty")
	}

	// Verify the bytes represent the same private key
	d := new(big.Int).SetBytes(privKeyBytes)
	if d.Cmp(keyPair.PrivateKey.D) != 0 {
		t.Error("Private key bytes don't match original")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test invalid PEM data
	invalidPEM := "invalid pem data"
	_, err := DeserializePrivateKey(invalidPEM)
	if err == nil {
		t.Error("Should fail to deserialize invalid PEM data")
	}

	_, err = DeserializePublicKey(invalidPEM)
	if err == nil {
		t.Error("Should fail to deserialize invalid PEM data")
	}

	// Test valid PEM but invalid DER
	invalidDERBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: []byte("invalid der"),
	}
	invalidDERPEM := pem.EncodeToMemory(invalidDERBlock)
	_, err = DeserializePrivateKey(string(invalidDERPEM))
	if err == nil {
		t.Error("Should fail to deserialize invalid DER")
	}

	invalidPubDERBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: []byte("invalid der"),
	}
	invalidPubDERPEM := pem.EncodeToMemory(invalidPubDERBlock)
	_, err = DeserializePublicKey(string(invalidPubDERPEM))
	if err == nil {
		t.Error("Should fail to deserialize invalid DER")
	}

	// Test ToStorage with nil private key
	testKeyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate test key pair: %v", err)
	}
	nilPrivKeyPair := &KeyPair{
		PrivateKey: nil,
		PublicKey:  testKeyPair.PublicKey,
		Address:    testKeyPair.Address,
	}
	_, err = nilPrivKeyPair.ToStorage()
	if err == nil {
		t.Error("Should fail to convert to storage with nil private key")
	}

	// Test ToStorage with nil public key
	nilPubKeyPair := &KeyPair{
		PrivateKey: testKeyPair.PrivateKey,
		PublicKey:  nil,
		Address:    testKeyPair.Address,
	}
	_, err = nilPubKeyPair.ToStorage()
	if err == nil {
		t.Error("Should fail to convert to storage with nil public key")
	}

	// Test loading non-existent file
	_, err = LoadFromFile("non_existent_file.json")
	if err == nil {
		t.Error("Should fail to load non-existent file")
	}

	// Test storage with mismatched keys
	keyPair1, _ := NewKeyPair()
	keyPair2, _ := NewKeyPair()

	invalidStorage := &KeyStorage{
		PrivateKeyPEM: func() string {
			pem, _ := SerializePrivateKey(keyPair1.PrivateKey)
			return pem
		}(),
		PublicKeyPEM: func() string {
			pem, _ := SerializePublicKey(keyPair2.PublicKey)
			return pem
		}(),
		Address: keyPair1.Address,
	}

	_, err = FromStorage(invalidStorage)
	if err == nil {
		t.Error("Should fail to restore from storage with mismatched keys")
	}

	// Test FromStorage with invalid private key PEM
	invalidPrivateStorage := &KeyStorage{
		PrivateKeyPEM: "invalid pem",
		PublicKeyPEM: func() string {
			pem, _ := SerializePublicKey(keyPair1.PublicKey)
			return pem
		}(),
		Address: keyPair1.Address,
	}
	_, err = FromStorage(invalidPrivateStorage)
	if err == nil {
		t.Error("Should fail to restore from storage with invalid private key PEM")
	}

	// Test FromStorage with invalid public key PEM
	invalidPublicStorage := &KeyStorage{
		PrivateKeyPEM: func() string {
			pem, _ := SerializePrivateKey(keyPair1.PrivateKey)
			return pem
		}(),
		PublicKeyPEM: "invalid pem",
		Address:      keyPair1.Address,
	}
	_, err = FromStorage(invalidPublicStorage)
	if err == nil {
		t.Error("Should fail to restore from storage with invalid public key PEM")
	}

	// Test loading file with invalid JSON
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidFile, []byte("{invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}
	_, err = LoadFromFile(invalidFile)
	if err == nil {
		t.Error("Should fail to load invalid JSON")
	}

	// Test saving to directory without write permission
	noWriteDir := filepath.Join(tempDir, "no_write")
	err = os.Mkdir(noWriteDir, 0000)
	if err != nil {
		t.Fatalf("Failed to create no-write dir: %v", err)
	}
	defer os.Chmod(noWriteDir, 0700) // restore for cleanup
	noWriteFile := filepath.Join(noWriteDir, "test.json")
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	err = keyPair.SaveToFile(noWriteFile)
	if err == nil {
		t.Error("Should fail to save to no-write directory")
	}

	// Test SaveToFile with MkdirAll error (trying to create dir where file exists)
	fileInsteadOfDir := filepath.Join(tempDir, "file_not_dir")
	err = os.WriteFile(fileInsteadOfDir, []byte("dummy"), 0600)
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	mkdirFailFile := filepath.Join(fileInsteadOfDir, "sub", "test.json")
	err = keyPair.SaveToFile(mkdirFailFile)
	if err == nil {
		t.Error("Should fail to save when MkdirAll fails")
	}

	// Test deserializing RSA private key as EC
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	rsaPrivateDER := x509.MarshalPKCS1PrivateKey(rsaPrivateKey)
	rsaPrivatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: rsaPrivateDER,
	})
	_, err = DeserializePrivateKey(string(rsaPrivatePEM))
	if err == nil {
		t.Error("Should fail to deserialize RSA private key as EC")
	}

	// Test deserializing RSA public key as EC
	rsaPublicKey := &rsaPrivateKey.PublicKey
	rsaPublicDER, err := x509.MarshalPKIXPublicKey(rsaPublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal RSA public key: %v", err)
	}
	rsaPublicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: rsaPublicDER,
	})
	_, err = DeserializePublicKey(string(rsaPublicPEM))
	if err == nil {
		t.Error("Should fail to deserialize RSA public key as EC")
	}
}

func TestKeyPairConsistency(t *testing.T) {
	// Generate a key pair and verify all components are consistent
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Verify address is derived correctly from public key
	expectedAddr, err := generateAddress(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to generate expected address: %v", err)
	}

	if keyPair.Address != expectedAddr {
		t.Errorf("Address inconsistency: expected %s, got %s", expectedAddr, keyPair.Address)
	}

	// Verify public key can be derived from private key
	derivedPubKey := DerivePublicKey(keyPair.PrivateKey)
	if !derivedPubKey.Equal(keyPair.PublicKey) {
		t.Error("Public key derivation failed")
	}

	// Test multiple serialization/deserialization cycles
	for i := 0; i < 3; i++ {
		storage, err := keyPair.ToStorage()
		if err != nil {
			t.Fatalf("Failed to convert to storage (cycle %d): %v", i, err)
		}

		restored, err := FromStorage(storage)
		if err != nil {
			t.Fatalf("Failed to restore from storage (cycle %d): %v", i, err)
		}

		if keyPair.Address != restored.Address {
			t.Errorf("Address mismatch after cycle %d", i)
		}
	}
}

func BenchmarkNewKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewKeyPair()
		if err != nil {
			b.Fatalf("Failed to generate key pair: %v", err)
		}
	}
}

func BenchmarkSerializeDeserialize(b *testing.B) {
	keyPair, err := NewKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	pemData, err := SerializePrivateKey(keyPair.PrivateKey)
	if err != nil {
		b.Fatalf("Failed to serialize private key: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DeserializePrivateKey(pemData)
		if err != nil {
			b.Fatalf("Failed to deserialize private key: %v", err)
		}
	}
}

func BenchmarkGenerateAddress(b *testing.B) {
	keyPair, err := NewKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generateAddress(keyPair.PublicKey)
		if err != nil {
			b.Fatalf("Failed to generate address: %v", err)
		}
	}
}
