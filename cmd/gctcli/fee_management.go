package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var exchangeFeeManagerCommand = &cli.Command{
	Name:      "fee",
	Usage:     "execute fee management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "getall",
			Usage:     "returns all fees associated with an exchange",
			ArgsUsage: "<exchange>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "exchange",
					Usage:    "the exchange to act on",
					Required: true,
				},
			},
			Action: getAllFees,
		},
		{
			Name:  "set",
			Usage: "sets new fee structure to running instance",
			Subcommands: []*cli.Command{
				{
					Name:      "global",
					Usage:     "sets new maker and taker values for an exchange",
					ArgsUsage: "<exchange> <maker> <taker>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "exchange",
							Usage:    "the exchange to act on",
							Required: true,
						},
						&cli.Float64Flag{
							Name:     "maker",
							Usage:    "the maker fee",
							Required: true,
						},
						&cli.Float64Flag{
							Name:     "taker",
							Usage:    "the taker fee",
							Required: true,
						},
					},
					Action: setGlobalFees,
				},
				{
					Name:      "transfer",
					Usage:     "sets new withdrawal and deposit values for an exchange",
					ArgsUsage: "<exchange> <currency> <asset> <withdraw> <deposit>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "exchange",
							Usage:    "the exchange to act on",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "currency",
							Usage:    "the currency for transfer",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "asset",
							Usage:    "the asset type",
							Required: true,
						},
						&cli.Float64Flag{
							Name:  "withdraw",
							Usage: "the withdraw fee",
						},
						&cli.Float64Flag{
							Name:  "deposit",
							Usage: "the deposit fee",
						},
					},
					Action: setTransferFees,
				},
			},
		},
	},
}

func getAllFees(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	fmt.Println(exchange)

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	// client := gctrpc.NewGoCryptoTraderClient(conn)
	// result, err := client.WebsocketGetInfo(context.Background(),
	// 	&gctrpc.WebsocketGetInfoRequest{Exchange: exchange})
	// if err != nil {
	// 	return err
	// }
	// jsonOutput(result)

	return nil
}

func setGlobalFees(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	fmt.Println(exchange)

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	// client := gctrpc.NewGoCryptoTraderClient(conn)
	// result, err := client.WebsocketGetInfo(context.Background(),
	// 	&gctrpc.WebsocketGetInfoRequest{Exchange: exchange})
	// if err != nil {
	// 	return err
	// }
	// jsonOutput(result)

	return nil
}

func setTransferFees(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	fmt.Println(exchange)

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	// client := gctrpc.NewGoCryptoTraderClient(conn)
	// result, err := client.WebsocketGetInfo(context.Background(),
	// 	&gctrpc.WebsocketGetInfoRequest{Exchange: exchange})
	// if err != nil {
	// 	return err
	// }
	// jsonOutput(result)

	return nil
}
