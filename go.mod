module github.com/rony4d/go-opera-asset

go 1.14

require (
	github.com/Fantom-foundation/lachesis-base v0.0.0-20221101131534-22299068014e
	github.com/certifi/gocertifi v0.0.0-20210507211836-431795d63e8d // indirect
	github.com/ethereum/go-ethereum v1.10.8
	github.com/evalphobia/logrus_sentry v0.8.2
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.7.2
	gopkg.in/urfave/cli.v1 v1.20.0 // gopkg.in/urfave/cli.v1 is a popular Go library for building rich command-line interfaces—think commands, subcommands, flags, usage text, help output, etc

)

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum v1.10.8-ftm-rc9
