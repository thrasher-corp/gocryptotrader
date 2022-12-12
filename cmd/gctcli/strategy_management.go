package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var (
	stratStartTime       string
	stratEndTime         string
	stratGranularity     int64
	stratMaxImpact       float64
	stratMaxSpread       float64
	stratSimulate        bool
	stratCandleAligned   bool
	stratRetries         int64
	stratTwapGranularity int64
)

var strategyManagementCommand = &cli.Command{
	Name:      "strategy",
	Usage:     "execute strategy management command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:        "manager",
			Usage:       "interacts with manager layer",
			ArgsUsage:   "<command> <args>",
			Subcommands: []*cli.Command{managerGetAll, managerStopStrategy},
		},
		{
			Name:        "dca",
			Usage:       "initiates a DCA (Dollar Cost Average) strategy to accumulate or decumulate your position",
			ArgsUsage:   "<command> <args>",
			Subcommands: []*cli.Command{dcaStream},
		},
		{
			Name:        "twap",
			Usage:       "initiates a TWAP (Time Weighted Average Price) strategy to accumulate or decumulate your position",
			ArgsUsage:   "<command> <args>",
			Subcommands: []*cli.Command{twapStream},
		},
	},
}

var (
	managerGetAll = &cli.Command{
		Name:      "getall",
		Usage:     "gets all strategies",
		ArgsUsage: "<running>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "running",
				Usage: "only returns running strategies",
			},
		},
		Action: getAllStrats,
	}
	managerStopStrategy = &cli.Command{
		Name:      "stopstrategy",
		Usage:     "stops a strategy by uuid",
		ArgsUsage: "<uuid>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "uuid",
				Usage: "the registered strategy's uuid",
			},
		},
		Action: stopStrategy,
	}

	dcaStream = &cli.Command{
		Name:      "stream",
		Usage:     "executes strategy while reporting all actions to the client, exiting will stop strategy NOTE: cli flag might need to be used to access underyling funds e.g. --apisubaccount='main' for ftx main sub account",
		ArgsUsage: "<exchange> <pair> <asset> <start> <end>",
		Action:    dcaStreamfunc,
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
				Name:        "simulate",
				Usage:       "puts the strategy in simulation mode and will not execute live orders, this is on by default",
				Value:       true,
				Destination: &stratSimulate,
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
				Value:       time.Now().Add(time.Minute * 5).Format(common.SimpleTimeFormat),
				Destination: &stratEndTime,
			},
			&cli.Int64Flag{
				Name:        "granularity",
				Aliases:     []string{"g"},
				Usage:       klineMessage,
				Value:       60,
				Destination: &stratGranularity,
			},
			&cli.Float64Flag{
				Name:  "amount",
				Usage: "if buying is how much quote to use, if selling is how much base to liquidate",
			},
			&cli.BoolFlag{
				Name:  "fullamount",
				Usage: "will use entire funding amount associated with the apikeys",
			},
			&cli.Float64Flag{
				Name:  "pricelimit",
				Usage: "enforces price limits if lifting the asks it will not execute an order above this price. If hitting the bids this will not execute an order below this price",
			},
			&cli.Float64Flag{
				Name:        "maximpact",
				Usage:       "will enforce no orderbook impact slippage beyond this percentage amount",
				Value:       1, // Default 1% slippage catch if not set.
				Destination: &stratMaxImpact,
			},
			&cli.Float64Flag{
				Name:  "maxnominal",
				Usage: "will enforce no orderbook nominal (your average order cost from initial order cost) slippage beyond this percentage amount",
			},
			&cli.BoolFlag{
				Name:  "buy",
				Usage: "whether you are buying base or selling base",
			},
			&cli.Float64Flag{
				Name:        "maxspread",
				Usage:       "will enforce no orderbook spread percentage beyond this amount. If there is massive spread it usually means liquidity issues",
				Value:       1, // Default 1% spread catch if not set.
				Destination: &stratMaxSpread,
			},
			&cli.BoolFlag{
				Name:        "aligned",
				Usage:       "aligns execution to candle open based on requested interval",
				Value:       true,
				Destination: &stratCandleAligned,
			},
			&cli.Int64Flag{
				Name:        "retries",
				Usage:       "how many order retries will occur before a fatal error results",
				Value:       3,
				Destination: &stratRetries,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "more verbose output",
			},
		},
	}

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
				Name:        "simulate",
				Usage:       "puts the strategy in simulation mode and will not execute live orders, this is on by default",
				Value:       true,
				Destination: &stratSimulate,
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
				Value:       time.Now().Add(time.Minute * 5).Format(common.SimpleTimeFormat),
				Destination: &stratEndTime,
			},
			&cli.Int64Flag{
				Name:        "granularity",
				Aliases:     []string{"g"},
				Usage:       klineMessage,
				Value:       60,
				Destination: &stratGranularity,
			},
			&cli.Float64Flag{
				Name:  "amount",
				Usage: "if buying is how much quote to use, if selling is how much base to liquidate",
			},
			&cli.BoolFlag{
				Name:  "fullamount",
				Usage: "will use entire funding amount associated with the apikeys",
			},
			&cli.Float64Flag{
				Name:  "pricelimit",
				Usage: "enforces price limits if lifting the asks it will not execute an order above this price. If hitting the bids this will not execute an order below this price",
			},
			&cli.Float64Flag{
				Name:        "maximpact",
				Usage:       "will enforce no orderbook impact slippage from *TWAP PRICE* beyond this percentage amount",
				Value:       1, // Default 1% slippage catch if not set.
				Destination: &stratMaxImpact,
			},
			&cli.Float64Flag{
				Name:  "maxnominal",
				Usage: "will enforce no orderbook nominal (your average order cost from initial order cost) slippage beyond this percentage amount",
			},
			&cli.BoolFlag{
				Name:  "buy",
				Usage: "whether you are buying base or selling base",
			},
			&cli.Float64Flag{
				Name:        "maxspread",
				Usage:       "will enforce no orderbook spread percentage beyond this amount. If there is massive spread it usually means liquidity issues",
				Value:       1, // Default 1% spread catch if not set.
				Destination: &stratMaxSpread,
			},
			&cli.BoolFlag{
				Name:        "aligned",
				Usage:       "aligns execution to candle open based on requested interval",
				Value:       true,
				Destination: &stratCandleAligned,
			},
			&cli.Int64Flag{
				Name:        "retries",
				Usage:       "how many order retries will occur before a fatal error results",
				Value:       3,
				Destination: &stratRetries,
			},
			&cli.Int64Flag{
				Name:        "twap",
				Usage:       "TWAP generated granularity:" + klineMessage,
				Value:       3600,
				Destination: &stratTwapGranularity,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "more verbose output",
			},
		},
	}
)

func getAllStrats(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	var running bool
	if c.IsSet("running") {
		running = c.Bool("running")
	} else {
		running, _ = strconv.ParseBool(c.Args().First())
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAllStrategies(c.Context, &gctrpc.GetAllStrategiesRequest{
		Running: running,
	})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func stopStrategy(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	var id string
	if c.IsSet("uuid") {
		id = c.String("uuid")
	} else {
		id = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.StopStrategy(c.Context, &gctrpc.StopStrategyRequest{
		Id: id,
	})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

type StrategyReponse struct {
	ID       string      `json:"id,omitempty"`
	Strategy string      `json:"strategy,omitempty"`
	Reason   string      `json:"reason,omitempty"`
	Time     string      `json:"time,omitempty"`
	Action   interface{} `json:"action,omitempty"`
	Finished bool        `json:"finished,omitempty"`
}

func jsonStrategyOutput(id, strategy, reason, timeOfBroadcast string, action []byte, finished bool) {
	var ready interface{}
	_ = json.Unmarshal(action, &ready)

	payload, _ := json.MarshalIndent(StrategyReponse{ID: id, Strategy: strategy, Action: ready, Finished: finished, Reason: reason, Time: timeOfBroadcast}, "", " ")
	fmt.Println(string(payload))
}

func dcaStreamfunc(c *cli.Context) error {
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

	if c.IsSet("simulate") {
		stratSimulate = c.Bool("simulate")
	} else {
		var arg bool
		arg, err = strconv.ParseBool(c.Args().Get(3))
		if err == nil {
			stratSimulate = arg
		}
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

	var fullAmount bool
	if c.IsSet("fullamount") {
		fullAmount = c.Bool("fullamount")
	} else {
		fullAmount, _ = strconv.ParseBool(c.Args().Get(8))
	}

	var priceLimit float64
	if c.IsSet("pricelimit") {
		priceLimit = c.Float64("pricelimit")
	} else if c.Args().Get(7) != "" {
		priceLimit, err = strconv.ParseFloat(c.Args().Get(9), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("maximpact") {
		stratMaxImpact = c.Float64("maximpact")
	} else if c.Args().Get(7) != "" {
		stratMaxImpact, err = strconv.ParseFloat(c.Args().Get(10), 64)
		if err != nil {
			return err
		}
	}

	var maxNominal float64
	if c.IsSet("maxnominal") {
		maxNominal = c.Float64("maxnominal")
	} else if c.Args().Get(7) != "" {
		maxNominal, err = strconv.ParseFloat(c.Args().Get(11), 64)
		if err != nil {
			return err
		}
	}

	if stratMaxImpact <= 0 && maxNominal <= 0 {
		log.Println("Warning: No slippage protection on strategy run, this can have dire consequences. Continue (y/n)?")
		input := ""
		if _, err := fmt.Scanln(&input); err != nil {
			return err
		}
		if !common.YesOrNo(input) {
			return nil
		}
	}

	var buy bool
	if c.IsSet("buy") {
		buy = c.Bool("buy")
	} else {
		buy, _ = strconv.ParseBool(c.Args().Get(12))
	}

	if c.IsSet("maxspread") {
		stratMaxSpread = c.Float64("maxspread")
	} else if c.Args().Get(7) != "" {
		stratMaxSpread, err = strconv.ParseFloat(c.Args().Get(13), 64)
		if err != nil {
			return err
		}
	}

	if stratMaxSpread <= 0 {
		log.Println("Warning: No max spread protection on strategy run, this can have dire consequences. Continue (y/n)?")
		input := ""
		if _, err := fmt.Scanln(&input); err != nil {
			return err
		}
		if !common.YesOrNo(input) {
			return nil
		}
	}

	if c.IsSet("aligned") {
		stratCandleAligned = c.Bool("aligned")
	} else if c.Args().Get(14) != "" {
		stratCandleAligned, _ = strconv.ParseBool(c.Args().Get(14))
	}

	if c.IsSet("retries") {
		stratRetries = c.Int64("retries")
	} else if c.Args().Get(15) != "" {
		stratRetries, err = strconv.ParseInt(c.Args().Get(15), 10, 64)
		if err != nil {
			return err
		}
	}

	var verbose bool
	if c.IsSet("verbose") {
		verbose = c.Bool("verbose")
	} else if c.Args().Get(16) != "" {
		verbose, _ = strconv.ParseBool(c.Args().Get(14))
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.DCAStream(c.Context, &gctrpc.DCARequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Simulate:            stratSimulate,
		Asset:               assetType,
		Start:               negateLocalOffsetTS(s),
		End:                 negateLocalOffsetTS(e),
		Interval:            stratGranularity * int64(time.Second),
		Amount:              amount,
		FullAmount:          fullAmount,
		PriceLimit:          priceLimit,
		MaxImpactSlippage:   stratMaxImpact,
		MaxNominalSlippage:  maxNominal,
		Buy:                 buy,
		MaxSpreadPercentage: stratMaxSpread,
		AlignedToInterval:   stratCandleAligned,
		RetryAttempts:       stratRetries,
		Verbose:             verbose,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}
		jsonStrategyOutput(resp.Id, resp.Strategy, resp.Reason, time.Now().String(), resp.Action, resp.Finished)
		if resp.Finished {
			return nil
		}
	}
}

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

	if c.IsSet("simulate") {
		stratSimulate = c.Bool("simulate")
	} else {
		var arg bool
		arg, err = strconv.ParseBool(c.Args().Get(3))
		if err == nil {
			stratSimulate = arg
		}
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

	var fullAmount bool
	if c.IsSet("fullamount") {
		fullAmount = c.Bool("fullamount")
	} else {
		fullAmount, _ = strconv.ParseBool(c.Args().Get(8))
	}

	var priceLimit float64
	if c.IsSet("pricelimit") {
		priceLimit = c.Float64("pricelimit")
	} else if c.Args().Get(7) != "" {
		priceLimit, err = strconv.ParseFloat(c.Args().Get(9), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("maximpact") {
		stratMaxImpact = c.Float64("maximpact")
	} else if c.Args().Get(7) != "" {
		stratMaxImpact, err = strconv.ParseFloat(c.Args().Get(10), 64)
		if err != nil {
			return err
		}
	}

	var maxNominal float64
	if c.IsSet("maxnominal") {
		maxNominal = c.Float64("maxnominal")
	} else if c.Args().Get(7) != "" {
		maxNominal, err = strconv.ParseFloat(c.Args().Get(11), 64)
		if err != nil {
			return err
		}
	}

	if stratMaxImpact <= 0 && maxNominal <= 0 {
		log.Println("Warning: No slippage protection on strategy run, this can have dire consequences. Continue (y/n)?")
		input := ""
		if _, err := fmt.Scanln(&input); err != nil {
			return err
		}
		if !common.YesOrNo(input) {
			return nil
		}
	}

	var buy bool
	if c.IsSet("buy") {
		buy = c.Bool("buy")
	} else {
		buy, _ = strconv.ParseBool(c.Args().Get(12))
	}

	if c.IsSet("maxspread") {
		stratMaxSpread = c.Float64("maxspread")
	} else if c.Args().Get(7) != "" {
		stratMaxSpread, err = strconv.ParseFloat(c.Args().Get(13), 64)
		if err != nil {
			return err
		}
	}

	if stratMaxSpread <= 0 {
		log.Println("Warning: No max spread protection on strategy run, this can have dire consequences. Continue (y/n)?")
		input := ""
		if _, err := fmt.Scanln(&input); err != nil {
			return err
		}
		if !common.YesOrNo(input) {
			return nil
		}
	}

	if c.IsSet("aligned") {
		stratCandleAligned = c.Bool("aligned")
	} else if c.Args().Get(14) != "" {
		stratCandleAligned, _ = strconv.ParseBool(c.Args().Get(14))
	}

	if c.IsSet("retries") {
		stratRetries = c.Int64("retries")
	} else if c.Args().Get(15) != "" {
		stratRetries, err = strconv.ParseInt(c.Args().Get(15), 10, 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("twap") {
		stratTwapGranularity = c.Int64("twap")
	} else if c.Args().Get(16) != "" {
		stratTwapGranularity, err = strconv.ParseInt(c.Args().Get(16), 10, 64)
		if err != nil {
			return err
		}
	}

	var verbose bool
	if c.IsSet("verbose") {
		verbose = c.Bool("verbose")
	} else if c.Args().Get(16) != "" {
		verbose, _ = strconv.ParseBool(c.Args().Get(14))
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
		Simulate:            stratSimulate,
		Asset:               assetType,
		Start:               negateLocalOffsetTS(s),
		End:                 negateLocalOffsetTS(e),
		Interval:            stratGranularity * int64(time.Second),
		Amount:              amount,
		FullAmount:          fullAmount,
		PriceLimit:          priceLimit,
		MaxImpactSlippage:   stratMaxImpact,
		MaxNominalSlippage:  maxNominal,
		Buy:                 buy,
		MaxSpreadPercentage: stratMaxSpread,
		AlignedToInterval:   stratCandleAligned,
		RetryAttempts:       stratRetries,
		TwapInterval:        stratTwapGranularity * int64(time.Second),
		Verbose:             verbose,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}
		jsonStrategyOutput(resp.Id, resp.Strategy, resp.Reason, time.Now().String(), resp.Action, resp.Finished)
		if resp.Finished {
			return nil
		}
	}
}
