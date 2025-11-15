package flags

import (
	"gopkg.in/urfave/cli.v1"
)

// NetworkFlags covers P2P and networking configuration.

func NetworkFlags() []cli.Flag {
	return []cli.Flag{
		cli.IntFlag{
			Name:  "port",
			Usage: "P2P networking port",
			Value: 5050,
		},
		cli.IntFlag{
			Name:  "maxpeers",
			Usage: "Maximum number of peer connections",
			Value: 50,
		},
		cli.StringFlag{
			Name:  "nat",
			Usage: "NAT mechanism (any|none|extip:<ip>|upnp|pmp|pmp:<addr>)",
		},
		cli.StringFlag{
			Name:  "bootnodes",
			Usage: "Comma-separated enode URLs for bootstrap peers",
		},
		cli.StringSliceFlag{
			Name:  "staticnodes",
			Usage: "List of enode URLs to maintain persistent connections with",
		},
		cli.StringSliceFlag{
			Name:  "trustednodes",
			Usage: "Whitelist of peers that bypass slot limits",
		},
		cli.BoolFlag{
			Name:  "nodiscover",
			Usage: "Disable the peer discovery mechanism (manual peers only)",
		},
		cli.BoolFlag{
			Name:  "discv5",
			Usage: "Enable discovery v5 (experimental)",
		},
		cli.StringFlag{
			Name:  "netrestrict",
			Usage: "Comma-separated CIDR block list to restrict communication to",
		},
		cli.StringFlag{
			Name:  "ipcdisable",
			Usage: "Disable the default IPC listener (mirrors --ipc=false)",
		},
	}
}

// TxPoolFlags isolates transaction-pool tuning knobs.
func TxPoolFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "txpool.journal",
			Usage: "Location of the transaction journal file",
			Value: "transactions.rlp",
		},
		cli.IntFlag{
			Name:  "txpool.localslots",
			Usage: "Number of executable transaction slots per account",
			Value: 16,
		},
		cli.IntFlag{
			Name:  "txpool.globalslots",
			Usage: "Maximum number of executable transactions total",
			Value: 4096,
		},
		cli.IntFlag{
			Name:  "txpool.localqueue",
			Usage: "Number of non-executable transaction slots per account",
			Value: 64,
		},
		cli.IntFlag{
			Name:  "txpool.globalqueue",
			Usage: "Maximum number of non-executable transactions total",
			Value: 1024,
		},
		cli.Uint64Flag{
			Name:  "txpool.pricelimit",
			Usage: "Minimum gas price (in wei) to accept a transaction",
			Value: 1,
		},
		cli.Uint64Flag{
			Name:  "txpool.pricebump",
			Usage: "Price bump percentage to replace an existing transaction",
			Value: 10,
		},
		cli.Uint64Flag{
			Name:  "txpool.lifetime",
			Usage: "Maximum transaction lifetime in the pool (seconds)",
			Value: 10800,
		},
	}
}
