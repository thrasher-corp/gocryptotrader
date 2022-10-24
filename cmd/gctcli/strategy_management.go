package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var (
	stratStartTime   string
	stratEndTime     string
	stratGranularity int64
)

var strategyManagementCommand = &cli.Command{
	Name:      "strategy",
	Usage:     "execute strategy management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:        "twap",
			Usage:       "initiates a twap strategy to accumulate or decumulate your position",
			ArgsUsage:   "<command> <args>",
			Subcommands: []*cli.Command{twapStream},
		},
	},
}

var (
	twapStream = &cli.Command{
		Name:      "stream",
		Usage:     "executes strategy while reporting all actions to the client, exiting will stop strategy NOTE: cli flag might need to be used to access underyling funds e.g. --apisubaccount='main' for ftx main sub account",
		ArgsUsage: "<exchange> <pair> <asset> <start> <end>",
		Action:    twapStreamfunc,
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
			&cli.StringFlag{
				Name:        "start",
				Usage:       "the start date - can be scheduled for future",
				Value:       time.Now().Format(common.SimpleTimeFormat),
				Destination: &stratStartTime,
			},
			&cli.StringFlag{
				Name:        "end",
				Usage:       "the end date",
				Value:       time.Now().AddDate(0, 0, 30).Format(common.SimpleTimeFormat),
				Destination: &stratEndTime,
			},
			&cli.Int64Flag{
				Name:        "granularity",
				Aliases:     []string{"g"},
				Usage:       klineMessage,
				Value:       86400,
				Destination: &stratGranularity,
			},
			&cli.Float64Flag{
				Name:  "amount",
				Usage: "if buying is how much quote to use, if selling is how much base to liquidate",
			},
			&cli.Int64Flag{
				Name:  "twapgranularity",
				Usage: "twap interval granularity - this will truncate and structure order execution to UTC alignment (for now)",
				// Destination: &stratLookback,
				Value: 30,
			},
			&cli.Int64Flag{
				Name:  "lookback",
				Usage: "how many candles previous from strategy interval to create signal",
				// Destination: &stratLookback,
				Value: 30,
			},
		},
	}
)

func twapStreamfunc(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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

	var accumulate bool
	if c.IsSet("buy") {
		accumulate = c.Bool("buy")
	} else {
		accumulate, _ = strconv.ParseBool(c.Args().Get(3))
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			stratStartTime = c.Args().Get(4)
		}
	} else {
		stratStartTime, _ = c.Value("start").(string)
	}

	s, err := time.Parse(common.SimpleTimeFormat, stratStartTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			stratEndTime = c.Args().Get(5)
		}
	} else {
		stratEndTime, _ = c.Value("end").(string)
	}

	e, err := time.Parse(common.SimpleTimeFormat, stratEndTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	err = common.StartEndTimeCheck(s, e)
	if err != nil && !errors.Is(err, common.ErrStartAfterTimeNow) {
		return err
	}

	if c.IsSet("granularity") {
		stratGranularity = c.Int64("granularity")
	} else if c.Args().Get(6) != "" {
		stratGranularity, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
		if err != nil {
			return err
		}
	}

	var amount float64
	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else if c.Args().Get(7) != "" {
		amount, err = strconv.ParseFloat(c.Args().Get(7), 64)
		if err != nil {
			return err
		}
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
		Asset:      assetType,
		Accumulate: accumulate,
		Start:      negateLocalOffsetTS(s),
		End:        negateLocalOffsetTS(e),
		Interval:   stratGranularity,
		Amount:     amount,
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
