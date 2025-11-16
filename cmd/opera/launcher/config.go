package launcher

import (
	"fmt"
	"os"

	// "path"
	"reflect"

	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/naoina/toml"
	"gopkg.in/urfave/cli.v1"
	//	Packages below are not built yet in the go-opera-asset project so I need to  comment them out for now
	//	TODO: Build these dependencies and uncomment them when ready
	// "github.com/Fantom-foundation/go-opera/evmcore"
	// "github.com/Fantom-foundation/go-opera/gossip"
	// "github.com/Fantom-foundation/go-opera/gossip/emitter"
	// "github.com/Fantom-foundation/go-opera/gossip/gasprice"
	// "github.com/Fantom-foundation/go-opera/integration"
	// "github.com/Fantom-foundation/go-opera/integration/makefakegenesis"
	// "github.com/Fantom-foundation/go-opera/opera/genesis"
	// "github.com/Fantom-foundation/go-opera/opera/genesisstore"
	// futils "github.com/Fantom-foundation/go-opera/utils"
	// "github.com/Fantom-foundation/go-opera/vecmt"
)

const (
	// DefaultCacheSize is calculated as memory consumption in a worst case scenario with default configuration
	// Average memory consumption might be 3-5 times lower than the maximum
	DefaultCacheSize  = 3600
	ConstantCacheSize = 600
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		Flags:       append(nodeFlags, testFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching",
		Value: DefaultCacheSize,
	}
)

func dumpConfig(ctx *cli.Context) error {
	cfg := makeAllConfigs(ctx)
	comment := ""

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func makeAllConfigs(ctx *cli.Context) *config {
	cfg, err := mayMakeAllConfigs(ctx)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	return cfg
}

func mayMakeAllConfigs(ctx *cli.Context) (*config, error) {
	return nil, fmt.Errorf("mayMakeAllConfigs is not implemented")
}

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		return fmt.Errorf("field '%s' is not defined in %s", field, rt.String())
	},
}

type config struct {
	Node node.Config
	// Opera         gossip.Config
	// Emitter       emitter.Config
	// TxPool        evmcore.TxPoolConfig
	// OperaStore    gossip.StoreConfig
	Lachesis      abft.Config
	LachesisStore abft.StoreConfig
	// VectorClock   vecmt.IndexConfig
	// DBs           integration.DBsConfig
}

// func mayMakeAllConfigs(ctx *cli.Context) (*config, error) {
// 	// Defaults (low priority)
// 	cacheRatio := cacheScaler(ctx)
// 	cfg := config{
// 		Node:          defaultNodeConfig(),
// 		Opera:         gossip.DefaultConfig(cacheRatio),
// 		Emitter:       emitter.DefaultConfig(),
// 		TxPool:        evmcore.DefaultTxPoolConfig,
// 		OperaStore:    gossip.DefaultStoreConfig(cacheRatio),
// 		Lachesis:      abft.DefaultConfig(),
// 		LachesisStore: abft.DefaultStoreConfig(cacheRatio),
// 		VectorClock:   vecmt.DefaultConfig(cacheRatio),
// 	}

// 	if ctx.GlobalIsSet(FakeNetFlag.Name) {
// 		_, num, _ := parseFakeGen(ctx.GlobalString(FakeNetFlag.Name))
// 		cfg.Emitter = emitter.FakeConfig(num)
// 		setBootnodes(ctx, []string{}, &cfg.Node)
// 	} else {
// 		// "asDefault" means set network defaults
// 		cfg.Node.P2P.BootstrapNodes = asDefault
// 		cfg.Node.P2P.BootstrapNodesV5 = asDefault
// 	}

// 	// Load config file (medium priority)
// 	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
// 		if err := loadAllConfigs(file, &cfg); err != nil {
// 			return &cfg, err
// 		}
// 	}
// 	// apply default for DB config if it wasn't touched by config file
// 	dbDefault := integration.DefaultDBsConfig(cacheRatio.U64, uint64(utils.MakeDatabaseHandles()))
// 	if len(cfg.DBs.Routing.Table) == 0 {
// 		cfg.DBs.Routing = dbDefault.Routing
// 	}
// 	if len(cfg.DBs.GenesisCache.Table) == 0 {
// 		cfg.DBs.GenesisCache = dbDefault.GenesisCache
// 	}
// 	if len(cfg.DBs.RuntimeCache.Table) == 0 {
// 		cfg.DBs.RuntimeCache = dbDefault.RuntimeCache
// 	}

// 	// Apply flags (high priority)
// 	var err error
// 	cfg.Opera, err = gossipConfigWithFlags(ctx, cfg.Opera)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.OperaStore, err = gossipStoreConfigWithFlags(ctx, cfg.OperaStore)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.Node = nodeConfigWithFlags(ctx, cfg.Node)
// 	cfg.DBs = setDBConfig(ctx, cfg.DBs, cacheRatio)

// 	err = setValidator(ctx, &cfg.Emitter)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if cfg.Emitter.Validator.ID != 0 && len(cfg.Emitter.PrevEmittedEventFile.Path) == 0 {
// 		cfg.Emitter.PrevEmittedEventFile.Path = cfg.Node.ResolvePath(path.Join("emitter", fmt.Sprintf("last-%d", cfg.Emitter.Validator.ID)))
// 	}
// 	setTxPool(ctx, &cfg.TxPool)

// 	if err := cfg.Opera.Validate(); err != nil {
// 		return nil, err
// 	}

// 	return &cfg, nil
// }

func cacheScaler(ctx *cli.Context) cachescale.Func {
	if !ctx.GlobalIsSet(CacheFlag.Name) {
		return cachescale.Identity
	}
	targetCache := ctx.GlobalInt(CacheFlag.Name)
	baseSize := DefaultCacheSize
	if targetCache < baseSize {
		log.Crit("Invalid flag", "flag", CacheFlag.Name, "err", fmt.Sprintf("minimum cache size is %d MB", baseSize))
	}
	return cachescale.Ratio{
		Base:   uint64(baseSize - ConstantCacheSize),
		Target: uint64(targetCache - ConstantCacheSize),
	}
}
