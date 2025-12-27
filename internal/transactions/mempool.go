package transactions

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MempoolConfig defines configuration for the mempool
type MempoolConfig struct {
	MaxSize       int           // Maximum number of transactions in pool
	MaxAge        time.Duration // Maximum age of transactions before removal
	MinFeeRate    float64       // Minimum fee rate to accept transactions
	MaxTxSize     int           // Maximum transaction size in bytes
	ValidateTx    bool          // Whether to validate transactions before adding
	CleanupInterval time.Duration // Interval for cleaning old transactions
}

// DefaultMempoolConfig returns default mempool configuration
func DefaultMempoolConfig() MempoolConfig {
	return MempoolConfig{
		MaxSize:        5000,
		MaxAge:         24 * time.Hour,
		MinFeeRate:     0.00001,
		MaxTxSize:      100000, // 100KB
		ValidateTx:     true,
		CleanupInterval: 10 * time.Minute,
	}
}

// MempoolEntry represents a transaction in the mempool with metadata
type MempoolEntry struct {
	Transaction *Transaction
	FeeRate     float64
	Size        int
	AddedAt     time.Time
	Priority    int
}

// Mempool manages pending transactions
type Mempool struct {
	config      MempoolConfig
	transactions map[string]*MempoolEntry // txID -> entry
	byAddress   map[string][]string      // address -> []txID
	priorityQueue *PriorityQueue
	mu         sync.RWMutex
	lastCleanup time.Time
}

// NewMempool creates a new mempool with default configuration
func NewMempool() *Mempool {
	return NewMempoolWithConfig(DefaultMempoolConfig())
}

// NewMempoolWithConfig creates a new mempool with custom configuration
func NewMempoolWithConfig(config MempoolConfig) *Mempool {
	mp := &Mempool{
		config:        config,
		transactions:  make(map[string]*MempoolEntry),
		byAddress:     make(map[string][]string),
		priorityQueue: &PriorityQueue{},
		lastCleanup:   time.Now(),
	}
	heap.Init(mp.priorityQueue)
	return mp
}

// AddTransaction adds a transaction to the mempool
func (mp *Mempool) AddTransaction(tx *Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if transaction already exists
	if _, exists := mp.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction %s already exists in mempool", tx.ID)
	}

	// Validate transaction if enabled
	if mp.config.ValidateTx {
		if err := tx.ValidateBasic(); err != nil {
			return fmt.Errorf("transaction validation failed: %w", err)
		}
	}

	// Check transaction size
	txSize := EstimateTransactionSize(len(tx.Inputs), len(tx.Outputs))
	if txSize > mp.config.MaxTxSize {
		return fmt.Errorf("transaction size %d exceeds maximum %d", txSize, mp.config.MaxTxSize)
	}

	// Calculate fee rate
	fee := tx.GetFee()
	if fee <= 0 {
		return fmt.Errorf("transaction must have positive fee")
	}

	feeRate := fee / float64(txSize)
	if feeRate < mp.config.MinFeeRate {
		return fmt.Errorf("fee rate %.8f below minimum %.8f", feeRate, mp.config.MinFeeRate)
	}

	// Check pool size limit
	if len(mp.transactions) >= mp.config.MaxSize {
		// Try to evict lowest priority transaction
		if err := mp.evictLowestPriority(); err != nil {
			return fmt.Errorf("mempool is full and cannot evict transactions: %w", err)
		}
	}

	// Create mempool entry
	entry := &MempoolEntry{
		Transaction: tx,
		FeeRate:     feeRate,
		Size:        txSize,
		AddedAt:     time.Now(),
		Priority:    mp.calculatePriority(tx, feeRate),
	}

	// Add to mempool
	mp.transactions[tx.ID] = entry

	// Update address index
	// For now, we'll use a simplified approach
	// In a real implementation, we'd need to resolve input addresses
	mp.byAddress["unknown"] = append(mp.byAddress["unknown"], tx.ID)

	for _, output := range tx.Outputs {
		mp.byAddress[output.Address] = append(mp.byAddress[output.Address], tx.ID)
	}

	// Add to priority queue
	heap.Push(mp.priorityQueue, entry)

	// Periodic cleanup
	if time.Since(mp.lastCleanup) > mp.config.CleanupInterval {
		go mp.cleanupOldTransactions()
		mp.lastCleanup = time.Now()
	}

	return nil
}

// GetTransaction retrieves a transaction from the mempool
func (mp *Mempool) GetTransaction(txID string) (*Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if entry, exists := mp.transactions[txID]; exists {
		return entry.Transaction, true
	}
	return nil, false
}

// RemoveTransaction removes a transaction from the mempool
func (mp *Mempool) RemoveTransaction(txID string) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	entry, exists := mp.transactions[txID]
	if !exists {
		return false
	}

	// Remove from main storage
	delete(mp.transactions, txID)

	// Remove from address index
	for _, txIDs := range mp.byAddress {
		for i, id := range txIDs {
			if id == txID {
				mp.byAddress[txIDs[0]] = append(txIDs[:i], txIDs[i+1:]...)
				break
			}
		}
	}

	// Remove from priority queue (mark as removed)
	entry.Priority = -1 // Mark as removed

	return true
}

// GetTransactionsByAddress returns all transactions involving an address
func (mp *Mempool) GetTransactionsByAddress(address string) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txIDs, exists := mp.byAddress[address]
	if !exists {
		return []*Transaction{}
	}

	transactions := make([]*Transaction, 0, len(txIDs))
	for _, txID := range txIDs {
		if entry, exists := mp.transactions[txID]; exists {
			transactions = append(transactions, entry.Transaction)
		}
	}

	return transactions
}

// GetTransactionsByFeeRate returns transactions sorted by fee rate
func (mp *Mempool) GetTransactionsByFeeRate(limit int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	entries := make([]*MempoolEntry, 0, len(mp.transactions))
	for _, entry := range mp.transactions {
		entries = append(entries, entry)
	}

	// Sort by fee rate (descending)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FeeRate > entries[j].FeeRate
	})

	// Apply limit
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	transactions := make([]*Transaction, len(entries))
	for i, entry := range entries {
		transactions[i] = entry.Transaction
	}

	return transactions
}

// GetTransactionsForBlock returns the best transactions for a new block
func (mp *Mempool) GetTransactionsForBlock(maxSize, maxCount int) []*Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Create a copy of the priority queue
	pq := &PriorityQueue{}
	heap.Init(pq)

	// Add all valid transactions
	for _, entry := range mp.transactions {
		if entry.Priority >= 0 { // Skip removed entries
			heap.Push(pq, entry)
		}
	}

	// Select transactions for block
	var selected []*Transaction
	var currentSize int

	for pq.Len() > 0 && (maxCount == 0 || len(selected) < maxCount) {
		entry := heap.Pop(pq).(*MempoolEntry)
		
		if currentSize+entry.Size > maxSize {
			continue
		}

		selected = append(selected, entry.Transaction)
		currentSize += entry.Size
	}

	return selected
}

// ValidateAndRemoveInvalid removes invalid transactions from the mempool
func (mp *Mempool) ValidateAndRemoveInvalid(utxoSet interface{}) []string { // TODO: Replace interface{} with UTXOSet type
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var removed []string

	for txID, entry := range mp.transactions {
		// Basic validation
		if err := entry.Transaction.ValidateBasic(); err != nil {
			delete(mp.transactions, txID)
			removed = append(removed, txID)
			continue
		}

		// TODO: Add UTXO validation when UTXOSet is properly imported
		// This would validate that all inputs are still unspent
	}

	return removed
}

// GetStats returns mempool statistics
func (mp *Mempool) GetStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := map[string]interface{}{
		"total_transactions": len(mp.transactions),
		"total_size":        mp.getTotalSize(),
		"total_fees":        mp.getTotalFees(),
		"average_fee_rate":  mp.getAverageFeeRate(),
		"oldest_transaction": mp.getOldestTransactionAge(),
		"newest_transaction": mp.getNewestTransactionAge(),
		"addresses":         len(mp.byAddress),
		"capacity_used":     float64(len(mp.transactions)) / float64(mp.config.MaxSize) * 100,
	}

	return stats
}

// Clear removes all transactions from the mempool
func (mp *Mempool) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.transactions = make(map[string]*MempoolEntry)
	mp.byAddress = make(map[string][]string)
	mp.priorityQueue = &PriorityQueue{}
	heap.Init(mp.priorityQueue)
}

// Size returns the number of transactions in the mempool
func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.transactions)
}

// IsEmpty returns true if the mempool is empty
func (mp *Mempool) IsEmpty() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.transactions) == 0
}

// calculatePriority calculates transaction priority based on fee rate and age
func (mp *Mempool) calculatePriority(tx *Transaction, feeRate float64) int {
	txTime := time.Unix(tx.Timestamp, 0)
	age := time.Since(txTime)
	return int(feeRate*1e8) + int(age.Seconds())
}

// evictLowestPriority removes the lowest priority transaction
func (mp *Mempool) evictLowestPriority() error {
	if mp.priorityQueue.Len() == 0 {
		return fmt.Errorf("no transactions to evict")
	}

	entry := heap.Pop(mp.priorityQueue).(*MempoolEntry)
	delete(mp.transactions, entry.Transaction.ID)
	return nil
}

// cleanupOldTransactions removes transactions older than MaxAge
func (mp *Mempool) cleanupOldTransactions() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	cutoff := time.Now().Add(-mp.config.MaxAge)
	var toRemove []string

	for txID, entry := range mp.transactions {
		if entry.AddedAt.Before(cutoff) {
			toRemove = append(toRemove, txID)
		}
	}

	for _, txID := range toRemove {
		delete(mp.transactions, txID)
	}
}

// Helper methods for statistics
func (mp *Mempool) getTotalSize() int {
	total := 0
	for _, entry := range mp.transactions {
		total += entry.Size
	}
	return total
}

func (mp *Mempool) getTotalFees() float64 {
	total := 0.0
	for _, entry := range mp.transactions {
		total += entry.Transaction.GetFee()
	}
	return total
}

func (mp *Mempool) getAverageFeeRate() float64 {
	if len(mp.transactions) == 0 {
		return 0
	}
	return mp.getTotalFees() / float64(mp.getTotalSize())
}

func (mp *Mempool) getOldestTransactionAge() time.Duration {
	if len(mp.transactions) == 0 {
		return 0
	}
	
	oldest := time.Now()
	for _, entry := range mp.transactions {
		if entry.AddedAt.Before(oldest) {
			oldest = entry.AddedAt
		}
	}
	return time.Since(oldest)
}

func (mp *Mempool) getNewestTransactionAge() time.Duration {
	if len(mp.transactions) == 0 {
		return 0
	}
	
	newest := time.Time{}
	for _, entry := range mp.transactions {
		if entry.AddedAt.After(newest) {
			newest = entry.AddedAt
		}
	}
	return time.Since(newest)
}

// PriorityQueue implements a priority queue for transactions
type PriorityQueue []*MempoolEntry

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority comes first
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*MempoolEntry)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}