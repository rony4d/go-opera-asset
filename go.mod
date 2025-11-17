module github.com/rony4d/go-opera-asset

go 1.14

require (
	github.com/ethereum/go-ethereum v1.10.8
	golang.org/x/sys v0.0.0-20210909193231-528a39cd75f3 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0 // gopkg.in/urfave/cli.v1 is a popular Go library for building rich command-line interfaces—think commands, subcommands, flags, usage text, help output, etc

)

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum v1.10.8-ftm-rc9
