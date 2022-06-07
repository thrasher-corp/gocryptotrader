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

var getOpenPositionsCommand = &cli.Command{
	Name:      "getopenpositions",
	Usage:     "will retrieve all futures positions in a timeframe, then calculate PNL based on that. Note, the dates have an impact on PNL calculations, ensure your start date is not after a new position is opened",
	ArgsUsage: "<exchange> <asset> <start> <end>",
	Action:    getOpenPositions,
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
}

func getOpenPositions(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getopenpositions")
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

	s, err := time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.Parse(common.SimpleTimeFormat, endTime)
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
			Exchange:  exchangeName,
			Asset:     assetType,
			StartDate: negateLocalOffset(s),
			EndDate:   negateLocalOffset(e),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getFuturesPositionsCommand = &cli.Command{
	Name:      "getfuturespositions",
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
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "includes all orders that make up a position in the response",
		},
		&cli.BoolFlag{
			Name:    "overwrite",
			Aliases: []string{"o"},
			Usage:   "if true, will overwrite futures results for the provided exchange, asset, pair",
		},
		&cli.BoolFlag{
			Name:    "getfundingdata",
			Aliases: []string{"f"},
			Usage:   "if true, will return funding rate summary",
		},
		&cli.BoolFlag{
			Name:    "getpositionstats",
			Aliases: []string{"stats"},
			Usage:   "if true, will return extra stats on the position",
		},
	},
}

func getFuturesPositions(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, "getfuturesposition")
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

	var verbose bool
	if c.IsSet("verbose") {
		verbose = c.Bool("verbose")
	} else if c.Args().Get(7) != "" {
		verbose, err = strconv.ParseBool(c.Args().Get(7))
		if err != nil {
			return err
		}
	}

	var overwrite bool
	if c.IsSet("overwrite") {
		overwrite = c.Bool("overwrite")
	} else if c.Args().Get(8) != "" {
		overwrite, err = strconv.ParseBool(c.Args().Get(8))
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

	var getPositionsStats bool
	if c.IsSet("getpositionstats") {
		getPositionsStats = c.Bool("getpositionstats")
	} else if c.Args().Get(10) != "" {
		getPositionsStats, err = strconv.ParseBool(c.Args().Get(10))
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
			StartDate:        negateLocalOffset(s),
			EndDate:          negateLocalOffset(e),
			Status:           status,
			PositionLimit:    int64(limit),
			Verbose:          verbose,
			Overwrite:        overwrite,
			GetFundingData:   getFundingData,
			GetPositionStats: getPositionsStats,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCollateralCommand = &cli.Command{
	Name:      "getcollateral",
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
