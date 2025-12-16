// Store is the primary persistence manager for the node.
// It wraps the underlying physical key-value database (kvdb) and provides high-level accessors
// for consensus data (DAG), block data, and state. It also manages the integration with the
// EVM storage (StateDB) and handles caching strategies.

package gossip

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/Fantom-foundation/lachesis-base/utils/wlru"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/rony4d/go-opera-asset/logger"
)

// Store is the primary persistence manager for the node.
// It wraps the underlying physical key-value database (kvdb) and provides high-level accessors
// for consensus data (DAG), block data, and state. It also manages the integration with the
// EVM storage (StateDB) and handles caching strategies.
type Store struct {
	// dbs is the producer for flushable database instances.
	dbs kvdb.FlushableDBProducer
	// cfg holds the configuration for the store (cache sizes, flush intervals, etc).
	cfg StoreConfig

	// snapshotedEVMDB is a wrapper that allows switching between different snapshots of the EVM database.
	snapshotedEVMDB *switchable.Snapshot
	// evm is the specific store for EVM-related data (accounts, storage slots).
	evm *evmstore.Store

	// table struct groups all logical KVDB tables used by the gossip protocol.
	// Each field represents a distinct prefix/bucket in the key-value store.
	table struct {
		// Version stores the database schema version.
		Version kvdb.Store `table:"_"`

		// Main DAG tables
		// BlockEpochState stores the current dynamic state of the epoch (validators, uptime, etc.).
		BlockEpochState kvdb.Store `table:"D"`
		// BlockEpochStateHistory stores historical snapshots of epoch states for rewinds/API.
		BlockEpochStateHistory kvdb.Store `table:"h"`
		// Events stores the DAG events (vertices in the graph).
		Events kvdb.Store `table:"e"`
		// Blocks stores the decided blocks.
		Blocks kvdb.Store `table:"b"`
		// EpochBlocks stores the mapping between epochs and their blocks.
		EpochBlocks kvdb.Store `table:"P"`
		// Genesis stores the genesis state.
		Genesis kvdb.Store `table:"g"`
		// UpgradeHeights stores block numbers where network upgrades occurred.
		UpgradeHeights kvdb.Store `table:"U"`

		// P2P-only
		// HighestLamport stores the highest Lamport timestamp seen so far (for synchronization).
		HighestLamport kvdb.Store `table:"l"`

		// Network version
		// NetworkVersion stores the protocol network ID/version.
		NetworkVersion kvdb.Store `table:"V"`

		// API-only
		// BlockHashes stores the mapping from block hash to block number.
		BlockHashes kvdb.Store `table:"B"`

		// LLR (Lachesis Light Reputation / Longest Living R...) tables
		// LlrState stores the current state of the LLR consensus.
		LlrState           kvdb.Store `table:"S"`
		LlrBlockResults    kvdb.Store `table:"R"`
		LlrEpochResults    kvdb.Store `table:"Q"`
		LlrBlockVotes      kvdb.Store `table:"T"`
		LlrBlockVotesIndex kvdb.Store `table:"J"`
		LlrEpochVotes      kvdb.Store `table:"E"`
		LlrEpochVoteIndex  kvdb.Store `table:"I"`
		LlrLastBlockVotes  kvdb.Store `table:"G"`
		LlrLastEpochVote   kvdb.Store `table:"F"`
	}

	// prevFlushTime tracks the last time the database was flushed to disk.
	prevFlushTime time.Time

	// epochStore is an atomic value holding the store for the current epoch's specific data.
	epochStore atomic.Value

	// cache struct holds in-memory LRU caches for frequently accessed data.
	cache struct {
		Events                 *wlru.Cache `cache:"-"` // store by pointer
		EventIDs               *eventid.Cache
		EventsHeaders          *wlru.Cache  `cache:"-"` // store by pointer
		Blocks                 *wlru.Cache  `cache:"-"` // store by pointer
		BlockHashes            *wlru.Cache  `cache:"-"` // store by pointer
		EvmBlocks              *wlru.Cache  `cache:"-"` // store by pointer
		BlockEpochStateHistory *wlru.Cache  `cache:"-"` // store by pointer
		BlockEpochState        atomic.Value // store by value
		HighestLamport         atomic.Value // store by value
		LastBVs                atomic.Value // store by pointer
		LastEV                 atomic.Value // store by pointer
		LlrState               atomic.Value // store by value
		KvdbEvmSnap            atomic.Value // store by pointer
		UpgradeHeights         atomic.Value // store by pointer
		Genesis                atomic.Value // store by value
		LlrBlockVotesIndex     *VotesCache  // store by pointer
		LlrEpochVoteIndex      *VotesCache  // store by pointer
	}

	// mutex protects concurrent access to specific fields.
	mutex struct {
		WriteLlrState sync.Mutex
	}

	// rlp is a helper for RLP encoding/decoding operations.
	rlp rlpstore.Helper

	// Instance embeds the logger for this store.
	logger.Instance
}

// NewStore creates and initializes a new Store backed by the provided persistent key-value database.
// It sets up the table structure, initializes caches, and prepares the EVM store.
func NewStore(dbs kvdb.FlushableDBProducer, cfg StoreConfig) *Store {
	s := &Store{
		dbs:           dbs,
		cfg:           cfg,
		Instance:      logger.New("gossip-store"),
		prevFlushTime: time.Now(),
		rlp:           rlpstore.Helper{logger.New("rlp")},
	}

	// Initialize logical tables with prefixes
	err := table.OpenTables(&s.table, dbs, "gossip")
	if err != nil {
		log.Crit("Failed to open DB", "name", "gossip", "err", err)
	}

	s.initCache()
	s.evm = evmstore.NewStore(dbs, cfg.EVM)

	// Perform any necessary data migrations for DB upgrades
	if err := s.migrateData(); err != nil {
		s.Log.Crit("Failed to migrate Gossip DB", "err", err)
	}

	return s
}

// initCache configures and initializes all the LRU (Least Recently Used) caches based on the store configuration.
func (s *Store) initCache() {
	s.cache.Events = s.makeCache(s.cfg.Cache.EventsSize, s.cfg.Cache.EventsNum)
	s.cache.Blocks = s.makeCache(s.cfg.Cache.BlocksSize, s.cfg.Cache.BlocksNum)

	blockHashesNum := s.cfg.Cache.BlocksNum
	blockHashesCacheSize := nominalSize * uint(blockHashesNum)
	s.cache.BlockHashes = s.makeCache(blockHashesCacheSize, blockHashesNum)

	eventsHeadersNum := s.cfg.Cache.EventsNum
	eventsHeadersCacheSize := nominalSize * uint(eventsHeadersNum)
	s.cache.EventsHeaders = s.makeCache(eventsHeadersCacheSize, eventsHeadersNum)

	s.cache.EventIDs = eventid.NewCache(s.cfg.Cache.EventsIDsNum)

	blockEpochStatesNum := s.cfg.Cache.BlockEpochStateNum
	blockEpochStatesSize := nominalSize * uint(blockEpochStatesNum)
	s.cache.BlockEpochStateHistory = s.makeCache(blockEpochStatesSize, blockEpochStatesNum)

	s.cache.LlrBlockVotesIndex = NewVotesCache(s.cfg.Cache.LlrBlockVotesIndexes, s.flushLlrBlockVoteWeight)
	s.cache.LlrEpochVoteIndex = NewVotesCache(s.cfg.Cache.LlrEpochVotesIndexes, s.flushLlrEpochVoteWeight)
}

// Close shuts down the underlying database and releases resources.
// It ensures all tables and caches are closed properly.
func (s *Store) Close() {
	setnil := func() interface{} {
		return nil
	}

	_ = table.CloseTables(&s.table)
	table.MigrateTables(&s.table, nil)
	table.MigrateCaches(&s.cache, setnil)

	_ = s.closeEpochStore()
	s.evm.Close()
}

// IsCommitNeeded checks if the database should be flushed to disk.
// It uses heuristics based on time since last flush and the amount of non-flushed data
// to decide when to commit.
func (s *Store) IsCommitNeeded() bool {
	// randomize flushing criteria for each epoch so that nodes would desynchronize flushes
	// This prevents all nodes from freezing for IO at the exact same time.
	ratio := 900 + randat.RandAt(uint64(s.GetEpoch()))%100
	return s.isCommitNeeded(ratio, ratio)
}

// isCommitNeeded implements the logic for checking flush criteria with randomized ratios.
func (s *Store) isCommitNeeded(sc, tc uint64) bool {
	period := s.cfg.MaxNonFlushedPeriod * time.Duration(sc) / 1000
	size := (uint64(s.cfg.MaxNonFlushedSize) / 2) * tc / 1000
	return time.Since(s.prevFlushTime) > period ||
		uint64(s.dbs.NotFlushedSizeEst()) > size
}

// commitEVM persists the current EVM state to the underlying database.
// It updates the state root to match the latest block.
func (s *Store) commitEVM(flush bool) {
	bs := s.GetBlockState()
	err := s.evm.Commit(bs.LastBlock.Idx, bs.FinalizedStateRoot, flush)
	if err != nil {
		s.Log.Crit("Failed to commit EVM storage", "err", err)
	}
	s.evm.Cap()
}

// cleanCommitEVM performs a commit that also cleans up old trie nodes
// that are no longer referenced, helping to keep the DB size manageable.
func (s *Store) cleanCommitEVM() {
	err := s.evm.CleanCommit(s.GetBlockState())
	if err != nil {
		s.Log.Crit("Failed to commit EVM storage", "err", err)
	}
	s.evm.Cap()
}

// GenerateSnapshotAt triggers the generation of an EVM state snapshot at a specific root hash.
// This allows for consistent reads of the state at that point in time.
func (s *Store) GenerateSnapshotAt(root common.Hash, async bool) (err error) {
	err = s.generateSnapshotAt(s.evm, root, true, async)
	if err != nil {
		s.Log.Error("EVM snapshot", "at", root, "err", err)
	} else {
		gen, _ := s.evm.Snaps.Generating()
		s.Log.Info("EVM snapshot", "at", root, "generating", gen)
	}
	return err
}

// generateSnapshotAt calls the underlying EVM store to generate the snapshot.
func (s *Store) generateSnapshotAt(evmStore *evmstore.Store, root common.Hash, rebuild, async bool) (err error) {
	return evmStore.GenerateEvmSnapshot(root, rebuild, async)
}

// Commit persists all pending changes in memory (caches, buffers) to the physical database.
// This includes flushing the block state, LLR state, votes, and the underlying DBs.
func (s *Store) Commit() error {
	s.FlushBlockEpochState()
	s.FlushHighestLamport()
	s.FlushLastBVs()
	s.FlushLastEV()
	s.FlushLlrState()
	s.cache.LlrBlockVotesIndex.FlushMutated(s.flushLlrBlockVoteWeight)
	s.cache.LlrEpochVoteIndex.FlushMutated(s.flushLlrEpochVoteWeight)
	es := s.getAnyEpochStore()
	if es != nil {
		es.FlushHeads()
		es.FlushLastEvents()
	}
	return s.flushDBs()
}

// flushDBs performs the low-level flush of the flushable database producer.
// It writes a flush ID based on the current timestamp.
func (s *Store) flushDBs() error {
	s.prevFlushTime = time.Now()
	flushID := bigendian.Uint64ToBytes(uint64(s.prevFlushTime.UnixNano()))
	return s.dbs.Flush(flushID)
}

// EvmStore returns the EVM storage manager associated with this Store.
func (s *Store) EvmStore() *evmstore.Store {
	return s.evm
}

// CaptureEvmKvdbSnapshot creates a "frozen" snapshot of the current KVDB state for the EVM.
// This is used to allow long-running read operations (like RPC calls) to see a consistent view
// of the database even while new blocks are being committed.
func (s *Store) CaptureEvmKvdbSnapshot() {
	if s.evm.Snaps == nil {
		return
	}
	gen, err := s.evm.Snaps.Generating()
	if err != nil {
		s.Log.Error("Failed to check EVM snapshot generation", "err", err)
		return
	}
	if gen {
		return
	}
	newEvmKvdbSnap, err := s.evm.EVMDB().GetSnapshot()
	if err != nil {
		s.Log.Error("Failed to initialize frozen KVDB", "err", err)
		return
	}
	if s.snapshotedEVMDB == nil {
		s.snapshotedEVMDB = switchable.Wrap(newEvmKvdbSnap)
	} else {
		old := s.snapshotedEVMDB.SwitchTo(newEvmKvdbSnap)
		// release only after DB is atomically switched
		if old != nil {
			old.Release()
		}
	}
	// Create a new EVM store instance backed by the frozen snapshot
	newStore := s.evm.ResetWithEVMDB(snap2kvdb.Wrap(s.snapshotedEVMDB))
	newStore.Snaps = nil
	root := s.GetBlockState().FinalizedStateRoot
	err = s.generateSnapshotAt(newStore, common.Hash(root), false, false)
	if err != nil {
		s.Log.Error("Failed to initialize EVM snapshot for frozen KVDB", "err", err)
		return
	}
	s.cache.KvdbEvmSnap.Store(newStore)
}

// LastKvdbEvmSnapshot returns the most recently captured frozen EVM store.
// If none exists, it returns the current live EVM store.
func (s *Store) LastKvdbEvmSnapshot() *evmstore.Store {
	if v := s.cache.KvdbEvmSnap.Load(); v != nil {
		return v.(*evmstore.Store)
	}
	return s.evm
}

/*
 * Utils:
 */

// makeCache creates a new weighted LRU cache with the specified parameters.
// It logs a critical error if cache creation fails.
func (s *Store) makeCache(weight uint, size int) *wlru.Cache {
	cache, err := wlru.New(weight, size)
	if err != nil {
		s.Log.Crit("Failed to create LRU cache", "err", err)
		return nil
	}
	return cache
}
