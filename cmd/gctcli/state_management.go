package main

import (
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var stateManagementCommand = &cli.Command{
	Name:      "state",
	Usage:     "execute exchange currency state management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "getall",
			Usage:     "fetch all currency states associated with an exchange",
			ArgsUsage: "<exchange>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the exchange to act on",
				},
			},
			Action: stateGetAll,
		},
		{
			Name:      "withdraw",
			Usage:     "returns if the currency can be withdrawn from the exchange",
			ArgsUsage: "<exchange> <code> <asset> <enabled>",
			Flags:     stateFlags,
			Action:    stateGetWithdrawal,
		},
		{
			Name:      "deposit",
			Usage:     "returns if the currency can be deposited onto an exchange",
			ArgsUsage: "<exchange> <code> <asset> <enabled>",
			Flags:     stateFlags,
			Action:    stateGetDeposit,
		},
		{
			Name:      "trade",
			Usage:     "returns if the currency can be deposited onto an exchange",
			ArgsUsage: "<exchange> <code> <asset> <enabled>",
			Flags:     stateFlags,
			Action:    stateGetTrading,
		},
	},
}

var stateFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "exchange",
		Usage: "the exchange to act on",
	},
	&cli.StringFlag{
		Name:  "code",
		Usage: "the currency code",
	},
	&cli.StringFlag{
		Name:  "asset",
		Usage: "the asset type",
	},
}

func stateGetAll(c *cli.Context) error {
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

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.StateGetAll(c.Context,
		&gctrpc.StateGetAllRequest{Exchange: exchange},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)

	return nil
}

func stateGetDeposit(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var code string
	if c.IsSet("code") {
		code = c.String("code")
	} else {
		code = c.Args().Get(1)
	}

	var a string
	if c.IsSet("asset") {
		a = c.String("asset")
	} else {
		a = c.Args().Get(2)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.StateDeposit(c.Context,
		&gctrpc.StateDepositRequest{
			Exchange: exchange,
			Code:     code,
			Asset:    a},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func stateGetWithdrawal(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var code string
	if c.IsSet("code") {
		code = c.String("code")
	} else {
		code = c.Args().Get(1)
	}

	var a string
	if c.IsSet("asset") {
		a = c.String("asset")
	} else {
		a = c.Args().Get(2)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.StateWithdraw(c.Context,
		&gctrpc.StateWithdrawRequest{
			Exchange: exchange,
			Code:     code,
			Asset:    a},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func stateGetTrading(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var code string
	if c.IsSet("code") {
		code = c.String("code")
	} else {
		code = c.Args().Get(1)
	}

	var a string
	if c.IsSet("asset") {
		a = c.String("asset")
	} else {
		a = c.Args().Get(2)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.StateTrading(c.Context,
		&gctrpc.StateTradingRequest{
			Exchange: exchange,
			Code:     code,
			Asset:    a},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
