package launcher

import (
	"errors"
	"fmt"

	"github.com/rony4d/go-opera-asset/flags"
)

var app = flags.NewApp()

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
