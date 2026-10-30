package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var commonFlag = []cli.Flag{
	&cli.StringFlag{
		Name:     exchangeFlag,
		Required: true,
		Usage:    exchangeUsage,
	},
	&cli.StringFlag{
		Name:     pairFlag,
		Required: true,
		Usage:    "currency pair",
	},
	&cli.StringFlag{
		Name:     assetFlag,
		Required: true,
		Usage:    assetFlagUsage,
	},
	&cli.Int64Flag{
		Name:    "granularity",
		Aliases: []string{"g"},
		Usage:   klineMessage,
		Value:   86400,
	},
	&cli.StringFlag{
		Name:  startFlag,
		Usage: "the start date",
		Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
	},
	&cli.StringFlag{
		Name:  endFlag,
		Usage: "the end date",
		Value: time.Now().Format(time.DateTime),
	},
}

var (
	periodFlag = &cli.Int64Flag{
		Name:  "period",
		Usage: "denotes period (rolling window) for technical analysis",
		Value: 9,
	}
	fastFlag = &cli.Int64Flag{
		Name:  "fastperiod",
		Usage: "denotes fast period (ema) for macd generation",
		Value: 12,
	}
	slowFlag = &cli.Int64Flag{
		Name:  "slowperiod",
		Usage: "denotes slow period (ema) for macd generation",
		Value: 26,
	}
	stdDevUpFlag = &cli.Float64Flag{
		Name:  "stddevup",
		Usage: "standard deviation limit for upper band",
		Value: 1.5,
	}
	stdDevDownFlag = &cli.Float64Flag{
		Name:  "stddevdown",
		Usage: "standard deviation limit for lower band",
		Value: 1.5,
	}
	maTypeFlag = &cli.StringFlag{
		Name:  "movingaveragetype",
		Usage: "defines the moving average type for underlying calculation ('ema'/'sma')",
		Value: "sma",
	}

	otherAssetFlag = []cli.Flag{
		&cli.StringFlag{
			Name:    "comparisonexchange",
			Usage:   "the other exchange to compare to - if not supplied will default to initial exchange",
			Aliases: []string{"ce", "cexchange", "oe", "otherexchange"},
		},
		&cli.StringFlag{
			Name:     "comparisonpair",
			Required: true,
			Usage:    "the other currency pair",
			Aliases:  []string{"cp", "cpair", "op", "otherpair"},
		},
		&cli.StringFlag{
			Name:    "comparisonasset",
			Usage:   "the other asset - if not supplied will default to initial exchange",
			Aliases: []string{"ca", "casset", "oa", "otherasset"},
		},
	}
)

var technicalAnalysisCommand = &cli.Command{
	Name:      "technicalanalysis",
	Usage:     "get technical analysis command",
	Aliases:   []string{"ta"},
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:   "twap",
			Usage:  "returns the time weighted average price",
			Flags:  commonFlag,
			Action: getTWAP,
		},
		{
			Name:   "vwap",
			Usage:  "returns the volume weighted average price",
			Flags:  commonFlag,
			Action: getVWAP,
		},
		{
			Name:   "atr",
			Usage:  "returns the average true range",
			Flags:  append(commonFlag, periodFlag),
			Action: getATR,
		},
		{
			Name:   "bbands",
			Usage:  "returns the bollinger bands",
			Flags:  append(commonFlag, periodFlag, stdDevUpFlag, stdDevDownFlag, maTypeFlag),
			Action: getBollingerBands,
		},
		{
			Name:   "coco",
			Usage:  "returns the correlation-coefficient",
			Flags:  append(commonFlag, append([]cli.Flag{periodFlag}, otherAssetFlag...)...),
			Action: getCoco,
		},
		{
			Name:   "sma",
			Usage:  "returns the simple moving average",
			Flags:  append(commonFlag, periodFlag),
			Action: getSMA,
		},
		{
			Name:   "ema",
			Usage:  "returns the exponential moving average",
			Flags:  append(commonFlag, periodFlag),
			Action: getEMA,
		},
		{
			Name:   "macd",
			Usage:  "returns the moving average convergence divergence",
			Flags:  append(commonFlag, periodFlag, fastFlag, slowFlag),
			Action: getMACD,
		},
		{
			Name:   "mfi",
			Usage:  "returns the money flow index",
			Flags:  append(commonFlag, periodFlag),
			Action: getMFI,
		},
		{
			Name:   "obv",
			Usage:  "returns the on balance volume",
			Flags:  commonFlag,
			Action: getOBV,
		},
		{
			Name:   "rsi",
			Usage:  "returns the relative strength index",
			Flags:  append(commonFlag, periodFlag),
			Action: getRSI,
		},
	},
}

func getTWAP(c *cli.Context) error {
	return getTecnicalAnalysis(c, "TWAP")
}

func getVWAP(c *cli.Context) error {
	return getTecnicalAnalysis(c, "VWAP")
}

func getATR(c *cli.Context) error {
	return getTecnicalAnalysis(c, "ATR")
}

func getSMA(c *cli.Context) error {
	return getTecnicalAnalysis(c, "SMA")
}

func getEMA(c *cli.Context) error {
	return getTecnicalAnalysis(c, "EMA")
}

func getMFI(c *cli.Context) error {
	return getTecnicalAnalysis(c, "MFI")
}

func getOBV(c *cli.Context) error {
	return getTecnicalAnalysis(c, "OBV")
}

func getRSI(c *cli.Context) error {
	return getTecnicalAnalysis(c, "RSI")
}

func getTecnicalAnalysis(c *cli.Context, algo string) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	pair, err := currency.NewPairFromString(c.String(pairFlag))
	if err != nil {
		return err
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	s, err := time.ParseInLocation(time.DateTime, c.String(startFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err := time.ParseInLocation(time.DateTime, c.String(endFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}
	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: c.String(exchangeFlag),
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:     asset,
		AlgorithmType: algo,
		Interval:      c.Int64("granularity") * int64(time.Second),
		Start:         timestamppb.New(s),
		End:           timestamppb.New(e),
		Period:        c.Int64("period"),
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTechnicalAnalysis(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getBollingerBands(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	pair, err := currency.NewPairFromString(c.String(pairFlag))
	if err != nil {
		return err
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	s, err := time.ParseInLocation(time.DateTime, c.String(startFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err := time.ParseInLocation(time.DateTime, c.String(endFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	var maType int64
	switch strings.ToLower(c.String("movingaveragetype")) {
	case "sma":
	case "ema":
		maType = 1
	default:
		return errors.New("invalid moving average type")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: c.String(exchangeFlag),
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:             asset,
		AlgorithmType:         "BBANDS",
		Interval:              c.Int64("granularity") * int64(time.Second),
		Start:                 timestamppb.New(s),
		End:                   timestamppb.New(e),
		Period:                c.Int64("period"),
		StandardDeviationUp:   c.Float64("stddevup"),
		StandardDeviationDown: c.Float64("stddevdown"),
		MovingAverageType:     maType,
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTechnicalAnalysis(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getMACD(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	pair, err := currency.NewPairFromString(c.String(pairFlag))
	if err != nil {
		return err
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	s, err := time.ParseInLocation(time.DateTime, c.String(startFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err := time.ParseInLocation(time.DateTime, c.String(endFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: c.String(exchangeFlag),
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:     asset,
		AlgorithmType: "MACD",
		Interval:      c.Int64("granularity") * int64(time.Second),
		Start:         timestamppb.New(s),
		End:           timestamppb.New(e),
		Period:        c.Int64("period"),
		SlowPeriod:    c.Int64("slowperiod"),
		FastPeriod:    c.Int64("fastperiod"),
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTechnicalAnalysis(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCoco(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	pair, err := currency.NewPairFromString(c.String(pairFlag))
	if err != nil {
		return err
	}

	asset := strings.ToLower(c.String(assetFlag))
	if !validAsset(asset) {
		return errInvalidAsset
	}

	s, err := time.ParseInLocation(time.DateTime, c.String(startFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err := time.ParseInLocation(time.DateTime, c.String(endFlag), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	oCpString := c.String("comparisonpair")

	if oCpString == "" {
		return errors.New("other pair is empty, to compare this must be specified")
	}
	otherPair, err := currency.NewPairFromString(oCpString)
	if err != nil {
		return err
	}

	otherAsset := strings.ToLower(c.String("comparisonasset"))
	if otherAsset != "" && !validAsset(otherAsset) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: c.String(exchangeFlag),
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:      asset,
		AlgorithmType:  "COCO",
		Interval:       c.Int64("granularity") * int64(time.Second),
		Start:          timestamppb.New(s),
		End:            timestamppb.New(e),
		Period:         c.Int64("period"),
		OtherExchange:  c.String("comparisonexchange"),
		OtherPair:      &gctrpc.CurrencyPair{Base: otherPair.Base.String(), Quote: otherPair.Quote.String()},
		OtherAssetType: otherAsset,
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTechnicalAnalysis(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
