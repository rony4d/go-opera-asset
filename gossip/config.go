// This file (config.go) defines the configuration structures (Config, ProtocolConfig, StoreConfig)
// and their default values. It serves as the central place to tune system parameters
// like cache sizes, network timeouts, gas limits, and resource semaphore limits.
// It also includes validation logic to ensure configuration parameters are consistent and safe.

package gossip

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Fantom-foundation/lachesis-base/gossip/dagprocessor"
	"github.com/Fantom-foundation/lachesis-base/gossip/itemsfetcher"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/rony4d/go-opera-asset/gossip/protocols/blockrecords/brprocessor"
	"github.com/rony4d/go-opera-asset/gossip/protocols/blockvotes/bvprocessor"
	"github.com/rony4d/go-opera-asset/gossip/protocols/epochpacks/epprocessor"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// nominalSize is a baseline constant used for cache sizing calculations.
const nominalSize uint = 1

type (
	// ProtocolConfig controls the behavior of the P2P gossip protocol.
	// It tunes how the node interacts with peers, handles latency vs throughput trade-offs,
	// and manages resource limits for incoming data.
	ProtocolConfig struct {
		// LatencyImportance vs ThroughputImportance allows fine-tuning the peer selection and data propagation strategy.
		// 0/M means "optimize only for throughput", N/0 means "optimize only for latency", N/M is a balanced mode.
		LatencyImportance    int
		ThroughputImportance int

		// Resource limits for various internal queues and semaphores.
		// These prevent the node from being overwhelmed by too much incoming data.
		EventsSemaphoreLimit dag.Metric
		BVsSemaphoreLimit    dag.Metric
		MsgsSemaphoreLimit   dag.Metric
		MsgsSemaphoreTimeout time.Duration

		// ProgressBroadcastPeriod defines how often the node broadcasts its current status/progress to peers.
		ProgressBroadcastPeriod time.Duration

		// Sub-configurations for specific protocol processors (DAG, Block Votes, Block Records, Epoch Packs).
		DagProcessor dagprocessor.Config
		BvProcessor  bvprocessor.Config
		BrProcessor  brprocessor.Config
		EpProcessor  epprocessor.Config

		// Configurations for fetching items and streaming data.
		DagFetcher       itemsfetcher.Config
		TxFetcher        itemsfetcher.Config
		DagStreamLeecher dagstreamleecher.Config
		DagStreamSeeder  dagstreamseeder.Config
		BvStreamLeecher  bvstreamleecher.Config
		BvStreamSeeder   bvstreamseeder.Config
		BrStreamLeecher  brstreamleecher.Config
		BrStreamSeeder   brstreamseeder.Config
		EpStreamLeecher  epstreamleecher.Config
		EpStreamSeeder   epstreamseeder.Config

		// Limits and periods for transaction propagation.
		MaxInitialTxHashesSend   int
		MaxRandomTxHashesSend    int
		RandomTxHashesSendPeriod time.Duration

		PeerCache PeerCacheConfig
	}

	// Config is the top-level configuration structure for the gossip service.
	Config struct {
		FilterAPI filters.Config

		// URLs for node discovery (bootnodes).
		// This can be set to list of enrtree:// URLs which will be queried for
		// for nodes to connect to.
		OperaDiscoveryURLs []string
		SnapDiscoveryURLs  []string

		// AllowSnapsync enables the snapshot synchronization mode.
		AllowSnapsync bool

		// TxIndex controls whether to enable indexing transactions and receipts.
		// If true, transaction history will be searchable.
		TxIndex bool

		// Protocol options
		Protocol ProtocolConfig

		// HeavyCheck configuration for intensive validation logic.
		HeavyCheck heavycheck.Config

		// Gas Price Oracle (GPO) options for estimating transaction fees.
		GPO gasprice.Config

		// RPCGasCap is the global gas cap for eth_call variants.
		RPCGasCap uint64 `toml:",omitempty"`

		// RPCTxFeeCap is the global transaction fee (price * gaslimit) cap for
		// send-transaction variants. The unit is ether.
		RPCTxFeeCap float64 `toml:",omitempty"`

		// RPCTimeout is a global time limit for RPC methods execution.
		RPCTimeout time.Duration

		// AllowUnprotectedTxs determines if non-EIP155 transactions (without chain ID) are allowed.
		AllowUnprotectedTxs bool

		// RPCBlockExt enables extended block information in RPC responses.
		RPCBlockExt bool
	}

	// StoreCacheConfig defines the sizes for various in-memory caches used by the store.
	// These are critical for performance, balancing memory usage against disk I/O.
	StoreCacheConfig struct {
		// Cache size for full events.
		EventsNum  int
		EventsSize uint
		// Cache size for event IDs
		EventsIDsNum int
		// Cache size for full blocks.
		BlocksNum  int
		BlocksSize uint
		// Cache size for history block/epoch states.
		BlockEpochStateNum int

		LlrBlockVotesIndexes int
		LlrEpochVotesIndexes int
	}

	// StoreConfig is the configuration for the persistent storage layer.
	StoreConfig struct {
		Cache StoreCacheConfig
		// EVM is EVM store config
		EVM evmstore.StoreConfig
		// MaxNonFlushedSize is the threshold of data in memory before a disk flush is triggered.
		MaxNonFlushedSize int
		// MaxNonFlushedPeriod is the maximum time data can stay in memory before flushing.
		MaxNonFlushedPeriod time.Duration
	}
)

// PeerCacheConfig defines limits for caching peer-related data to prevent DoS attacks.
type PeerCacheConfig struct {
	MaxKnownTxs    int // Maximum transactions hashes to keep in the known list (prevent DOS)
	MaxKnownEvents int // Maximum event hashes to keep in the known list (prevent DOS)
	// MaxQueuedItems is the maximum number of items to queue up before
	// dropping broadcasts. This is a sensitive number as a transaction list might
	// contain a single transaction, or thousands.
	MaxQueuedItems idx.Event
	MaxQueuedSize  uint64
}

// DefaultConfig returns the standard configuration for the gossip service.
// It accepts a `scale` function to adjust parameters based on the system's resources (e.g., RAM size).
func DefaultConfig(scale cachescale.Func) Config {
	cfg := Config{
		FilterAPI: filters.DefaultConfig(),

		TxIndex: true,

		HeavyCheck: heavycheck.DefaultConfig(),

		Protocol: ProtocolConfig{
			LatencyImportance:    60,
			ThroughputImportance: 40,
			MsgsSemaphoreLimit: dag.Metric{
				Num:  scale.Events(1000),
				Size: scale.U64(30 * opt.MiB),
			},
			EventsSemaphoreLimit: dag.Metric{
				Num:  scale.Events(10000),
				Size: scale.U64(30 * opt.MiB),
			},
			BVsSemaphoreLimit: dag.Metric{
				Num:  scale.Events(5000),
				Size: scale.U64(15 * opt.MiB),
			},
			MsgsSemaphoreTimeout:    10 * time.Second,
			ProgressBroadcastPeriod: 10 * time.Second,

			DagProcessor: dagprocessor.DefaultConfig(scale),
			BvProcessor:  bvprocessor.DefaultConfig(scale),
			BrProcessor:  brprocessor.DefaultConfig(scale),
			EpProcessor:  epprocessor.DefaultConfig(scale),
			DagFetcher: itemsfetcher.Config{
				ForgetTimeout:       1 * time.Minute,
				ArriveTimeout:       1000 * time.Millisecond,
				GatherSlack:         100 * time.Millisecond,
				HashLimit:           20000,
				MaxBatch:            scale.I(512),
				MaxQueuedBatches:    scale.I(32),
				MaxParallelRequests: 192,
			},
			TxFetcher: itemsfetcher.Config{
				ForgetTimeout:       1 * time.Minute,
				ArriveTimeout:       1000 * time.Millisecond,
				GatherSlack:         100 * time.Millisecond,
				HashLimit:           10000,
				MaxBatch:            scale.I(512),
				MaxQueuedBatches:    scale.I(32),
				MaxParallelRequests: 64,
			},
			DagStreamLeecher:         dagstreamleecher.DefaultConfig(),
			DagStreamSeeder:          dagstreamseeder.DefaultConfig(scale),
			BvStreamLeecher:          bvstreamleecher.DefaultConfig(),
			BvStreamSeeder:           bvstreamseeder.DefaultConfig(scale),
			BrStreamLeecher:          brstreamleecher.DefaultConfig(),
			BrStreamSeeder:           brstreamseeder.DefaultConfig(scale),
			EpStreamLeecher:          epstreamleecher.DefaultConfig(),
			EpStreamSeeder:           epstreamseeder.DefaultConfig(scale),
			MaxInitialTxHashesSend:   20000,
			MaxRandomTxHashesSend:    128,
			RandomTxHashesSendPeriod: 20 * time.Second,
			PeerCache:                DefaultPeerCacheConfig(scale),
		},

		GPO: gasprice.Config{
			MaxGasPrice:      gasprice.DefaultMaxGasPrice,
			MinGasPrice:      new(big.Int),
			DefaultCertainty: 0.5 * gasprice.DecimalUnit,
		},

		RPCBlockExt: true,

		RPCGasCap:   50000000,
		RPCTxFeeCap: 100, // 100 FTM
		RPCTimeout:  5 * time.Second,
	}
	// Apply dynamic adjustments to protocol limits based on session configuration
	sessionCfg := cfg.Protocol.DagStreamLeecher.Session
	cfg.Protocol.DagProcessor.EventsBufferLimit.Num = idx.Event(sessionCfg.ParallelChunksDownload)*
		idx.Event(sessionCfg.DefaultChunkItemsNum) + softLimitItems
	cfg.Protocol.DagProcessor.EventsBufferLimit.Size = uint64(sessionCfg.ParallelChunksDownload)*sessionCfg.DefaultChunkItemsSize + 8*opt.MiB
	cfg.Protocol.DagStreamLeecher.MaxSessionRestart = 4 * time.Minute
	cfg.Protocol.DagFetcher.ArriveTimeout = 4 * time.Second
	cfg.Protocol.DagFetcher.HashLimit = 10000
	cfg.Protocol.TxFetcher.HashLimit = 10000

	return cfg
}

// Validate checks the consistency of the configuration parameters.
// It ensures that buffer limits are large enough to handle maximum message sizes
// and that related parameters are logically coherent (e.g., semaphore limits >= buffer limits).
func (c *Config) Validate() error {
	p := c.Protocol
	defaultChunkSize := dag.Metric{idx.Event(p.DagStreamLeecher.Session.DefaultChunkItemsNum), p.DagStreamLeecher.Session.DefaultChunkItemsSize}
	if defaultChunkSize.Num > hardLimitItems-1 {
		return fmt.Errorf("DefaultChunkSize.Num has to be at not greater than %d", hardLimitItems-1)
	}
	if defaultChunkSize.Size > protocolMaxMsgSize/2 {
		return fmt.Errorf("DefaultChunkSize.Num has to be at not greater than %d", protocolMaxMsgSize/2)
	}
	if p.EventsSemaphoreLimit.Num < 2*defaultChunkSize.Num ||
		p.EventsSemaphoreLimit.Size < 2*defaultChunkSize.Size {
		return fmt.Errorf("EventsSemaphoreLimit has to be at least 2 times greater than %s (DefaultChunkSize)", defaultChunkSize.String())
	}
	if p.EventsSemaphoreLimit.Num < 2*p.DagProcessor.EventsBufferLimit.Num ||
		p.EventsSemaphoreLimit.Size < 2*p.DagProcessor.EventsBufferLimit.Size {
		return fmt.Errorf("EventsSemaphoreLimit has to be at least 2 times greater than %s (EventsBufferLimit)", p.DagProcessor.EventsBufferLimit.String())
	}
	if p.EventsSemaphoreLimit.Size < 2*protocolMaxMsgSize {
		return fmt.Errorf("EventsSemaphoreLimit.Size has to be at least %d", 2*protocolMaxMsgSize)
	}
	if p.MsgsSemaphoreLimit.Size < protocolMaxMsgSize {
		return fmt.Errorf("MsgsSemaphoreLimit.Size has to be at least %d", protocolMaxMsgSize)
	}
	if p.DagProcessor.EventsBufferLimit.Size < protocolMaxMsgSize {
		return fmt.Errorf("EventsBufferLimit.Size has to be at least %d", protocolMaxMsgSize)
	}

	return nil
}

// DefaultStoreConfig returns the default configuration for the storage layer.
// It sizes caches appropriate for a production environment, scaling with system resources.
func DefaultStoreConfig(scale cachescale.Func) StoreConfig {
	return StoreConfig{
		Cache: StoreCacheConfig{
			EventsNum:            scale.I(5000),
			EventsSize:           scale.U(6 * opt.MiB),
			EventsIDsNum:         scale.I(100000),
			BlocksNum:            scale.I(5000),
			BlocksSize:           scale.U(512 * opt.KiB),
			BlockEpochStateNum:   scale.I(8),
			LlrBlockVotesIndexes: scale.I(100),
			LlrEpochVotesIndexes: scale.I(5),
		},
		EVM:                 evmstore.DefaultStoreConfig(scale),
		MaxNonFlushedSize:   21*opt.MiB + scale.I(2*opt.MiB),
		MaxNonFlushedPeriod: 30 * time.Minute,
	}
}

// LiteStoreConfig provides a lightweight configuration suitable for tests or low-resource environments.
// It effectively divides the default resource usage by 10.
func LiteStoreConfig() StoreConfig {
	return DefaultStoreConfig(cachescale.Ratio{Base: 10, Target: 1})
}

// DefaultPeerCacheConfig returns the default limits for per-peer caches.
func DefaultPeerCacheConfig(scale cachescale.Func) PeerCacheConfig {
	return PeerCacheConfig{
		MaxKnownTxs:    24576*3/4 + scale.I(24576/4),
		MaxKnownEvents: 24576*3/4 + scale.I(24576/4),
		MaxQueuedItems: 4096*3/4 + scale.Events(4096/4),
		MaxQueuedSize:  protocolMaxMsgSize*3/4 + 1024 + scale.U64(protocolMaxMsgSize/4),
	}
}
