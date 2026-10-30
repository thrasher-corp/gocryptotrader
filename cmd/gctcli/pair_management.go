package main

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var exchangePairManagerCommand = &cli.Command{
	Name:      pairFlag,
	Usage:     "execute exchange pair management command",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:  "get",
			Usage: "returns all enabled and available pairs by asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Usage:    assetFlagUsage,
				},
			},
			Action: getExchangePairs,
		},
		{
			Name:  "disableasset",
			Usage: "disables asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Usage:    assetFlagUsage,
				},
			},
			Action: enableDisableExchangeAsset,
		},
		{
			Name:  "enableasset",
			Usage: "enables asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Usage:    assetFlagUsage,
				},
				&cli.BoolFlag{
					Name:   enableFlag,
					Hidden: true,
					Value:  true,
				},
			},
			Action: enableDisableExchangeAsset,
		},
		{
			Name:  "disable",
			Usage: "disable pairs by asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:     "pairs",
					Required: true,
					Usage:    "either a single currency pair string or comma delimiter string of pairs e.g. \"BTC-USD,XRP-USD\"",
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Usage:    assetFlagUsage,
				},
			},
			Action: enableDisableExchangePair,
		},
		{
			Name:  enableFlag,
			Usage: "enable pairs by asset type",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:     "pairs",
					Required: true,
					Usage:    "either a single currency pair string or comma delimiter string of pairs e.g. \"BTC-USD,XRP-USD\"",
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Usage:    assetFlagUsage,
				},
				&cli.BoolFlag{
					Name:   enableFlag,
					Hidden: true,
					Value:  true,
				},
			},
			Action: enableDisableExchangePair,
		},
		{
			Name:  "enableall",
			Usage: "enable all pairs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.BoolFlag{
					Name:   enableFlag,
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
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: enableDisableAllExchangePairs,
		},
		{
			Name:  "update",
			Usage: "fetches supported pairs from the exchange and updates available pairs and removes unsupported enable pairs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: updateExchangeSupportedPairs,
		},
		{
			Name:  "getassets",
			Usage: "fetches supported assets",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: getExchangeAssets,
		},
	},
}

func enableDisableExchangePair(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	pairList := strings.Split(c.String("pairs"), ",")

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
			Exchange:  c.String(exchangeFlag),
			Pairs:     validPairs,
			AssetType: asset,
			Enable:    c.Bool(enableFlag),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getExchangePairs(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	asset := strings.ToLower(c.String(assetFlag))
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
			Exchange: c.String(exchangeFlag),
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
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
			Exchange: c.String(exchangeFlag),
			Asset:    asset,
			Enable:   c.Bool(enableFlag),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func enableDisableAllExchangePairs(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetAllExchangePairs(c.Context,
		&gctrpc.SetExchangeAllPairsRequest{
			Exchange: c.String(exchangeFlag),
			Enable:   c.Bool(enableFlag),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func updateExchangeSupportedPairs(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.UpdateExchangeSupportedPairs(c.Context,
		&gctrpc.UpdateExchangeSupportedPairsRequest{
			Exchange: c.String(exchangeFlag),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getExchangeAssets(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeAssets(c.Context,
		&gctrpc.GetExchangeAssetsRequest{
			Exchange: c.String(exchangeFlag),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
