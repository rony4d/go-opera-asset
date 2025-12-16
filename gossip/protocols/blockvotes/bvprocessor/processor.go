// Package bvprocessor (Block Votes Processor) implements the logic for concurrently
// processing, validating, and applying block votes received from the p2p network.
// It uses a pipelined approach with worker pools to handle incoming votes efficiently,
// separating the validation (check) phase from the application (insert) phase while
// respecting resource limits via semaphores.
package bvprocessor

import (
	"errors"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/rony4d/go-opera-asset/inter"
)

var (
	// ErrBusy is returned when the processor cannot accept new items because
	// the resource semaphore is exhausted.
	ErrBusy = errors.New("failed to acquire events semaphore")
)

// Processor orchestrates the validation and processing of incoming block votes.
// It manages concurrency, resource usage, and the lifecycle of processing tasks.
type Processor struct {
	cfg Config

	// quit is used to signal shutdown to worker goroutines.
	quit chan struct{}
	// wg waits for all worker goroutines to finish on shutdown.
	wg sync.WaitGroup

	// callback holds the external functions to invoke for checking and processing items.
	callback Callback

	// inserter is a worker pool for the second stage: applying the votes (inserting into DB/state).
	inserter *workers.Workers

	// checker is a worker pool for the first stage: validating the votes (signature checks, etc.).
	checker *workers.Workers

	// itemsSemaphore limits the total number/size of items currently being processed to prevent memory overflow.
	itemsSemaphore *datasemaphore.DataSemaphore
}

// ItemCallback defines the hooks for processing a single block vote batch.
type ItemCallback struct {
	// Process is called to apply valid votes to the system (e.g., write to DB).
	Process func(bvs inter.LlrSignedBlockVotes) error
	// Released is called when processing is finished (success or failure) to release resources/notify peers.
	Released func(bvs inter.LlrSignedBlockVotes, peer string, err error)
	// Check is called to validate the votes before processing (e.g., check signatures).
	Check func(bvs inter.LlrSignedBlockVotes, checked func(error))
}

// Callback wraps the item callbacks.
type Callback struct {
	Item ItemCallback
}

// New creates a new Processor instance.
// It initializes the worker pools and wraps the release callback to ensure
// the semaphore is always released when an item finishes processing.
func New(itemsSemaphore *datasemaphore.DataSemaphore, cfg Config, callback Callback) *Processor {
	f := &Processor{
		cfg:            cfg,
		quit:           make(chan struct{}),
		itemsSemaphore: itemsSemaphore,
	}
	// Wrap the Released callback to automatically release the semaphore resources
	released := callback.Item.Released
	callback.Item.Released = func(bvs inter.LlrSignedBlockVotes, peer string, err error) {
		f.itemsSemaphore.Release(dag.Metric{Num: 1, Size: uint64(bvs.Size())})
		if released != nil {
			released(bvs, peer, err)
		}
	}
	f.callback = callback
	f.inserter = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	f.checker = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	return f
}

// Start boots up the background worker routines.
// It starts the workers for both the checking and insertion phases.
func (f *Processor) Start() {
	f.inserter.Start(1)
	f.checker.Start(1)
}

// Stop initiates a graceful shutdown of the processor.
// It signals workers to stop, waits for them to finish, and terminates the semaphore.
func (f *Processor) Stop() {
	close(f.quit)
	f.itemsSemaphore.Terminate()
	f.wg.Wait()
}

// Overloaded checks if the processor is under heavy load.
// It returns true if the number of queued tasks exceeds 75% of the maximum capacity.
func (f *Processor) Overloaded() bool {
	return f.TasksCount() > f.cfg.MaxTasks*3/4
}

// checkRes holds the result of the validation phase.
type checkRes struct {
	bvs inter.LlrSignedBlockVotes
	err error
	pos idx.Event // Preserves the original order in the batch
}

// Enqueue submits a batch of block votes for processing.
// It acquires the necessary semaphore resources, then pipelines the items through
// the 'checker' (validation) and 'inserter' (application) stages.
// Items are validated in parallel but applied in the order they were received to maintain consistency.
func (f *Processor) Enqueue(peer string, items []inter.LlrSignedBlockVotes, done func()) error {
	totalSize := uint64(0)
	for _, v := range items {
		totalSize += v.Size()
	}

	// Try to acquire resources; fail immediately if busy
	if !f.itemsSemaphore.Acquire(dag.Metric{Num: idx.Event(len(items)), Size: totalSize}, f.cfg.SemaphoreTimeout) {
		return ErrBusy
	}

	// Channel to pass validation results from checker to inserter
	checkedC := make(chan *checkRes, len(items))

	// Stage 1: Enqueue tasks to the 'checker' worker pool for parallel validation
	err := f.checker.Enqueue(func() {
		for i, v := range items {
			pos := idx.Event(i)
			bvs := v
			f.callback.Item.Check(bvs, func(err error) {
				checkedC <- &checkRes{
					bvs: bvs,
					err: err,
					pos: pos,
				}
			})
		}
	})
	if err != nil {
		return err
	}

	itemsLen := len(items)

	// Stage 2: Enqueue a task to the 'inserter' worker pool to collect results and apply them.
	// This runs in the 'inserter' pool to ensure that the actual application logic is controlled there.
	return f.inserter.Enqueue(func() {
		if done != nil {
			defer done()
		}

		// Buffer to reorder results because parallel validation might finish out of order
		var orderedResults = make([]*checkRes, itemsLen)
		var processed int

		for processed < itemsLen {
			select {
			case res := <-checkedC:
				orderedResults[res.pos] = res

				// Process all contiguous results that are ready, starting from the current 'processed' index
				for i := processed; processed < len(orderedResults) && orderedResults[i] != nil; i++ {
					f.process(peer, orderedResults[i].bvs, orderedResults[i].err)
					orderedResults[i] = nil // free the memory
					processed++
				}

			case <-f.quit:
				return
			}
		}
	})
}

// process handles the final step for a single item after validation.
// If validation failed, it releases the item with error.
// If validation succeeded, it calls the Process callback to apply the item.
func (f *Processor) process(peer string, bvs inter.LlrSignedBlockVotes, resErr error) {
	// release item if failed validation
	if resErr != nil {
		f.callback.Item.Released(bvs, peer, resErr)
		return
	}
	// process item (apply to state/DB)
	err := f.callback.Item.Process(bvs)
	f.callback.Item.Released(bvs, peer, err)
}

// TasksCount returns the total number of tasks currently queued or running in both worker pools.
func (f *Processor) TasksCount() int {
	return f.inserter.TasksCount() + f.checker.TasksCount()
}
