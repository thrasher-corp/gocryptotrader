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
	},
}

func getTwap(c *cli.Context) error {
	return getPrice(c, "TWAP")
}

func getVwap(c *cli.Context) error {
	return getPrice(c, "VWAP")
}

var priceStartTime string
var priceEndTime string
var priceGranularity int64

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
	}

	fmt.Println(req)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAveragePrice(c.Context, req)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
