// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/rony4d/go-opera-asset/inter"
)

var FakeGenesisTime = inter.Timestamp(1608600000 * time.Second)

// ApplyFakeGenesis writes or updates the genesis block in db.
func ApplyFakeGenesis(statedb *state.StateDB, time inter.Timestamp, balances map[common.Address]*big.Int) (*EvmBlock, error) {
	for acc, balance := range balances {
		statedb.SetBalance(acc, balance)
	}

	// initial block
	root, err := flush(statedb, true)
	if err != nil {
		return nil, err
	}
	block := genesisBlock(time, root)

	return block, nil
}
