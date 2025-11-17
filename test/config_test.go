package test

import (
	// "gopkg.in/urfave/cli.v1"
	// "runtime"
	"strings"

	"path/filepath"
	"testing"

	"gopkg.in/urfave/cli.v1"

	"github.com/rony4d/go-opera-asset/cmd/opera/launcher"
	"github.com/rony4d/go-opera-asset/flags"
)

// helper to run makeAllConfigs with a synthetic CLI context.

func runConfigFromArgs(t *testing.T, args []string) launcher.Config {

	t.Helper()

	app := cli.NewApp()

	app.HideHelp = true
	app.HideVersion = true

	// Register the subset of flags we want to exercise.

	networkFlags := flags.NetworkFlags()
	txPoolFlags := flags.TxPoolFlags()
	commonFlags := flags.CommonFlags()
	nodeFlags := flags.NodeFlags()

	app.Flags = append(app.Flags, networkFlags...)
	app.Flags = append(app.Flags, txPoolFlags...)
	app.Flags = append(app.Flags, commonFlags...)
	app.Flags = append(app.Flags, nodeFlags...)

	//	Get an instance of the Config struct that we want to bind to the flags
	var got launcher.Config

	app.Action = func(c *cli.Context) error {
		got = launcher.MakeAllConfigs(c)
		return nil
	}

	if err := app.Run(append([]string{"opera"}, args...)); err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
	return got
}

// TestMakeAllConfigs_flagOverrides verifies that every command-line flag we declare
// in the launcher correctly overrides the corresponding field in the aggregated
// Config struct. The test iterates through representative flag combinations and
// asserts that MakeAllConfigs applies them as expected.
//
// Each sub-test feeds custom CLI arguments into a synthetic app, invokes
// launcher.MakeAllConfigs, and checks the bits of the resulting struct that should
// have changed.
func TestMakeAllConfigs_flagOverrides(t *testing.T) {

	projectRoot := launcher.GuessProjectRoot()

	// t.Skip("Remove when MakeAllConfigs is wired with the placeholder structs") // skip until MakeAllConfigs is ready

	tests := []struct {
		name string                                  // descriptive name for the scenario
		args []string                                // CLI arguments to feed into makeAllConfigs
		want func(t *testing.T, cfg launcher.Config) // assertion helper examining the final config
	}{
		{
			name: "datadir and identity",
			args: []string{"--datadir", projectRoot + "/opera-asset-chain/devnet/node-data", "--identity", "ugo-node"}, //	NOTE: this is the command line argument that overrides the default data dir and identity
			want: func(t *testing.T, cfg launcher.Config) {

				// Expect the datadir to be overridden by the --datadir flag command line argument.
				if cfg.Node.DataDir != filepath.Join(projectRoot+"/opera-asset-chain/devnet/node-data") {
					t.Fatalf("Datadir = %q, want %q", cfg.Node.DataDir, filepath.Join(projectRoot+"/opera-asset-chain/devnet/node-data"))
				}
				t.Logf("cfg.Node.DataDir = %q", cfg.Node.DataDir) //	NOTE: this will only be printed if the test fails

				// Expect the identity to be overridden by the --identity flag command line argument.
				if cfg.Node.Name != "ugo-node" {
					t.Fatalf("Identity = %q, want go-opera", cfg.Node.Name)
				}
				t.Logf("cfg.Node.Name = %q", cfg.Node.Name) //	NOTE: this will only be printed if the test fails

			},
		},

		{
			name: "P2P and bootnodes",
			args: []string{"--port", "5151", "--maxpeers", "99", "--bootnodes", "enode://abc@1.2.3.4:5050,enode://def@5.6.7.8:5050"},
			want: func(t *testing.T, cfg launcher.Config) {
				// port -> ListenPort override
				if cfg.Node.P2P.ListenPort != 5151 {
					t.Fatalf("ListenPort = %d, want 5151", cfg.Node.P2P.ListenPort)
				}
				// maxpeers -> MaxPeers
				if cfg.Node.P2P.MaxPeers != 99 {
					t.Fatalf("MaxPeers = %d, want 99", cfg.Node.P2P.MaxPeers)
				}
				// bootnodes list should split on comma and trim whitespace.
				if len(cfg.Node.P2P.Bootnodes) != 2 || cfg.Node.P2P.Bootnodes[0] != "enode://abc@1.2.3.4:5050" {
					t.Fatalf("Bootnodes = %#v, want two entries", cfg.Node.P2P.Bootnodes)
				}
			},
		},
		{
			name: "RPC toggle and APIs",
			args: []string{
				"--http", "--http.addr", "0.0.0.0", "--http.port", "19545", "--http.api", "eth,net,web3,ftm",
				"--ws", "--ws.addr", "0.0.0.0", "--ws.port", "19546", "--ws.api", "eth,net",
				"--ipc", "--ipc.path", "/tmp/opera.ipc",
			},
			want: func(t *testing.T, cfg launcher.Config) {

				// HTTP flags toggled and addresses applied.
				if !cfg.Node.RPC.HTTPEnabled || cfg.Node.RPC.HTTPAddr != "0.0.0.0" || cfg.Node.RPC.HTTPPort != 19545 {
					t.Fatalf("HTTP config not applied: %#v", cfg.Node.RPC)
				}
				// API list trimmed and split.
				if strings.Join(cfg.Node.RPC.HTTPAPI, ",") != "eth,net,web3,ftm" {
					t.Fatalf("HTTP API = %v", cfg.Node.RPC.HTTPAPI)
				}
				// WS overrides valid.
				if !cfg.Node.RPC.EnableWS || cfg.Node.RPC.WSPort != 19546 {
					t.Fatalf("WS config not applied: %#v", cfg.Node.RPC)
				}
				// IPC flag + path.
				if !cfg.Node.RPC.EnableIPC || cfg.Node.RPC.IPCPath != "/tmp/opera.ipc" {
					t.Fatalf("IPC config not applied: %#v", cfg.Node.RPC)
				}
			},
		},

		{
			name: "Txpool overrides",
			args: []string{
				"--txpool.journal", "custom.rlp",
				"--txpool.pricelimit", "5",
				"--txpool.pricebump", "20",
				"--txpool.localslots", "42",
				"--txpool.globalslots", "999",
				"--txpool.localqueue", "12",
				"--txpool.globalqueue", "777",
				"--txpool.lifetime", "3600",
			},
			want: func(t *testing.T, cfg launcher.Config) {
				got := cfg.TxPool
				// Strings and scalar overrides should pass through verbatim.
				if got.Journal != "custom.rlp" || got.PriceLimit != 5 || got.PriceBump != 20 {
					t.Fatalf("TxPool basic overrides failed: %#v", got)
				}
				// Numeric fields for slots/queues/lifetime rely on conversions; verify each.
				if got.AccountSlots != 42 || got.GlobalSlots != 999 {
					t.Fatalf("TxPool slots mismatch: %#v", got)
				}
				if got.AccountQueue != 12 || got.GlobalQueue != 777 {
					t.Fatalf("TxPool queue mismatch: %#v", got)
				}
				if got.TxLifetimeSec != 3600 {
					t.Fatalf("TxPool TxLifetime = %d", got.TxLifetimeSec)
				}
			},
		},
		{
			name: "Genesis flags",
			args: []string{"--genesis", "/tmp/genesis.toml"},
			want: func(t *testing.T, cfg launcher.Config) {
				// Path to genesis file should copy directly.
				if cfg.Genesis.Path != "/tmp/genesis.toml" {
					t.Fatalf("Genesis path = %q", cfg.Genesis.Path)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := runConfigFromArgs(t, test.args) // build config using the test helper
			test.want(t, cfg)                      // apply the scenario-specific assertions
			t.Logf("args = %#v", test.args)        //	NOTE: this will only be printed if the test fails
		})

	}

}
