

### Installation

```bash
# Clone the repository
git clone https://github.com/Aliexe-code/MxM-Chain.git
cd MxM-Chain

# Download dependencies
go mod download
```

### Building

```bash
# Build main application
make build

# Build mining CLI
make miner

# Build all
make build miner
```

### Running

#### Mining CLI

```bash
# Start mining with default settings
go run cmd/miner/main.go start

# Start mining with custom settings
go run cmd/miner/main.go start -miner alice -difficulty 3

# Show mining status
go run cmd/miner/main.go status

# Show detailed statistics
go run cmd/miner/main.go stats

# Stop mining
go run cmd/miner/main.go stop

# Set difficulty
go run cmd/miner/main.go set-difficulty 4
```

#### Storage CLI

```bash
# Show blockchain info (file storage)
go run cmd/storage/main.go info file

# Show blockchain info (database storage)
go run cmd/storage/main.go info db

# Export blockchain
go run cmd/storage/main.go export file ./backup/blockchain.json

# Import blockchain
go run cmd/storage/main.go import file ./backup/blockchain.json

# Validate blockchain
go run cmd/storage/main.go validate file

# Create backup
go run cmd/storage/main.go backup file

# Cleanup storage
go run cmd/storage/main.go cleanup file
```

## 🔐 Blockchain Concepts

### Blocks and Hashing

Each block contains:
- **Timestamp**: When the block was created
- **Data**: Transaction information
- **PrevHash**: Hash of the previous block (links the chain)
- **Hash**: Current block's hash (SHA-256)
- **Nonce**: Number used once for mining

### Proof of Work

Mining involves finding a hash that meets a difficulty target:
```
Block Data + Nonce → SHA-256 → Hash
```

The hash must start with a specific number of zeros (determined by difficulty).

### UTXO Model

Unlike traditional account balances, MxM-Chain uses Unspent Transaction Outputs (UTXOs):
- Each transaction consumes existing UTXOs as inputs
- Creates new UTXOs as outputs
- Prevents double-spending through cryptographic verification

### Consensus

- **Longest Chain Rule**: Nodes follow the chain with the most cumulative work
- **Fork Resolution**: Automatic reconciliation when chains diverge
- **Network Sync**: Nodes synchronize with peers to maintain consistency

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make dev-test

# Run specific package tests
go test -v ./internal/blockchain/
go test -v ./internal/transactions/
go test -v ./internal/network/
```

## 📊 Features

### Mining
- Configurable difficulty (1-8)
- Real-time statistics
- Mining rewards based on difficulty
- Graceful shutdown

### Storage
- JSON file-based storage
- SQLite database support
- Automatic backups
- Import/export functionality
- Data integrity validation

### Network
- P2P TCP networking
- Peer discovery
- Message protocol
- Chain synchronization
- Consensus enforcement
- Partition handling

### Transactions
- Digital signatures (ECDSA)
- UTXO model
- Transaction pool (mempool)
- Validation and verification
- Double-spend protection

### Wallet
- Multiple wallet support
- Balance tracking
- Address generation
- Key management

## 🛠️ Development

### Makefile Targets

```bash
make help        # Show available targets
make build       # Build main application
make miner       # Build mining CLI
make test        # Run all tests
make clean       # Clean build artifacts
make install     # Install miner CLI to system
make dev-test    # Run tests with coverage
make dev-lint    # Run linter
make dev-fmt     # Format code
make quick-start # Quick start with default settings
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test -v ./...

# Run benchmarks
go test -bench=. ./...
```


## 🔧 Dependencies

- `github.com/mattn/go-sqlite3` - SQLite database driver
- `github.com/stretchr/testify` - Testing framework
- `golang.org/x/crypto` - Cryptographic functions

## 🤝 Contributing

This is a learning and development project. Contributions are welcome!

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request


## 🔬 Blockchain Concepts Explained

### 1. Blocks and Chains

#### Block Structure
A block is a container for transaction data with the following properties:

```go
// Conceptual structure (not actual code yet)
type Block struct {
    Timestamp    // When the block was created
    Data         // Transaction information
    PrevHash     // Hash of the previous block
    Hash         // Current block's hash
    Nonce        // Number used once (for mining)
}
```

**Why this matters:**
- **Immutability**: Each block contains the hash of the previous block
- **Chain Security**: Changing any block invalidates all subsequent blocks
- **Chronological Order**: Timestamps ensure proper sequencing

#### Hash Functions
Hash functions are the foundation of blockchain security:

**Properties:**
- **Deterministic**: Same input always produces same output
- **One-way**: Cannot reverse engineer input from hash
- **Fixed size**: Always produces output of same length
- **Avalanche effect**: Small input change = completely different output

**In Blockchain:**
- Links blocks together
- Ensures data integrity
- Makes tampering evident

### 2. Proof of Work (PoW)

#### The Mining Process
Mining is the process of finding a hash that meets certain criteria:

```
Block Data + Nonce → Hash Function → Hash
```

**Difficulty Target**: Hash must start with certain number of zeros
**Nonce**: Number we change until we find a valid hash

**Why PoW:**
- **Security**: Makes it expensive to attack the network
- **Consensus**: Helps nodes agree on the valid chain
- **Fairness**: Rewards computational work

### 3. Transactions and UTXO

#### Transaction Model
Instead of account balances, we use Unspent Transaction Outputs (UTXO):

**Traditional Banking:**
```
Account A: $100
Account B: $50
```

**UTXO Model:**
```
UTXO 1: $100 (owned by A)
UTXO 2: $50 (owned by B)
```

#### Transaction Flow
1. **Inputs**: Reference existing UTXOs you own
2. **Outputs**: Create new UTXOs for recipients
3. **Change**: Return any remaining amount to yourself

**Benefits:**
- **Parallel Processing**: No account locking
- **Privacy**: Harder to track ownership
- **Security**: Each transaction is cryptographically verified

### 4. Cryptography in Blockchain

#### Public/Private Key Pairs
- **Private Key**: Secret key for signing transactions
- **Public Key**: Derived from private key, used for verification
- **Address**: Hashed version of public key

#### Digital Signatures
1. **Signing**: Use private key to sign transaction data
2. **Verification**: Use public key to verify signature
3. **Security**: Proves ownership without revealing private key

### 5. Network and Consensus

#### Peer-to-Peer Network
- **Decentralization**: No central authority
- **Redundancy**: Multiple copies of blockchain
- **Fault Tolerance**: Network continues if some nodes fail

#### Consensus Mechanisms
How nodes agree on the valid blockchain:
- **Longest Chain Rule**: Follow the chain with most work
- **Fork Resolution**: Handle temporary splits in the network
- **Finality**: Ensure transactions cannot be reversed

### 6. Smart Contracts

#### Virtual Machine
- **Deterministic**: Same input produces same output on all nodes
- **Sandboxed**: Limited access to system resources
- **Gas System**: Prevents infinite loops and resource abuse

#### Contract Lifecycle
1. **Deployment**: Upload contract code to blockchain
2. **Creation**: Initialize contract with constructor
3. **Execution**: Call contract functions with transactions
4. **State**: Contract maintains its own storage

---


## 🧪 Testing Strategy

### Unit Tests
- Test each component in isolation
- Ensure correctness of algorithms
- Validate edge cases and error conditions

### Integration Tests
- Test component interactions
- Validate end-to-end workflows
- Ensure system reliability

### Network Tests
- Test multi-node scenarios
- Validate consensus mechanisms
- Ensure network resilience

---

## 📖 Learning Resources

### Go Documentation
- [Official Go Tour](https://tour.golang.org/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go by Example](https://gobyexample.com/)

### Blockchain Resources
- [Mastering Bitcoin](https://github.com/bitcoinbook/bitcoinbook) - Technical foundation
- [Ethereum White Paper](https://ethereum.org/en/whitepaper/) - Smart contracts
- [Blockchain Basics](https://github.com/blockchain-guide/blockchain-basics) - Conceptual understanding

### Cryptography
- [Crypto 101](https://www.crypto101.io/) - Cryptography fundamentals
- [Go Crypto Packages](https://pkg.go.dev/crypto) - Go's crypto libraries

---

