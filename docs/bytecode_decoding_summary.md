# Quick Reference: Bytecode Decoding & Generation

## Decoding the NodeDriver Bytecode

### Function Selectors Found

The bytecode contains **16 function selectors**:

```
0x07690b2a  0x0aeeca00  0x18f628d4  0x1e702f83
0x242a6e3f  0x267ab446  0x39e503ab  0x485cc955
0x4feb92f3  0xa4066fbe  0xb9cc6b1c  0xd6a0c7af
0xda7fc24f  0xe08d7e66  0xe30443bc  0xebdf104c
```

### Quick Decode Methods

1. **Online Decompiler**: https://ethervm.io/decompile
   - Paste full bytecode → Get Solidity-like code

2. **4byte Directory**: https://www.4byte.directory/
   - Search `0x07690b2a` → Find function signatures

3. **Foundry Cast**:
   ```bash
   cast 4byte 0x07690b2a  # Reverse lookup selector
   cast sig "functionName()"  # Calculate selector
   ```

## Generating Pre-Deployment Bytecode

### Quick Steps

```bash
# 1. Write contract (MyContract.sol)
# 2. Compile with runtime bytecode
solc --bin-runtime MyContract.sol -o output/

# 3. Get runtime bytecode (NOT creation bytecode!)
cat output/MyContract.bin-runtime

# 4. Use in Go
func GetContractBin() []byte {
    return hexutil.MustDecode("0x<your-bytecode>")
}
```

### Key Points

- ✅ Use `--bin-runtime` (runtime bytecode for predeployment)
- ❌ Don't use `--bin` (creation bytecode includes constructor)
- ✅ Match compiler version (0.5.17 for NodeDriver)
- ✅ Match optimization runs (10000 for NodeDriver)

### Example: Simple Predeploy Contract

```solidity
// SimplePredeploy.sol
pragma solidity ^0.5.17;

contract SimplePredeploy {
    uint256 public value;
    
    function setValue(uint256 _value) external {
        value = _value;
    }
    
    function getValue() external view returns (uint256) {
        return value;
    }
}
```

Compile:
```bash
solc --bin-runtime SimplePredeploy.sol
```

Use output in Go:
```go
package simple

import "github.com/ethereum/go-ethereum/common/hexutil"

func GetContractBin() []byte {
    return hexutil.MustDecode("0x608060405234801561001057600080fd5b...")
}
```

