package test

import (
	// "gopkg.in/urfave/cli.v1"
	// "runtime"
	// "strings"

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
			args: []string{"--datadir", projectRoot + launcher.DefaultConfig().Node.DataDir, "--identity", "go-opera"},
			want: func(t *testing.T, cfg launcher.Config) {

				// Expect the datadir to be the project root + the default data dir.
				if cfg.Node.DataDir != filepath.Join(projectRoot+launcher.DefaultConfig().Node.DataDir) {
					t.Fatalf("Datadir = %q, want ~/.opera", cfg.Node.DataDir)
				}

				t.Logf("cfg.Node.DataDir = %q", cfg.Node.DataDir) //	NOTE: this will only be printed if the test fails
				// Expect the identity to be the default name (go-opera).
				if cfg.Node.Name != "go-opera" {
					t.Fatalf("Identity = %q, want go-opera", cfg.Node.Name)
				}
				t.Logf("cfg.Node.Name = %q", cfg.Node.Name) //	NOTE: this will only be printed if the test fails

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
