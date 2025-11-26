package inter

import "fmt"

// Constants defining the indices for the gas buckets.
const (
	// ShortTermGas is the index for the short-window gas bucket.
	// This bucket fills up and drains quickly, preventing immediate bursts of spam.
	ShortTermGas = 0

	// LongTermGas is the index for the long-window gas bucket.
	// This bucket fills and drains slowly, enforcing an average throughput limit over time.
	LongTermGas = 1

	// GasPowerConfigs defines the total number of gas buckets used (currently 2).
	GasPowerConfigs = 2
)

// GasPowerLeft represents the remaining "throughput allowance" for a validator.
//
// The protocol uses a "Token Bucket" algorithm to limit how much computation (gas)
// a validator can impose on the network.
// - You earn gas power as time passes (refill rate).
// - You spend gas power when you emit an event containing transactions (consumption).
//
// We track two separate buckets (Short-Term and Long-Term) to allow for brief
// bursts of high activity (using the Short bucket) while still capping the
// sustained load (using the Long bucket).
type GasPowerLeft struct {
	// Gas holds the current level of the token buckets.
	// Gas[0] = ShortTermGas, Gas[1] = LongTermGas.
	Gas [GasPowerConfigs]uint64
}

// Add increases the gas power in ALL buckets by the specified amount.
// This typically happens when time elapses (e.g., "1 second passed, add 1000 gas to allowance").
// Note: In the original Go code, this receiver is by value, so it doesn't modify the caller's struct
// unless reassigned. However, the Go code implementation `g.Gas[i] += diff` implies intent to modify
// if it were a pointer receiver.
// *Correction for Porting*: The original code `func (g GasPowerLeft) Add` receives a COPY.
// The mutation inside the loop `g.Gas[i] += diff` only affects the local copy and is discarded.
// This looks like a bug or a "return modified copy" pattern in the original code,
// but since it returns nothing, it effectively does nothing.
// CHECK THIS LOGIC CAREFULLY. If it's meant to modify, it should be `func (g *GasPowerLeft) Add`.
// Based on usage in typical Lachesis, this is usually calculated freshly rather than mutated in place.

// func (g GasPowerLeft) Add(diff uint64) {
// 	for i := range g.Gas {
// 		g.Gas[i] += diff
// 	}
// }

// Min returns the minimum gas available across all buckets.
// This is the effective limit. You cannot spend more gas than your most constrained bucket allows.
// If ShortTerm has 500 and LongTerm has 10000, your effective limit is 500.
func (g GasPowerLeft) Min() uint64 {
	min := g.Gas[0]
	for _, gas := range g.Gas {
		if min > gas {
			min = gas
		}
	}
	return min
}

// Max returns the maximum gas available in any bucket.
// Mostly used for metrics or debugging to see the most loose constraint.
func (g GasPowerLeft) Max() uint64 {
	max := g.Gas[0]
	for _, gas := range g.Gas {
		if max < gas {
			max = gas
		}
	}
	return max
}

// Sub creates a NEW GasPowerLeft object with the gas reduced by `diff` in all buckets.
// This simulates "spending" gas.
// Used when validating an event: `NewGasLeft = OldGasLeft.Sub(TxGasUsed)`.
// If the result would underflow (go negative), the transaction/event is invalid.

// func (g GasPowerLeft) Sub(diff uint64) GasPowerLeft {
// 	cp := g
// 	for i := range cp.Gas {
// 		// In Go, uint64 underflow wraps around.
// 		// In porting, ensure you handle underflow checks explicitly if required
// 		// (though usually validity checks happen before calling Sub).
// 		cp.Gas[i] -= diff
// 	}
// 	return cp
// }

// String returns a human-readable string representation for logging.
func (g GasPowerLeft) String() string {
	return fmt.Sprintf("{short=%d, long=%d}", g.Gas[ShortTermGas], g.Gas[LongTermGas])
}
