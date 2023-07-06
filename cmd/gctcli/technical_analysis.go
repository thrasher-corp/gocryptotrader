package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	taStartTime         string
	taEndTime           string
	taGranularity       int64
	taPeriod            int64
	taFastPeriod        int64
	taSlowPeriod        int64
	taMovingAverageType string
	taStdDevUp          float64
	taStdDevDown        float64
)

var commonFlag = []cli.Flag{
	&cli.StringFlag{
		Name:  "exchange",
		Usage: "the exchange to act on",
	},
	&cli.StringFlag{
		Name:  "pair",
		Usage: "currency pair",
	},
	&cli.StringFlag{
		Name:  "asset",
		Usage: "asset",
	},
	&cli.Int64Flag{
		Name:        "granularity",
		Aliases:     []string{"g"},
		Usage:       klineMessage,
		Value:       86400,
		Destination: &taGranularity,
	},
	&cli.StringFlag{
		Name:        "start",
		Usage:       "the start date",
		Value:       time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		Destination: &taStartTime,
	},
	&cli.StringFlag{
		Name:        "end",
		Usage:       "the end date",
		Value:       time.Now().Format(time.DateTime),
		Destination: &taEndTime,
	},
}

var (
	periodFlag = &cli.Int64Flag{
		Name:        "period",
		Usage:       "denotes period (rolling window) for technical analysis",
		Value:       9,
		Destination: &taPeriod,
	}
	fastFlag = &cli.Int64Flag{
		Name:        "fastperiod",
		Usage:       "denotes fast period (ema) for macd generation",
		Value:       12,
		Destination: &taFastPeriod,
	}
	slowFlag = &cli.Int64Flag{
		Name:        "slowperiod",
		Usage:       "denotes slow period (ema) for macd generation",
		Value:       26,
		Destination: &taSlowPeriod,
	}
	stdDevUpFlag = &cli.Float64Flag{
		Name:        "stddevup",
		Usage:       "standard deviation limit for upper band",
		Value:       1.5,
		Destination: &taStdDevUp,
	}
	stdDevDownFlag = &cli.Float64Flag{
		Name:        "stddevdown",
		Usage:       "standard deviation limit for lower band",
		Value:       1.5,
		Destination: &taStdDevDown,
	}
	maTypeFlag = &cli.StringFlag{
		Name:        "movingaveragetype",
		Usage:       "defines the moving average type for underlying calculation ('ema'/'sma')",
		Value:       "sma",
		Destination: &taMovingAverageType,
	}

	otherAssetFlag = []cli.Flag{
		&cli.StringFlag{
			Name:    "comparisonexchange",
			Usage:   "the other exchange to compare to - if not supplied will default to initial exchange",
			Aliases: []string{"ce", "cexchange", "oe", "otherexchange"},
		},
		&cli.StringFlag{
			Name:    "comparisonpair",
			Usage:   "the other currency pair",
			Aliases: []string{"cp", "cpair", "op", "otherpair"},
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
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "twap",
			Usage:     "returns the time weighted average price",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getTWAP,
		},
		{
			Name:      "vwap",
			Usage:     "returns the volume weighted average price",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getVWAP,
		},
		{
			Name:      "atr",
			Usage:     "returns the average true range",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period>",
			Flags:     append(commonFlag, periodFlag),
			Action:    getATR,
		},
		{
			Name:      "bbands",
			Usage:     "returns the bollinger bands",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period> <std deviation up> <std deviation down> <moving average type>",
			Flags:     append(commonFlag, periodFlag, stdDevUpFlag, stdDevDownFlag, maTypeFlag),
			Action:    getBollingerBands,
		},
		{
			Name:      "coco",
			Usage:     "returns the correlation-coefficient",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <other exchange> <other asset> <other pair>",
			Flags:     append(commonFlag, append([]cli.Flag{periodFlag}, otherAssetFlag...)...),
			Action:    getCoco,
		},
		{
			Name:      "sma",
			Usage:     "returns the simple moving average",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period>",
			Flags:     append(commonFlag, periodFlag),
			Action:    getSMA,
		},
		{
			Name:      "ema",
			Usage:     "returns the exponential moving average",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period>",
			Flags:     append(commonFlag, periodFlag),
			Action:    getEMA,
		},
		{
			Name:      "macd",
			Usage:     "returns the moving average convergence divergence",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period> <fast period> <slow period>",
			Flags:     append(commonFlag, periodFlag, fastFlag, slowFlag),
			Action:    getMACD,
		},
		{
			Name:      "mfi",
			Usage:     "returns the money flow index",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period>",
			Flags:     append(commonFlag, periodFlag),
			Action:    getMFI,
		},
		{
			Name:      "obv",
			Usage:     "returns the on balance volume",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getOBV,
		},
		{
			Name:      "rsi",
			Usage:     "returns the relative strength index",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end> <period>",
			Flags:     append(commonFlag, periodFlag),
			Action:    getRSI,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var cpString string
	if c.IsSet("pair") {
		cpString = c.String("pair")
	} else {
		cpString = c.Args().Get(1)
	}

	pair, err := currency.NewPairFromString(cpString)
	if err != nil {
		return err
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(2)
	}

	asset = strings.ToLower(asset)
	if !validAsset(asset) {
		return errInvalidAsset
	}

	if c.IsSet("granularity") {
		taGranularity = c.Int64("granularity")
	} else if c.Args().Get(3) != "" {
		taGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			taStartTime = c.Args().Get(4)
		}
	} else {
		taStartTime, _ = c.Value("start").(string)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			taEndTime = c.Args().Get(5)
		}
	} else {
		taEndTime, _ = c.Value("end").(string)
	}

	s, err := time.ParseInLocation(time.DateTime, taStartTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, taEndTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}
	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if !c.IsSet("period") {
		if c.Args().Get(6) != "" {
			taPeriod, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taPeriod, _ = c.Value("period").(int64)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:     asset,
		AlgorithmType: algo,
		Interval:      taGranularity * int64(time.Second),
		Start:         timestamppb.New(s),
		End:           timestamppb.New(e),
		Period:        taPeriod,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var cpString string
	if c.IsSet("pair") {
		cpString = c.String("pair")
	} else {
		cpString = c.Args().Get(1)
	}

	pair, err := currency.NewPairFromString(cpString)
	if err != nil {
		return err
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(2)
	}

	asset = strings.ToLower(asset)
	if !validAsset(asset) {
		return errInvalidAsset
	}

	if c.IsSet("granularity") {
		taGranularity = c.Int64("granularity")
	} else if c.Args().Get(3) != "" {
		taGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			taStartTime = c.Args().Get(4)
		}
	} else {
		taStartTime, _ = c.Value("start").(string)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			taEndTime = c.Args().Get(5)
		}
	} else {
		taEndTime, _ = c.Value("end").(string)
	}

	s, err := time.ParseInLocation(time.DateTime, taStartTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, taEndTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if !c.IsSet("period") {
		if c.Args().Get(6) != "" {
			taPeriod, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taPeriod, _ = c.Value("period").(int64)
	}

	if !c.IsSet("stddevup") {
		if c.Args().Get(7) != "" {
			taStdDevUp, err = strconv.ParseFloat(c.Args().Get(7), 64)
			if err != nil {
				return err
			}
		}
	} else {
		taStdDevUp, _ = c.Value("stddevup").(float64)
	}

	if !c.IsSet("stddevdown") {
		if c.Args().Get(8) != "" {
			taStdDevDown, err = strconv.ParseFloat(c.Args().Get(8), 64)
			if err != nil {
				return err
			}
		}
	} else {
		taStdDevDown, _ = c.Value("stddevdown").(float64)
	}

	if !c.IsSet("movingaveragetype") && c.Args().Get(9) != "" {
		taMovingAverageType = c.Args().Get(9)
	} else {
		taMovingAverageType, _ = c.Value("movingaveragetype").(string)
	}

	var maType int64
	switch strings.ToLower(taMovingAverageType) {
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
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:             asset,
		AlgorithmType:         "BBANDS",
		Interval:              taGranularity * int64(time.Second),
		Start:                 timestamppb.New(s),
		End:                   timestamppb.New(e),
		Period:                taPeriod,
		StandardDeviationUp:   taStdDevUp,
		StandardDeviationDown: taStdDevDown,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var cpString string
	if c.IsSet("pair") {
		cpString = c.String("pair")
	} else {
		cpString = c.Args().Get(1)
	}

	pair, err := currency.NewPairFromString(cpString)
	if err != nil {
		return err
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(2)
	}

	asset = strings.ToLower(asset)
	if !validAsset(asset) {
		return errInvalidAsset
	}

	if c.IsSet("granularity") {
		taGranularity = c.Int64("granularity")
	} else if c.Args().Get(3) != "" {
		taGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			taStartTime = c.Args().Get(4)
		}
	} else {
		taStartTime, _ = c.Value("start").(string)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			taEndTime = c.Args().Get(5)
		}
	} else {
		taEndTime, _ = c.Value("end").(string)
	}

	s, err := time.ParseInLocation(time.DateTime, taStartTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, taEndTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if !c.IsSet("period") {
		if c.Args().Get(6) != "" {
			taPeriod, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taPeriod, _ = c.Value("period").(int64)
	}

	if !c.IsSet("fastperiod") {
		if c.Args().Get(7) != "" {
			taFastPeriod, err = strconv.ParseInt(c.Args().Get(7), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taFastPeriod, _ = c.Value("fastperiod").(int64)
	}

	if !c.IsSet("slowperiod") {
		if c.Args().Get(8) != "" {
			taSlowPeriod, err = strconv.ParseInt(c.Args().Get(8), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taSlowPeriod, _ = c.Value("slowperiod").(int64)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:     asset,
		AlgorithmType: "MACD",
		Interval:      taGranularity * int64(time.Second),
		Start:         timestamppb.New(s),
		End:           timestamppb.New(e),
		Period:        taPeriod,
		SlowPeriod:    taSlowPeriod,
		FastPeriod:    taFastPeriod,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var cpString string
	if c.IsSet("pair") {
		cpString = c.String("pair")
	} else {
		cpString = c.Args().Get(1)
	}

	pair, err := currency.NewPairFromString(cpString)
	if err != nil {
		return err
	}

	var asset string
	if c.IsSet("asset") {
		asset = c.String("asset")
	} else {
		asset = c.Args().Get(2)
	}

	asset = strings.ToLower(asset)
	if !validAsset(asset) {
		return errInvalidAsset
	}

	if c.IsSet("granularity") {
		taGranularity = c.Int64("granularity")
	} else if c.Args().Get(3) != "" {
		taGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			taStartTime = c.Args().Get(4)
		}
	} else {
		taStartTime, _ = c.Value("start").(string)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			taEndTime = c.Args().Get(5)
		}
	} else {
		taEndTime, _ = c.Value("end").(string)
	}

	s, err := time.ParseInLocation(time.DateTime, taStartTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, taEndTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if !c.IsSet("period") {
		if c.Args().Get(6) != "" {
			taPeriod, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		taPeriod, _ = c.Value("period").(int64)
	}

	var otherExchange string
	if c.IsSet("comparisonexchange") {
		otherExchange = c.String("comparisonexchange")
	} else {
		otherExchange = c.Args().Get(7)
	}

	var oCpString string
	if c.IsSet("comparisonpair") {
		oCpString = c.String("comparisonpair")
	} else {
		oCpString = c.Args().Get(8)
	}

	if oCpString == "" {
		return errors.New("other pair is empty, to compare this must be specified")
	}
	otherPair, err := currency.NewPairFromString(oCpString)
	if err != nil {
		return err
	}

	var otherAsset string
	if c.IsSet("comparisonasset") {
		otherAsset = c.String("comparisonasset")
	} else {
		otherAsset = c.Args().Get(9)
	}

	otherAsset = strings.ToLower(otherAsset)
	if otherAsset != "" && !validAsset(otherAsset) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:      asset,
		AlgorithmType:  "COCO",
		Interval:       taGranularity * int64(time.Second),
		Start:          timestamppb.New(s),
		End:            timestamppb.New(e),
		Period:         taPeriod,
		OtherExchange:  otherExchange,
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
