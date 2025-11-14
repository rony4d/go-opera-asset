package launcher

import (
	"errors"
	"fmt"

	"github.com/rony4d/go-opera-asset/flags"
)

var app = flags.NewApp()

// Launch is a stub; it will eventually parse flags and start the node.
func Launch(args []string) error {

	if err := app.Run(args); err != nil {
		fmt.Println("App Run Error:", err)
		return err
	}
	return errors.New("opera launcher not implemented yet")
}
