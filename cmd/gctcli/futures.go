package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
			Name:      "getfuturespositions",
			Aliases:   []string{"positions", "p"},
			Usage:     "will retrieve all futures positions in a timeframe, then calculate PNL based on that. Note, the dates have an impact on PNL calculations, ensure your start date is not after a new position is opened",
			ArgsUsage: "<exchange> <asset> <pair> <start> <end> <limit> <status> <overwrite> <includeorderdetails> <getpositionstats> <getfundingdata> <includefundingentries> <includepredictedrate>",
			Action:    getFuturesPositions,
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
				&cli.IntFlag{
					Name:        "limit",
					Aliases:     []string{"l"},
					Usage:       "the number of positions (not orders) to return",
					Value:       86400,
					Destination: &limit,
				},
				&cli.StringFlag{
					Name:    "status",
					Aliases: []string{"s"},
					Usage:   "limit return to position statuses - open, closed, any",
					Value:   "ANY",
				},
				&cli.BoolFlag{
					Name:    "overwrite",
					Aliases: []string{"o"},
					Usage:   "if true, will overwrite futures results for the provided exchange, asset, pair",
				},
				&cli.BoolFlag{
					Name:    "includeorderdetails",
					Aliases: []string{"orders"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getpositionstats",
					Aliases: []string{"stats"},
					Usage:   "if true, will return extra stats on the position from the exchange",
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

func getFuturesPositions(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName          string
		assetType             string
		currencyPair          string
		err                   error
		includeOrderDetails   bool
		status                string
		overwrite             bool
		getFundingData        bool
		includeFundingEntries bool
		getPositionsStats     bool
		includePredicted      bool
		s, e                  time.Time
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
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
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
	if c.IsSet("limit") {
		limit = c.Int("limit")
	} else if c.Args().Get(5) != "" {
		var limit64 int64
		limit64, err = strconv.ParseInt(c.Args().Get(5), 10, 64)
		if err != nil {
			return err
		}
		limit = int(limit64)
	}
	if limit <= 0 {
		return errors.New("limit must be greater than 0")
	}

	if c.IsSet("status") {
		status = c.String("status")
	} else if c.Args().Get(6) != "" {
		status = c.Args().Get(6)
	}
	if !strings.EqualFold(status, "any") &&
		!strings.EqualFold(status, "open") &&
		!strings.EqualFold(status, "closed") &&
		status != "" {
		return errors.New("unrecognised status")
	}

	if c.IsSet("overwrite") {
		overwrite = c.Bool("overwrite")
	} else if c.Args().Get(7) != "" {
		overwrite, err = strconv.ParseBool(c.Args().Get(7))
		if err != nil {
			return err
		}
	}

	if c.IsSet("includeorderdetails") {
		includeOrderDetails = c.Bool("includeorderdetails")
	} else if c.Args().Get(8) != "" {
		includeOrderDetails, err = strconv.ParseBool(c.Args().Get(8))
		if err != nil {
			return err
		}
	}
	if c.IsSet("getpositionstats") {
		getPositionsStats = c.Bool("getpositionstats")
	} else if c.Args().Get(9) != "" {
		getPositionsStats, err = strconv.ParseBool(c.Args().Get(9))
		if err != nil {
			return err
		}
	}
	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	} else if c.Args().Get(10) != "" {
		getFundingData, err = strconv.ParseBool(c.Args().Get(10))
		if err != nil {
			return err
		}
	}
	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	} else if c.Args().Get(11) != "" {
		includeFundingEntries, err = strconv.ParseBool(c.Args().Get(11))
		if err != nil {
			return err
		}
	}
	if c.IsSet("includepredictedrate") {
		includePredicted = c.Bool("includepredictedrate")
	} else if c.Args().Get(12) != "" {
		includePredicted, err = strconv.ParseBool(c.Args().Get(12))
		if err != nil {
			return err
		}
	}
	err = order.CheckFundingRatePrerequisites(getFundingData, includePredicted, includeFundingEntries)
	if err != nil {
		return err
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
	result, err := client.GetFuturesPositions(c.Context,
		&gctrpc.GetFuturesPositionsRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			StartDate:               s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:                 e.Format(common.SimpleTimeFormatWithTimezone),
			Status:                  status,
			PositionLimit:           int64(limit),
			Overwrite:               overwrite,
			GetPositionStats:        getPositionsStats,
			IncludeFullOrderData:    includeOrderDetails,
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
			IncludePredictedRate:    includePredicted,
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
