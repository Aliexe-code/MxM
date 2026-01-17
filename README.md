# MxM-Chain: Blockchain Implementation in Go

A production-ready blockchain implementation in Go featuring proof-of-work consensus, peer-to-peer networking, and comprehensive transaction management.

## 🎯 Project Overview

MxM-Chain is a fully functional cryptocurrency blockchain written in Go. It demonstrates core blockchain concepts including immutable data structures, cryptographic hashing, proof-of-work mining, UTXO transaction model, wallet management, persistent storage, peer-to-peer networking, and consensus mechanisms.

**Version:** 0.1.0  
**Current Branch:** sprint/06-network-consensus  
**Go Version:** 1.25.5

## ✅ Current Implementation Status

The project has completed 6 sprints of development, implementing the following core features:

### ✅ Sprint 1: Blockchain Fundamentals
- **Block Structure**: Immutable blocks with hash linking
- **Blockchain**: Chain management with validation
- **Proof of Work**: Configurable mining difficulty (1-8)
- **JSON Serialization**: Import/export functionality

### ✅ Sprint 2: Mining System
- **Mining CLI**: Command-line interface for mining operations
- **Mining Statistics**: Real-time tracking of blocks mined, rewards, and performance
- **Difficulty Adjustment**: Configurable mining difficulty
- **Mining Rewards**: Reward calculation based on difficulty

### ✅ Sprint 3: Transaction Foundation
- **Cryptographic Keys**: ECDSA key pair generation
- **Transaction Structure**: Input/output model with validation
- **Digital Signatures**: Transaction signing and verification

### ✅ Sprint 4: Wallet & UTXO
- **Wallet Management**: Multiple wallet support with balance tracking
- **UTXO Model**: Unspent transaction output management
- **Transaction Pool**: Mempool for pending transactions
- **Transaction Validation**: Double-spend protection

### ✅ Sprint 5: Persistence & Storage
- **File Storage**: JSON-based blockchain persistence
- **Database Storage**: SQLite integration for efficient querying
- **Storage CLI**: Data export, import, validation, and backup
- **Backup System**: Automatic backup creation and management

### ✅ Sprint 6: Network & Consensus
- **P2P Networking**: TCP-based peer-to-peer communication
- **Peer Discovery**: Bootstrap nodes and peer exchange
- **Message Protocol**: Standardized network messages
- **Chain Synchronization**: Automatic sync with peers
- **Consensus Rules**: Longest chain rule and fork resolution
- **Network Partitions**: Graceful handling of network splits

## 🏗️ Project Structure

```
MxM-Chain/
├── cmd/
│   ├── miner/              # Mining CLI application
│   │   ├── main.go
│   │   ├── config.go
│   │   └── miner_test.go
│   └── storage/            # Storage management CLI
│       ├── main.go
│       └── storage_test.go
├── internal/
│   ├── blockchain/         # Core blockchain logic
│   │   ├── block.go        # Block structure
│   │   ├── blockchain.go  # Blockchain management
│   │   └── proof.go       # Proof of Work
│   ├── consensus/          # Consensus mechanisms
│   │   ├── network.go      # Network consensus integration
│   │   ├── partition.go    # Partition handling
│   │   ├── rules.go        # Consensus rules
│   │   └── sync.go         # Chain synchronization
│   ├── crypto/             # Cryptographic operations
│   │   └── keys.go         # Key generation
│   ├── network/            # P2P networking
│   │   ├── tcp.go          # TCP server/client
│   │   ├── message.go      # Message protocol
│   │   └── discovery.go    # Peer discovery
│   ├── storage/            # Data persistence
│   │   ├── file.go         # File-based storage
│   │   └── database.go     # SQLite storage
│   ├── transactions/       # Transaction system
│   │   ├── transaction.go  # Transaction structure
│   │   ├── utxo.go         # UTXO management
│   │   ├── mempool.go      # Transaction pool
│   │   ├── signature.go    # Digital signatures
│   │   ├── validation.go   # Transaction validation
│   │   └── utils.go        # Helper functions
│   └── wallet/             # Wallet functionality
│       └── wallet.go       # Wallet implementation
├── main.go                 # Entry point
├── Makefile                # Build targets
├── go.mod                  # Go module definition
└── README.md               # This file
```

## 🚀 Getting Started

### Prerequisites
- Go 1.25.5 or higher
- Basic understanding of Go programming
- Familiarity with blockchain concepts

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

## 📚 Documentation

- **SPRINTS.md**: Detailed sprint roadmap and development guide
- **README.md**: This file - project overview and getting started

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

## 📝 License

This project is open source and available under the MIT License.

## 🎯 Future Roadmap

### Sprint 7: Smart Contracts Foundation
- Virtual machine implementation
- Contract deployment
- Contract execution
- Gas system

### Sprint 8: Advanced Features
- Merkle trees for transaction verification
- Performance monitoring
- Optimization and benchmarking
- Advanced security features

### Sprint 9: Production Readiness
- Comprehensive testing
- Documentation
- Deployment guides
- Performance tuning

## 📧 Contact

For questions or feedback, please open an issue on GitHub.

---

**Built with ❤️ using Go**

## 🏗️ Technical Architecture

### Project Structure
```
crypto-chain/
├── cmd/                    # Command-line interfaces
│   ├── wallet/            # Wallet management CLI
│   ├── node/              # Node operation CLI
│   └── cli/               # General blockchain CLI
├── internal/
│   ├── blockchain/        # Core blockchain logic
│   │   ├── block.go       # Block structure and methods
│   │   ├── blockchain.go  # Blockchain implementation
│   │   └── proof.go       # Proof of Work algorithm
│   ├── crypto/            # Cryptographic operations
│   │   ├── hash.go        # Hash functions
│   │   ├── keys.go        # Key generation
│   │   └── signature.go   # Digital signatures
│   ├── transaction/       # Transaction system
│   │   ├── transaction.go # Transaction structure
│   │   ├── utxo.go        # UTXO management
│   │   └── mempool.go     # Transaction pool
│   ├── network/           # P2P networking
│   │   ├── peer.go        # Peer management
│   │   ├── message.go     # Message protocol
│   │   └── sync.go        # Chain synchronization
│   ├── vm/                # Virtual machine
│   │   ├── vm.go          # VM implementation
│   │   ├── contract.go    # Smart contracts
│   │   └── gas.go         # Gas calculation
│   ├── wallet/            # Wallet functionality
│   │   ├── wallet.go      # Wallet structure
│   │   └── address.go     # Address generation
│   └── storage/           # Data persistence
│       ├── file.go        # File-based storage
│       └── serialization.go # JSON serialization
├── pkg/                   # Public APIs
│   └── cryptochain/       # External library interface
├── configs/               # Configuration files
│   ├── default.json       # Default configuration
│   └── test.json          # Test configuration
├── scripts/               # Helper scripts
│   ├── setup.sh           # Environment setup
│   └── test.sh            # Testing scripts
└── docs/                  # Documentation
    ├── api.md             # API documentation
    └── concepts.md        # Concept explanations
```

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

## 🛠️ Go Programming Concepts You'll Master

### Phase 1: Go Fundamentals
- **Structs and Methods**: Custom data types and their behaviors
- **Slices**: Dynamic arrays for blockchain storage
- **JSON Marshaling**: Serialization for storage and networking
- **Error Handling**: Proper error propagation and handling
- **Testing**: Unit tests for blockchain components

### Phase 2: Intermediate Go
- **Interfaces**: Abstract behavior definition
- **Pointers**: Memory management and efficiency
- **File I/O**: Persistent data storage
- **Concurrency Basics**: Goroutines for parallel operations
- **Packages**: Code organization and modularity

### Phase 3: Advanced Go
- **Concurrency Patterns**: Channels and select statements
- **Network Programming**: TCP sockets and HTTP servers
- **Reflection**: Dynamic type inspection
- **Performance Optimization**: Profiling and optimization
- **CLI Development**: Command-line interface creation

---

## 📋 Implementation Roadmap

### Week 1: Project Setup & Basic Blocks
- [ ] Initialize Go module
- [ ] Set up project structure
- [ ] Implement basic Block struct
- [ ] Create hash calculation function
- [ ] Write unit tests

### Week 2: Blockchain & Mining
- [ ] Implement Blockchain struct
- [ ] Add block creation logic
- [ ] Implement Proof of Work
- [ ] Create chain validation
- [ ] Add genesis block

### Week 3: Transaction Foundation
- [ ] Design Transaction struct
- [ ] Implement basic cryptography
- [ ] Create key generation
- [ ] Add digital signatures
- [ ] Test transaction validation

### Week 4: Wallet & UTXO
- [ ] Implement wallet creation
- [ ] Build UTXO management system
- [ ] Add transaction pool (mempool)
- [ ] Create transaction selection logic
- [ ] Implement change calculation

### Week 5: Persistence & CLI
- [ ] Add file-based storage
- [ ] Implement JSON serialization
- [ ] Create CLI commands
- [ ] Add blockchain exploration
- [ ] Build wallet management interface

### Week 6: Network Foundation
- [ ] Implement basic TCP server
- [ ] Create message protocol
- [ ] Add peer discovery
- [ ] Implement basic message propagation
- [ ] Test node connectivity

### Week 7: Consensus & Sync
- [ ] Implement chain synchronization
- [ ] Add longest chain rule
- [ ] Handle fork resolution
- [ ] Create consensus mechanism
- [ ] Test network scenarios

### Week 8: Smart Contracts
- [ ] Design virtual machine
- [ ] Implement basic instruction set
- [ ] Add contract deployment
- [ ] Create contract execution
- [ ] Implement gas system

### Week 9: Advanced Features
- [ ] Add Merkle trees
- [ ] Implement advanced validation
- [ ] Create monitoring system
- [ ] Optimize performance
- [ ] Add comprehensive tests

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

## 🎯 Success Criteria

### Beginner Level Completion
- [ ] Create a blockchain with 10+ valid blocks
- [ ] Mine blocks with adjustable difficulty
- [ ] Validate entire chain integrity
- [ ] Handle basic edge cases

### Intermediate Level Completion
- [ ] Create and sign valid transactions
- [ ] Manage multiple wallets
- [ ] Maintain accurate UTXO set
- [ ] Persist blockchain to disk

### Advanced Level Completion
- [ ] Connect 5+ nodes in a network
- [ ] Achieve consensus on blockchain state
- [ ] Deploy and execute smart contracts
- [ ] Handle network partitions and recovery

---

## 🚀 Getting Started

### Prerequisites
- Go 1.19+ installed
- Basic understanding of Go syntax
- Familiarity with command line
- Curiosity about blockchain technology

### Environment Setup
```bash
# Clone the project
git clone <your-repo-url>
cd crypto-chain

# Initialize Go module
go mod init crypto-chain

# Run tests
go test ./...

# Build the project
go build ./cmd/cli
```

### First Steps
1. Read through this documentation completely
2. Understand the concepts before coding
3. Start with Phase 1 implementation
4. Test each component thoroughly
5. Progress to next phase only after mastery

---

## 🤝 Contributing & Learning

This is a learning project. Focus on:
- Understanding concepts deeply
- Writing clean, idiomatic Go code
- Testing thoroughly
- Documenting your learning
- Asking questions when stuck

Remember: The goal is learning, not just completion. Take your time with each concept and ensure you understand both the "how" and the "why."

---

## 📝 Next Steps

Ready to start coding? Begin with:

1. **Phase 1**: Implement the basic Block and Blockchain structures
2. **Test**: Verify your implementation with unit tests
3. **Learn**: Understand each concept before moving forward
4. **Build**: Progress through each phase systematically

Happy coding and learning! 🎉