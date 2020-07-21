package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/urfave/cli/v2"
)

var seedExchangeCommand = &cli.Command{
	Name:  "exchange",
	Usage: "seed exchange data",
	Subcommands: []*cli.Command{
		{
			Name:      "file",
			Usage:     "seed exchange data from a file",
			ArgsUsage: "<flags>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "filename",
					Usage: "<filename>",
				},
			},
			Action: seedExchangeFromFile,
		},
		{
			Name:      "add",
			Usage:     "add a single exchange",
			ArgsUsage: "<flags>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "name",
				},
			},
			Action: addSingleExchange,
		},
	},
}

func ExchangesFromDefaultList() error {
	var allExchanges []exchangeDB.Details
	for x := range exchange.Exchanges {
		allExchanges = append(allExchanges, exchangeDB.Details{
			Name: strings.Title(exchange.Exchanges[x]),
		})
	}
	return exchangeDB.InsertMany(allExchanges)
}

func Exchanges(in []exchangeDB.Details) error {
	return exchangeDB.InsertMany(in)
}

func seedExchangeFromFile(c *cli.Context) error {
	var fileName string
	if c.IsSet("name") {
		fileName = c.String("name")
	} else if c.Args().Get(0) != "" {
		fileName = c.Args().Get(0)
	}

	_, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	err = Load(c.String("config"))
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
	var exchangeName string
	if c.IsSet("name") {
		exchangeName = c.String("name")
	} else if c.Args().Get(0) != "" {
		exchangeName = c.Args().Get(0)
	}

	err := Load(c.String("config"))
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
