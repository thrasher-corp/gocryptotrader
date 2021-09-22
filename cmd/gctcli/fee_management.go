package main

import (
	"context"
	"strconv"

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
			Usage: "sets new fee structure to running instance",
			Subcommands: []*cli.Command{
				{
					Name:      "commission",
					Usage:     "sets new maker and taker values for an exchange",
					ArgsUsage: "<exchange> <asset> <maker> <taker> <percentage>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "exchange",
							Usage: "the exchange to act on",
						},
						&cli.StringFlag{
							Name:  "asset",
							Usage: "the currency asset type",
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
							Name:   "percentage",
							Usage:  "if the fees are a set value or percentage",
							Value:  true, // Default to true
							Hidden: true,
						},
					},
					Action: setCommissionFees,
				},
				{
					Name:      "transfer",
					Usage:     "sets new withdrawal and deposit values for an exchange",
					ArgsUsage: "<exchange> <currency> <asset> <withdraw> <deposit> <setvalue>",
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
							Name:   "setvalue",
							Usage:  "if the fees are a set value or percentage",
							Value:  true, // Default to a set value
							Hidden: true,
						},
					},
					Action: setTransferFees,
				},
				{
					Name:      "banktransfer",
					Usage:     "sets new withdrawal and deposit values for an exchange bank transfer",
					ArgsUsage: "<exchange> <currency> <banktype> <withdraw> <deposit> <setvalue>",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "exchange",
							Usage: "the exchange to act on",
						},
						&cli.StringFlag{
							Name:  "currency",
							Usage: "the currency for transfer",
						},
						&cli.IntFlag{
							Name:  "banktype",
							Usage: "banking type refer too fee.BankTransaction type",
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
							Name:   "setvalue",
							Usage:  "if the fees are a set value or percentage",
							Value:  true, // Default to a set value
							Hidden: true,
						},
					},
					Action: setBankTransferFees,
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

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetAllFees(context.Background(),
		&gctrpc.GetAllFeesRequest{Exchange: exchange})
	if err != nil {
		return err
	}
	jsonOutput(result)

	return nil
}

func setCommissionFees(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(1)
	}

	var maker float64
	if c.IsSet("maker") {
		maker = c.Float64("maker")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(2), 64)
		if err != nil {
			return err
		}
		maker = f
	}

	var taker float64
	if c.IsSet("taker") {
		taker = c.Float64("taker")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(3), 64)
		if err != nil {
			return err
		}
		taker = f
	}

	var percentage bool
	if c.IsSet("percentage") {
		percentage = c.Bool("percentage")
	} else {
		b, err := strconv.ParseBool(c.Args().Get(4))
		if err != nil {
			return err
		}
		percentage = b
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.SetCommission(context.Background(),
		&gctrpc.SetCommissionRequest{
			Exchange:    exchange,
			Asset:       asset,
			Maker:       maker,
			Taker:       taker,
			IsSetAmount: !percentage,
		})
	if err != nil {
		return err
	}
	jsonOutput(result)

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

	var code string
	if c.IsSet("currency") {
		code = c.String("currency")
	} else {
		code = c.Args().Get(1)
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(2)
	}

	var withdraw float64
	if c.IsSet("withdraw") {
		withdraw = c.Float64("withdraw")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(3), 64)
		if err != nil {
			return err
		}
		withdraw = f
	}

	var deposit float64
	if c.IsSet("deposit") {
		deposit = c.Float64("deposit")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
		deposit = f
	}

	var setValue bool
	if c.IsSet("setvalue") {
		setValue = c.Bool("setvalue")
	} else {
		b, err := strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
		setValue = b
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.SetTransferFee(context.Background(),
		&gctrpc.SetTransferFeeRequest{
			Exchange:     exchange,
			Currency:     code,
			Asset:        asset,
			Withdraw:     withdraw,
			Deposit:      deposit,
			IsPercentage: !setValue,
		})
	if err != nil {
		return err
	}
	jsonOutput(result)

	return nil
}

func setBankTransferFees(c *cli.Context) error {
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
	if c.IsSet("currency") {
		code = c.String("currency")
	} else {
		code = c.Args().Get(1)
	}

	var bank int64
	if c.IsSet("banktype") {
		bank = c.Int64("banktype")
	} else {
		i, err := strconv.ParseInt(c.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}
		bank = i
	}

	var withdraw float64
	if c.IsSet("withdraw") {
		withdraw = c.Float64("withdraw")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(3), 64)
		if err != nil {
			return err
		}
		withdraw = f
	}

	var deposit float64
	if c.IsSet("deposit") {
		deposit = c.Float64("deposit")
	} else {
		f, err := strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
		deposit = f
	}

	var setValue bool
	if c.IsSet("setvalue") {
		setValue = c.Bool("setvalue")
	} else {
		b, err := strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
		setValue = b
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.SetBankTransferFee(context.Background(),
		&gctrpc.SetBankTransferFeeRequest{
			Exchange:     exchange,
			Currency:     code,
			BankType:     int32(bank),
			Withdraw:     withdraw,
			Deposit:      deposit,
			IsPercentage: !setValue,
		})
	if err != nil {
		return err
	}
	jsonOutput(result)

	return nil
}
