# Merkle Patricia Trie (MPT) Explained

## What is a Merkle Trie?

A **Merkle Patricia Trie** (also called MPT or Trie) is a data structure that combines:
- **Patricia Trie**: A radix tree optimized for key-value storage
- **Merkle Tree**: Cryptographic hashing for integrity verification

It's the core data structure used in Ethereum to store:
- **State Trie**: All account balances and contract storage
- **Transaction Trie**: All transactions in a block
- **Receipt Trie**: Transaction receipts

## Key Concepts

### 1. Trie Structure

```
Root Node (hash of entire tree)
├── Branch Node (16 children + value)
│   ├── [0-9, a-f] → Child Node
│   └── value (if key ends here)
├── Extension Node (shared prefix + next node)
│   └── "abc" → points to next node
└── Leaf Node (final key-value pair)
    └── "abc123" → value
```

### 2. Node Types

**Branch Node**: Has 16 children (one for each hex digit 0-f) plus an optional value
```
Branch: [child0, child1, ..., child15, value]
```

**Extension Node**: Compresses shared prefixes
```
Extension: "abc" → points to next node
```

**Leaf Node**: Final key-value pair
```
Leaf: "abc123" → "value"
```

### 3. Merkle Hashing

Each node is hashed (SHA-3/Keccak-256), creating a cryptographic fingerprint:
- **Root Hash**: Single hash representing entire tree
- **Change Detection**: Any change in data changes the root hash
- **Proof**: Can prove a value exists without downloading entire tree

## How It Works

### Example: Storing Key-Value Pairs

Let's store:
- `"abc"` → `"value1"`
- `"abd"` → `"value2"`
- `"xyz"` → `"value3"`

**Step 1**: Convert keys to hex nibbles
```
"abc" → [0x61, 0x62, 0x63] → [6,1,6,2,6,3]
"abd" → [0x61, 0x62, 0x64] → [6,1,6,2,6,4]
"xyz" → [0x78, 0x79, 0x7a] → [7,8,7,9,7,a]
```

**Step 2**: Build trie
```
Root (Branch)
├── [6] → Extension "16"
│   └── Branch
│       ├── [1] → Extension "626"
│       │   └── Branch
│       │       ├── [3] → Leaf "value1"
│       │       └── [4] → Leaf "value2"
│       └── ...
└── [7] → Extension "8797a"
    └── Leaf "value3"
```

**Step 3**: Hash each node bottom-up
```
Leaf nodes → hashed
Branch nodes → hash(children hashes)
Root → final hash (state root)
```

## Ethereum's Usage

### State Root
```go
// All account states
stateRoot := trie.HashRoot(accountStates)
// Stored in block header
block.Header.Root = stateRoot
```

### Transaction Root
```go
// All transactions in block
txRoot := types.DeriveSha(transactions, trie.NewStackTrie(nil))
// Stored in block header
block.Header.TxHash = txRoot
```

## Benefits

1. **Cryptographic Integrity**: Root hash proves entire tree state
2. **Efficient Updates**: Only changed nodes need recomputation
3. **Proof Generation**: Can prove value exists with minimal data
4. **Deterministic**: Same data always produces same root hash

