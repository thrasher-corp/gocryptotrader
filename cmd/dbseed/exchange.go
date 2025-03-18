package main

import (
	"fmt"
	"log"
	"os"

	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/urfave/cli/v2"
)

var seedExchangeCommand = &cli.Command{
	Name:  "exchange",
	Usage: "seed exchange data",
	Subcommands: []*cli.Command{
		{
			Name:  "file",
			Usage: "seed exchange data from a file",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:      "filename",
					Usage:     "CSV file to load exchanges from",
					TakesFile: true,
					FilePath:  workingDir,
				},
			},
			Action: seedExchangeFromFile,
		},
		{
			Name:  "add",
			Usage: "add a single exchange",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "name",
				},
			},
			Action: addSingleExchange,
		},
		{
			Name:   "default",
			Usage:  "seed exchange from default list",
			Action: seedExchangeFromDefaultList,
		},
	},
}

func seedExchangeFromDefaultList(c *cli.Context) error {
	err := load(c)
	if err != nil {
		return err
	}
	allExchanges := make([]exchangeDB.Details, len(exchange.Exchanges))
	for x := range exchange.Exchanges {
		allExchanges[x] = exchangeDB.Details{
			Name: exchange.Exchanges[x],
		}
	}
	err = exchangeDB.InsertMany(allExchanges)
	if err != nil {
		return err
	}
	fmt.Println("command completed successfully")
	return nil
}

func seedExchangeFromFile(c *cli.Context) error {
	if c.NumFlags() == 0 && c.NArg() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var fileName string
	if c.IsSet("filename") {
		fileName = c.String("filename")
	} else if c.Args().Get(0) != "" {
		fileName = c.Args().Get(0)
	}

	_, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	err = load(c)
	if err != nil {
		return err
	}

	exchangeList, err := exchangeDB.LoadCSV(fileName)
	if err != nil {
		return err
	}
	err = exchangeDB.InsertMany(exchangeList)
	if err != nil {
		return err
	}
	for x := range exchangeList {
		fmt.Printf("Added exchange: %v\n", exchangeList[x].Name)
	}

	return nil
}

func addSingleExchange(c *cli.Context) error {
	if c.NumFlags() == 0 && c.NArg() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("name") {
		exchangeName = c.String("name")
	} else if c.Args().Get(0) != "" {
		exchangeName = c.Args().Get(0)
	}

	err := load(c)
	if err != nil {
		return err
	}

	err = exchangeDB.Insert(exchangeDB.Details{
		Name: exchangeName,
	})
	if err != nil {
		return err
	}

	log.Printf("Added new exchange: %v", exchangeName)
	return nil
}
