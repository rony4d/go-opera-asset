package launcher

// Defaults bundles the baseline configuration values the launcher will use
// before flags/config files override them. Fill these out as the project evolves.

type Defaults struct {
	Node      NodeDefaults
	Network   NetworkDefaults
	Storage   StorageDefaults
	RPC       RPCDefaults
	Metrics   MetricsDefaults
	Validator ValidatorDefaults
}

// NodeDefaults captures top-level node settings (datadir, identity, etc).

type NodeDefaults struct {
	DataDir       string   //	Filesystem root where the node stores everything (chaindata, keystore, logs, errlock). Changing it lets you run multiple nodes or keep test data isolated.
	Name          string   //	Human-readable node identity advertised via enode:// and logs; helps peers/operator distinguish instances
	LightKDF      bool     //	When true, uses a weaker key-derivation function for keystore passwords so unlocking accounts is faster (good for dev/test, insecure for production).
	NoUSB         bool     //  Disables scanning hardware wallets over USB; avoids needing libusb/hid support or interacting with physical devices.
	SyncMode      string   //  Strategy for syncing the blockchain (e.g., full, snap, light); impacts what data the node downloads and how it validates history.
	MaxPeers      int      //  Upper bound on concurrent P2P peers; protects CPU/bandwidth and controls network exposure.
	ListenAddr    string   //  IP/interface the node binds to for incoming p2p connections (e.g., 0.0.0.0 for all interfaces or 127.0.0.1 for local-only).
	ListenPort    int      //  TCP/UDP port used for p2p discovery and DevP2P traffic.
	ExternalIP    string   //  Public IP advertised to peers when NAT discovery isn’t available; helps others connect back to you.
	StaticNodes   []string //  List of enode URLs the node always attempts to connect to; useful for bootstrapping or pinning trusted peers
	TrustedNodes  []string //  Peers allowed to stay connected even when above MaxPeers; ensures whitelisted validators/operators retain connectivity.
	DiscoveryURLs []string //   DNS discovery endpoints (EIP-1459 style) the node polls to discover bootnodes; complements static bootnode lists.

}

// NetworkDefaults holds chain rules and bootnode info.
type NetworkDefaults struct {
	NetworkID   uint64   //  Unique identifier for the network (e.g., 1 for mainnet, 2 for testnet, 3 for devnet). The numerical chain identifier used to distinguish this network from others. It’s embedded in consensus rules, transactions, and RPC responses. Matching NetworkIDs across nodes ensures they only sync with peers on the same Opera network (e.g., mainnet vs fakenet).
	ChainName   string   //  Human-readable name for the network (e.g., “Mainnet”, “Testnet”, “Devnet”). This is displayed in logs and user interfaces to help operators identify which network they’re running on. Human-friendly name for the network preset (e.g., mainnet, testnet, fakenet). The name is surfaced in logs, config dumps, and RPC responses so operators know which network they’re attached to.
	Bootnodes   []string //  Enode URLs the node dials during startup to discover peers. These are hard-coded P2P endpoints that seed the discovery table; without them a fresh node might have no way to join the network.
	FakeNetSize int      //  Specific to the deterministic fakenet helper; it tells the launcher how many validator slots exist in the synthetic network so it can derive the correct validator key pairs and genesis parameters. For example, 1 yields a single-validator PoA chain, while 5 would generate five validator configs.
}

// StorageDefaults configures database/cache behaviour.
type StorageDefaults struct {
	CacheSizeMB int    //	Amount of memory (in megabytes) reserved for on-disk database caches (LevelDB/pebble) and in-memory state caches. Larger values reduce disk I/O but increase RAM footprint; CacheSizeMB tunes this balance.
	Handles     int    //	Number of file handles the node opens for database operations; higher values allow more concurrent operations but risk running out of OS resources. Handles tunes this balance between concurrency and resource usage.
	GCMode      string //	Garbage-collection strategy for historical state data. Typical values mirror geth, e.g. full (keep all receipts/state), archive (no pruning), or light. This setting dictates whether old state is pruned during runtime or kept for archival queries.
	DBPreset    string //	Database preset to use (e.g., default, light); impacts the database schema and indexing strategy. DBPreset customizes this for different use cases (e.g., full node vs light client).
}

// RPCDefaults captures HTTP/WS/IPC options.
type RPCDefaults struct {
	EnableHTTP bool     //	Toggle for the JSON-RPC HTTP server; when true the node listens for HTTP requests (Metamask, curl, etc.).
	HTTPAddr   string   //	IP/interface the HTTP server binds to for incoming requests (e.g., 0.0.0.0 for all interfaces or 127.0.0.1 for local-only).
	HTTPPort   int      //	TCP port clients connect to for HTTP RPC; default 18545 to avoid colliding with Geth’s 8545.
	HTTPAPI    []string //	API modules exposed via HTTP; e.g., eth, web3, debug, txpool, etc. This list determines which RPC endpoints are available to clients.

	EnableWS bool     //	Toggle for the JSON-RPC WebSocket server; when true the node listens for WebSocket requests (Metamask, websocat, etc.).
	WSAddr   string   //	IP/interface the WebSocket server binds to for incoming connections (e.g., 0.0.0.0 for all interfaces or 127.0.0.1 for local-only).
	WSPort   int      //	TCP port clients connect to for WebSocket RPC; default 18546 to avoid colliding with Geth’s 8546.
	WSAPI    []string //	API modules exposed via WebSocket; e.g., eth, web3, debug, txpool, etc. This list determines which RPC endpoints are available to clients.

	EnableIPC bool   //	Toggle for the JSON-RPC IPC (Inter-Process Communication) server; when true the node listens for local socket requests (e.g., geth attach). IPC stands for Inter-Process Communication. On Opera/go-ethereum style nodes it refers to the local Unix-domain socket (opera.ipc) that client tools (like opera attach) connect to for JSON-RPC calls. It never leaves the machine—unlike HTTP/WS, it’s a filesystem socket—so commands run locally can talk to the node without exposing ports over the network.
	IPCPath   string //	Path to the local Unix-domain socket file that IPC clients (e.g., opera attach) connect to. This is where the node listens for local JSON-RPC requests from tools like opera attach. It’s a filesystem socket so it never leaves the machine—unlike HTTP/WS, it’s a local-only communication channel.
	GraphQL   bool   //	Toggle for the GraphQL server; when true the node exposes a GraphQL endpoint for querying the blockchain.
}

type MetricsDefaults struct {
	Enable          bool   //	Toggle for the metrics server; when true the node exposes Prometheus-compatible metrics on the specified IP/port.
	EnableExpensive bool   //	Toggle for expensive metrics; when true the node exposes additional metrics that may impact performance (e.g., block processing stats).
	HTTPAddr        string //	IP/interface the metrics server binds to for incoming requests (e.g., 0.0.0.0 for all interfaces or 127.0.0.1 for local-only).
	HTTPPort        int    //	TCP port clients connect to for metrics; default 6060.
	InfluxEnabled   bool   //	Toggle for InfluxDB metrics; when true the node sends metrics to InfluxDB.
}

// ValidatorDefaults stores defaults for validator-related CLI.
type ValidatorDefaults struct {
	Enabled        bool     //	Whether validator mode should start by default (emit blocks/events).
	ID             uint32   //	Validator index in the genesis/fakenet configuration; tells the emitter which validator slot to take.
	PubKeyHex      string   //	Hex-encoded validator BLS/EC  public key expected by the network. Used to match the local keystore key.
	SignerPassword string   //	Password to unlock the validator key inline (not recommended; better use a file).
	PasswordFile   string   //	Path to a file containing the validator’s password. This is used to unlock the validator key.
	UnlockAccounts []string //	List of account addresses to unlock automatically when the node starts.
}

// TxPoolDefaults tunes the transaction pool.
type TxPoolDefaults struct {
	Journal       string
	PriceLimit    uint64
	PriceBump     uint64
	AccountSlots  uint64
	GlobalSlots   uint64
	AccountQueue  uint64
	GlobalQueue   uint64
	TxLifetimeSec uint64
}
