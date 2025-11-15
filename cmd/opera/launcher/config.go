package launcher

import (
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/!fantom-foundation/lachesis-base/abft"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/naoina/toml"
	"gopkg.in/urfave/cli.v1"
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

func mayMakeAllConfigs(ctx *cli.Context) (*config, error) {
	// Defaults (low priority)
	cacheRatio := cacheScaler(ctx)
	cfg := config{
		Node:          defaultNodeConfig(),
		Opera:         gossip.DefaultConfig(cacheRatio),
		Emitter:       emitter.DefaultConfig(),
		TxPool:        evmcore.DefaultTxPoolConfig,
		OperaStore:    gossip.DefaultStoreConfig(cacheRatio),
		Lachesis:      abft.DefaultConfig(),
		LachesisStore: abft.DefaultStoreConfig(cacheRatio),
		VectorClock:   vecmt.DefaultConfig(cacheRatio),
	}

	if ctx.GlobalIsSet(FakeNetFlag.Name) {
		_, num, _ := parseFakeGen(ctx.GlobalString(FakeNetFlag.Name))
		cfg.Emitter = emitter.FakeConfig(num)
		setBootnodes(ctx, []string{}, &cfg.Node)
	} else {
		// "asDefault" means set network defaults
		cfg.Node.P2P.BootstrapNodes = asDefault
		cfg.Node.P2P.BootstrapNodesV5 = asDefault
	}

	// Load config file (medium priority)
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadAllConfigs(file, &cfg); err != nil {
			return &cfg, err
		}
	}
	// apply default for DB config if it wasn't touched by config file
	dbDefault := integration.DefaultDBsConfig(cacheRatio.U64, uint64(utils.MakeDatabaseHandles()))
	if len(cfg.DBs.Routing.Table) == 0 {
		cfg.DBs.Routing = dbDefault.Routing
	}
	if len(cfg.DBs.GenesisCache.Table) == 0 {
		cfg.DBs.GenesisCache = dbDefault.GenesisCache
	}
	if len(cfg.DBs.RuntimeCache.Table) == 0 {
		cfg.DBs.RuntimeCache = dbDefault.RuntimeCache
	}

	// Apply flags (high priority)
	var err error
	cfg.Opera, err = gossipConfigWithFlags(ctx, cfg.Opera)
	if err != nil {
		return nil, err
	}
	cfg.OperaStore, err = gossipStoreConfigWithFlags(ctx, cfg.OperaStore)
	if err != nil {
		return nil, err
	}
	cfg.Node = nodeConfigWithFlags(ctx, cfg.Node)
	cfg.DBs = setDBConfig(ctx, cfg.DBs, cacheRatio)

	err = setValidator(ctx, &cfg.Emitter)
	if err != nil {
		return nil, err
	}
	if cfg.Emitter.Validator.ID != 0 && len(cfg.Emitter.PrevEmittedEventFile.Path) == 0 {
		cfg.Emitter.PrevEmittedEventFile.Path = cfg.Node.ResolvePath(path.Join("emitter", fmt.Sprintf("last-%d", cfg.Emitter.Validator.ID)))
	}
	setTxPool(ctx, &cfg.TxPool)

	if err := cfg.Opera.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
