package flags

import (
	"gopkg.in/urfave/cli.v1"
)

// NodeFlags holds knobs specific to the local node instance (datadir, sync mode, identity, etc.).

func NodeFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "identity",
			Usage: "Custom node name to advertise over the network",
		},
		cli.StringFlag{
			Name:  "syncmode",
			Usage: "Blockchain sync mode (full|snap|light)",
			Value: "full",
		},
		cli.IntFlag{
			Name:  "cache",
			Usage: "Megabytes of memory allocated to internal caching",
			Value: 1024,
		},
		cli.BoolFlag{
			Name:  "nousb",
			Usage: "Disable monitoring for new USB hardware wallets",
		},
		cli.BoolFlag{
			Name:  "lightkdf",
			Usage: "Reduce key-derivation hardness (faster account unlock, insecure for prod)",
		},
		cli.StringFlag{
			Name:  "keystore",
			Usage: "Directory for storing encrypted account keys",
		},
		cli.StringFlag{
			Name:  "datadir.chaindata",
			Usage: "Override path to the chaindata DB (defaults to <datadir>/chaindata)",
		},
		cli.StringFlag{
			Name:  "datadir.errlock",
			Usage: "Override path to the errlock file (defaults to <datadir>)",
		},
	}
}
