package main

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli"
)

var websocketManagerCommand = cli.Command{
	Name:      "websocket",
	Usage:     "execute websocket management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []cli.Command{
		{
			Name:   "info",
			Usage:  "returns all exchange websocket information",
			Action: getwebsocketInfo,
		},
		{
			Name:  "disable",
			Usage: "disables websocket connection for an exchange",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			// Action: disableWebsocket,
		},
		{
			Name:  "enable",
			Usage: "enables websocket connection for an exchange",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			// Action: enableWebsocket,
		},
		{
			Name:  "getSubs",
			Usage: "returns current subscriptions for an exchange",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			// Action: getSubscriptions,
		},
		{
			Name:  "setproxy",
			Usage: "sets exchange websocket proxy, flushes and reroutes connection",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
				cli.StringFlag{
					Name:  "proxy",
					Usage: "proxy address to change to",
				},
			},
			// Action: setProxy,
		},
		{
			Name:  "seturl",
			Usage: "sets exchange websocket connection, flushes and reconnects",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
				cli.StringFlag{
					Name:  "url",
					Usage: "url string to change to",
				},
			},
			// Action: setURL,
		},
	},
}

func getwebsocketInfo(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetExchangePairs(context.Background(),
		&gctrpc.GetExchangePairsRequest{
			Exchange: exchange,
			Asset:    asset,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
