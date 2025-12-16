// Package brprocessor (Block Records Processor) implements the logic for processing
// full block records received via the gossip protocol.
//
// Unlike block votes which are small and frequent, block records contain full transaction
// data and receipts. This processor manages the ingestion of these heavier items,
// ensuring that the system's memory usage is controlled via semaphores and that
// processing happens concurrently within defined limits.
package brprocessor

import (
	"errors"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/rony4d/go-opera-asset/inter/ibr"
)

var (
	// ErrBusy is returned when the processor cannot accept new items because
	// the resource semaphore is exhausted/full.
	ErrBusy = errors.New("failed to acquire events semaphore")
)

// Processor handles the lifecycle of incoming block records.
// It manages a worker pool to process records and a semaphore to limit concurrent resource usage.
type Processor struct {
	cfg Config

	// quit signals worker goroutines to shut down.
	quit chan struct{}
	// wg waits for all workers to complete during shutdown.
	wg sync.WaitGroup

	// callback holds the external logic for processing and releasing items.
	callback Callback

	// inserter is the worker pool responsible for executing the processing tasks.
	inserter *workers.Workers

	// itemsSemaphore tracks and limits the total size/count of items currently being processed.
	itemsSemaphore *datasemaphore.DataSemaphore
}

// ItemCallback defines the hooks for processing a single block record.
type ItemCallback struct {
	// Process is called to validate and apply the block record (e.g., save to DB).
	Process func(br ibr.LlrIdxFullBlockRecord) error
	// Released is called after processing finishes (success or failure) to notify the source peer
	// and perform cleanup.
	Released func(br ibr.LlrIdxFullBlockRecord, peer string, err error)
}

// Callback wraps the specific item callbacks.
type Callback struct {
	Item ItemCallback
}

// New creates a new Block Records Processor.
// It initializes the semaphore, configuration, and worker pool.
func New(itemsSemaphore *datasemaphore.DataSemaphore, cfg Config, callback Callback) *Processor {
	f := &Processor{
		cfg:            cfg,
		quit:           make(chan struct{}),
		itemsSemaphore: itemsSemaphore,
	}
	f.callback = callback
	f.inserter = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	return f
}

// Start activates the background worker pool to begin processing queued tasks.
func (f *Processor) Start() {
	f.inserter.Start(1)
}

// Stop gracefully shuts down the processor.
// It closes the quit channel, terminates the semaphore (waking up any waiting goroutines),
// and waits for all active workers to finish.
func (f *Processor) Stop() {
	close(f.quit)
	f.itemsSemaphore.Terminate()
	f.wg.Wait()
}

// Overloaded checks if the processor is under heavy load.
// It returns true if the number of queued tasks exceeds 75% of the configured maximum.
func (f *Processor) Overloaded() bool {
	return f.inserter.TasksCount() > f.cfg.MaxTasks*3/4
}

// Enqueue adds a batch of block records to the processing queue.
// It attempts to acquire the necessary semaphore resources first. If successful,
// it schedules a task on the worker pool to process the items sequentially.
// The 'totalSize' parameter is pre-calculated by the caller to avoid re-scanning the items.
func (f *Processor) Enqueue(peer string, items []ibr.LlrIdxFullBlockRecord, totalSize uint64, done func()) error {
	// Calculate resource usage for this batch
	metric := dag.Metric{Num: idx.Event(len(items)), Size: totalSize}

	// Try to acquire semaphore resources; fail if busy/full
	if !f.itemsSemaphore.Acquire(metric, f.cfg.SemaphoreTimeout) {
		return ErrBusy
	}

	// Schedule processing task
	return f.inserter.Enqueue(func() {
		if done != nil {
			defer done()
		}
		// Ensure resources are released when this task completes
		defer f.itemsSemaphore.Release(metric)

		for i, item := range items {
			// Process the individual block record
			err := f.callback.Item.Process(item)

			// Clear heavy payload fields to free memory immediately after processing,
			// before the callback returns or the loop continues.
			items[i].Txs = nil
			items[i].Receipts = nil

			// Notify completion
			f.callback.Item.Released(item, peer, err)
		}
	})
}

// TasksCount returns the number of tasks currently queued or running in the worker pool.
func (f *Processor) TasksCount() int {
	return f.inserter.TasksCount()
}
