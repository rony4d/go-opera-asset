module github.com/rony4d/go-opera-asset

go 1.14

require (
	github.com/Fantom-foundation/lachesis-base v0.0.0-20221101131534-22299068014e
	github.com/ethereum/go-ethereum v1.10.8
	github.com/naoina/toml v0.1.2-0.20170918210437-9fafd6967416
	gopkg.in/urfave/cli.v1 v1.20.0 // gopkg.in/urfave/cli.v1 is a popular Go library for building rich command-line interfaces—think commands, subcommands, flags, usage text, help output, etc

)

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum v1.10.8-ftm-rc9
