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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "Exchange name of supplied candle data",
				},
				&cli.StringFlag{
					Name:  "base",
					Usage: "Base currency of supplied candle data",
				},
				&cli.StringFlag{
					Name:  "quote",
					Usage: "Quote currency of supplied candle data",
				},
				&cli.StringFlag{
					Name:  "interval",
					Usage: "Interval of supplied candle data",
				},
				&cli.StringFlag{
					Name:  "asset",
					Usage: "Asset type of supplied data (spot/margin/futures for example)",
				},
				&cli.StringFlag{
					Name:      "filename",
					Usage:     "CSV file to load candle data from",
					TakesFile: true,
					FilePath:  workingDir,
				},
			},
			Action: seedCandleFromFile,
		},
	},
}

func seedCandleFromFile(c *cli.Context) error {
	if c.NumFlags() == 0 && c.NArg() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
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
		asset = c.String("asset")
	} else if c.Args().Get(4) != "" {
		asset = c.Args().Get(4)
	}

	var fileName string
	if c.IsSet("filename") {
		fileName = c.String("filename")
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
