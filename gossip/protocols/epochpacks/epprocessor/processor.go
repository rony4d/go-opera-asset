// Package epprocessor (Epoch Packs Processor) implements the logic for processing
// epoch packs, which consist of an epoch record (the state) and a set of votes
// confirming that state.
//
// The processor manages the validation of votes and the application of the epoch record
// using concurrent worker pools. It separates signature verification (checking) from
// the sequential application logic (inserting) to maximize throughput while maintaining consistency.
package epprocessor

import (
	"errors"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/datasemaphore"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/inter/iep"
	"github.com/rony4d/go-opera-asset/inter/ier"
)

var (
	// ErrBusy is returned when the processor cannot accept new items because
	// the resource semaphore is exhausted/full.
	ErrBusy = errors.New("failed to acquire events semaphore")
)

// Processor orchestrates the validation and application of epoch packs.
// It manages two stages of workers: 'checker' for parallel validation of votes,
// and 'inserter' for applying the validated votes and the epoch record itself.
type Processor struct {
	cfg Config

	// quit is used to signal shutdown to worker goroutines.
	quit chan struct{}
	// wg waits for all workers to finish during shutdown.
	wg sync.WaitGroup

	// callback holds the external logic for processing items.
	callback Callback

	// inserter is a worker pool for applying the epoch data (votes and record) to storage.
	inserter *workers.Workers
	// checker is a worker pool for validating signatures on epoch votes.
	checker *workers.Workers

	// itemsSemaphore tracks and limits the total size/count of items currently being processed.
	itemsSemaphore *datasemaphore.DataSemaphore
}

// ItemCallback defines the hooks for processing parts of an epoch pack.
type ItemCallback struct {
	// ProcessEV applies a valid epoch vote.
	ProcessEV func(ev inter.LlrSignedEpochVote) error
	// ProcessER applies the full epoch record (state).
	ProcessER func(er ier.LlrIdxFullEpochRecord) error
	// ReleasedEV notifies when an epoch vote has been processed or dropped.
	ReleasedEV func(ev inter.LlrSignedEpochVote, peer string, err error)
	// ReleasedER notifies when an epoch record has been processed or dropped.
	ReleasedER func(er ier.LlrIdxFullEpochRecord, peer string, err error)
	// CheckEV performs validation checks (like signatures) on an epoch vote.
	CheckEV func(ev inter.LlrSignedEpochVote, checked func(error))
}

// Callback wraps the item callbacks.
type Callback struct {
	Item ItemCallback
}

// New creates a new Epoch Pack Processor.
// It initializes the worker pools and synchronization primitives.
func New(itemsSemaphore *datasemaphore.DataSemaphore, cfg Config, callback Callback) *Processor {
	f := &Processor{
		cfg:            cfg,
		quit:           make(chan struct{}),
		itemsSemaphore: itemsSemaphore,
		callback:       callback,
	}
	f.callback = callback
	f.inserter = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	f.checker = workers.New(&f.wg, f.quit, cfg.MaxTasks)
	return f
}

// Start boots up the background worker pools.
func (f *Processor) Start() {
	f.inserter.Start(1)
	f.checker.Start(1)
}

// Stop gracefully shuts down the processor.
// It signals workers to stop, waits for them to finish, and terminates the semaphore.
func (f *Processor) Stop() {
	close(f.quit)
	f.itemsSemaphore.Terminate()
	f.wg.Wait()
}

// Overloaded checks if the processor is under heavy load.
// It returns true if the total number of queued tasks exceeds 75% of maximum capacity.
func (f *Processor) Overloaded() bool {
	return f.TasksCount() > f.cfg.MaxTasks*3/4
}

// checkRes holds the result of validating a single epoch vote.
type checkRes struct {
	ev  inter.LlrSignedEpochVote
	err error
	pos idx.Event // Position in the original batch to maintain ordering
}

// Enqueue submits a batch of epoch packs for processing.
// It acquires resources and then schedules the work on the 'inserter' pool.
// Within that task, it uses the 'checker' pool to validate votes in parallel.
func (f *Processor) Enqueue(peer string, eps []iep.LlrEpochPack, totalSize uint64, done func()) error {
	if len(eps) == 0 {
		if done != nil {
			done()
		}
		return nil
	}

	// Calculate resource cost for the batch
	metric := dag.Metric{Num: idx.Event(len(eps)), Size: totalSize}

	// Try to acquire semaphore resources
	if !f.itemsSemaphore.Acquire(metric, f.cfg.SemaphoreTimeout) {
		return ErrBusy
	}

	// Schedule the main processing task on the inserter pool.
	// We handle each EpochPack sequentially here to maintain logical order,
	// but validate the votes within each pack in parallel.
	return f.inserter.Enqueue(func() {
		if done != nil {
			defer done()
		}
		defer f.itemsSemaphore.Release(metric)

		for _, ep := range eps {
			items := ep.Votes
			record := ep.Record

			// Channel to collect validation results from the checker pool
			checkedC := make(chan *checkRes, len(items))

			// Schedule parallel validation for all votes in this pack
			err := f.checker.Enqueue(func() {
				for i, v := range items {
					pos := idx.Event(i)
					ev := v
					f.callback.Item.CheckEV(ev, func(err error) {
						checkedC <- &checkRes{
							ev:  ev,
							err: err,
							pos: pos,
						}
					})
				}
			})
			if err != nil {
				return
			}

			// Collect results and apply them in order
			itemsLen := len(items)
			var orderedResults = make([]*checkRes, itemsLen)
			var processed int

			for processed < itemsLen {
				select {
				case res := <-checkedC:
					orderedResults[res.pos] = res

					// Process all contiguous results that are ready
					for i := processed; processed < len(orderedResults) && orderedResults[i] != nil; i++ {
						f.processEV(peer, orderedResults[i].ev, orderedResults[i].err)
						orderedResults[i] = nil // free memory
						processed++
					}

				case <-f.quit:
					return
				}
			}

			// After all votes are processed, process the epoch record itself
			f.processER(peer, record)
		}
	})
}

// processEV handles the result of a single vote validation.
// It calls the appropriate callback (Process or Released) based on the validation result.
func (f *Processor) processEV(peer string, ev inter.LlrSignedEpochVote, resErr error) {
	// release item if failed validation
	if resErr != nil {
		f.callback.Item.ReleasedEV(ev, peer, resErr)
		return
	}
	// process item (apply vote)
	err := f.callback.Item.ProcessEV(ev)
	f.callback.Item.ReleasedEV(ev, peer, err)
}

// processER handles the application of the epoch record.
func (f *Processor) processER(peer string, er ier.LlrIdxFullEpochRecord) {
	// process item (apply epoch record)
	err := f.callback.Item.ProcessER(er)
	f.callback.Item.ReleasedER(er, peer, err)
}

// TasksCount returns the total number of tasks currently queued or running in both pools.
func (f *Processor) TasksCount() int {
	return f.inserter.TasksCount() + f.checker.TasksCount()
}
