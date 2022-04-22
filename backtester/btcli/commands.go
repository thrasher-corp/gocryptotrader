package main

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/urfave/cli/v2"
)

var executeStrategyFromFileCommand = &cli.Command{
	Name:      "executestrategyfromfile",
	Usage:     "runs the strategy from a config file",
	ArgsUsage: "<path>",
	Action:    executeStrategyFromFile,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "path",
			Aliases: []string{"p"},
			Usage:   "the filepath to a strategy to execute",
		},
	},
}

func executeStrategyFromFile(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "executestrategyfromfile")
	}

	var path string
	if c.IsSet("path") {
		path = c.String("path")
	} else {
		path = c.Args().First()
	}

	client := btrpc.NewBacktesterClient(conn)
	result, err := client.ExecuteStrategyFromFile(
		c.Context,
		&btrpc.ExecuteStrategyFromFileRequest{
			StrategyFilePath: path,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
