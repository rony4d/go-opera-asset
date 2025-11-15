package flags

import (
	"time"

	"gopkg.in/urfave/cli.v1"
)

// CommonFlags returns the base set of CLI flags shared across commands.

func CommonFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "datadir",
			Usage: "Data directory for the Opera Asset Chain Node",
			Value: "~/.opera",
		},
		cli.StringFlag{
			Name:  "log.format",
			Usage: "Log output format (text|json)",
			Value: "text",
		},
		cli.IntFlag{
			Name:  "log.verbosity",
			Usage: "Logging verbosity (0=fatal,1=error,2=warn,3=info,4=debug,5=trace)",
			Value: 3,
		},
		cli.BoolFlag{
			Name:  "log.color",
			Usage: "Enable colored log output",
		},
		cli.BoolFlag{
			Name:  "http",
			Usage: "Enable HTTP JSON-RPC server",
		},
		cli.StringFlag{
			Name:  "http.addr",
			Usage: "HTTP-RPC server listening interface",
			Value: "127.0.0.1",
		},
		cli.IntFlag{
			Name:  "http.port",
			Usage: "HTTP-RPC server listening port",
			Value: 18545,
		},
		cli.StringFlag{
			Name:  "http.api",
			Usage: "Comma-separated list of HTTP-RPC APIs to enable",
			Value: "eth,net,web3",
		},
		cli.BoolFlag{
			Name:  "ws",
			Usage: "Enable WebSocket JSON-RPC server",
		},
		cli.StringFlag{
			Name:  "ws.addr",
			Usage: "WebSocket-RPC listening interface",
			Value: "127.0.0.1",
		},
		cli.IntFlag{
			Name:  "ws.port",
			Usage: "WebSocket-RPC listening port",
			Value: 18546,
		},
		cli.StringFlag{
			Name:  "ws.api",
			Usage: "Comma-separated list of WebSocket APIs to enable",
			Value: "eth,net,web3",
		},
		cli.BoolFlag{
			Name:  "ipc",
			Usage: "Enable IPC (Unix socket) JSON-RPC server",
		},
		cli.StringFlag{
			Name:  "ipc.path",
			Usage: "Filename for IPC socket/pipe",
			Value: "opera.ipc",
		},
		cli.BoolFlag{
			Name:  "metrics",
			Usage: "Enable collection of Prometheus-compatible metrics",
		},
		cli.StringFlag{
			Name:  "metrics.addr",
			Usage: "Metrics server listening interface",
			Value: "127.0.0.1",
		},
		cli.IntFlag{
			Name:  "metrics.port",
			Usage: "Metrics server listening port",
			Value: 6060,
		},
		cli.DurationFlag{
			Name:  "rpc.timeout",
			Usage: "Global JSON-RPC request timeout",
			Value: 30 * time.Second,
		},
	}
}
