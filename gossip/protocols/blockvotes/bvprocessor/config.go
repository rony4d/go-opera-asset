// Package bvprocessor (Block Votes Processor) handles the processing and validation of incoming block votes.
// This file (config.go) defines the configuration settings for the Block Vote Processor,
// controlling resource limits such as buffer sizes and timeouts to ensure stability and performance.
package bvprocessor

import (
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Config holds the tunable parameters for the Block Vote Processor.
type Config struct {
	// BufferLimit defines the maximum number and total size of block votes
	// that can be buffered in memory before processing.
	// This prevents memory exhaustion during high network activity.
	BufferLimit dag.Metric

	// SemaphoreTimeout is the maximum duration to wait for acquiring resources (e.g., a lock or slot)
	// before timing out. This prevents the processor from hanging indefinitely.
	SemaphoreTimeout time.Duration

	// MaxTasks is the maximum number of concurrent processing tasks allowed.
	// This controls parallelism to avoid CPU or I/O saturation.
	MaxTasks int
}

// DefaultConfig returns the standard configuration values for the Block Vote Processor.
// It takes a `scale` function to adjust memory-related limits based on available system resources.
func DefaultConfig(scale cachescale.Func) Config {
	return Config{
		BufferLimit: dag.Metric{
			Num:  3000,                    // Maximum number of votes to buffer
			Size: scale.U64(15 * opt.MiB), // Maximum total size of buffered votes (scaled)
		},
		SemaphoreTimeout: 10 * time.Second, // Default timeout for resource acquisition
		MaxTasks:         512,              // Default limit for concurrent tasks
	}
}
