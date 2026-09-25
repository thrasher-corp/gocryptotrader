package main

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:    "getmanagedposition",
			Aliases: []string{"managedposition", "mp"},
			Usage:   "retrieves an open position monitored by the order manager",
			Action:  getManagedPosition,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    "the currency pair of the position",
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
			Name:    "getallmanagedpositions",
			Aliases: []string{"managedpositions", "mps"},
			Usage:   "retrieves all open positions monitored by the order manager",
			Action:  getAllManagedPositions,
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
			Name:    "getcollateral",
			Aliases: []string{"collateral", "c"},
			Usage:   "returns total collateral for an exchange asset, with optional per currency breakdown",
			Action:  getCollateral,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
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
			Name:    "getfundingrates",
			Aliases: []string{"funding", "f"},
			Usage:   "returns funding rate data between two dates",
			Action:  getFundingRates,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    "currency pair",
				},
				&cli.StringFlag{
					Name:        startFlag,
					Aliases:     []string{"sd"},
					Usage:       "<start> rounded down to the nearest hour",
					Value:       time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
					Destination: &startTime,
				},
				&cli.StringFlag{
					Name:        endFlag,
					Aliases:     []string{"ed"},
					Usage:       "<end>",
					Value:       time.Now().Format(time.DateTime),
					Destination: &endTime,
				},
				&cli.StringFlag{
					Name:    "paymentcurrency",
					Aliases: []string{"pc"},
					Usage:   "optional - if you are paid in a currency that isn't easily inferred from the Pair, eg BTCUSD-PERP use this field",
				},
				&cli.BoolFlag{
					Name:    "includepredicted",
					Aliases: []string{"ip", "predicted"},
					Usage:   "optional - include the predicted next funding rate",
				},
				&cli.BoolFlag{
					Name:    "includepayments",
					Aliases: []string{"pay"},
					Usage:   "optional - include funding rate payments, must be authenticated",
				},
				&cli.BoolFlag{
					Name:    "respecthistorylimits",
					Aliases: []string{"respect", "r"},
					Usage:   "optional - if true, will change the starting date to the maximum allowable limit if start date exceeds it",
				},
			},
		},
		{
			Name:    "getlatestfundingrate",
			Aliases: []string{"latestrate", "lr", "r8"},
			Usage:   "returns the latest funding rate data",
			Action:  getLatestFundingRate,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    "currency pair",
				},
				&cli.BoolFlag{
					Name:    "includepredicted",
					Aliases: []string{"ip", "predicted"},
					Usage:   "optional - include the predicted next funding rate",
				},
			},
		},
		{
			Name:    "getcollateralmode",
			Aliases: []string{"gcm"},
			Usage:   "gets the collateral mode for an exchange asset",
			Action:  getCollateralMode,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
			},
		},
		{
			Name:    "setcollateralmode",
			Aliases: []string{"scm"},
			Usage:   "sets the collateral mode for an exchange asset",
			Action:  setCollateralMode,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     "collateralmode",
					Required: true,
					Aliases:  []string{"collateral", "cm", "c"},
					Usage:    "the collateral mode type, such as 'single', 'multi' or 'global'",
				},
			},
		},
		{
			Name:    "setleverage",
			Aliases: []string{"sl"},
			Usage:   "sets the initial leverage level for an exchange currency pair",
			Action:  setLeverage,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
				&cli.Float64Flag{
					Name:     "leverage",
					Required: true,
					Aliases:  []string{"l"},
					Usage:    "the level of leverage you want, increase it to lose your capital faster",
				},
				&cli.StringFlag{
					Name:    "orderside",
					Aliases: []string{sideFlag, "os", "o"},
					Usage:   "optional - some exchanges distinguish between order side",
				},
			},
		},
		{
			Name:    "getleverage",
			Aliases: []string{"gl"},
			Usage:   "gets the initial leverage level for an exchange currency pair",
			Action:  getLeverage,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
				},
				&cli.StringFlag{
					Name:    "margintype",
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
				&cli.StringFlag{
					Name:    "orderside",
					Aliases: []string{sideFlag, "os", "o"},
					Usage:   "optional - some exchanges distinguish between order side",
				},
			},
		},
		{
			Name:    "changepositionmargin",
			Aliases: []string{"cpm"},
			Usage:   "sets isolated margin levels for an existing position",
			Action:  changePositionMargin,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
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
					Name:     "newallocatedmargin",
					Required: true,
					Aliases:  []string{"nac"},
					Usage:    "the new allocated margin level you desire",
				},
				&cli.StringFlag{
					Name:    "marginside",
					Aliases: []string{sideFlag, "ms"},
					Usage:   "optional - the margin side, typically 'buy' or 'sell'",
				},
			},
		},
		{
			Name:    "getfuturespositionsummary",
			Aliases: []string{"summary", "fps"},
			Usage:   "return a summary of your futures position",
			Action:  getFuturesPositionSummary,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
				},
				&cli.StringFlag{
					Name:    "underlyingpair",
					Aliases: []string{"up"},
					Usage:   "optional - used to distinguish the underlying currency of a futures pair eg pair is BTCUSD-1337-C, the underlying pair could be BTC-USD, or if pair is LTCUSD-PERP the underlying pair could be LTC-USD",
				},
			},
		},
		{
			Name:    "getfuturepositionorders",
			Aliases: []string{"orders", "fpo"},
			Usage:   "return a slice of orders that make up your position",
			Action:  getFuturePositionOrders,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
				},
				&cli.StringFlag{
					Name:        startFlag,
					Aliases:     []string{"sd"},
					Usage:       "<start> rounded down to the nearest hour",
					Value:       time.Now().AddDate(0, 0, -7).Truncate(time.Hour).Format(time.DateTime),
					Destination: &startTime,
				},
				&cli.StringFlag{
					Name:        endFlag,
					Aliases:     []string{"ed"},
					Usage:       "<end>",
					Value:       time.Now().Format(time.DateTime),
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
			Name:    "setmargintype",
			Aliases: []string{"smt"},
			Usage:   "sets the margin type for a exchange asset pair",
			Action:  setMarginType,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    futuresExchangeUsage,
				},
				&cli.StringFlag{
					Name:     assetFlag,
					Required: true,
					Aliases:  []string{"a"},
					Usage:    futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:     pairFlag,
					Required: true,
					Aliases:  []string{"p"},
					Usage:    pairUsage,
				},
				&cli.StringFlag{
					Name:     "margintype",
					Required: true,
					Aliases:  []string{"margin", "mt", "m"},
					Usage:    "the margin type, such as 'isolated', 'multi' or 'cross'",
				},
			},
		},
		{
			Name:    "getopeninterest",
			Aliases: []string{"goi", "oi"},
			Usage:   "gets the open interest for provided exchange asset pair, if asset pair is not present, return all available if supported",
			Action:  getOpenInterest,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Aliases:  []string{"e"},
					Usage:    "the exchange to retrieve open interest from",
				},
				&cli.StringFlag{
					Name:    assetFlag,
					Aliases: []string{"a"},
					Usage:   "optional - the asset type of the currency pair, must be a futures type",
				},
				&cli.StringFlag{
					Name:    pairFlag,
					Aliases: []string{"p"},
					Usage:   "optional - the currency pair",
				},
			},
		},
	},
}

func getManagedPosition(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	var assetType string
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}
	err := isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	var currencyPair string
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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
	}

	var getFundingData bool
	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	}

	var includeFundingEntries bool
	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	}

	var includePredictedRate bool
	if c.IsSet("includepredictedrate") {
		includePredictedRate = c.Bool("includepredictedrate")
	}

	err = futures.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries)
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
	}

	if c.IsSet("getfundingdata") {
		getFundingData = c.Bool("getfundingdata")
	}

	if c.IsSet("includefundingentries") {
		includeFundingEntries = c.Bool("includefundingentries")
	}

	if c.IsSet("includepredictedrate") {
		includePredictedRate = c.Bool("includepredictedrate")
	}

	err = futures.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries)
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType                               string
		calculateOffline, includeBreakdown, includeZeroValues bool
		err                                                   error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}
	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("calculateoffline") {
		calculateOffline = c.Bool("calculateoffline")
	}

	if c.IsSet("includebreakdown") {
		includeBreakdown = c.Bool("includebreakdown")
	}

	if c.IsSet("includezerovalues") {
		includeZeroValues = c.Bool("includezerovalues")
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, paymentCurrency             string
		includePredicted, includePayments, respectFundingRateHistoryLimits bool
		p                                                                  currency.Pair
		s, e                                                               time.Time
		err                                                                error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}
	if !validPair(currencyPair) {
		return errInvalidPair
	}
	p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("paymentcurrency") {
		paymentCurrency = c.String("paymentcurrency")
	}

	if c.IsSet("includepredicted") {
		includePredicted = c.Bool("includepredicted")
	}
	if c.IsSet("includepayments") {
		includePayments = c.Bool("includepayments")
	}
	if c.IsSet("respecthistorylimits") {
		respectFundingRateHistoryLimits = c.Bool("respecthistorylimits")
	}

	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetFundingRates(c.Context,
		&gctrpc.GetFundingRatesRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			StartDate:            s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:              e.Format(common.SimpleTimeFormatWithTimezone),
			IncludePredicted:     includePredicted,
			IncludePayments:      includePayments,
			RespectHistoryLimits: respectFundingRateHistoryLimits,
			PaymentCurrency:      paymentCurrency,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLatestFundingRate(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair string
		includePredicted                      bool
		p                                     currency.Pair
		err                                   error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}
	if !validPair(currencyPair) {
		return errInvalidPair
	}
	p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("includepredicted") {
		includePredicted = c.Bool("includepredicted")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetLatestFundingRate(c.Context,
		&gctrpc.GetLatestFundingRateRequest{
			Exchange: exchangeName,
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludePredicted: includePredicted,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateralMode(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType string
		err                     error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, collateralMode string
		err                                     error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet("collateralmode") {
		collateralMode = c.String("collateralmode")
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType, orderSide string
		leverage                                                     float64
		err                                                          error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	if c.IsSet("leverage") {
		leverage = c.Float64("leverage")
	}

	if c.IsSet("orderside") {
		orderSide = c.String("orderside")
	}
	if orderSide != "" {
		_, err = order.StringToOrderSide(orderSide)
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
			OrderSide:  orderSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLeverage(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType, orderSide string
		err                                                          error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	if c.IsSet("orderside") {
		orderSide = c.String("orderside")
	}
	if orderSide != "" {
		_, err = order.StringToOrderSide(orderSide)
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
			OrderSide:  orderSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func changePositionMargin(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType, marginSide string
		originalAllocatedMargin, newAllocatedMargin                   float64
		err                                                           error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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
	}
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	if c.IsSet("originalallocatedmargin") {
		originalAllocatedMargin = c.Float64("originalallocatedmargin")
	}

	if c.IsSet("newallocatedmargin") {
		newAllocatedMargin = c.Float64("newallocatedmargin")
	}

	if c.IsSet("marginside") {
		marginSide = c.String("marginside")
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, underlyingPair string
		err                                                   error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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
				Base:      underlying.Base.Upper().String(),
				Quote:     underlying.Quote.Upper().String(),
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturePositionOrders(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, underlyingPair string
		respectOrderHistoryLimits, syncWithOrderManager       bool
		s, e                                                  time.Time
		err                                                   error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}

	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	if c.IsSet("respectorderhistorylimits") {
		respectOrderHistoryLimits = c.Bool("respectorderhistorylimits")
	}

	if c.IsSet("underlyingpair") {
		underlyingPair = c.String("underlyingpair")
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
				Base:      underlying.Base.Upper().String(),
				Quote:     underlying.Quote.Upper().String(),
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair, marginType string
		err                                               error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	err = isFuturesAsset(assetType)
	if err != nil {
		return err
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
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

func getOpenInterest(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		exchangeName, assetType, currencyPair string
		err                                   error
	)
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	if assetType != "" {
		err = isFuturesAsset(assetType)
		if err != nil {
			return err
		}
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}
	var pair currency.Pair
	if currencyPair != "" {
		if !validPair(currencyPair) {
			return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
		}
		pair, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
		if err != nil {
			return err
		}
	}

	data := make([]*gctrpc.OpenInterestDataRequest, 0, 1)
	if !pair.IsEmpty() {
		data = append(data, &gctrpc.OpenInterestDataRequest{
			Asset: assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
		})
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOpenInterest(c.Context,
		&gctrpc.GetOpenInterestRequest{
			Exchange: exchangeName,
			Data:     data,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
