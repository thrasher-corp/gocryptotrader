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
			ArgsUsage: "<exchange> <asset> <start> <end>",
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
					Aliases: []string{"orders", "o"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getfundingdata",
					Aliases: []string{"funding", "f"},
					Usage:   "if true, will return funding rate summary",
				},
				&cli.BoolFlag{
					Name:    "includefundingentries",
					Aliases: []string{"allfunding", "af"},
					Usage:   "if true, will return all funding rate entries - requires --getfundingdata",
				},
			},
		},
		{
			Name:      "getallmanagedpositions",
			Aliases:   []string{"managedpositions", "mps"},
			Usage:     "retrieves all open positions monitored by the order manager",
			ArgsUsage: "<includeorderdetails> <getfundingdata> <includefundingentries>",
			Action:    getAllManagedPositions,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "includeorderdetails",
					Aliases: []string{"orders", "o"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getfundingdata",
					Aliases: []string{"funding", "f"},
					Usage:   "if true, will return funding rate summary",
				},
				&cli.BoolFlag{
					Name:    "includefundingentries",
					Aliases: []string{"allfunding", "af"},
					Usage:   "if true, will return all funding rate entries - requires --getfundingdata",
				},
			},
		},

		{
			Name:      "getfuturespositions",
			Aliases:   []string{"positions", "p"},
			Usage:     "will retrieve all futures positions in a timeframe, then calculate PNL based on that. Note, the dates have an impact on PNL calculations, ensure your start date is not after a new position is opened",
			ArgsUsage: "<exchange> <pair> <asset> <start> <end> <limit> <status> <verbose> <overwrite>",
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
					Aliases: []string{"orders", "o"},
					Usage:   "includes all orders that make up a position in the response",
				},
				&cli.BoolFlag{
					Name:    "getfundingdata",
					Aliases: []string{"funding", "f"},
					Usage:   "if true, will return funding rate summary",
				},
				&cli.BoolFlag{
					Name:    "includefundingentries",
					Aliases: []string{"allfunding", "af"},
					Usage:   "if true, will return all funding rate entries - requires --getfundingdata",
				},
				&cli.BoolFlag{
					Name:    "getpositionstats",
					Aliases: []string{"stats"},
					Usage:   "if true, will return extra stats on the position from the exchange",
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
			Name:      "getfundingpayments",
			Aliases:   []string{"funding", "f"},
			Usage:     "returns funding rate data between two dates",
			ArgsUsage: "<exchange> <asset> <pair> <start> <end>",
			Action:    getFundingPayments,
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
			},
		},
	},
}

func getManagedPosition(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getmanagedposition")
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
	if !validAsset(assetType) {
		return errInvalidAsset
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
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getAllManagedPositions(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getallmanagedpositions")
	}

	var err error
	var includeOrderDetails bool
	if c.IsSet("includeorderdetails") {
		includeOrderDetails = c.Bool("includeorderdetails")
	} else if c.Args().Get(0) != "" {
		includeOrderDetails, err = strconv.ParseBool(c.Args().Get(0))
		if err != nil {
			return err
		}
	}

	var getFundingData bool
	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	} else if c.Args().Get(1) != "" {
		getFundingData, err = strconv.ParseBool(c.Args().Get(1))
		if err != nil {
			return err
		}
	}

	var includeFundingEntries bool
	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	} else if c.Args().Get(2) != "" {
		includeFundingEntries, err = strconv.ParseBool(c.Args().Get(2))
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
	result, err := client.GetAllManagedPositions(c.Context,
		&gctrpc.GetAllManagedPositionsRequest{
			IncludeFullOrderData:    includeOrderDetails,
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturesPositions(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getfuturespositions")
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

	if !validAsset(assetType) {
		return errInvalidAsset
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

	var status string
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

	var overwrite bool
	if c.IsSet("overwrite") {
		overwrite = c.Bool("overwrite")
	} else if c.Args().Get(7) != "" {
		overwrite, err = strconv.ParseBool(c.Args().Get(7))
		if err != nil {
			return err
		}
	}

	var includeOrderDetails bool
	if c.IsSet("includeorderdetails") {
		includeOrderDetails = c.Bool("includeorderdetails")
	} else if c.Args().Get(8) != "" {
		includeOrderDetails, err = strconv.ParseBool(c.Args().Get(8))
		if err != nil {
			return err
		}
	}

	var getFundingData bool
	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	} else if c.Args().Get(9) != "" {
		getFundingData, err = strconv.ParseBool(c.Args().Get(9))
		if err != nil {
			return err
		}
	}

	var includeFundingEntries bool
	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	} else if c.Args().Get(10) != "" {
		includeFundingEntries, err = strconv.ParseBool(c.Args().Get(10))
		if err != nil {
			return err
		}
	}

	var getPositionsStats bool
	if c.IsSet("getpositionstats") {
		getPositionsStats = c.Bool("getpositionstats")
	} else if c.Args().Get(11) != "" {
		getPositionsStats, err = strconv.ParseBool(c.Args().Get(11))
		if err != nil {
			return err
		}
	}

	var s, e time.Time
	s, err = time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.Parse(common.SimpleTimeFormat, endTime)
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
			StartDate:               negateLocalOffset(s),
			EndDate:                 negateLocalOffset(e),
			Status:                  status,
			PositionLimit:           int64(limit),
			Overwrite:               overwrite,
			GetPositionStats:        getPositionsStats,
			IncludeFullOrderData:    includeOrderDetails,
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateral(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
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

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var err error
	var calculateOffline bool
	if c.IsSet("calculateoffline") {
		calculateOffline = c.Bool("calculateoffline")
	} else if c.Args().Get(2) != "" {
		calculateOffline, err = strconv.ParseBool(c.Args().Get(2))
		if err != nil {
			return err
		}
	}

	var includeBreakdown bool
	if c.IsSet("includebreakdown") {
		includeBreakdown = c.Bool("includebreakdown")
	} else if c.Args().Get(3) != "" {
		includeBreakdown, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	var includeZeroValues bool
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

func getFundingPayments(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getfundingpayments")
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

	if !validAsset(assetType) {
		return errInvalidAsset
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

	var s, e time.Time
	s, err = time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.Parse(common.SimpleTimeFormat, endTime)
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
	result, err := client.GetFundingPayments(c.Context,
		&gctrpc.GetFundingPaymentsRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			StartDate: negateLocalOffset(s),
			EndDate:   negateLocalOffset(e),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
