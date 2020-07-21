package main

import (
	"log"
	"os"

	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/urfave/cli/v2"
)

var seedCandleCommand = &cli.Command{
	Name:  "candle",
	Usage: "seed candle data",
	Subcommands: []*cli.Command{
		{
			Name:      "file",
			Usage:     "seed candle data from a file",
			ArgsUsage: "<flags>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "<exchange>",
				},
				&cli.StringFlag{
					Name:  "base",
					Usage: "<base>",
				},
				&cli.StringFlag{
					Name:  "quote",
					Usage: "<quote>",
				},
				&cli.StringFlag{
					Name:  "interval",
					Usage: "<interval>",
				},
				&cli.StringFlag{
					Name:  "asset",
					Usage: "<asset>",
				},
				&cli.StringFlag{
					Name:  "filename",
					Usage: "<filename>",
				},
			},
			Action: seedCandleFromFile,
		},
	},
}

func seedCandleFromFile(c *cli.Context) error {
	var exchangeName string
	if c.IsSet("name") {
		exchangeName = c.String("name")
	} else if c.Args().Get(0) != "" {
		exchangeName = c.Args().Get(0)
	}

	var base string
	if c.IsSet("base") {
		base = c.String("base")
	} else if c.Args().Get(1) != "" {
		base = c.Args().Get(1)
	}

	var quote string
	if c.IsSet("quote") {
		quote = c.String("quote")
	} else if c.Args().Get(2) != "" {
		quote = c.Args().Get(2)
	}

	var interval string
	if c.IsSet("interval") {
		interval = c.String("interval")
	} else if c.Args().Get(3) != "" {
		interval = c.Args().Get(3)
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("name")
	} else if c.Args().Get(4) != "" {
		asset = c.Args().Get(4)
	}

	var fileName string
	if c.IsSet("name") {
		fileName = c.String("name")
	} else if c.Args().Get(5) != "" {
		fileName = c.Args().Get(5)
	}

	_, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	err = Load(c)
	if err != nil {
		return err
	}

	totalInserted, err := candle.InsertFromCSV(exchangeName,
		base, quote, interval, asset,
		fileName)
	if err != nil {
		return err
	}

	log.Printf("Inserted: %v records", totalInserted)
	return nil
}
