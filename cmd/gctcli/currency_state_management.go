package main

import (
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var currencyStateManagementCommand = &cli.Command{
	Name:      "currencystate",
	Usage:     "execute exchange currency state management command",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:  "getall",
			Usage: "fetch all currency states associated with an exchange",
			Flags: []cli.Flag{
				requiredExchangeFlag,
			},
			Action: stateGetAll,
		},
		{
			Name:   "withdraw",
			Usage:  "returns if the currency can be withdrawn from the exchange",
			Flags:  stateFlags,
			Action: stateGetWithdrawal,
		},
		{
			Name:   "deposit",
			Usage:  "returns if the currency can be deposited onto an exchange",
			Flags:  stateFlags,
			Action: stateGetDeposit,
		},
		{
			Name:   "trade",
			Usage:  "returns if the currency can be traded on the exchange",
			Flags:  stateFlags,
			Action: stateGetTrading,
		},
		{
			Name:  "tradepair",
			Usage: "returns if the currency pair can be traded on the exchange",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredPairFlag,
				requiredAssetFlag,
			},
			Action: stateGetPairTrading,
		},
	},
}

var stateFlags = []cli.Flag{
	requiredExchangeFlag,
	&cli.StringFlag{
		Name:     "code",
		Required: true,
		Usage:    "the currency code",
	},
	requiredAssetFlag,
}

func stateGetAll(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CurrencyStateGetAll(c.Context,
		&gctrpc.CurrencyStateGetAllRequest{Exchange: c.String(exchangeFlagName)},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)

	return nil
}

func stateGetDeposit(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CurrencyStateDeposit(c.Context,
		&gctrpc.CurrencyStateDepositRequest{
			Exchange: c.String(exchangeFlagName),
			Code:     c.String("code"),
			Asset:    c.String(assetFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func stateGetWithdrawal(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CurrencyStateWithdraw(c.Context,
		&gctrpc.CurrencyStateWithdrawRequest{
			Exchange: c.String(exchangeFlagName),
			Code:     c.String("code"),
			Asset:    c.String(assetFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func stateGetTrading(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CurrencyStateTrading(c.Context,
		&gctrpc.CurrencyStateTradingRequest{
			Exchange: c.String(exchangeFlagName),
			Code:     c.String("code"),
			Asset:    c.String(assetFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func stateGetPairTrading(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CurrencyStateTradingPair(c.Context,
		&gctrpc.CurrencyStateTradingPairRequest{
			Exchange: c.String(exchangeFlagName),
			Pair:     c.String(pairFlagName),
			Asset:    c.String(assetFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
