// This file maps CLI context to config struct; placeholders for node/p2p/app configs

// NOTE: This file is a placeholder and most of the data may change as the project evolves

package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

// Config aggregates every subsystem’s configuration the launcher needs.
type Config struct {
	Node          NodeConfig
	Opera         OperaConfig
	Emitter       EmitterConfig
	TxPool        TxPoolConfig
	OperaStore    StoreConfig
	Lachesis      LachesisConfig
	LachesisStore LachesisStoreConfig
	VectorClock   VectorClockConfig
	DBs           DBsConfig
}

// MakeConfig merges defaults, optional config file, then CLI flag overrides.

type NodeConfig struct {
	DataDir string
	Name    string
	P2P     P2PConfig
	RPC     RPCConfig
	Logging LoggingConfig
}

type P2PConfig struct {
	ListenAddr string
	ListenPort int
	MaxPeers   int
	Bootnodes  []string
}

type RPCConfig struct {
	HTTPEnabled bool
	HTTPAddr    string
	HTTPPort    int
	HTTPAPI     []string

	EnableWS bool
	WSAddr   string
	WSPort   int
	WSAPI    []string

	EnableIPC bool
	IPCPath   string
}

type LoggingConfig struct {
	Verbosity int
	Format    string
	Color     bool
}

type OperaConfig struct {
	NetworkName string
	NetworkID   uint64
	FakeNet     bool
}

type EmitterConfig struct {
	Enabled        bool
	ValidatorID    uint32
	ValidatorKey   string // hex public key for now
	Password       string // TODO: replace with secure keystore handling
	PasswordFile   string
	UnlockAccounts []string
}

type TxPoolConfig struct {
	Journal       string
	PriceLimit    uint64
	PriceBump     uint64
	AccountSlots  uint64
	GlobalSlots   uint64
	AccountQueue  uint64
	GlobalQueue   uint64
	TxLifetimeSec uint64
}

type StoreConfig struct {
	Path    string
	CacheMB int
}

type LachesisConfig struct {
	MaxEpochBlocks uint64
	MaxEpochTime   string // use duration strings until the engine is ready
}

type LachesisStoreConfig struct {
	CacheMB int
}

type VectorClockConfig struct {
	CacheSize uint32
}

type DBsConfig struct {
	RootDir      string
	RuntimeCache int
	Routing      map[string]string
}

// -----------------------------------------------------------------------------
// Default config + builders
// -----------------------------------------------------------------------------

//	Default config function creates a default config object using the DefaultConfig function from defaults.go file in launcher package
//	This keeps this main config file clean and in sync with the defaults.go file

func defaultConfig() Config {
	home := GuessHomeDir()
	return Config{
		Node: NodeConfig{
			DataDir: filepath.Join(home, ".opera"),
			Name:    DefaultConfig().Node.Name,
			P2P: P2PConfig{
				ListenAddr: DefaultConfig().Node.ListenAddr,
				ListenPort: DefaultConfig().Node.ListenPort,
				MaxPeers:   DefaultConfig().Node.MaxPeers,
				Bootnodes:  DefaultConfig().Network.Bootnodes,
			},
			RPC: RPCConfig{
				HTTPEnabled: true,
				HTTPAddr:    DefaultConfig().RPC.HTTPAddr,
				HTTPPort:    DefaultConfig().RPC.HTTPPort,
				HTTPAPI:     DefaultConfig().RPC.HTTPAPI,
				EnableWS:    DefaultConfig().RPC.EnableWS,
				WSAddr:      DefaultConfig().RPC.WSAddr,
				WSPort:      DefaultConfig().RPC.WSPort,
				WSAPI:       DefaultConfig().RPC.WSAPI,
				EnableIPC:   DefaultConfig().RPC.EnableIPC,
				IPCPath:     DefaultConfig().RPC.IPCPath,
			},
			Logging: LoggingConfig{
				Verbosity: DefaultConfig().Logging.Verbosity,
				Format:    DefaultConfig().Logging.Format,
				Color:     DefaultConfig().Logging.Color,
			},
		},
		Opera: OperaConfig{
			NetworkName: DefaultConfig().Network.ChainName,
			NetworkID:   DefaultConfig().Network.NetworkID,
			FakeNet:     DefaultConfig().Network.FakeNet,
		},
		Emitter: EmitterConfig{},
		TxPool: TxPoolConfig{
			Journal:       DefaultConfig().TxPool.Journal,
			PriceLimit:    DefaultConfig().TxPool.PriceLimit,
			PriceBump:     DefaultConfig().TxPool.PriceBump,
			AccountSlots:  DefaultConfig().TxPool.AccountSlots,
			GlobalSlots:   DefaultConfig().TxPool.GlobalSlots,
			AccountQueue:  DefaultConfig().TxPool.AccountQueue,
			GlobalQueue:   DefaultConfig().TxPool.GlobalQueue,
			TxLifetimeSec: DefaultConfig().TxPool.TxLifetimeSec,
		},
		OperaStore:    StoreConfig{Path: "chaindata", CacheMB: 1024},
		Lachesis:      LachesisConfig{MaxEpochBlocks: 1000, MaxEpochTime: "24h"},
		LachesisStore: LachesisStoreConfig{CacheMB: 512},
		VectorClock:   VectorClockConfig{CacheSize: 64 * 1024},
		DBs:           DBsConfig{RootDir: "databases", RuntimeCache: 1024, Routing: map[string]string{}},
	}
}

// makeAllConfigs mirrors the launcher’s current behaviour: merge defaults,
// config-file values, and CLI overrides into a single config struct.

func MakeAllConfigs(ctx *cli.Context) Config {
	cfg := defaultConfig()

	if file := ctx.String("config"); file != "" {
		if err := loadConfigFile(file, &cfg); err != nil {
			// In this placeholder we simply panic; in the real launcher return the error.
			panic(fmt.Errorf("failed to load config file %s: %w", file, err))
		}
	}

	applyCLIOverrides(ctx, &cfg)

	if err := ensureDir(cfg.Node.DataDir); err != nil {
		panic(err)
	}
	return cfg
}

// -----------------------------------------------------------------------------
// Config-file / CLI wiring
// -----------------------------------------------------------------------------

func loadConfigFile(path string, cfg *Config) error {
	// TODO: when ready, decode TOML into cfg using naoinna/toml or encoding/json.
	return nil
}

func applyCLIOverrides(ctx *cli.Context, cfg *Config) {
	if ctx.IsSet("datadir") {
		cfg.Node.DataDir = resolvePath(ctx.String("datadir"))
	}
	if ctx.IsSet("identity") {
		cfg.Node.Name = ctx.String("identity")
	}

	if ctx.IsSet("port") {
		cfg.Node.P2P.ListenPort = ctx.Int("port")
	}
	if ctx.IsSet("maxpeers") {
		cfg.Node.P2P.MaxPeers = ctx.Int("maxpeers")
	}
	if ctx.IsSet("bootnodes") {
		cfg.Node.P2P.Bootnodes = splitCSV(ctx.String("bootnodes"))
	}

	if ctx.Bool("http") {
		cfg.Node.RPC.HTTPEnabled = true
	}
	if ctx.IsSet("http.addr") {
		cfg.Node.RPC.HTTPAddr = ctx.String("http.addr")
	}
	if ctx.IsSet("http.port") {
		cfg.Node.RPC.HTTPPort = ctx.Int("http.port")
	}
	if ctx.IsSet("http.api") {
		cfg.Node.RPC.HTTPAPI = splitCSV(ctx.String("http.api"))
	}
	if ctx.Bool("ws") {
		cfg.Node.RPC.EnableWS = true
	}
	if ctx.IsSet("ws.addr") {
		cfg.Node.RPC.WSAddr = ctx.String("ws.addr")
	}
	if ctx.IsSet("ws.port") {
		cfg.Node.RPC.WSPort = ctx.Int("ws.port")
	}
	if ctx.IsSet("ws.api") {
		cfg.Node.RPC.WSAPI = splitCSV(ctx.String("ws.api"))
	}
	if ctx.IsSet("ipc") {
		cfg.Node.RPC.EnableIPC = ctx.Bool("ipc")
	}
	if ctx.IsSet("ipc.path") {
		cfg.Node.RPC.IPCPath = ctx.String("ipc.path")
	}

	if ctx.IsSet("log.format") {
		cfg.Node.Logging.Format = ctx.String("log.format")
	}
	if ctx.IsSet("log.verbosity") {
		cfg.Node.Logging.Verbosity = ctx.Int("log.verbosity")
	}
	if ctx.IsSet("log.color") {
		cfg.Node.Logging.Color = ctx.Bool("log.color")
	}

	if ctx.IsSet("txpool.journal") {
		cfg.TxPool.Journal = ctx.String("txpool.journal")
	}
	if ctx.IsSet("txpool.pricelimit") {
		cfg.TxPool.PriceLimit = ctx.Uint64("txpool.pricelimit")
	}
	if ctx.IsSet("txpool.pricebump") {
		cfg.TxPool.PriceBump = ctx.Uint64("txpool.pricebump")
	}
	if ctx.IsSet("txpool.localslots") {
		cfg.TxPool.AccountSlots = uint64(ctx.Int("txpool.localslots"))
	}
	if ctx.IsSet("txpool.globalslots") {
		cfg.TxPool.GlobalSlots = uint64(ctx.Int("txpool.globalslots"))
	}
	if ctx.IsSet("txpool.localqueue") {
		cfg.TxPool.AccountQueue = uint64(ctx.Int("txpool.localqueue"))
	}
	if ctx.IsSet("txpool.globalqueue") {
		cfg.TxPool.GlobalQueue = uint64(ctx.Int("txpool.globalqueue"))
	}
	if ctx.IsSet("txpool.lifetime") {
		cfg.TxPool.TxLifetimeSec = ctx.Uint64("txpool.lifetime")
	}

	if ctx.IsSet("genesis") {
		// cfg.Genesis.Path = ctx.String("genesis")
	}
	if ctx.IsSet("fakenet") {
		cfg.Opera.FakeNet = true
		cfg.Opera.NetworkName = "fakenet"
		cfg.Opera.NetworkID = uint64(ctx.Int("fakenet"))
	}
	if ctx.IsSet("cache") {
		cfg.OperaStore.CacheMB = ctx.Int("cache")
		cfg.DBs.RuntimeCache = ctx.Int("cache")
	}
	if ctx.IsSet("gcmode") {
		cfg.OperaStore.Path = ctx.String("gcmode") // placeholder; replace with real GC mode handling
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create datadir %s: %w", dir, err)
	}
	return nil
}

func resolvePath(p string) string {
	if strings.HasPrefix(p, "~") {
		return filepath.Join(GuessHomeDir(), strings.TrimPrefix(p, "~"))
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(GuessWorkDir(), p)
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func GuessWorkDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func GuessHomeDir() string {
	if dir, err := os.UserHomeDir(); err == nil {
		return dir
	}
	return "."
}

func GuessProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd // hit filesystem root without finding go.mod
		}
		dir = parent
	}
}
