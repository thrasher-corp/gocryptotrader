package main

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var exchangePairManagerCommand = &cli.Command{
	Name:      "pair",
	Usage:     "execute exchange pair management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "get",
			Usage:     "returns all enabled and available pairs by asset type",
			ArgsUsage: "<asset>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
				&cli.StringFlag{
					Name:  "asset",
					Usage: "asset",
				},
			},
			Action: getExchangePairs,
		},
		{
			Name:  "disableasset",
			Usage: "disables asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
				&cli.StringFlag{
					Name:  "asset",
					Usage: "asset",
				},
			},
			Action: enableDisableExchangeAsset,
		},
		{
			Name:   "enableasset",
			Usage:  "enables asset type",
			Flags:  FlagsFromStruct(&EnableDisableExchangeAsset{Enable: true}),
			Action: enableDisableExchangeAsset,
		},
		{
			Name:   "disable",
			Usage:  "disable pairs by asset type",
			Flags:  FlagsFromStruct(&EnableDisableExchangePair{}),
			Action: enableDisableExchangePair,
		},
		{
			Name:  "enable",
			Usage: "enable pairs by asset type",
			Flags: FlagsFromStruct(&EnableDisableExchangePair{
				Enable: true,
			}),
			Action: enableDisableExchangePair,
		},
		{
			Name:  "enableall",
			Usage: "enable all pairs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
				&cli.BoolFlag{
					Name:   "enable",
					Hidden: true,
					Value:  true,
				},
			},
			Action: enableDisableAllExchangePairs,
		},
		{
			Name:  "disableall",
			Usage: "disable all pairs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			Action: enableDisableAllExchangePairs,
		},
		{
			Name:  "update",
			Usage: "fetches supported pairs from the exchange and updates available pairs and removes unsupported enable pairs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			Action: updateExchangeSupportedPairs,
		},
		{
			Name:  "getassets",
			Usage: "fetches supported assets",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			Action: getExchangeAssets,
		},
	},
}

func enableDisableExchangePair(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &EnableDisableExchangePair{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	pairList := strings.Split(arg.Pairs, ",")

	validPairs := make([]*gctrpc.CurrencyPair, len(pairList))
	for i := range pairList {
		if !validPair(pairList[i]) {
			return errInvalidPair
		}

		p, err := currency.NewPairFromString(pairList[i])
		if err != nil {
			return err
		}

		validPairs[i] = &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.SetExchangePair(c.Context,
		&gctrpc.SetExchangePairRequest{
			Exchange:  arg.Exchange,
			Pairs:     validPairs,
			AssetType: arg.Asset,
			Enable:    arg.Enable,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getExchangePairs(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	var asset string

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(1)
	}

	asset = strings.ToLower(asset)
	if !validAsset(asset) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangePairs(c.Context,
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

func enableDisableExchangeAsset(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &EnableDisableExchangeAsset{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}
	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetExchangeAsset(c.Context,
		&gctrpc.SetExchangeAssetRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Enable:   arg.Enable,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func enableDisableAllExchangePairs(c *cli.Context) error {
	enable := c.Bool("enable")
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetAllExchangePairs(c.Context,
		&gctrpc.SetExchangeAllPairsRequest{
			Exchange: exchange,
			Enable:   enable,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func updateExchangeSupportedPairs(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.UpdateExchangeSupportedPairs(c.Context,
		&gctrpc.UpdateExchangeSupportedPairsRequest{
			Exchange: exchange,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getExchangeAssets(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeAssets(c.Context,
		&gctrpc.GetExchangeAssetsRequest{
			Exchange: exchange,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
