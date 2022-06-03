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
		Destination: &priceGranularity,
	},
	&cli.StringFlag{
		Name:        "start",
		Usage:       "the start date",
		Value:       time.Now().AddDate(0, -1, 0).Format(common.SimpleTimeFormat),
		Destination: &priceStartTime,
	},
	&cli.StringFlag{
		Name:        "end",
		Usage:       "the end date",
		Value:       time.Now().Format(common.SimpleTimeFormat),
		Destination: &priceEndTime,
	},
	&cli.Int64Flag{
		Name:        "period",
		Usage:       "denotes period (rolling window) for technical analysis",
		Value:       9,
		Destination: &pricePeriod,
	},
}

var pricingCommand = &cli.Command{
	Name:      "price",
	Usage:     "get weighted pricing command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "twap",
			Usage:     "returns the time weighted average price",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getTwap,
		},
		{
			Name:      "vwap",
			Usage:     "returns the volume weighted average price",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getVwap,
		},
		{
			Name:      "atr",
			Usage:     "returns the average true range",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getATR,
		},
		// { // TODO: With comparison exchange pairs
		// 	Name:      "coco",
		// 	Usage:     "returns the correlation-coefficient",
		// 	ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
		// 	Flags:     commonFlag,
		// 	Action:    getCOCO,
		// },
		{
			Name:      "sma",
			Usage:     "returns the simple moving average",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getSMA,
		},
		{
			Name:      "ema",
			Usage:     "returns the exponential moving average",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getEMA,
		},
		{
			Name:      "macd",
			Usage:     "returns the moving average convergence divergence",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getMACD,
		},
		{
			Name:      "mfi",
			Usage:     "returns the money flow index",
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
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
			ArgsUsage: "<exchange> <pair> <asset> <granularity> <start> <end>",
			Flags:     commonFlag,
			Action:    getRSI,
		},
	},
}

func getTwap(c *cli.Context) error {
	return getPrice(c, "TWAP")
}

func getVwap(c *cli.Context) error {
	return getPrice(c, "VWAP")
}

func getATR(c *cli.Context) error {
	return getPrice(c, "ATR")
}

func getBBANDS(c *cli.Context) error {
	return getPrice(c, "BBANDS")
}

// TODO: I'm in love with the coco
// func getCOCO(c *cli.Context) error {
// 	return getPrice(c, "COCO")
// }

func getSMA(c *cli.Context) error {
	return getPrice(c, "SMA")
}

func getEMA(c *cli.Context) error {
	return getPrice(c, "EMA")
}

func getMACD(c *cli.Context) error {
	return getPrice(c, "MACD")
}

func getMFI(c *cli.Context) error {
	return getPrice(c, "MFI")
}

func getOBV(c *cli.Context) error {
	return getPrice(c, "OBV")
}

func getRSI(c *cli.Context) error {
	return getPrice(c, "RSI")
}

var priceStartTime string
var priceEndTime string
var priceGranularity int64
var pricePeriod int64

func getPrice(c *cli.Context, algo string) error {
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
		priceGranularity = c.Int64("granularity")
	} else if c.Args().Get(4) != "" {
		priceGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			priceStartTime = c.Args().Get(4)
		}
	} else {
		priceStartTime = c.Value("start").(string)
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			priceEndTime = c.Args().Get(5)
		}
	} else {
		priceEndTime = c.Value("end").(string)
	}

	s, err := time.Parse(common.SimpleTimeFormat, priceStartTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.Parse(common.SimpleTimeFormat, priceEndTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if e.Before(s) {
		return errors.New("start cannot be after end")
	}

	if !c.IsSet("period") {
		if c.Args().Get(6) != "" {
			pricePeriod, err = strconv.ParseInt(c.Args().Get(6), 10, 64)
			if err != nil {
				return err
			}
		}
	} else {
		pricePeriod = c.Value("period").(int64)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	req := &gctrpc.GetAveragePriceRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  pair.Base.String(),
			Quote: pair.Quote.String(),
		},
		AssetType:     asset,
		AlgorithmType: algo,
		Interval:      priceGranularity * int64(time.Second),
		Start:         timestamppb.New(s),
		End:           timestamppb.New(e),
		Period:        pricePeriod,
	}

	fmt.Println("Request: ", req)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAveragePrice(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
