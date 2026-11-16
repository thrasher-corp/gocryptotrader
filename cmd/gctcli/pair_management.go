package main

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var exchangePairManagerCommand = &cli.Command{
	Name:      pairFlagName,
	Usage:     "execute exchange pair management command",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:  "get",
			Usage: "returns all enabled and available pairs by asset type",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredAssetFlag,
			},
			Action: getExchangePairs,
		},
		{
			Name:  "disableasset",
			Usage: "disables asset type",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredAssetFlag,
			},
			Action: enableDisableExchangeAsset,
		},
		{
			Name:  "enableasset",
			Usage: "enables asset type",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredAssetFlag,
				&cli.BoolFlag{
					Name:   enableFlagName,
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
				requiredExchangeFlag,
				requiredPairsFlag,
				requiredAssetFlag,
			},
			Action: enableDisableExchangePair,
		},
		{
			Name:  enableFlagName,
			Usage: "enable pairs by asset type",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredPairsFlag,
				requiredAssetFlag,
				&cli.BoolFlag{
					Name:   enableFlagName,
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
				requiredExchangeFlag,
				&cli.BoolFlag{
					Name:   enableFlagName,
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
				requiredExchangeFlag,
			},
			Action: enableDisableAllExchangePairs,
		},
		{
			Name:  "update",
			Usage: "fetches supported pairs from the exchange and updates available pairs and removes unsupported enable pairs",
			Flags: []cli.Flag{
				requiredExchangeFlag,
			},
			Action: updateExchangeSupportedPairs,
		},
		{
			Name:  "getassets",
			Usage: "fetches supported assets",
			Flags: []cli.Flag{
				requiredExchangeFlag,
			},
			Action: getExchangeAssets,
		},
	},
}

func enableDisableExchangePair(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	asset := strings.ToLower(c.String(assetFlagName))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	pairList := strings.Split(c.String(pairsFlagName), ",")

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
			Exchange:  c.String(exchangeFlagName),
			Pairs:     validPairs,
			AssetType: asset,
			Enable:    c.Bool(enableFlagName),
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

	asset := strings.ToLower(c.String(assetFlagName))
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
			Exchange: c.String(exchangeFlagName),
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

	asset := strings.ToLower(c.String(assetFlagName))
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
			Exchange: c.String(exchangeFlagName),
			Asset:    asset,
			Enable:   c.Bool(enableFlagName),
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
			Exchange: c.String(exchangeFlagName),
			Enable:   c.Bool(enableFlagName),
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
			Exchange: c.String(exchangeFlagName),
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
			Exchange: c.String(exchangeFlagName),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
