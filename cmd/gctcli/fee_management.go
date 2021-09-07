package main

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
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
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			Action: getAllFees,
		},
		{
			Name:  "set",
			Usage: "sets new fee structure to running instance, this enforces a custom state which enhibits fee manager updates",
			Subcommands: []*cli.Command{
				{
					Name:      "global",
					Usage:     "sets new maker and taker values for an exchange",
					ArgsUsage: "<exchange> <maker> <taker>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "exchange",
							Usage: "the exchange to act on",
						},
						&cli.Float64Flag{
							Name:  "maker",
							Usage: "the maker fee",
						},
						&cli.Float64Flag{
							Name:  "taker",
							Usage: "the taker fee",
						},
						&cli.BoolFlag{
							Name:   "ratio",
							Usage:  "if the fees are a set value or ratio",
							Value:  true, // Default to true
							Hidden: true,
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
							Name:  "exchange",
							Usage: "the exchange to act on",
						},
						&cli.StringFlag{
							Name:  "currency",
							Usage: "the currency for transfer",
						},
						&cli.StringFlag{
							Name:  "asset",
							Usage: "the asset type",
						},
						&cli.Float64Flag{
							Name:  "withdraw",
							Usage: "the withdraw fee",
						},
						&cli.Float64Flag{
							Name:  "deposit",
							Usage: "the deposit fee",
						},
						&cli.BoolFlag{
							Name:   "ratio",
							Usage:  "if the fees are a set value or ratio",
							Value:  false, // Default to a set value
							Hidden: true,
						},
					},
					Action: setTransferFees,
				},
				{
					Name:      "custom",
					Usage:     "if enabled this stops the periodic update from the fee manager for a given exchange",
					ArgsUsage: "<exchange>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "exchange",
							Usage: "the exchange to act on",
						},
						&cli.BoolFlag{
							Name:  "enabled",
							Usage: "if enabled or disabled",
						},
					},
					Action: yieldToFeeManager,
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

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetAllFees(context.Background(),
		&gctrpc.GetAllFeesRequest{Exchange: exchange})
	if err != nil {
		return err
	}
	jsonOutput(result)

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

func yieldToFeeManager(c *cli.Context) error {
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
