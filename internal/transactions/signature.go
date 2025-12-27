package transactions

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

func (tx *Transaction) SignTransaction(inputIndex int, privateKey *ecdsa.PrivateKey, referencedTxOutputs []TxOutput) error {
	if inputIndex >= len(tx.Inputs) {
		return fmt.Errorf("input index %d out of range", inputIndex)
	}

	// Get the referenced output for this input
	if tx.Inputs[inputIndex].TxID == "" {
		return fmt.Errorf("input %d has empty tx ID", inputIndex)
	}

	// Create the message to sign (transaction data + referenced output)
	message, err := tx.getSigningMessage(inputIndex, referencedTxOutputs)
	if err != nil {
		return fmt.Errorf("failed to create signing message: %w", err)
	}

	// Sign the message
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, message)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Store signature and public key
	tx.Inputs[inputIndex].Signature = hex.EncodeToString(signature)
	tx.Inputs[inputIndex].PublicKey = encodePublicKey(&privateKey.PublicKey)

	return nil
}

// getSigningMessage creates the message to be signed
func (tx *Transaction) getSigningMessage(inputIndex int, referencedTxOutputs []TxOutput) ([]byte, error) {
	// Hash the transaction data (excluding signatures and public keys)
	txCopy := tx.copyWithEmptySignatures()
	txData, err := json.Marshal(txCopy)
	if err != nil {
		return nil, err
	}

	// Include the referenced output data
	if inputIndex < len(referencedTxOutputs) {
		outputData, err := json.Marshal(referencedTxOutputs[inputIndex])
		if err != nil {
			return nil, err
		}

		// Combine transaction data and referenced output data
		combinedData := append(txData, outputData...)
		hash := sha256.Sum256(combinedData)
		return hash[:], nil
	}

	// If no referenced output, just hash the transaction data
	hash := sha256.Sum256(txData)
	return hash[:], nil
}

// copyWithEmptySignatures creates a copy of the transaction with empty signatures
func (tx *Transaction) copyWithEmptySignatures() *Transaction {
	copy := &Transaction{
		ID:        tx.ID,
		Inputs:    make([]TxInput, len(tx.Inputs)),
		Outputs:   tx.Outputs,
		Timestamp: tx.Timestamp,
	}

	for i, input := range tx.Inputs {
		copy.Inputs[i] = TxInput{
			TxID:      input.TxID,
			Index:     input.Index,
			Signature: "", // Empty for signing
			PublicKey: "", // Empty for signing
		}
	}

	return copy
}

// VerifyInputSignature verifies the signature of a transaction input
func (tx *Transaction) VerifyInputSignature(inputIndex int, referencedTxOutputs []TxOutput) error {
	if inputIndex >= len(tx.Inputs) {
		return fmt.Errorf("input index %d out of range", inputIndex)
	}

	input := tx.Inputs[inputIndex]

	// Decode signature
	signature, err := hex.DecodeString(input.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Decode public key
	publicKey, err := decodePublicKey(input.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Create the message that was signed
	message, err := tx.getSigningMessage(inputIndex, referencedTxOutputs)
	if err != nil {
		return fmt.Errorf("failed to create signing message: %w", err)
	}

	// Verify the signature
	if !ecdsa.VerifyASN1(publicKey, message, signature) {
		return fmt.Errorf("invalid signature for input %d", inputIndex)
	}

	return nil
}

// encodePublicKey encodes a public key to hex string
func encodePublicKey(publicKey *ecdsa.PublicKey) string {
	// Use uncompressed format: 0x04 + X + Y
	keyBytes := make([]byte, 0, 65)
	keyBytes = append(keyBytes, 0x04)
	keyBytes = append(keyBytes, publicKey.X.Bytes()...)
	keyBytes = append(keyBytes, publicKey.Y.Bytes()...)
	return hex.EncodeToString(keyBytes)
}

// decodePublicKey decodes a hex string to public key
func decodePublicKey(hexKey string) (*ecdsa.PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	if len(keyBytes) != 65 || keyBytes[0] != 0x04 {
		return nil, fmt.Errorf("invalid public key format")
	}

	// Extract X and Y coordinates
	x := new(big.Int).SetBytes(keyBytes[1:33])
	y := new(big.Int).SetBytes(keyBytes[33:65])

	// Create public key
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	// Verify the point is on the curve
	if !publicKey.Curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("public key point is not on the curve")
	}

	return publicKey, nil
}

// GetInputPublicKey extracts the public key from a transaction input
func (tx *Transaction) GetInputPublicKey(inputIndex int) (*ecdsa.PublicKey, error) {
	if inputIndex >= len(tx.Inputs) {
		return nil, fmt.Errorf("input index %d out of range", inputIndex)
	}

	return decodePublicKey(tx.Inputs[inputIndex].PublicKey)
}

// SignAllInputs signs all inputs of a transaction with the provided private keys
func (tx *Transaction) SignAllInputs(privateKeys []*ecdsa.PrivateKey, referencedTxOutputs [][]TxOutput) error {
	if len(privateKeys) != len(tx.Inputs) {
		return fmt.Errorf("number of private keys (%d) doesn't match number of inputs (%d)",
			len(privateKeys), len(tx.Inputs))
	}

	for i, privateKey := range privateKeys {
		var outputs []TxOutput
		if i < len(referencedTxOutputs) {
			outputs = referencedTxOutputs[i]
		}

		err := tx.SignTransaction(i, privateKey, outputs)
		if err != nil {
			return fmt.Errorf("failed to sign input %d: %w", i, err)
		}
	}

	return nil
}

// VerifyAllSignatures verifies all signatures in the transaction
func (tx *Transaction) VerifyAllSignatures(referencedTxOutputs [][]TxOutput) error {
	if tx.IsCoinbase() {
		return nil // Coinbase transactions don't need signatures
	}

	for i := range tx.Inputs {
		var outputs []TxOutput
		if i < len(referencedTxOutputs) {
			outputs = referencedTxOutputs[i]
		}

		err := tx.VerifyInputSignature(i, outputs)
		if err != nil {
			return fmt.Errorf("failed to verify input %d: %w", i, err)
		}
	}

	return nil
}

// GetSignatureInfo returns information about transaction signatures
func (tx *Transaction) GetSignatureInfo() map[string]interface{} {
	info := map[string]interface{}{
		"total_inputs":    len(tx.Inputs),
		"signed_inputs":   0,
		"unsigned_inputs": 0,
		"is_coinbase":     tx.IsCoinbase(),
	}

	if tx.IsCoinbase() {
		return info
	}

	for _, input := range tx.Inputs {
		if input.Signature != "" && input.PublicKey != "" {
			info["signed_inputs"] = info["signed_inputs"].(int) + 1
		} else {
			info["unsigned_inputs"] = info["unsigned_inputs"].(int) + 1
		}
	}

	return info
}
