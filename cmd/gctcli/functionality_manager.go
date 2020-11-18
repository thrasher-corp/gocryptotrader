package main

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli"
)

var functionalityManagerCommand = cli.Command{
	Name:      "functionality",
	Usage:     "execute protocol functionality management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []cli.Command{
		// Add global setting of functionality
		{
			Name:  "get",
			Usage: "get protocol functionality",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "protocol",
					Usage:    "either websocket/rest",
					Required: true,
				},
				cli.StringFlag{
					Name:     "exchange",
					Usage:    "name of exchange",
					Required: true,
				},
			},
			Action: getProtocolFunctionality,
		},
		{
			Name:  "set",
			Usage: "sets protocol functionality",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "protocol",
					Usage:    "either websocket/rest",
					Required: true,
				},
				cli.StringFlag{
					Name:     "exchange",
					Usage:    "name of exchange",
					Required: true,
				},
				cli.BoolFlag{
					Name:  "tickerfetch",
					Usage: "sets ticker fetching on protocol enabled or disabled",
				},
				cli.BoolFlag{
					Name:  "orderbookfetch",
					Usage: "sets orderbook fetching on protocol enabled or disabled",
				},
				cli.BoolFlag{
					Name:  "klinefetch",
					Usage: "sets kline fetching on protocol enabled or disabled",
				},
				cli.BoolFlag{
					Name:  "tradefetch",
					Usage: "sets trade fetching on protocol enabled or disabled",
				},
			},
			Action: setProtocolFunctionality,
		},
	},
}

func getProtocolFunctionality(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var protocolType string
	if c.IsSet("protocol") {
		protocolType = c.String("protocol")
	} else {
		protocolType = c.Args().First()
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().Get(1)
	}

	if !validExchange(exchange) {
		return fmt.Errorf("[%s] is not a valid exchange", exchange)
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetFunctionality(context.Background(), &gctrpc.FunctionalityGetRequest{
		Exchange: exchange,
		Protocol: protocolType,
	})

	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setProtocolFunctionality(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var protocolType string
	if c.IsSet("protocol") {
		protocolType = c.String("protocol")
	} else {
		protocolType = c.Args().First()
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().Get(1)
	}

	if !validExchange(exchange) {
		return fmt.Errorf("[%s] is not a valid exchange", exchange)
	}

	var tickerFetching string
	if c.IsSet("tickerfetch") {
		if c.Bool("tickerfetch") {
			tickerFetching = "true"
		} else {
			tickerFetching = "false"
		}
	}

	var orderbookFetching string
	if c.IsSet("orderbookfetch") {
		if c.Bool("orderbookfetch") {
			orderbookFetching = "true"
		} else {
			orderbookFetching = "false"
		}
	}

	var klineFetching string
	if c.IsSet("klinefetch") {
		if c.Bool("klinefetch") {
			klineFetching = "true"
		} else {
			klineFetching = "false"
		}
	}

	var tradeFetching string
	if c.IsSet("tradefetch") {
		if c.Bool("tradefetch") {
			tradeFetching = "true"
		} else {
			tradeFetching = "false"
		}
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.SetFunctionality(context.Background(),
		&gctrpc.FunctionalitySetRequest{
			Exchange:          exchange,
			Protocol:          protocolType,
			TickerFetching:    tickerFetching,
			OrderbookFetching: orderbookFetching,
			KlingFetching:     klineFetching,
			TradeFetching:     tradeFetching,
		})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
