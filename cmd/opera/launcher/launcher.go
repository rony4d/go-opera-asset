/*
	The launcher is the main entry point for the go-opera-asset command-line interface.
	It wires together CLI flags, config-file decoding, genesis/fakenet handling, node + consensus setup,
	and helper commands (dumpconfig, checkconfig).
*/

package launcher

import (
	"errors"
	"fmt"

	"github.com/rony4d/go-opera-asset/flags"
	"gopkg.in/urfave/cli.v1"
)

const (
	// clientIdentifier to advertise over the network.
	clientIdentifier = "go-opera"
)

var (

	// Git SHA1 commit hash of the release (set via linker flags).
	gitCommit = ""
	gitDate   = ""

	// The app that holds all commands and flags.
	app = flags.NewApp(gitCommit, gitDate, "the go-opera-asset command line interface")

	nodeFlags        []cli.Flag
	testFlags        []cli.Flag
	gpoFlags         []cli.Flag
	accountFlags     []cli.Flag
	performanceFlags []cli.Flag
	networkingFlags  []cli.Flag
	txpoolFlags      []cli.Flag
	operaFlags       []cli.Flag
	legacyRpcFlags   []cli.Flag
	rpcFlags         []cli.Flag
	metricsFlags     []cli.Flag
)

func initFlags() {

}

// Launch is a stub; it will eventually parse flags and start the node.
func Launch(args []string) error {

	app.Flags = append(app.Flags, flags.CommonFlags()...)  //	Add the common flags to the app
	app.Flags = append(app.Flags, flags.NetworkFlags()...) //	Add the network flags to the app
	app.Flags = append(app.Flags, flags.NodeFlags()...)    //	Add the node flags to the app
	app.Flags = append(app.Flags, flags.TxPoolFlags()...)  //	Add the txpool flags to the app

	if err := app.Run(args); err != nil {
		fmt.Println("App Run Error:", err)
		return err
	}
	return errors.New("opera launcher not implemented yet")
}
