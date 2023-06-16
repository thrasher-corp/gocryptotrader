package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

// futuresCommands contains all commands related to futures
// position data, funding rates, collateral, pnl etc
var futuresCommands = &cli.Command{
	Name:      "futures",
	Aliases:   []string{"f"},
	Usage:     "contains all futures based rpc commands",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "getmanagedposition",
			Aliases:   []string{"managedposition", "mp"},
			Usage:     "retrieves an open position monitored by the order manager",
			ArgsUsage: "<exchange> <asset> <pair> <includeorderdetails> <getfundingdata> <includefundingentries> <includepredictedrate>",
			Action:    getManagedPosition,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair of the position",
				},
				&cli.BoolFlag{
					Name:    "includeorderdetails",
					Aliases: []string{"orders"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getfundingdata",
					Aliases: []string{"funding", "fd"},
					Usage:   "if true, will return funding rate summary",
				},
				&cli.BoolFlag{
					Name:    "includefundingentries",
					Aliases: []string{"allfunding", "af"},
					Usage:   "if true, will return all funding rate entries - requires --getfundingdata",
				},
				&cli.BoolFlag{
					Name:    "includepredictedrate",
					Aliases: []string{"predicted", "pr"},
					Usage:   "if true, will return the predicted funding rate - requires --getfundingdata",
				},
			},
		},
		{
			Name:      "getallmanagedpositions",
			Aliases:   []string{"managedpositions", "mps"},
			Usage:     "retrieves all open positions monitored by the order manager",
			ArgsUsage: "<includeorderdetails> <getfundingdata> <includefundingentries> <includepredictedrate>",
			Action:    getAllManagedPositions,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "includeorderdetails",
					Aliases: []string{"orders"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getfundingdata",
					Aliases: []string{"funding", "fd"},
					Usage:   "if true, will return funding rate summary",
				},
				&cli.BoolFlag{
					Name:    "includefundingentries",
					Aliases: []string{"allfunding", "af"},
					Usage:   "if true, will return all funding rate entries - requires --getfundingdata",
				},
				&cli.BoolFlag{
					Name:    "includepredictedrate",
					Aliases: []string{"predicted", "pr"},
					Usage:   "if true, will return the predicted funding rate - requires --getfundingdata",
				},
			},
		},
		{
			Name:      "getcollateral",
			Aliases:   []string{"collateral", "c"},
			Usage:     "returns total collateral for an exchange asset, with optional per currency breakdown",
			ArgsUsage: "<exchange> <asset> <calculateoffline> <includebreakdown> <includezerovalues>",
			Action:    getCollateral,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.BoolFlag{
					Name:    "calculateoffline",
					Aliases: []string{"c"},
					Usage:   "use local scaling calculations instead of requesting the collateral values directly, depending on individual exchange support",
				},
				&cli.BoolFlag{
					Name:    "includebreakdown",
					Aliases: []string{"i"},
					Usage:   "include a list of each held currency and its contribution to the overall collateral value",
				},
				&cli.BoolFlag{
					Name:    "includezerovalues",
					Aliases: []string{"z"},
					Usage:   "include collateral values that are zero",
				},
			},
		},
		{
			Name:      "getfundingrates",
			Aliases:   []string{"funding", "f"},
			Usage:     "returns funding rate data between two dates",
			ArgsUsage: "<exchange> <asset> <pairs> <start> <end> <includepredicted> <includepayments>",
			Action:    getFundingRates,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringSliceFlag{
					Name:    "pairs",
					Aliases: []string{"p"},
					Usage:   "comma delimited list of pairs you wish to get funding rate data for",
				},
				&cli.StringFlag{
					Name:        "start",
					Aliases:     []string{"sd"},
					Usage:       "<start> rounded down to the nearest hour, ensure your starting position is within this window for accurate calculations",
					Value:       time.Now().AddDate(-1, 0, 0).Truncate(time.Hour).Format(common.SimpleTimeFormat),
					Destination: &startTime,
				},
				&cli.StringFlag{
					Name:        "end",
					Aliases:     []string{"ed"},
					Usage:       "<end> rounded down to the nearest hour, ensure your last position is within this window for accurate calculations",
					Value:       time.Now().Format(common.SimpleTimeFormat),
					Destination: &endTime,
				},
				&cli.BoolFlag{
					Name:    "includepredicted",
					Aliases: []string{"ip", "predicted"},
					Usage:   "include the predicted next funding rate",
				},
				&cli.BoolFlag{
					Name:    "includepayments",
					Aliases: []string{"pay"},
					Usage:   "include funding rate payments",
				},
			},
		},
		{
			Name:      "getcollateralmode",
			Aliases:   []string{"gcm"},
			Usage:     "gets the collateral mode for an exchange asset",
			ArgsUsage: "<exchange> <asset>",
			Action:    getCollateralMode,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
			},
		},
		{
			Name:      "setcollateralmode",
			Aliases:   []string{"scm"},
			Usage:     "sets the collateral mode for an exchange asset",
			ArgsUsage: "<exchange> <asset> <collateralmode>",
			Action:    setCollateralMode,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},

				&cli.StringFlag{
					Name:    "collateralmode",
					Aliases: []string{"collateral", "cm", "c"},
					Usage:   "the collateral mode type, such as 'single', 'multi' or 'global'",
				},
			},
		},
		{
			Name:      "setleverage",
			Aliases:   []string{"sl"},
			Usage:     "sets the initial leverage level for an exchange currency pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype> <leverage>",
			Action:    setLeverage,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
				&cli.Float64Flag{
					Name:    "leverage",
					Aliases: []string{"l", "riskon", "uponly", "yolo", "steadylads"},
					Usage:   "the level of leverage you want, increase it to lose your capital faster",
				},
			},
		},
		{
			Name:      "getleverage",
			Aliases:   []string{"gl"},
			Usage:     "gets the initial leverage level for an exchange currency pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype>",
			Action:    getLeverage,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
			},
		},
		{
			Name:      "changepositionmargin",
			Aliases:   []string{"cpm"},
			Usage:     "sets isolated margin levels for an existing position",
			ArgsUsage: "<exchange> <asset> <pair> <start>",
			Action:    changePositionMargin,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, most likely 'isolated'",
				},
				&cli.Float64Flag{
					Name:    "originalallocatedmargin",
					Aliases: []string{"oac"},
					Usage:   "the original allocated margin, is used by some exchanges to determine differences to apply",
				},
				&cli.Float64Flag{
					Name:    "newallocatedmargin",
					Aliases: []string{"nac"},
					Usage:   "the new allocated margin level you desire",
				},
				&cli.StringFlag{
					Name:    "marginside",
					Aliases: []string{"side", "ms"},
					Usage:   "optional - the margin side, typically 'buy' or 'sell'",
				},
			},
		},
		{
			Name:      "getfuturespositionsummary",
			Aliases:   []string{"summary", "fps"},
			Usage:     "return a summary of your futures position",
			ArgsUsage: "<exchange> <asset> <pair> <start>",
			Action:    getFuturesPositionSummary,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:    "underlyingpair",
					Aliases: []string{"up"},
					Usage:   "optional - the underlying currency pair eg if pair is BTCUSD-1984-C, the underlying pair could be BTC-USD",
				},
			},
		},
		{
			Name:      "getfuturepositionorders",
			Aliases:   []string{"orders", "fpo"},
			Usage:     "return a slice of orders that make up your position",
			ArgsUsage: "<exchange> <asset> <pair> <start>",
			Action:    getFuturePositionOrders,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:        "start",
					Aliases:     []string{"sd"},
					Usage:       "<start> rounded down to the nearest hour",
					Value:       time.Now().AddDate(0, 0, -7).Truncate(time.Hour).Format(common.SimpleTimeFormat),
					Destination: &startTime,
				},
				&cli.StringFlag{
					Name:        "end",
					Aliases:     []string{"ed"},
					Usage:       "<end> rounded down to the nearest hour",
					Value:       time.Now().Truncate(time.Hour).Format(common.SimpleTimeFormat),
					Destination: &endTime,
				},
				&cli.BoolFlag{
					Name:    "respectorderhistorylimits",
					Aliases: []string{"r"},
					Usage:   "recommended true - if set to true, will not request orders beyond its API limits, preventing errors",
				},
				&cli.StringFlag{
					Name:    "underlyingpair",
					Aliases: []string{"up"},
					Usage:   "optional - the underlying currency pair",
				},
				&cli.BoolFlag{
					Name:    "syncwithordermanager",
					Aliases: []string{"sync", "s"},
					Usage:   "if true, will sync the orders with the order manager if supported",
				},
			},
		},
		{
			Name:      "setmargintype",
			Aliases:   []string{"smt"},
			Usage:     "sets the margin type for a exchange asset pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype> <leverage>",
			Action:    setMarginType,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "exchange",
					Aliases: []string{"e"},
					Usage:   "the exchange to retrieve futures positions from",
				},
				&cli.StringFlag{
					Name:    "asset",
					Aliases: []string{"a"},
					Usage:   "the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    "pair",
					Aliases: []string{"p"},
					Usage:   "the currency pair",
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
			},
		},
	},
}

func getManagedPosition(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}
	err := isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	var currencyPair string
	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var includeOrderDetails bool
	if c.IsSet("includeorderdetails") {
		includeOrderDetails = c.Bool("includeorderdetails")
	} else if c.Args().Get(3) != "" {
		includeOrderDetails, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	var getFundingData bool
	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	} else if c.Args().Get(4) != "" {
		getFundingData, err = strconv.ParseBool(c.Args().Get(4))
		if err != nil {
			return err
		}
	}

	var includeFundingEntries bool
	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	} else if c.Args().Get(5) != "" {
		includeFundingEntries, err = strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
	}

	var includePredictedRate bool
	if c.IsSet("includepredictedrate") {
		includePredictedRate = c.Bool("includepredictedrate")
	} else if c.Args().Get(6) != "" {
		includePredictedRate, err = strconv.ParseBool(c.Args().Get(6))
		if err != nil {
			return err
		}
	}

	err = order.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetManagedPosition(c.Context,
		&gctrpc.GetManagedPositionRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludeFullOrderData:    includeOrderDetails,
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
			IncludePredictedRate:    includePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getAllManagedPositions(c *cli.Context) error {
	var (
		err                   error
		includeOrderDetails   bool
		getFundingData        bool
		includeFundingEntries bool
		includePredictedRate  bool
	)
	if c.IsSet("includeorderdetails") {
		includeOrderDetails = c.Bool("includeorderdetails")
	} else if c.Args().Get(0) != "" {
		includeOrderDetails, err = strconv.ParseBool(c.Args().Get(0))
		if err != nil {
			return err
		}
	}

	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	} else if c.Args().Get(1) != "" {
		getFundingData, err = strconv.ParseBool(c.Args().Get(1))
		if err != nil {
			return err
		}
	}

	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	} else if c.Args().Get(2) != "" {
		includeFundingEntries, err = strconv.ParseBool(c.Args().Get(2))
		if err != nil {
			return err
		}
	}

	if c.IsSet("includepredictedrate") {
		includePredictedRate = c.Bool("includepredictedrate")
	} else if c.Args().Get(2) != "" {
		includePredictedRate, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	err = order.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAllManagedPositions(c.Context,
		&gctrpc.GetAllManagedPositionsRequest{
			IncludeFullOrderData:    includeOrderDetails,
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
			IncludePredictedRate:    includePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateral(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType                               string
		calculateOffline, includeBreakdown, includeZeroValues bool
		err                                                   error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}
	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("calculateoffline") {
		calculateOffline = c.Bool("calculateoffline")
	} else if c.Args().Get(2) != "" {
		calculateOffline, err = strconv.ParseBool(c.Args().Get(2))
		if err != nil {
			return err
		}
	}

	if c.IsSet("includebreakdown") {
		includeBreakdown = c.Bool("includebreakdown")
	} else if c.Args().Get(3) != "" {
		includeBreakdown, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	if c.IsSet("includezerovalues") {
		includeZeroValues = c.Bool("includezerovalues")
	} else if c.Args().Get(4) != "" {
		includeZeroValues, err = strconv.ParseBool(c.Args().Get(4))
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
	result, err := client.GetCollateral(c.Context,
		&gctrpc.GetCollateralRequest{
			Exchange:          exchangeName,
			Asset:             assetType,
			IncludeBreakdown:  includeBreakdown,
			CalculateOffline:  calculateOffline,
			IncludeZeroValues: includeZeroValues,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFundingRates(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType           string
		currencyPairs                     []string
		includePredicted, includePayments bool
		p                                 currency.Pair
		s, e                              time.Time
		err                               error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	if c.IsSet("pairs") {
		currencyPairs = c.StringSlice("pairs")
	} else {
		currencyPairs = strings.Split(c.Args().Get(2), ",")
	}
	for i := range currencyPairs {
		if !validPair(currencyPairs[i]) {
			return errInvalidPair
		}
		p, err = currency.NewPairDelimiter(currencyPairs[i], pairDelimiter)
		if err != nil {
			return err
		}
		currencyPairs[i] = p.String()
	}
	if !c.IsSet("start") {
		if c.Args().Get(3) != "" {
			startTime = c.Args().Get(3)
		}
	}
	if !c.IsSet("end") {
		if c.Args().Get(4) != "" {
			endTime = c.Args().Get(4)
		}
	}
	if c.IsSet("includepredicted") {
		includePredicted = c.Bool("includepredicted")
	} else if c.Args().Get(5) != "" {
		includePredicted, err = strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
	}
	if c.IsSet("includepayments") {
		includePayments = c.Bool("includepayments")
	} else if c.Args().Get(6) != "" {
		includePayments, err = strconv.ParseBool(c.Args().Get(6))
		if err != nil {
			return err
		}
	}
	s, err = time.ParseInLocation(common.SimpleTimeFormat, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(common.SimpleTimeFormat, endTime, time.Local)
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

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetFundingRates(c.Context,
		&gctrpc.GetFundingRatesRequest{
			Exchange:         exchangeName,
			Asset:            assetType,
			Pairs:            currencyPairs,
			StartDate:        s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:          e.Format(common.SimpleTimeFormatWithTimezone),
			IncludePredicted: includePredicted,
			IncludePayments:  includePayments,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateralMode(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType string
		err                     error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCollateralMode(c.Context,
		&gctrpc.GetCollateralModeRequest{
			Exchange: exchangeName,
			Asset:    assetType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setCollateralMode(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, collateralMode string
		err                                     error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("collateralmode") {
		collateralMode = c.String("collateralmode")
	} else {
		collateralMode = c.Args().Get(2)
	}

	if !collateral.IsValidCollateralModeString(collateralMode) {
		return fmt.Errorf("invalid collateral mode: %v", collateralMode)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetCollateralMode(c.Context,
		&gctrpc.SetCollateralModeRequest{
			Exchange:       exchangeName,
			Asset:          assetType,
			CollateralMode: collateralMode,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setLeverage(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType string
		leverage                                          float64
		err                                               error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	} else {
		marginType = c.Args().Get(3)
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	if c.IsSet("leverage") {
		leverage = c.Float64("leverage")
	} else {
		leverage, err = strconv.ParseFloat(c.Args().Get(4), 64)
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
	result, err := client.SetLeverage(c.Context,
		&gctrpc.SetLeverageRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
			Leverage:   leverage,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLeverage(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType string
		err                                               error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	} else {
		marginType = c.Args().Get(3)
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetLeverage(c.Context,
		&gctrpc.GetLeverageRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func changePositionMargin(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType, marginSide string
		originalAllocatedMargin, newAllocatedMargin                   float64
		err                                                           error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	} else {
		marginType = c.Args().Get(3)
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	if c.IsSet("originalallocatedmargin") {
		originalAllocatedMargin = c.Float64("originalallocatedmargin")
	} else {
		originalAllocatedMargin, err = strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("newallocatedmargin") {
		newAllocatedMargin = c.Float64("newallocatedmargin")
	} else {
		newAllocatedMargin, err = strconv.ParseFloat(c.Args().Get(5), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("marginside") {
		marginSide = c.String("marginside")
	} else {
		marginSide = c.Args().Get(6)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.ChangePositionMargin(c.Context,
		&gctrpc.ChangePositionMarginRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType:              marginType,
			OriginalAllocatedMargin: originalAllocatedMargin,
			NewAllocatedMargin:      newAllocatedMargin,
			MarginSide:              marginSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturesPositionSummary(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, underlyingPair string
		err                                                   error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("underlyingpair") {
		underlyingPair = c.String("underlyingpair")
	} else {
		underlyingPair = c.Args().Get(3)
	}
	var underlying currency.Pair
	if underlyingPair != "" {
		underlying, err = currency.NewPairDelimiter(underlyingPair, pairDelimiter)
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
	result, err := client.GetFuturesPositionsSummary(c.Context,
		&gctrpc.GetFuturesPositionsSummaryRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			UnderlyingPair: &gctrpc.CurrencyPair{
				Delimiter: underlying.Delimiter,
				Base:      underlying.Base.String(),
				Quote:     underlying.Quote.String(),
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturePositionOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, underlyingPair string
		respectOrderHistoryLimits, syncWithOrderManager       bool
		s, e                                                  time.Time
		err                                                   error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if !c.IsSet("start") {
		if c.Args().Get(3) != "" {
			startTime = c.Args().Get(3)
		}
	}
	s, err = time.ParseInLocation(common.SimpleTimeFormat, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}

	if !c.IsSet("end") {
		if c.Args().Get(4) != "" {
			endTime = c.Args().Get(4)
		}
	}
	e, err = time.ParseInLocation(common.SimpleTimeFormat, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if c.IsSet("respectorderhistorylimits") {
		respectOrderHistoryLimits = c.Bool("respectorderhistorylimits")
	} else if c.Args().Get(5) != "" {
		respectOrderHistoryLimits, err = strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
	}

	if c.IsSet("underlyingpair") {
		underlyingPair = c.String("underlyingpair")
	} else {
		underlyingPair = c.Args().Get(6)
	}
	var underlying currency.Pair
	if underlyingPair != "" {
		underlying, err = currency.NewPairDelimiter(underlyingPair, pairDelimiter)
		if err != nil {
			return err
		}
	}
	if c.IsSet("syncwithordermanager") {
		syncWithOrderManager = c.Bool("syncwithordermanager")
	} else if c.Args().Get(7) != "" {
		syncWithOrderManager, err = strconv.ParseBool(c.Args().Get(7))
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
	result, err := client.GetFuturesPositionsOrders(c.Context,
		&gctrpc.GetFuturesPositionsOrdersRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			StartDate: s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:   e.Format(common.SimpleTimeFormatWithTimezone),
			UnderlyingPair: &gctrpc.CurrencyPair{
				Delimiter: underlying.Delimiter,
				Base:      underlying.Base.String(),
				Quote:     underlying.Quote.String(),
			},
			SyncWithOrderManager:      syncWithOrderManager,
			RespectOrderHistoryLimits: respectOrderHistoryLimits,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setMarginType(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType string
		err                                               error
	)
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	} else {
		marginType = c.Args().Get(3)
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetMarginType(c.Context,
		&gctrpc.SetMarginTypeRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
