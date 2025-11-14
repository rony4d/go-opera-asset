// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package flags

import (
	"os"

	cli "gopkg.in/urfave/cli.v1"
)

func NewApp() *cli.App {

	app := cli.NewApp()
	app.Name = "opera-asset"
	app.Usage = "Asset Chain Opera Node (stub)"
	app.Action = func(c *cli.Context) error {
		return nil
	}
	app.Version = "0.1.0"
	app.Writer = os.Stdout
	return app

}
