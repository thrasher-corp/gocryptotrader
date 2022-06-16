package main

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var strategyManagementCommand = &cli.Command{
	Name:      "strategy",
	Usage:     "execute strategy management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:        "twap",
			Usage:       "initiates a twap strategy to accumulate or decumulate your position",
			ArgsUsage:   "<exchange> <pair> <asset>",
			Subcommands: []*cli.Command{twapStream},
		},
	},
}

var (
	twapStream = &cli.Command{
		Name:   "stream",
		Usage:  "executes strategy while reporting all actions to the client, exiting will stop strategy",
		Action: twapStreamfunc,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "exchange",
				Usage: "the exchange to act on",
			},
			&cli.StringFlag{
				Name:  "pair",
				Usage: "curreny pair",
			},
			&cli.StringFlag{
				Name:  "asset",
				Usage: "asset",
			},
			&cli.BoolFlag{
				Name:  "buy",
				Usage: "whether you are buying base or selling base",
			},
		},
	}
)

func twapStreamfunc(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "twap")
	}

	var exchangeName string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var pair string
	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(1)
	}

	cp, err := currency.NewPairDelimiter(pair, pairDelimiter)
	if err != nil {
		return err
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.TWAPStream(c.Context, &gctrpc.TWAPRequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Asset: assetType,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}

		jsonOutput(resp)

		// if resp.Finished {
		// 	fmt.Println("TWAP HAS COMPLETED")
		// 	return nil
		// }
	}
}
